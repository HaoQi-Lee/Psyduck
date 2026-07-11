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
