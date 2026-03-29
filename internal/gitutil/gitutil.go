package gitutil

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	gitLogTimeout      = 1000 * time.Millisecond
	staleLockThreshold = 5 * time.Minute
)

// StatusResult holds parsed git status --porcelain=v2 output.
type StatusResult struct {
	Branch    string
	Staged    int
	Unstaged  int
	Untracked int
}

// TotalChanges returns the sum of all change counts.
func (s StatusResult) TotalChanges() int {
	return s.Staged + s.Unstaged + s.Untracked
}

// ParsePorcelainV2 parses the output of `git status --porcelain=v2 --branch`.
func ParsePorcelainV2(output string) StatusResult {
	var result StatusResult
	for line := range strings.SplitSeq(output, "\n") {
		if strings.HasPrefix(line, "# branch.head ") {
			result.Branch = line[len("# branch.head "):]
			continue
		}
		if strings.HasPrefix(line, "1 ") || strings.HasPrefix(line, "2 ") {
			if len(line) >= 4 {
				if line[2] != '.' {
					result.Staged++
				}
				if line[3] != '.' {
					result.Unstaged++
				}
			}
			continue
		}
		if strings.HasPrefix(line, "? ") {
			result.Untracked++
			continue
		}
		if strings.HasPrefix(line, "u ") {
			// Unmerged — count as both staged and unstaged
			result.Staged++
			result.Unstaged++
		}
	}
	return result
}

// GetStatus runs `git -c color.status=never status --porcelain=v2 --branch` in the given dir
// and returns the parsed result. Returns zero StatusResult on error.
func GetStatus(dir string) (StatusResult, error) {
	cmd := exec.Command("git", "-c", "color.status=never", "status", "--porcelain=v2", "--branch")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return StatusResult{}, err
	}
	return ParsePorcelainV2(string(out)), nil
}

// GetLastCommit returns the subject line of the most recent commit in dir.
// Returns "" on error or timeout.
func GetLastCommit(dir string) string {
	ctx, cancel := context.WithTimeout(context.Background(), gitLogTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "log", "-1", "--format=%s", "--no-walk")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// HandleStaleGitLock checks for a .git/index.lock file in dir.
// If the lock is older than staleLockThreshold it is removed and false is returned.
// Returns true if a fresh lock exists (git operation in progress), false otherwise.
func HandleStaleGitLock(dir string) bool {
	lockPath := filepath.Join(dir, ".git", "index.lock")
	info, err := os.Stat(lockPath)
	if err != nil {
		return false // no lock
	}
	age := time.Since(info.ModTime())
	if age > staleLockThreshold {
		_ = os.Remove(lockPath)
		fmt.Fprintf(os.Stderr, "[session-start] Removed stale git lock (age: %ds)\n", int(age.Seconds()))
		return false
	}
	return true // lock exists and is fresh
}
