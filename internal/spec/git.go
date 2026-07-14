package spec

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// VCS is the source-control seam used by Check. Methods take repo-root-
// relative paths. Production impl is gitVCS; tests inject fakes.
type VCS interface {
	// ListFiles returns git-tracked files under relDir ("" = whole repo),
	// repo-root-relative and slash-separated.
	ListFiles(relDir string) ([]string, error)
	// LastCommitTime returns the latest commit time touching relPath.
	// ok is false when the path has no commit history.
	LastCommitTime(relPath string) (t time.Time, ok bool, err error)
	// LastCommit returns the hash of the latest commit touching relPath; ok is
	// false when the path has no commit history. Used as the drift anchor (the
	// point since which post-sync changes are measured).
	LastCommit(relPath string) (hash string, ok bool, err error)
	// DiffNameStatus returns the net name-status changes to files under relDir
	// between fromCommit and HEAD (repo-root-relative paths, slash-separated).
	DiffNameStatus(fromCommit, relDir string) ([]NameStatus, error)
}

// NameStatus is one entry of a `git diff --name-status` range: the net change
// to a single path between two commits. Status is the code letter(s) with any
// score stripped ("A", "D", "M", "R", "C", "T", ...). Path is the new path
// (repo-root-relative, slash); OldPath is the prior path for R/C, else "".
type NameStatus struct {
	Status  string
	Path    string
	OldPath string
}

// RepoRoot returns the absolute repository root of the current working
// directory via `git rev-parse --show-toplevel`.
func RepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository (or git unavailable): %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

type gitVCS struct{ root string }

// newGitVCS builds a VCS backed by system git, rooted at root.
func newGitVCS(root string) VCS { return &gitVCS{root: root} }

func (g *gitVCS) ListFiles(relDir string) ([]string, error) {
	args := []string{"ls-files"}
	if relDir != "" {
		args = append(args, relDir)
	}
	out, err := runGit(g.root, args...)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, line := range strings.Split(string(out), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			files = append(files, filepath.ToSlash(line))
		}
	}
	return files, nil
}

func (g *gitVCS) LastCommitTime(relPath string) (time.Time, bool, error) {
	out, err := runGit(g.root, "log", "-1", "--format=%ct", "--", relPath)
	if err != nil {
		return time.Time{}, false, err
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return time.Time{}, false, nil
	}
	sec, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("parse %q: %w", s, err)
	}
	return time.Unix(sec, 0), true, nil
}

func (g *gitVCS) LastCommit(relPath string) (string, bool, error) {
	out, err := runGit(g.root, "log", "-1", "--format=%H", "--", relPath)
	if err != nil {
		return "", false, err
	}
	h := strings.TrimSpace(string(out))
	if h == "" {
		return "", false, nil
	}
	return h, true, nil
}

func (g *gitVCS) DiffNameStatus(fromCommit, relDir string) ([]NameStatus, error) {
	// --no-renames: a rename is reported as a delete + add, which classify maps
	// to the same drift (removed old + added new) as an explicit R would. This
	// keeps the status alphabet stable at A/D/M/T and the output deterministic.
	out, err := runGit(g.root, "diff", "--no-renames", "--name-status", fromCommit, "HEAD", "--", relDir)
	if err != nil {
		return nil, err
	}
	return parseNameStatus(string(out)), nil
}

// parseNameStatus parses `git diff --name-status` output into NameStatus
// entries. Each line is "<code>[\t<old>]\t<new>"; a rename/copy carries both
// paths, any other status a single path. Trailing CR is trimmed per line.
func parseNameStatus(s string) []NameStatus {
	var out []NameStatus
	for _, line := range strings.Split(s, "\n") {
		if line = strings.TrimRight(line, "\r"); line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			continue
		}
		ns := NameStatus{Status: stripDiffScore(fields[0])}
		if len(fields) >= 3 {
			ns.OldPath = filepath.ToSlash(fields[1])
			ns.Path = filepath.ToSlash(fields[2])
		} else {
			ns.Path = filepath.ToSlash(fields[1])
		}
		out = append(out, ns)
	}
	return out
}

// stripDiffScore strips the similarity/degradation score suffix from a
// name-status code: "R100" -> "R", "C90" -> "C", "M" -> "M".
func stripDiffScore(code string) string {
	i := 0
	for i < len(code) && (code[i] < '0' || code[i] > '9') {
		i++
	}
	return code[:i]
}

// runGit runs git in repoRoot and returns stdout.
func runGit(repoRoot string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			return nil, fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return out, nil
}
