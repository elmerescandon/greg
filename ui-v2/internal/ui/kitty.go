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
)

// IsKittySupported returns true if the terminal supports the Kitty graphics protocol.
func IsKittySupported() bool {
	tp := os.Getenv("TERM_PROGRAM")
	return tp == "ghostty" || tp == "kitty" || os.Getenv("KITTY_WINDOW_ID") != ""
}

// kittyInlineImage encodes img as a Kitty protocol inline image occupying
// exactly `rows` terminal rows and `cols` terminal columns.
// The terminal scales the image to fit those cells.
func kittyInlineImage(img image.Image, rows, cols int) string {
	var buf bytes.Buffer
	png.Encode(&buf, img)
	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	const chunkSize = 4096
	var sb strings.Builder
	for i := 0; i < len(b64); i += chunkSize {
		end := i + chunkSize
		if end > len(b64) {
			end = len(b64)
		}
		chunk := b64[i:end]
		more := 1
		if end >= len(b64) {
			more = 0
		}
		if i == 0 {
			// First chunk: transmit+display, PNG format, quiet, specify cell dims
			sb.WriteString(fmt.Sprintf("\x1b_Ga=T,f=100,q=2,r=%d,c=%d,m=%d;%s\x1b\\",
				rows, cols, more, chunk))
		} else {
			sb.WriteString(fmt.Sprintf("\x1b_Gm=%d;%s\x1b\\", more, chunk))
		}
	}
	return sb.String()
}

// kittyImageBlock returns the view-string lines for a kitty image.
//
// Strategy: put the kitty sequence on the LAST line of the block, prefixed
// with a cursor-up of (rows-1) rows. bubbletea renders lines top-to-bottom
// using absolute cursor positioning:
//   1. Lines 0..rows-2 (empty): bubbletea clears those rows.
//   2. Line rows-1: bubbletea writes "\033[rows-1A" + kittySeq.
//      - cursor-up moves to the top of the block.
//      - kittySeq renders the image (rows tall), advancing cursor to rows-1+1.
//      - bubbletea's internal cursor also lands at the same row. ✓
//
// On subsequent ticks, none of these lines change → bubbletea skips them
// entirely → the image persists without flicker.
func kittyImageBlock(img image.Image, rows, cols int) []string {
	seq := kittyInlineImage(img, rows, cols)
	// cursor-up by (rows-1) so the image renders starting at the top of the block
	upMove := fmt.Sprintf("\033[%dA", rows-1)

	lines := make([]string, rows)
	for i := 0; i < rows-1; i++ {
		lines[i] = "" // cleared by bubbletea before the image renders
	}
	lines[rows-1] = upMove + seq
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

// DrawHourlyChart renders a bar chart for 24-hour session distribution.
// colorLow/colorHigh are CSS hex strings for the bar gradient.
func DrawHourlyChart(values [24]float64, imgW, imgH int, colorLow, colorHigh string) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, imgW, imgH))
	bg := hexRGBA(colorBg)
	lo := hexRGBA(colorLow)
	hi := hexRGBA(colorHigh)
	baseline := hexRGBA(colorBorder)

	// Background
	fillRect(img, 0, 0, imgW, imgH, bg)

	// Baseline at bottom
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
		x0 := i*(barW+gap)
		x1 := x0 + barW
		y1 := imgH - baseH
		y0 := y1 - barH

		for y := y0; y < y1; y++ {
			// Gradient: dim at bottom, bright at top
			rowT := float64(y1-y) / float64(max(barH, 1))
			c := lerpRGBA(lo, hi, rowT*t+0.15)
			for x := x0; x < x1; x++ {
				img.SetNRGBA(x, y, color.NRGBA{R: c.R, G: c.G, B: c.B, A: 255})
			}
		}
	}

	return img
}

// DrawDailyChart renders a bar chart for daily activity (sessions or cost).
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
		x0 := i*(barW+gap)
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
