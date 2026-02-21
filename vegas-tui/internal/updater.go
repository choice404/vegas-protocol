package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Version is the current application version. Update this on each release.
const Version = "0.1.0"

// RestartAfterUpdate is set to true when an update completes successfully.
// main() checks this after the TUI exits to re-exec the new binary.
var RestartAfterUpdate bool

const (
	githubRepo    = "choice404/vegas-protocol"
	installPath   = "github.com/choice404/vegas-protocol/vegas-tui/cmd/vegas"
	githubAPIBase = "https://api.github.com/repos/" + githubRepo
)

// GitHubRelease holds the relevant fields from a GitHub releases API response.
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	HTMLURL string `json:"html_url"`
}

// Message types for the update flow.

type updateCheckMsg struct {
	release   *GitHubRelease
	hasUpdate bool
	err       error
}

type updateDoneMsg struct {
	version string
	err     error
}

// compareVersions compares two semver strings. Returns -1 if v1 < v2, 0 if equal, 1 if v1 > v2.
func compareVersions(v1, v2 string) int {
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := max(len(parts1), len(parts2))

	for len(parts1) < maxLen {
		parts1 = append(parts1, "0")
	}
	for len(parts2) < maxLen {
		parts2 = append(parts2, "0")
	}

	for i := range maxLen {
		var num1, num2 int
		fmt.Sscanf(parts1[i], "%d", &num1)
		fmt.Sscanf(parts2[i], "%d", &num2)

		if num1 < num2 {
			return -1
		} else if num1 > num2 {
			return 1
		}
	}

	return 0
}

// checkForUpdatesCmd returns a tea.Cmd that checks GitHub for a newer release.
func checkForUpdatesCmd() tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{Timeout: 10 * time.Second}

		req, err := http.NewRequest("GET", githubAPIBase+"/releases/latest", nil)
		if err != nil {
			return updateCheckMsg{err: err}
		}
		req.Header.Set("Accept", "application/vnd.github.v3+json")
		req.Header.Set("User-Agent", "vegas-protocol-tui")

		resp, err := client.Do(req)
		if err != nil {
			return updateCheckMsg{err: err}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return updateCheckMsg{err: fmt.Errorf("GitHub API returned status %d", resp.StatusCode)}
		}

		var release GitHubRelease
		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			return updateCheckMsg{err: err}
		}

		currentVersion := strings.TrimPrefix(Version, "v")
		latestVersion := strings.TrimPrefix(release.TagName, "v")
		hasUpdate := compareVersions(currentVersion, latestVersion) < 0

		return updateCheckMsg{release: &release, hasUpdate: hasUpdate}
	}
}

// doUpdateCmd returns a tea.Cmd that runs go install to update the binary.
func doUpdateCmd(version string) tea.Cmd {
	return func() tea.Msg {
		installCmd := exec.Command("go", "install", fmt.Sprintf("%s@%s", installPath, version))
		if err := installCmd.Run(); err != nil {
			return updateDoneMsg{version: version, err: fmt.Errorf("go install failed: %w", err)}
		}
		return updateDoneMsg{version: version}
	}
}
