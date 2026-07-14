package spec

import (
	"reflect"
	"testing"
)

func TestClassify(t *testing.T) {
	goVocab := map[string]bool{".go": true}
	mdGoVocab := map[string]bool{".go": true, ".md": true}

	cases := []struct {
		name     string
		changes  []NameStatus
		declared map[string]bool
		vocab    map[string]bool
		added    []string
		removed  []string
		modified []string
	}{
		{
			name:     "added source type unlisted -> drift",
			changes:  []NameStatus{{Status: "A", Path: "new.go"}},
			declared: map[string]bool{"root.go": true},
			vocab:    goVocab,
			added:    []string{"new.go"},
		},
		{
			name:     "added resource type -> ignored",
			changes:  []NameStatus{{Status: "A", Path: "assets/logo.png"}},
			declared: map[string]bool{"root.go": true},
			vocab:    goVocab,
		},
		{
			name:     "added source type already declared -> ignored",
			changes:  []NameStatus{{Status: "A", Path: "root.go"}},
			declared: map[string]bool{"root.go": true},
			vocab:    goVocab,
		},
		{
			name:     "deleted but still declared -> drift",
			changes:  []NameStatus{{Status: "D", Path: "old.go"}},
			declared: map[string]bool{"old.go": true, "root.go": true},
			vocab:    goVocab,
			removed:  []string{"old.go"},
		},
		{
			name:     "deleted not declared -> ignored",
			changes:  []NameStatus{{Status: "D", Path: "data.json"}},
			declared: map[string]bool{"root.go": true},
			vocab:    goVocab,
		},
		{
			name:     "modified -> stale",
			changes:  []NameStatus{{Status: "M", Path: "root.go"}},
			declared: map[string]bool{"root.go": true},
			vocab:    goVocab,
			modified: []string{"root.go"},
		},
		{
			name:     "rename: old removed + new added",
			changes:  []NameStatus{{Status: "R", OldPath: "a.go", Path: "b.go"}},
			declared: map[string]bool{"a.go": true},
			vocab:    goVocab,
			added:    []string{"b.go"},
			removed:  []string{"a.go"},
		},
		{
			name:     "rename: old undeclared -> only new added",
			changes:  []NameStatus{{Status: "R", OldPath: "x.go", Path: "y.go"}},
			declared: map[string]bool{},
			vocab:    goVocab,
			added:    []string{"y.go"},
		},
		{
			name:     "rename to resource type -> old removed only",
			changes:  []NameStatus{{Status: "R", OldPath: "a.go", Path: "icon.png"}},
			declared: map[string]bool{"a.go": true},
			vocab:    goVocab,
			removed:  []string{"a.go"},
		},
		{
			name:     "copy -> new path added",
			changes:  []NameStatus{{Status: "C", OldPath: "root.go", Path: "copy.go"}},
			declared: map[string]bool{"root.go": true},
			vocab:    goVocab,
			added:    []string{"copy.go"},
		},
		{
			name:     "type change -> stale",
			changes:  []NameStatus{{Status: "T", Path: "file.go"}},
			declared: map[string]bool{"file.go": true},
			vocab:    goVocab,
			modified: []string{"file.go"},
		},
		{
			name:     "unknown status -> stale (conservative)",
			changes:  []NameStatus{{Status: "X", Path: "weird.go"}},
			declared: map[string]bool{"weird.go": true},
			vocab:    goVocab,
			modified: []string{"weird.go"},
		},
		{
			name: "mixed, results sorted",
			changes: []NameStatus{
				{Status: "M", Path: "z.go"},
				{Status: "A", Path: "b.go"},
				{Status: "A", Path: "a.go"},
				{Status: "D", Path: "d.go"},
			},
			declared: map[string]bool{"z.go": true, "d.go": true},
			vocab:    mdGoVocab,
			added:    []string{"a.go", "b.go"},
			removed:  []string{"d.go"},
			modified: []string{"z.go"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			added, removed, modified := classify(c.changes, c.declared, c.vocab)
			if !reflect.DeepEqual(added, c.added) {
				t.Errorf("added = %v, want %v", added, c.added)
			}
			if !reflect.DeepEqual(removed, c.removed) {
				t.Errorf("removed = %v, want %v", removed, c.removed)
			}
			if !reflect.DeepEqual(modified, c.modified) {
				t.Errorf("modified = %v, want %v", modified, c.modified)
			}
		})
	}
}
