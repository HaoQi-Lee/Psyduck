package cli

import (
	_ "embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// claudeMdSection is the psyduck instruction block appended to CLAUDE.md.
// It teaches Claude Code to auto-invoke /psy-syncing at the right moment.
// The content lives in claudemd/section.md so it can be edited as Markdown.
//
//go:embed claudemd/section.md
var claudeMdSection string

func newInitCmd() *cobra.Command {
	var installPlugins bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize psyduck in this repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return err
			}
			// install-plugins runs independently of init so it works on
			// already-initialized repos too.
			if installPlugins {
				if err := installPluginsToHome(cmd); err != nil {
					return err
				}
			}
			return runInit(cmd, dir)
		},
	}
	cmd.Flags().BoolVar(&installPlugins, "install-plugins", false, "install skills globally to ~/.claude/skills/<name>/SKILL.md (slash-command form)")
	return cmd
}

func runInit(cmd *cobra.Command, dir string) error {
	psyDir := filepath.Join(dir, ".psy")
	if _, err := os.Stat(psyDir); err == nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "init: already initialized at %s\n", psyDir)
		return fmt.Errorf("init: already initialized at %s", psyDir)
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(psyDir, 0o755); err != nil {
		return fmt.Errorf("init: mkdir: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "initialized .psy/ at %s\n", psyDir)

	// Append SPEC.md reading rule to CLAUDE.md so Claude Code loads specs
	// into context every session.
	if err := ensureClaudeMd(dir); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not update CLAUDE.md: %v\n", err)
	}

	return nil
}

// ensureClaudeMd appends the psyduck SPEC.md reading rule to the project's
// CLAUDE.md. If CLAUDE.md does not exist, it is created. If the rule is
// already present (detected by a sentinel marker), the file is left unchanged.
func ensureClaudeMd(dir string) error {
	path := filepath.Join(dir, "CLAUDE.md")
	const marker = "<!-- psyduck -->"

	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Already has the section — skip.
	if err == nil && strings.Contains(string(data), marker) {
		return nil
	}

	content := marker + claudeMdSection
	if err == nil {
		// Append to existing file.
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err := f.WriteString(content); err != nil {
			return err
		}
	} else {
		// Create new file.
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// installPluginsToHome resolves the user's home directory and delegates to
// installPluginsToDir.
func installPluginsToHome(cmd *cobra.Command) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("install-plugins: home dir: %w", err)
	}
	return installPluginsToDir(cmd, homeDir)
}

// installPluginsToDir writes embedded skill files to
// <baseDir>/.claude/skills/<name>/SKILL.md so Claude Code discovers them as
// /slash commands. Existing directories are skipped so user customisations are
// never overwritten.
func installPluginsToDir(cmd *cobra.Command, baseDir string) error {
	targetBase := filepath.Join(baseDir, ".claude", "skills")
	if err := os.MkdirAll(targetBase, 0o755); err != nil {
		return fmt.Errorf("install-plugins: mkdir: %w", err)
	}

	entries, err := fs.ReadDir(skillFiles, "skills")
	if err != nil {
		return fmt.Errorf("install-plugins: read embedded skills: %w", err)
	}

	installed := 0
	skipped := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		skillDir := filepath.Join(targetBase, name)
		if _, err := os.Stat(skillDir); err == nil {
			skipped++
			fmt.Fprintf(cmd.OutOrStdout(), "  skipped %s (already exists)\n", name)
			continue
		}

		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			return fmt.Errorf("install-plugins: mkdir %s: %w", name, err)
		}
		data, err := fs.ReadFile(skillFiles, "skills/"+e.Name())
		if err != nil {
			return fmt.Errorf("install-plugins: read %s: %w", name, err)
		}
		skillPath := filepath.Join(skillDir, "SKILL.md")
		if err := os.WriteFile(skillPath, data, 0o644); err != nil {
			return fmt.Errorf("install-plugins: write %s: %w", name, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", name)
		installed++
	}

	if installed > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "installed %d plugin(s) to %s\n", installed, targetBase)
	}
	if skipped > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "skipped %d (already exists)\n", skipped)
	}
	return nil
}
