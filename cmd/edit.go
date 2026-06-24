package cmd

import (
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/clierr"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/date"
	"github.com/antopolskiy/kanban-md/internal/output"
	"github.com/antopolskiy/kanban-md/internal/task"
)

var editCmd = &cobra.Command{
	Use:   "edit ID[,ID,...]",
	Short: "Edit a task",
	Long: `Modifies fields of an existing task. Only specified fields are changed.
Multiple IDs can be provided as a comma-separated list.`,
	Args: cobra.ExactArgs(1),
	RunE: runEdit,
}

func init() {
	editCmd.Flags().String("title", "", "new title")
	editCmd.Flags().String("status", "", "new status")
	editCmd.Flags().String("priority", "", "new priority")
	editCmd.Flags().String("assignee", "", "new assignee")
	editCmd.Flags().StringSlice("add-tag", nil, "add tags")
	editCmd.Flags().StringSlice("remove-tag", nil, "remove tags")
	editCmd.Flags().String("due", "", "new due date (YYYY-MM-DD)")
	editCmd.Flags().Bool("clear-due", false, "clear due date")
	editCmd.Flags().String("estimate", "", "new time estimate")
	editCmd.Flags().String("body", "", "new body text (replaces entire body)")
	editCmd.Flags().StringP("append-body", "a", "", "append text to task body")
	editCmd.Flags().BoolP("timestamp", "t", false, "prefix a timestamp line when appending")
	editCmd.Flags().String("started", "", "set started date (YYYY-MM-DD)")
	editCmd.Flags().Bool("clear-started", false, "clear started timestamp")
	editCmd.Flags().String("completed", "", "set completed date (YYYY-MM-DD)")
	editCmd.Flags().Bool("clear-completed", false, "clear completed timestamp")
	editCmd.Flags().Int("parent", 0, "set parent task ID")
	editCmd.Flags().Bool("clear-parent", false, "clear parent")
	editCmd.Flags().IntSlice("add-dep", nil, "add dependency task IDs")
	editCmd.Flags().IntSlice("remove-dep", nil, "remove dependency task IDs")
	editCmd.Flags().String("block", "", "mark task as blocked with reason")
	editCmd.Flags().Bool("unblock", false, "clear blocked state")
	editCmd.Flags().String("claim", "", "claim task for an agent")
	editCmd.Flags().Bool("release", false, "release claim on task")
	editCmd.Flags().String("class", "", "set class of service")
	rootCmd.AddCommand(editCmd)
}

func runEdit(cmd *cobra.Command, args []string) error {
	ids, err := parseIDs(args[0])
	if err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Single ID: preserve exact current behavior.
	if len(ids) == 1 {
		return editSingleTask(cfg, ids[0], cmd)
	}

	// Batch mode.
	return runBatch(ids, func(id int) error {
		_, _, err := executeEdit(cfg, id, cmd)
		return err
	})
}

// editSingleTask handles a single task edit with full output.
func editSingleTask(cfg *config.Config, id int, cmd *cobra.Command) error {
	t, newPath, err := executeEdit(cfg, id, cmd)
	if err != nil {
		return err
	}

	if outputFormat() == output.FormatJSON {
		t.File = newPath
		return output.JSON(os.Stdout, t)
	}

	output.Messagef(os.Stdout, "Updated task #%d: %s", t.ID, t.Title)
	return nil
}

// executeEdit performs the core edit via board.Edit.
// Returns the modified task and its new file path.
func executeEdit(cfg *config.Config, id int, cmd *cobra.Command) (*task.Task, string, error) {
	claimant, _ := cmd.Flags().GetString("claim")
	release, _ := cmd.Flags().GetBool("release")

	result, err := board.Edit(cfg, id, claimant, release,
		func(t *task.Task) (bool, error) {
			return applyEditChanges(cmd, t, cfg, claimant, release)
		}, time.Now())
	if err != nil {
		return nil, "", err
	}

	return result.Task, result.NewPath, nil
}

// validateEditClaim checks claim ownership and require_claim before allowing edits.
// The --release flag bypasses claim checks since its intent is to release a claim.
func validateEditClaim(cfg *config.Config, t *task.Task, cmd *cobra.Command) (string, bool, error) { //nolint:unparam // claimant used by tests
	claimant, _ := cmd.Flags().GetString("claim")
	release, _ := cmd.Flags().GetBool("release")
	if !release {
		if err := checkClaim(t, claimant, cfg.ClaimTimeoutDuration()); err != nil {
			return "", false, err
		}
	}
	if cfg.StatusRequiresClaim(t.Status) && claimant == "" && !release {
		return "", false, task.ValidateClaimRequired(t.Status)
	}
	return claimant, release, nil
}

// validateEditPost runs post-edit validations: deps, require_claim for new status, WIP limits.
func validateEditPost(cfg *config.Config, t *task.Task, oldStatus, claimant string) error { //nolint:unparam // oldStatus varies in tests
	if err := validateDeps(cfg, t); err != nil {
		return err
	}
	if t.Status != oldStatus && cfg.StatusRequiresClaim(t.Status) && claimant == "" {
		return task.ValidateClaimRequired(t.Status)
	}
	if t.Status != oldStatus {
		if t.Class != "" && len(cfg.Classes) > 0 {
			return enforceWIPLimitForClass(cfg, t, oldStatus, t.Status)
		}
		return enforceWIPLimit(cfg, oldStatus, t.Status)
	}
	return nil
}

// logEditActivity logs the edit and any block/unblock/claim/release transitions.
func logEditActivity(cfg *config.Config, t *task.Task, wasBlocked bool, wasClaimedBy string) {
	board.LogMutation(cfg.Dir(), "edit", t.ID, t.Title)
	if !wasBlocked && t.Blocked {
		board.LogMutation(cfg.Dir(), "block", t.ID, t.BlockReason)
	}
	if wasBlocked && !t.Blocked {
		board.LogMutation(cfg.Dir(), "unblock", t.ID, t.Title)
	}
	if wasClaimedBy == "" && t.ClaimedBy != "" {
		board.LogMutation(cfg.Dir(), "claim", t.ID, t.ClaimedBy)
	}
	if wasClaimedBy != "" && t.ClaimedBy == "" {
		board.LogMutation(cfg.Dir(), "release", t.ID, wasClaimedBy)
	}
}

// applyEditChanges applies field edits and claim/release flags.
func applyEditChanges(cmd *cobra.Command, t *task.Task, cfg *config.Config, claimant string, release bool) (bool, error) {
	changed, err := applyEditFlags(cmd, t, cfg)
	if err != nil {
		return false, err
	}
	if c, claimErr := applyClaimFlags(cmd, t, claimant, release); claimErr != nil {
		return false, claimErr
	} else if c {
		changed = true
	}
	return changed, nil
}

// applyClaimFlags handles --claim and --release flags.
func applyClaimFlags(cmd *cobra.Command, t *task.Task, claimant string, release bool) (bool, error) {
	claimSet := cmd.Flags().Changed("claim")
	if claimSet && release {
		return false, clierr.New(clierr.StatusConflict, "cannot use --claim and --release together")
	}
	if claimSet {
		if claimant == "" {
			return false, clierr.New(clierr.InvalidInput, "claim name is required (use --claim NAME)")
		}
		now := time.Now()
		t.ClaimedBy = claimant
		t.ClaimedAt = &now
		return true, nil
	}
	if release {
		t.ClaimedBy = ""
		t.ClaimedAt = nil
		return true, nil
	}
	return false, nil
}

func applyEditFlags(cmd *cobra.Command, t *task.Task, cfg *config.Config) (bool, error) {
	changed, err := applySimpleEditFlags(cmd, t, cfg)
	if err != nil {
		return false, err
	}

	// Apply grouped flag helpers, each returning (bool, error).
	for _, fn := range []func(*cobra.Command, *task.Task) (bool, error){
		applyTimestampFlags,
		applyTagDueFlags,
		applyDepFlags,
		applyBlockFlags,
	} {
		c, fnErr := fn(cmd, t)
		if fnErr != nil {
			return false, fnErr
		}
		if c {
			changed = true
		}
	}

	return changed, nil
}

func applySimpleEditFlags(cmd *cobra.Command, t *task.Task, cfg *config.Config) (bool, error) {
	changed := false

	if v, _ := cmd.Flags().GetString("title"); v != "" {
		t.Title = v
		changed = true
	}
	if v, _ := cmd.Flags().GetString("status"); v != "" {
		if err := task.ValidateStatus(v, cfg.StatusNames()); err != nil {
			return false, err
		}
		t.Status = v
		changed = true
	}
	if v, _ := cmd.Flags().GetString("priority"); v != "" {
		if err := task.ValidatePriority(v, cfg.Priorities); err != nil {
			return false, err
		}
		t.Priority = v
		changed = true
	}
	if v, _ := cmd.Flags().GetString("assignee"); v != "" {
		t.Assignee = v
		changed = true
	}
	if v, _ := cmd.Flags().GetString("estimate"); v != "" {
		t.Estimate = v
		changed = true
	}
	bodySet := cmd.Flags().Changed("body")
	appendSet := cmd.Flags().Changed("append-body")
	if bodySet && appendSet {
		return false, clierr.New(clierr.StatusConflict, "cannot use --body and --append-body together")
	}
	if bodySet {
		v, _ := cmd.Flags().GetString("body")
		t.Body = v
		changed = true
	}
	if appendSet {
		v, _ := cmd.Flags().GetString("append-body")
		ts, _ := cmd.Flags().GetBool("timestamp")
		t.Body = board.AppendBody(t.Body, v, ts)
		changed = true
	}
	if v, _ := cmd.Flags().GetString("class"); v != "" {
		if err := task.ValidateClass(v, cfg.ClassNames()); err != nil {
			return false, err
		}
		t.Class = v
		changed = true
	}

	return changed, nil
}

func applyTimestampFlags(cmd *cobra.Command, t *task.Task) (bool, error) {
	changed := false

	startedSet := cmd.Flags().Changed("started")
	clearStarted, _ := cmd.Flags().GetBool("clear-started")
	completedSet := cmd.Flags().Changed("completed")
	clearCompleted, _ := cmd.Flags().GetBool("clear-completed")

	if startedSet && clearStarted {
		return false, clierr.New(clierr.StatusConflict, "cannot use --started and --clear-started together")
	}
	if completedSet && clearCompleted {
		return false, clierr.New(clierr.StatusConflict, "cannot use --completed and --clear-completed together")
	}

	if startedSet {
		v, _ := cmd.Flags().GetString("started")
		d, err := date.Parse(v)
		if err != nil {
			return false, task.ValidateDate("started", v, err)
		}
		ts := d.Time
		t.Started = &ts
		changed = true
	}
	if clearStarted {
		t.Started = nil
		changed = true
	}
	if completedSet {
		v, _ := cmd.Flags().GetString("completed")
		d, err := date.Parse(v)
		if err != nil {
			return false, task.ValidateDate("completed", v, err)
		}
		ts := d.Time
		t.Completed = &ts
		changed = true
	}
	if clearCompleted {
		t.Completed = nil
		changed = true
	}

	return changed, nil
}

func applyTagDueFlags(cmd *cobra.Command, t *task.Task) (bool, error) {
	changed := false

	if v, _ := cmd.Flags().GetStringSlice("add-tag"); len(v) > 0 {
		t.Tags = appendUnique(t.Tags, v...)
		changed = true
	}
	if v, _ := cmd.Flags().GetStringSlice("remove-tag"); len(v) > 0 {
		t.Tags = removeAll(t.Tags, v...)
		changed = true
	}
	if v, _ := cmd.Flags().GetString("due"); v != "" {
		d, err := date.Parse(v)
		if err != nil {
			return false, task.FormatDueDate(v, err)
		}
		t.Due = &d
		changed = true
	}
	if clearDue, _ := cmd.Flags().GetBool("clear-due"); clearDue {
		t.Due = nil
		changed = true
	}

	return changed, nil
}

func applyDepFlags(cmd *cobra.Command, t *task.Task) (bool, error) {
	changed := false

	parentSet := cmd.Flags().Changed("parent")
	clearParent, _ := cmd.Flags().GetBool("clear-parent")

	if parentSet && clearParent {
		return false, clierr.New(clierr.StatusConflict, "cannot use --parent and --clear-parent together")
	}
	if parentSet {
		v, _ := cmd.Flags().GetInt("parent")
		t.Parent = &v
		changed = true
	}
	if clearParent {
		t.Parent = nil
		changed = true
	}

	if v, _ := cmd.Flags().GetIntSlice("add-dep"); len(v) > 0 {
		t.DependsOn = appendUniqueInts(t.DependsOn, v...)
		changed = true
	}
	if v, _ := cmd.Flags().GetIntSlice("remove-dep"); len(v) > 0 {
		t.DependsOn = removeInts(t.DependsOn, v...)
		changed = true
	}

	return changed, nil
}

func appendUniqueInts(slice []int, items ...int) []int {
	seen := make(map[int]bool, len(slice))
	for _, v := range slice {
		seen[v] = true
	}
	for _, item := range items {
		if !seen[item] {
			slice = append(slice, item)
			seen[item] = true
		}
	}
	return slice
}

func removeInts(slice []int, items ...int) []int {
	remove := make(map[int]bool, len(items))
	for _, item := range items {
		remove[item] = true
	}
	result := make([]int, 0, len(slice))
	for _, v := range slice {
		if !remove[v] {
			result = append(result, v)
		}
	}
	return result
}

func applyBlockFlags(cmd *cobra.Command, t *task.Task) (bool, error) {
	blockReason, _ := cmd.Flags().GetString("block")
	unblock, _ := cmd.Flags().GetBool("unblock")
	blockSet := cmd.Flags().Changed("block")

	if blockSet && unblock {
		return false, clierr.New(clierr.StatusConflict, "cannot use --block and --unblock together")
	}
	if blockSet {
		if blockReason == "" {
			return false, clierr.New(clierr.InvalidInput, "block reason is required (use --block REASON)")
		}
		t.Blocked = true
		t.BlockReason = blockReason
		return true, nil
	}
	if unblock {
		t.Blocked = false
		t.BlockReason = ""
		return true, nil
	}
	return false, nil
}

func appendUnique(slice []string, items ...string) []string {
	seen := make(map[string]bool, len(slice))
	for _, s := range slice {
		seen[s] = true
	}
	for _, item := range items {
		if !seen[item] {
			slice = append(slice, item)
			seen[item] = true
		}
	}
	return slice
}

func removeAll(slice []string, items ...string) []string {
	remove := make(map[string]bool, len(items))
	for _, item := range items {
		remove[item] = true
	}
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if !remove[s] {
			result = append(result, s)
		}
	}
	return result
}

