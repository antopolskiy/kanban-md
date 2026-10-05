package tui

import (
	"testing"

	"github.com/antopolskiy/kanban-md/internal/task"
)

func TestSearchTitleOrIndividualTag(t *testing.T) {
	tests := []struct {
		name  string
		task  task.Task
		query string
		want  bool
	}{
		{name: "tag only", task: task.Task{Title: "Fix login", Tags: []string{"Backend"}}, query: "backend", want: true},
		{name: "title only", task: task.Task{Title: "Backend cleanup"}, query: "backend", want: true},
		{name: "tag substring and trim", task: task.Task{Tags: []string{"frontend", "Backend"}}, query: "  END  ", want: true},
		{name: "whole query", task: task.Task{Tags: []string{"red", "blue"}}, query: "red blue", want: false},
		{name: "no comma join", task: task.Task{Tags: []string{"red", "blue"}}, query: "red,blue", want: false},
		{name: "body excluded", task: task.Task{Body: "backend"}, query: "backend", want: false},
		{name: "literal punctuation", task: task.Task{Tags: []string{"api.v2"}}, query: "api.v2", want: true},
		{name: "not regex", task: task.Task{Title: "anything"}, query: ".*", want: false},
		{name: "nil tags", task: task.Task{Title: "login"}, query: "backend", want: false},
		{name: "empty", query: " ", want: true},
		{name: "ID prefix", task: task.Task{ID: 121}, query: "#12", want: true},
		{name: "ID exact rejects prefix", task: task.Task{ID: 121}, query: "#12 ", want: false},
		{name: "ID exact", task: task.Task{ID: 12}, query: "#12 ", want: true},
		{name: "bare hash", task: task.Task{ID: 99}, query: "#", want: true},
		{name: "ID ignores tags", task: task.Task{ID: 99, Tags: []string{"#12"}}, query: "#12", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchesFilter(&tt.task, tt.query); got != tt.want {
				t.Fatalf("matchesFilter(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}
