package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// ── File loading messages ────────────────────────────────────────────────────

type loadedFile struct {
	path  string
	lines []string
}

type filesLoadedMsg struct {
	files []loadedFile
	err   error
}

// ── Async file reading command ───────────────────────────────────────────────

func readFilesCmd(paths []string) tea.Cmd {
	return func() tea.Msg {
		var files []loadedFile
		for _, p := range paths {
			resolved, err := resolvePath(p)
			if err != nil {
				return filesLoadedMsg{err: fmt.Errorf("%s: %w", p, err)}
			}
			if err := validateFilePath(resolved); err != nil {
				return filesLoadedMsg{err: err}
			}
			lines, err := ReadMDLines(resolved)
			if err != nil {
				return filesLoadedMsg{err: err}
			}
			files = append(files, loadedFile{path: resolved, lines: lines})
		}
		return filesLoadedMsg{files: files}
	}
}

// ── Path helpers (cross-platform) ────────────────────────────────────────────

// resolvePath expands ~/ and returns an absolute, cleaned path.
func resolvePath(p string) (string, error) {
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		p = filepath.Join(home, p[2:])
	}
	abs, err := filepath.Abs(filepath.Clean(p))
	if err != nil {
		return "", fmt.Errorf("resolve absolute path: %w", err)
	}
	return abs, nil
}

// validateFilePath checks that the path exists and is a regular file.
func validateFilePath(p string) error {
	info, err := os.Stat(p)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", p)
		}
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("expected a file, got directory: %s", p)
	}
	return nil
}
