package internal

import (
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// GitStatus holds parsed output from git status --porcelain=v2 --branch.
type GitStatus struct {
	IsRepo    bool
	Branch    string
	HasRemote bool
	Ahead     int
	Behind    int
	Staged    []string
	Unstaged  []string
	Untracked []string
	Clean     bool
}

// Message types returned by git tea.Cmd functions.

type gitStatusMsg struct {
	status GitStatus
	err    error
}

type gitStageMsg struct {
	err error
}

type gitCommitMsg struct {
	output string
	err    error
}

type gitPushMsg struct {
	output string
	err    error
}

type gitPullMsg struct {
	output string
	err    error
}

type gitFetchMsg struct {
	output string
	err    error
}

// fetchGitStatus returns a tea.Cmd that checks if dir is a git repo
// and parses the current status.
func fetchGitStatus(dir string) tea.Cmd {
	return func() tea.Msg {
		var status GitStatus

		// Check if inside a git work tree
		out, err := exec.Command("git", "-C", dir, "rev-parse", "--is-inside-work-tree").CombinedOutput()
		if err != nil || strings.TrimSpace(string(out)) != "true" {
			return gitStatusMsg{status: status}
		}
		status.IsRepo = true

		// Get porcelain v2 status with branch info
		out, err = exec.Command("git", "-C", dir, "status", "--porcelain=v2", "--branch").CombinedOutput()
		if err != nil {
			return gitStatusMsg{status: status, err: err}
		}

		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			// Branch headers
			if strings.HasPrefix(line, "# branch.head ") {
				status.Branch = strings.TrimPrefix(line, "# branch.head ")
			}
			if strings.HasPrefix(line, "# branch.upstream ") {
				status.HasRemote = true
			}
			if strings.HasPrefix(line, "# branch.ab ") {
				parts := strings.Fields(line)
				// Format: # branch.ab +N -M
				for _, p := range parts {
					if strings.HasPrefix(p, "+") {
						var n int
						for _, c := range p[1:] {
							n = n*10 + int(c-'0')
						}
						status.Ahead = n
					}
					if strings.HasPrefix(p, "-") {
						var n int
						for _, c := range p[1:] {
							n = n*10 + int(c-'0')
						}
						status.Behind = n
					}
				}
			}

			// Changed entries (porcelain v2)
			// "1 XY ..." ordinary changed entry
			// "2 XY ..." renamed/copied entry
			if strings.HasPrefix(line, "1 ") || strings.HasPrefix(line, "2 ") {
				parts := strings.Fields(line)
				if len(parts) >= 9 {
					xy := parts[1]
					// Get the file path (last field, may contain spaces for renamed)
					filePath := parts[len(parts)-1]

					x := xy[0] // index (staged) status
					y := xy[1] // worktree status

					if x != '.' {
						status.Staged = append(status.Staged, string(x)+" "+filePath)
					}
					if y != '.' {
						status.Unstaged = append(status.Unstaged, string(y)+" "+filePath)
					}
				}
			}

			// Untracked
			if strings.HasPrefix(line, "? ") {
				filePath := strings.TrimPrefix(line, "? ")
				status.Untracked = append(status.Untracked, filePath)
			}
		}

		status.Clean = len(status.Staged) == 0 && len(status.Unstaged) == 0 && len(status.Untracked) == 0
		return gitStatusMsg{status: status}
	}
}

// gitStageAll runs git add -A in the given directory.
func gitStageAll(dir string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("git", "-C", dir, "add", "-A")
		_, err := cmd.CombinedOutput()
		return gitStageMsg{err: err}
	}
}

// gitCommit runs git commit -m in the given directory.
func gitCommit(dir, message string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("git", "-C", dir, "commit", "-m", message)
		out, err := cmd.CombinedOutput()
		return gitCommitMsg{output: strings.TrimSpace(string(out)), err: err}
	}
}

// gitPush runs git push with GIT_TERMINAL_PROMPT=0 to prevent auth hangs.
func gitPush(dir string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("git", "-C", dir, "push")
		cmd.Env = append(cmd.Environ(), "GIT_TERMINAL_PROMPT=0")
		out, err := cmd.CombinedOutput()
		return gitPushMsg{output: strings.TrimSpace(string(out)), err: err}
	}
}

// gitPull runs git pull with GIT_TERMINAL_PROMPT=0 to prevent auth hangs.
func gitPull(dir string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("git", "-C", dir, "pull")
		cmd.Env = append(cmd.Environ(), "GIT_TERMINAL_PROMPT=0")
		out, err := cmd.CombinedOutput()
		return gitPullMsg{output: strings.TrimSpace(string(out)), err: err}
	}
}

// gitFetch runs git fetch with GIT_TERMINAL_PROMPT=0 to prevent auth hangs.
func gitFetch(dir string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("git", "-C", dir, "fetch")
		cmd.Env = append(cmd.Environ(), "GIT_TERMINAL_PROMPT=0")
		out, err := cmd.CombinedOutput()
		return gitFetchMsg{output: strings.TrimSpace(string(out)), err: err}
	}
}
