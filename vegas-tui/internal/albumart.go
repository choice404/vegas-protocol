package internal

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"strings"
	"time"

	"github.com/choice404/vegas-protocol/vegas-tui/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	artWidth  = 20
	artHeight = 10
)

type albumArtMsg struct {
	Art      string
	ImageURL string
}

// fetchAlbumArtCmd downloads the image at imageURL, decodes it, and renders
// it as a block-character ASCII art string with green CRT colors.
func fetchAlbumArtCmd(imageURL string) tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(imageURL)
		if err != nil {
			return albumArtMsg{Art: placeholderArt(), ImageURL: imageURL}
		}
		defer resp.Body.Close()

		img, _, err := image.Decode(resp.Body)
		if err != nil {
			return albumArtMsg{Art: placeholderArt(), ImageURL: imageURL}
		}

		art := renderASCII(img)
		return albumArtMsg{Art: art, ImageURL: imageURL}
	}
}

// renderASCII resizes the image to artWidth x artHeight using nearest-neighbor
// sampling and maps pixel brightness to block characters with green CRT colors.
func renderASCII(img image.Image) string {
	bounds := img.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	dimGreenStyle := lipgloss.NewStyle().Foreground(theme.DimGreen)
	darkGreenStyle := lipgloss.NewStyle().Foreground(theme.DarkGreen)
	greenStyle := lipgloss.NewStyle().Foreground(theme.Green)

	var b strings.Builder
	for y := 0; y < artHeight; y++ {
		for x := 0; x < artWidth; x++ {
			// Map output pixel to source pixel (nearest-neighbor)
			srcX := bounds.Min.X + (x*srcW)/artWidth
			srcY := bounds.Min.Y + (y*srcH)/artHeight

			r, g, bl, _ := img.At(srcX, srcY).RGBA()
			// Brightness as 0.0-1.0 (using luminance formula)
			brightness := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(bl)) / 65535.0

			switch {
			case brightness < 0.2:
				b.WriteString(" ")
			case brightness < 0.4:
				b.WriteString(dimGreenStyle.Render("░"))
			case brightness < 0.7:
				b.WriteString(darkGreenStyle.Render("▒"))
			default:
				b.WriteString(greenStyle.Render("█"))
			}
		}
		if y < artHeight-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// placeholderArt returns a blank art block used when the image can't be fetched.
func placeholderArt() string {
	dimStyle := lipgloss.NewStyle().Foreground(theme.DimGreen)
	var lines []string
	for y := 0; y < artHeight; y++ {
		if y == artHeight/2 {
			pad := (artWidth - 8) / 2
			line := strings.Repeat(" ", pad) + dimStyle.Render("NO IMAGE") + strings.Repeat(" ", artWidth-pad-8)
			lines = append(lines, line)
		} else {
			lines = append(lines, strings.Repeat(" ", artWidth))
		}
	}
	return strings.Join(lines, "\n")
}

// albumArtBox wraps rendered album art in a border for the connected view.
func albumArtBox(art string) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.DimGreen).
		Padding(0, 1)
	return fmt.Sprintf("%s", style.Render(art))
}
