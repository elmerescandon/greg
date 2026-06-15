package ui

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/kitty"
	tea "charm.land/bubbletea/v2"
)

const (
	kittyImgHourly = 1
	kittyImgDaily  = 2
)

func IsKittySupported() bool {
	tp := os.Getenv("TERM_PROGRAM")
	return tp == "ghostty" || tp == "kitty" || os.Getenv("KITTY_WINDOW_ID") != ""
}

// kittyTransmitCmd returns a tea.Cmd that transmits a PNG image to the terminal
// using the Kitty graphics protocol, then creates a unicode virtual placement.
// The image is stored in terminal memory with the given ID.
func kittyTransmitCmd(img image.Image, id, rows, cols int) tea.Cmd {
	var buf bytes.Buffer
	png.Encode(&buf, img)
	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	var cmds []tea.Cmd

	// Delete any existing image with this ID first
	cmds = append(cmds, tea.Raw(
		ansi.KittyGraphics(nil, fmt.Sprintf("a=d,d=i,i=%d,q=2", id)),
	))

	// Transmit in chunks (max 4096 base64 chars per chunk)
	const chunkSize = 4096
	chunks := splitB64(b64, chunkSize)
	for i, chunk := range chunks {
		more := 1
		if i == len(chunks)-1 {
			more = 0
		}
		if i == 0 {
			// First chunk: transmit (store, don't display), PNG format
			cmds = append(cmds, tea.Raw(
				ansi.KittyGraphics(
					[]byte(chunk),
					fmt.Sprintf("a=t,f=%d,i=%d,q=2,m=%d", kitty.PNG, id, more),
				),
			))
		} else {
			cmds = append(cmds, tea.Raw(
				ansi.KittyGraphics([]byte(chunk), fmt.Sprintf("m=%d", more)),
			))
		}
	}

	// Create unicode virtual placement (U=1 enables unicode placeholders)
	cmds = append(cmds, tea.Raw(
		ansi.KittyGraphics(nil, fmt.Sprintf("a=p,U=1,i=%d,c=%d,r=%d,q=2", id, cols, rows)),
	))

	return tea.Batch(cmds...)
}

func splitB64(s string, size int) []string {
	var chunks []string
	for i := 0; i < len(s); i += size {
		end := i + size
		if end > len(s) {
			end = len(s)
		}
		chunks = append(chunks, s[i:end])
	}
	if len(chunks) == 0 {
		chunks = []string{""}
	}
	return chunks
}

// kittyPlaceholderLines generates lines of U+10EEEE placeholder characters
// that the terminal replaces with the image tile at (row, col).
// The foreground color encodes the image ID (RGB where R=0, G=0, B=id).
// Each cell contains the placeholder char + a combining diacritic for the row.
func kittyPlaceholderLines(id, rows, cols int) []string {
	// Foreground color encodes the image ID.
	// Kitty protocol: fg color = (r << 16 | g << 8 | b), image_id = that value.
	// For small IDs, use B channel.
	fgHex := fmt.Sprintf("#%06x", id)
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(fgHex))

	lines := make([]string, rows)
	for row := 0; row < rows; row++ {
		var sb strings.Builder
		rowDiacritic := kitty.Diacritic(row)
		cell := string(kitty.Placeholder) + string(rowDiacritic)
		for c := 0; c < cols; c++ {
			sb.WriteString(cell)
		}
		lines[row] = style.Render(sb.String())
	}
	return lines
}

// ── Chart drawing ─────────────────────────────────────────────────────────────

func hexRGBA(hex string) color.RGBA {
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}
	if len(hex) != 6 {
		return color.RGBA{A: 255}
	}
	var r, g, b uint8
	fmt.Sscanf(hex[0:2], "%x", &r)
	fmt.Sscanf(hex[2:4], "%x", &g)
	fmt.Sscanf(hex[4:6], "%x", &b)
	return color.RGBA{R: r, G: g, B: b, A: 255}
}

func lerpRGBA(a, b color.RGBA, t float64) color.RGBA {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return color.RGBA{
		R: uint8(float64(a.R)*(1-t) + float64(b.R)*t),
		G: uint8(float64(a.G)*(1-t) + float64(b.G)*t),
		B: uint8(float64(a.B)*(1-t) + float64(b.B)*t),
		A: 255,
	}
}

func fillRect(img *image.NRGBA, x0, y0, x1, y1 int, c color.RGBA) {
	nc := color.NRGBA{R: c.R, G: c.G, B: c.B, A: c.A}
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			img.SetNRGBA(x, y, nc)
		}
	}
}

func DrawHourlyChart(values [24]float64, imgW, imgH int, colorLow, colorHigh string) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, imgW, imgH))
	bg := hexRGBA(colorBg)
	lo := hexRGBA(colorLow)
	hi := hexRGBA(colorHigh)
	baseline := hexRGBA(colorBorder)

	fillRect(img, 0, 0, imgW, imgH, bg)
	baseH := 2
	fillRect(img, 0, imgH-baseH, imgW, imgH, baseline)

	maxV := 0.0
	for _, v := range values {
		if v > maxV {
			maxV = v
		}
	}
	if maxV == 0 {
		return img
	}

	n := 24
	gap := max(2, imgW/120)
	totalGap := gap * (n - 1)
	barW := (imgW - totalGap) / n
	if barW < 1 {
		barW = 1
	}
	chartH := imgH - baseH

	for i := 0; i < n; i++ {
		t := values[i] / maxV
		barH := int(math.Round(t * float64(chartH)))
		if barH < 1 && values[i] > 0 {
			barH = 1
		}
		x0 := i * (barW + gap)
		x1 := x0 + barW
		y1 := imgH - baseH
		y0 := y1 - barH

		for y := y0; y < y1; y++ {
			rowT := float64(y1-y) / float64(max(barH, 1))
			c := lerpRGBA(lo, hi, rowT*t+0.15)
			for x := x0; x < x1; x++ {
				img.SetNRGBA(x, y, color.NRGBA{R: c.R, G: c.G, B: c.B, A: 255})
			}
		}
	}

	return img
}

func DrawDailyChart(values []float64, imgW, imgH int, colorLow, colorHigh string) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, imgW, imgH))
	bg := hexRGBA(colorBg)
	lo := hexRGBA(colorLow)
	hi := hexRGBA(colorHigh)
	baseline := hexRGBA(colorBorder)

	fillRect(img, 0, 0, imgW, imgH, bg)
	baseH := 2
	fillRect(img, 0, imgH-baseH, imgW, imgH, baseline)

	n := len(values)
	if n == 0 {
		return img
	}
	maxV := 0.0
	for _, v := range values {
		if v > maxV {
			maxV = v
		}
	}
	if maxV == 0 {
		return img
	}

	gap := max(1, imgW/200)
	totalGap := gap * (n - 1)
	barW := (imgW - totalGap) / n
	if barW < 1 {
		barW = 1
	}
	chartH := imgH - baseH

	for i, v := range values {
		t := v / maxV
		barH := int(math.Round(t * float64(chartH)))
		if barH < 1 && v > 0 {
			barH = 1
		}
		x0 := i * (barW + gap)
		x1 := x0 + barW
		y1 := imgH - baseH
		y0 := y1 - barH

		for y := y0; y < y1; y++ {
			rowT := float64(y1-y) / float64(max(barH, 1))
			c := lerpRGBA(lo, hi, rowT*t+0.1)
			for x := x0; x < x1; x++ {
				img.SetNRGBA(x, y, color.NRGBA{R: c.R, G: c.G, B: c.B, A: 255})
			}
		}
	}

	return img
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
