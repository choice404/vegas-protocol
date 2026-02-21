package internal

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/choice404/vegas-protocol/vegas-tui/internal/settings"
	"github.com/choice404/vegas-protocol/vegas-tui/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"github.com/zmb3/spotify/v2"
	"golang.org/x/oauth2"
)

type radioState int

const (
	radioDisconnected radioState = iota
	radioAuthenticating
	radioConnected
	radioError
)

type radioTickMsg struct{}
type radioPollingMsg struct{}

type RadioModel struct {
	state radioState
	auth  *spotifyauth.Authenticator
	client *spotify.Client
	token  *oauth2.Token

	errorMsg   string
	retryCount int

	trackName  string
	artistName string
	albumName  string
	playing    bool
	progressMs int
	durationMs int
	deviceOK   bool

	eqBars []int

	appSettings *settings.Settings
}

func NewRadioModel(s *settings.Settings) RadioModel {
	return RadioModel{
		state:       radioDisconnected,
		eqBars:      make([]int, 16),
		appSettings: s,
	}
}

func (m RadioModel) Init() tea.Cmd {
	tok := settings.LoadSpotifyToken()
	if tok != nil {
		// We have a saved token -- send it as if auth just completed
		return func() tea.Msg {
			return spotifyAuthCompleteMsg{Token: tok}
		}
	}
	return nil
}

func radioTickCmd() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(t time.Time) tea.Msg {
		return radioTickMsg{}
	})
}

func radioPollingCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return radioPollingMsg{}
	})
}

func (m RadioModel) Update(msg tea.Msg) (RadioModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", " ":
			switch m.state {
			case radioDisconnected:
				return m.startAuth()
			case radioConnected:
				return m.togglePlayPause()
			case radioError:
				return m.retry()
			}
		case "n":
			if m.state == radioConnected && m.client != nil {
				return m, spotifyNextCmd(m.client)
			}
		case "p":
			if m.state == radioConnected && m.client != nil {
				return m, spotifyPrevCmd(m.client)
			}
		}

	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
			switch m.state {
			case radioDisconnected:
				if zone.Get("radio-connect").InBounds(msg) {
					return m.startAuth()
				}
			case radioConnected:
				if zone.Get("radio-play").InBounds(msg) {
					return m.togglePlayPause()
				}
				if zone.Get("radio-next").InBounds(msg) && m.client != nil {
					return m, spotifyNextCmd(m.client)
				}
				if zone.Get("radio-prev").InBounds(msg) && m.client != nil {
					return m, spotifyPrevCmd(m.client)
				}
			case radioError:
				if zone.Get("radio-retry").InBounds(msg) {
					return m.retry()
				}
			}
		}

	case spotifyAuthCompleteMsg:
		if msg.Err != nil {
			m.state = radioError
			m.errorMsg = msg.Err.Error()
			return m, nil
		}
		m.token = msg.Token
		m.auth = newSpotifyAuth()
		if m.auth == nil {
			m.state = radioError
			m.errorMsg = "SPOTIFY_ID and SPOTIFY_SECRET env vars not set"
			return m, nil
		}
		m.client = newSpotifyClient(m.auth, m.token)
		m.state = radioConnected
		m.retryCount = 0
		return m, tea.Batch(
			saveSpotifyTokenCmd(m.token),
			fetchSpotifyState(m.client),
			radioTickCmd(),
			radioPollingCmd(),
		)

	case spotifyStateMsg:
		if msg.Err != nil {
			m.retryCount++
			if m.retryCount >= 5 {
				m.state = radioError
				m.errorMsg = msg.Err.Error()
			}
			return m, nil
		}
		m.retryCount = 0
		m.trackName = msg.Track
		m.artistName = msg.Artist
		m.albumName = msg.Album
		m.playing = msg.Playing
		m.progressMs = msg.Progress
		m.durationMs = msg.Duration
		m.deviceOK = msg.DeviceOK
		// Re-save token in case oauth2 transport refreshed it
		if m.token != nil {
			return m, saveSpotifyTokenCmd(m.token)
		}
		return m, nil

	case spotifyActionMsg:
		if msg.Err != nil {
			m.errorMsg = msg.Err.Error()
		}
		// Always re-fetch state after an action
		if m.client != nil {
			return m, fetchSpotifyState(m.client)
		}
		return m, nil

	case radioPollingMsg:
		if m.state == radioConnected && m.client != nil {
			return m, tea.Batch(
				fetchSpotifyState(m.client),
				radioPollingCmd(),
			)
		}
		return m, nil

	case radioTickMsg:
		if m.state == radioConnected {
			if m.playing {
				for i := range m.eqBars {
					m.eqBars[i] = rand.Intn(8) + 1
				}
				m.progressMs += 150
				if m.progressMs > m.durationMs && m.durationMs > 0 {
					m.progressMs = m.durationMs
				}
			} else {
				for i := range m.eqBars {
					m.eqBars[i] = 0
				}
			}
			return m, radioTickCmd()
		}
		return m, nil

	case spotifyTokenSavedMsg:
		// Nothing to do
		return m, nil
	}

	return m, nil
}

func (m RadioModel) startAuth() (RadioModel, tea.Cmd) {
	m.auth = newSpotifyAuth()
	if m.auth == nil {
		m.state = radioError
		m.errorMsg = "SPOTIFY_ID and SPOTIFY_SECRET env vars not set"
		return m, nil
	}
	m.state = radioAuthenticating
	return m, spotifyAuthCmd(m.auth)
}

func (m RadioModel) togglePlayPause() (RadioModel, tea.Cmd) {
	if m.client == nil {
		return m, nil
	}
	if m.playing {
		return m, spotifyPauseCmd(m.client)
	}
	return m, spotifyPlayCmd(m.client)
}

func (m RadioModel) retry() (RadioModel, tea.Cmd) {
	m.retryCount = 0
	m.errorMsg = ""
	// If we have a token, try reconnecting directly
	if m.token != nil {
		m.auth = newSpotifyAuth()
		if m.auth == nil {
			m.state = radioError
			m.errorMsg = "SPOTIFY_ID and SPOTIFY_SECRET env vars not set"
			return m, nil
		}
		m.client = newSpotifyClient(m.auth, m.token)
		m.state = radioConnected
		return m, tea.Batch(
			fetchSpotifyState(m.client),
			radioTickCmd(),
			radioPollingCmd(),
		)
	}
	// Otherwise start fresh auth
	return m.startAuth()
}

func (m RadioModel) View(width, height int) string {
	var b strings.Builder

	b.WriteString(theme.TitleStyle.Render(" RADIO "))
	b.WriteString("\n\n")

	switch m.state {
	case radioDisconnected:
		b.WriteString(m.viewDisconnected())
	case radioAuthenticating:
		b.WriteString(m.viewAuthenticating())
	case radioConnected:
		b.WriteString(m.viewConnected(width))
	case radioError:
		b.WriteString(m.viewError())
	}

	return b.String()
}

func (m RadioModel) viewDisconnected() string {
	var b strings.Builder
	b.WriteString(theme.AmberStyle.Render("  SPOTIFY NOT CONNECTED"))
	b.WriteString("\n\n")
	b.WriteString("  ")
	b.WriteString(zone.Mark("radio-connect", theme.AmberStyle.Render("[ CONNECT SPOTIFY ]")))
	b.WriteString("\n\n")
	b.WriteString(theme.DimStyle.Render("  [Enter] Connect  |  Requires SPOTIFY_ID & SPOTIFY_SECRET env vars"))
	return b.String()
}

func (m RadioModel) viewAuthenticating() string {
	var b strings.Builder
	b.WriteString(theme.AmberStyle.Render("  AUTHENTICATING..."))
	b.WriteString("\n\n")
	b.WriteString(theme.DimStyle.Render("  Waiting for callback on http://127.0.0.1:8888/callback ..."))
	return b.String()
}

func (m RadioModel) viewConnected(width int) string {
	var b strings.Builder

	// Now playing info
	b.WriteString(theme.DimStyle.Render("  NOW PLAYING:"))
	b.WriteString("\n")

	trackDisplay := m.trackName
	if trackDisplay == "" {
		trackDisplay = "No track"
	}
	b.WriteString(fmt.Sprintf("  %s\n", theme.BoldStyle.Render(trackDisplay)))

	artistDisplay := m.artistName
	if artistDisplay == "" {
		artistDisplay = "Unknown Artist"
	}
	b.WriteString(fmt.Sprintf("  %s\n", theme.AmberStyle.Render(artistDisplay)))

	albumDisplay := m.albumName
	if albumDisplay == "" {
		albumDisplay = "Unknown Album"
	}
	b.WriteString(fmt.Sprintf("  %s\n", theme.DimStyle.Render(albumDisplay)))
	b.WriteString("\n")

	// Progress bar
	barWidth := width - 20
	if barWidth < 10 {
		barWidth = 10
	}
	if barWidth > 50 {
		barWidth = 50
	}

	progress := 0.0
	if m.durationMs > 0 {
		progress = float64(m.progressMs) / float64(m.durationMs)
		if progress > 1.0 {
			progress = 1.0
		}
	}
	filled := int(progress * float64(barWidth))
	empty := barWidth - filled

	progressBar := fmt.Sprintf("  %s %s%s %s",
		theme.BaseStyle.Render(formatDuration(m.progressMs)),
		theme.BaseStyle.Render(strings.Repeat("█", filled)),
		theme.DimStyle.Render(strings.Repeat("░", empty)),
		theme.DimStyle.Render(formatDuration(m.durationMs)),
	)
	b.WriteString(progressBar)
	b.WriteString("\n\n")

	// Equalizer
	b.WriteString("  ")
	for _, h := range m.eqBars {
		b.WriteString(theme.BaseStyle.Render(strings.Repeat("▮", h)))
		b.WriteString(theme.DimStyle.Render(strings.Repeat("▯", 8-h)))
		b.WriteString(" ")
	}
	b.WriteString("\n\n")

	// Controls
	playBtn := "[ PLAY ]"
	if m.playing {
		playBtn = "[ PAUSE ]"
	}
	controls := fmt.Sprintf("  %s  %s  %s",
		zone.Mark("radio-prev", theme.BaseStyle.Render("[ PREV ]")),
		zone.Mark("radio-play", theme.AmberStyle.Render(playBtn)),
		zone.Mark("radio-next", theme.BaseStyle.Render("[ NEXT ]")),
	)
	b.WriteString(controls)
	b.WriteString("\n\n")

	b.WriteString(theme.DimStyle.Render("  [Enter/Space] Play/Pause  [n] Next  [p] Prev"))

	return b.String()
}

func (m RadioModel) viewError() string {
	var b strings.Builder
	b.WriteString(theme.RedStyle.Render("  SIGNAL LOST"))
	b.WriteString("\n\n")
	b.WriteString(theme.RedStyle.Render(fmt.Sprintf("  ERROR: %s", m.errorMsg)))
	b.WriteString("\n\n")
	b.WriteString("  ")
	b.WriteString(zone.Mark("radio-retry", theme.AmberStyle.Render("[ RETRY ]")))
	b.WriteString("\n\n")
	b.WriteString(theme.DimStyle.Render("  [Enter] Retry"))
	return b.String()
}

func formatDuration(ms int) string {
	if ms < 0 {
		ms = 0
	}
	totalSec := ms / 1000
	min := totalSec / 60
	sec := totalSec % 60
	return fmt.Sprintf("%d:%02d", min, sec)
}
