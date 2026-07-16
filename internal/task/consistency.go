package task

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/antopolskiy/kanban-md/internal/config"
)

// ConsistencyReport summarizes consistency warnings and repairs.
type ConsistencyReport struct {
	Warnings []ReadWarning
	Repairs  []string
}

// EnsureConsistency checks tasks for ID/filename inconsistencies and repairs
// them in place. It also advances next_id to avoid future collisions.
func EnsureConsistency(cfg *config.Config) (ConsistencyReport, error) {
	tasks, warnings, err := ReadAllLenient(cfg.TasksPath())
	if err != nil {
		return ConsistencyReport{}, err
	}

	report := ConsistencyReport{Warnings: warnings}
	if len(tasks) == 0 {
		return report, nil
	}

	sortTasksByFile(tasks)
	usedIDs, nextID := initializeIDState(tasks, cfg.NextID)

	nextID, duplicateRepairs := repairDuplicateIDs(tasks, nextID, usedIDs)
	report.Repairs = append(report.Repairs, duplicateRepairs...)

	renameRepairs, err := repairFilenameMismatches(tasks, cfg.TasksPath())
	if err != nil {
		return ConsistencyReport{}, err
	}
	report.Repairs = append(report.Repairs, renameRepairs...)

	nextIDRepair, err := syncNextID(cfg, tasks, nextID)
	if err != nil {
		return ConsistencyReport{}, err
	}
	if nextIDRepair != "" {
		report.Repairs = append(report.Repairs, nextIDRepair)
	}

	// Fix file permissions to match claim state (silent, best-effort).
	repairFilePermissions(tasks, cfg.ClaimTimeoutDuration())

	return report, nil
}

func initializeIDState(tasks []*Task, cfgNextID int) (map[int]bool, int) {
	usedIDs := make(map[int]bool, len(tasks))
	maxID := 0
	for _, t := range tasks {
		usedIDs[t.ID] = true
		if t.ID > maxID {
			maxID = t.ID
		}
	}

	nextID := cfgNextID
	if nextID <= maxID {
		nextID = maxID + 1
	}
	return usedIDs, nextID
}

func repairDuplicateIDs(tasks []*Task, startNextID int, usedIDs map[int]bool) (int, []string) {
	nextID := startNextID
	var repairs []string
	duplicateIDs := duplicateTaskIDs(tasks)
	for _, id := range duplicateIDs {
		group := tasksWithID(tasks, id)
		sortTasksByFile(group)
		keeper := selectDuplicateKeeper(group, id)
		for i, t := range group {
			if i == keeper {
				continue
			}
			newID := nextAvailableID(nextID, usedIDs)
			nextID = newID + 1
			oldID := t.ID
			t.ID = newID
			t.Updated = time.Now()
			repairs = append(repairs,
				fmt.Sprintf("reassigned duplicate ID %d in %s to %d",
					oldID, filepath.Base(t.File), newID))
		}
	}
	return nextID, repairs
}

func repairFilenameMismatches(tasks []*Task, tasksDir string) ([]string, error) {
	occupied, err := occupiedTaskPaths(tasksDir)
	if err != nil {
		return nil, err
	}

	sortTasksByFile(tasks)
	var repairs []string
	for _, t := range tasks {
		if !needsFilenameRepair(t) {
			continue
		}

		oldPath := t.File
		oldName := filepath.Base(oldPath)
		targetPath := chooseTaskPath(tasksDir, t, oldPath, occupied)
		t.File = targetPath
		t.Updated = time.Now()

		if err := Write(targetPath, t); err != nil {
			return nil, fmt.Errorf("rewriting task file %s: %w", oldName, err)
		}
		if oldPath != targetPath {
			if err := os.Remove(oldPath); err != nil {
				return nil, fmt.Errorf("removing old task file %s: %w", oldName, err)
			}
		}

		delete(occupied, oldPath)
		occupied[targetPath] = true
		repairs = append(repairs,
			fmt.Sprintf("renamed %s to %s to match task ID %d",
				oldName, filepath.Base(targetPath), t.ID))
	}

	return repairs, nil
}

func needsFilenameRepair(t *Task) bool {
	fileID, idErr := ExtractIDFromFilename(filepath.Base(t.File))
	return idErr != nil || fileID != t.ID
}

func syncNextID(cfg *config.Config, tasks []*Task, candidateNext int) (string, error) {
	desiredNext := cfg.NextID
	maxID := maxTaskID(tasks)
	if desiredNext <= maxID {
		desiredNext = maxID + 1
	}
	if candidateNext > desiredNext {
		desiredNext = candidateNext
	}
	if desiredNext == cfg.NextID {
		return "", nil
	}

	oldNext := cfg.NextID
	cfg.NextID = desiredNext
	if err := cfg.Save(); err != nil {
		return "", fmt.Errorf("saving config: %w", err)
	}

	return fmt.Sprintf("updated next_id from %d to %d", oldNext, desiredNext), nil
}

func maxTaskID(tasks []*Task) int {
	maxID := 0
	for _, t := range tasks {
		if t.ID > maxID {
			maxID = t.ID
		}
	}
	return maxID
}

func duplicateTaskIDs(tasks []*Task) []int {
	counts := make(map[int]int, len(tasks))
	for _, t := range tasks {
		counts[t.ID]++
	}

	var ids []int
	for id, count := range counts {
		if count > 1 {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	return ids
}

func tasksWithID(tasks []*Task, id int) []*Task {
	group := make([]*Task, 0)
	for _, t := range tasks {
		if t.ID == id {
			group = append(group, t)
		}
	}
	return group
}

func selectDuplicateKeeper(group []*Task, id int) int {
	for i, t := range group {
		fileID, err := ExtractIDFromFilename(filepath.Base(t.File))
		if err == nil && fileID == id {
			return i
		}
	}
	return 0
}

func sortTasksByFile(tasks []*Task) {
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].File < tasks[j].File
	})
}

func nextAvailableID(start int, used map[int]bool) int {
	id := start
	for used[id] {
		id++
	}
	used[id] = true
	return id
}

func occupiedTaskPaths(tasksDir string) (map[string]bool, error) {
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]bool{}, nil
		}
		return nil, fmt.Errorf("reading tasks directory: %w", err)
	}

	occupied := make(map[string]bool, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != taskFileExt {
			continue
		}
		occupied[filepath.Join(tasksDir, entry.Name())] = true
	}
	return occupied, nil
}

func chooseTaskPath(tasksDir string, t *Task, currentPath string, occupied map[string]bool) string {
	slug := GenerateSlug(t.Title)
	if slug == "" {
		slug = "task"
	}
	base := GenerateFilename(t.ID, slug)
	candidate := filepath.Join(tasksDir, base)
	if candidate == currentPath || !occupied[candidate] {
		return candidate
	}
	for i := 1; ; i++ {
		candidate = filepath.Join(tasksDir, fmt.Sprintf("%03d-%s-%d%s", t.ID, slug, i, taskFileExt))
		if candidate == currentPath || !occupied[candidate] {
			return candidate
		}
	}
}

// repairFilePermissions ensures file permissions match claim state.
// Claimed (non-expired) tasks should be read-only (0o444).
// Unclaimed (or expired) tasks should be writable (0o600).
// Errors are silently ignored (best-effort for filesystems without Unix permissions).
func repairFilePermissions(tasks []*Task, timeout time.Duration) {
	for _, t := range tasks {
		if t.File == "" {
			continue
		}
		claimed := isActiveClaim(t, timeout)
		info, err := os.Stat(t.File)
		if err != nil {
			continue
		}
		isWritable := info.Mode().Perm()&0o200 != 0 //nolint:mnd // owner-write bit
		if claimed && isWritable {
			_ = os.Chmod(t.File, fileModeReadOnly)
		} else if !claimed && !isWritable {
			_ = os.Chmod(t.File, fileMode)
		}
	}
}

// isActiveClaim returns true if the task has a non-expired claim.
// Unlike CheckClaim, this is a pure read-only check that does not mutate the task.
func isActiveClaim(t *Task, timeout time.Duration) bool {
	if t.ClaimedBy == "" {
		return false
	}
	if timeout > 0 && t.ClaimedAt != nil && time.Since(*t.ClaimedAt) > timeout {
		return false
	}
	return true
}
