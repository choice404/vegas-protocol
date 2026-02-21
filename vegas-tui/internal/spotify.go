package internal

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/choice404/vegas-protocol/vegas-tui/internal/settings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
)

const spotifyRedirectURI = "http://127.0.0.1:8888/callback"

// --- Message types ---

type spotifyStateMsg struct {
	Track    string
	Artist   string
	Album    string
	Playing  bool
	Progress int // ms
	Duration int // ms
	DeviceOK bool
	Err      error
}

type spotifyActionMsg struct {
	Action string // "play", "pause", "next", "prev"
	Err    error
}

type spotifyAuthCompleteMsg struct {
	Token *oauth2.Token
	Err   error
}

type spotifyTokenSavedMsg struct{}

// --- Auth & Client ---

func newSpotifyAuth() *spotifyauth.Authenticator {
	id := os.Getenv("SPOTIFY_ID")
	secret := os.Getenv("SPOTIFY_SECRET")
	if id == "" || secret == "" {
		return nil
	}
	auth := spotifyauth.New(
		spotifyauth.WithClientID(id),
		spotifyauth.WithClientSecret(secret),
		spotifyauth.WithRedirectURL(spotifyRedirectURI),
		spotifyauth.WithScopes(
			spotifyauth.ScopeUserReadPlaybackState,
			spotifyauth.ScopeUserModifyPlaybackState,
			spotifyauth.ScopeUserReadCurrentlyPlaying,
		),
	)
	return auth
}

func newSpotifyClient(auth *spotifyauth.Authenticator, tok *oauth2.Token) *spotify.Client {
	httpClient := auth.Client(context.Background(), tok)
	client := spotify.New(httpClient)
	return client
}

// --- tea.Cmd functions ---

func spotifyAuthCmd(auth *spotifyauth.Authenticator) tea.Cmd {
	return func() tea.Msg {
		state := "vegas-protocol-auth"
		url := auth.AuthURL(state)

		// Try to open browser
		_ = exec.Command("xdg-open", url).Start()

		// Start temporary HTTP server for callback
		tokenCh := make(chan *oauth2.Token, 1)
		errCh := make(chan error, 1)

		mux := http.NewServeMux()
		mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
			tok, err := auth.Token(r.Context(), state, r)
			if err != nil {
				errCh <- fmt.Errorf("auth token exchange: %w", err)
				fmt.Fprintf(w, "Error: %v. You can close this tab.", err)
				return
			}
			tokenCh <- tok
			fmt.Fprint(w, "Authenticated! You can close this tab and return to the terminal.")
		})

		srv := &http.Server{
			Addr:    ":8888",
			Handler: mux,
		}

		go func() {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				errCh <- err
			}
		}()

		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = srv.Shutdown(ctx)
		}()

		select {
		case tok := <-tokenCh:
			return spotifyAuthCompleteMsg{Token: tok}
		case err := <-errCh:
			return spotifyAuthCompleteMsg{Err: err}
		case <-time.After(2 * time.Minute):
			return spotifyAuthCompleteMsg{Err: fmt.Errorf("authentication timed out (2 min)")}
		}
	}
}

func fetchSpotifyState(client *spotify.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		state, err := client.PlayerState(ctx)
		if err != nil {
			return spotifyStateMsg{Err: err}
		}

		msg := spotifyStateMsg{
			Playing:  state.Playing,
			Progress: int(state.Progress),
			DeviceOK: state.Device.ID != "",
		}

		if state.Item != nil {
			msg.Track = state.Item.Name
			msg.Duration = int(state.Item.Duration)

			artists := ""
			for i, a := range state.Item.Artists {
				if i > 0 {
					artists += ", "
				}
				artists += a.Name
			}
			msg.Artist = artists
			msg.Album = state.Item.Album.Name
		}

		return msg
	}
}

func spotifyPlayCmd(client *spotify.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := client.Play(ctx)
		return spotifyActionMsg{Action: "play", Err: err}
	}
}

func spotifyPauseCmd(client *spotify.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := client.Pause(ctx)
		return spotifyActionMsg{Action: "pause", Err: err}
	}
}

func spotifyNextCmd(client *spotify.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := client.Next(ctx)
		return spotifyActionMsg{Action: "next", Err: err}
	}
}

func spotifyPrevCmd(client *spotify.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := client.Previous(ctx)
		return spotifyActionMsg{Action: "prev", Err: err}
	}
}

func saveSpotifyTokenCmd(tok *oauth2.Token) tea.Cmd {
	return func() tea.Msg {
		_ = settings.SaveSpotifyToken(tok)
		return spotifyTokenSavedMsg{}
	}
}
