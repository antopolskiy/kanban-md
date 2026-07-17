//go:build !windows

package e2e_test

import (
	"strings"
	"testing"
)

func TestE2E_TUIStatusLinePrioritizesHelpInNormalAndMouseModes(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "normal",
			want: "4 cards | ? help | create",
		},
		{
			name: "mouse",
			args: []string{"--mouse"},
			want: "4 cards | ? help | mouse | create",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kanbanDir := initBoardWithSeededTasks(t)
			session := startTUIProcessWithOptions(t, kanbanDir, tuiProcessOptions{
				args: tt.args,
				cols: 80,
				rows: 24,
			})
			session.waitForOutput(tt.want)

			output := session.output()
			for _, unwanted := range []string{"Test Board |", "n/p:status", "c:create"} {
				if strings.Contains(output, unwanted) {
					t.Fatalf("status output contains removed text %q: %q", unwanted, output)
				}
			}

			session.pressKeys("q")
			session.waitForExit()
		})
	}
}
