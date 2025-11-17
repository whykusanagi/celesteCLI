package main

import (
	"fmt"
	"image"
	"image/gif"
	"os"
	"time"
)

// PixelBlockAnimator handles GIF animation by rendering frames as colored pixel blocks
type PixelBlockAnimator struct {
	frames []*image.Paletted
	delays []time.Duration
	width  int
}

// LoadGIFAnimation loads a GIF file and prepares it for animation
func LoadGIFAnimation(gifPath string, displayWidth int) (*PixelBlockAnimator, error) {
	file, err := os.Open(gifPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open GIF: %v", err)
	}
	defer file.Close()

	g, err := gif.DecodeAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode GIF: %v", err)
	}

	delays := make([]time.Duration, len(g.Delay))
	for i, d := range g.Delay {
		if d <= 0 {
			d = 10 // Default 100ms if not specified
		}
		delays[i] = time.Duration(d) * 10 * time.Millisecond
	}

	return &PixelBlockAnimator{
		frames: g.Image,
		delays: delays,
		width:  displayWidth,
	}, nil
}

// RenderFrameToBlocks renders a single image frame using half-pixel characters for high detail
// Uses Unicode block elements to achieve 2x vertical resolution per character
func (a *PixelBlockAnimator) RenderFrameToBlocks(frameIdx int) string {
	if frameIdx >= len(a.frames) {
		return ""
	}

	frame := a.frames[frameIdx]
	bounds := frame.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	// Scale to desired width - we use 2x height detail with half-blocks
	// So we double the height calculation
	newHeight := int(float64(h)/float64(w)*float64(a.width)*0.5) + 1
	if newHeight < 1 {
		newHeight = 1
	}

	// Double height for half-pixel detail (top and bottom of each char)
	scaledHeight := newHeight * 2

	// Simple nearest-neighbor scaling
	xStep := float64(w) / float64(a.width)
	yStep := float64(h) / float64(scaledHeight)

	var output string

	// Process pixels in pairs (top/bottom for each character row)
	for y := 0; y < newHeight; y++ {
		row := ""
		for x := 0; x < a.width; x++ {
			srcX := int(float64(x) * xStep)
			if srcX >= w {
				srcX = w - 1
			}

			// Top pixel (upper half of character)
			topY := int(float64(y*2) * yStep)
			if topY >= h {
				topY = h - 1
			}
			topR, topG, topB, topA := frame.At(bounds.Min.X+srcX, bounds.Min.Y+topY).RGBA()
			topR8 := uint8(topR >> 8)
			topG8 := uint8(topG >> 8)
			topB8 := uint8(topB >> 8)
			topA8 := uint8(topA >> 8)

			// Bottom pixel (lower half of character)
			botY := int(float64(y*2+1) * yStep)
			if botY >= h {
				botY = h - 1
			}
			botR, botG, botB, botA := frame.At(bounds.Min.X+srcX, bounds.Min.Y+botY).RGBA()
			botR8 := uint8(botR >> 8)
			botG8 := uint8(botG >> 8)
			botB8 := uint8(botB >> 8)
			botA8 := uint8(botA >> 8)

			// Skip if both pixels are fully transparent
			if topA8 < 128 && botA8 < 128 {
				row += " "
				continue
			}

			// Determine which half-block character to use
			topTransparent := topA8 < 128
			botTransparent := botA8 < 128

			var char string
			var r8, g8, b8 uint8

			if topTransparent && !botTransparent {
				// Only bottom visible: lower half block
				char = "▄"
				r8, g8, b8 = botR8, botG8, botB8
			} else if !topTransparent && botTransparent {
				// Only top visible: upper half block
				char = "▀"
				r8, g8, b8 = topR8, topG8, topB8
			} else if !topTransparent && !botTransparent {
				// Both visible: need to blend or choose based on similarity
				// For now, use full block with average color
				if topR8 == botR8 && topG8 == botG8 && topB8 == botB8 {
					// Same color - use full block
					char = "█"
					r8, g8, b8 = topR8, topG8, topB8
				} else {
					// Different colors - use full block with top color (more visible)
					char = "█"
					r8, g8, b8 = topR8, topG8, topB8
				}
			} else {
				// Both transparent - shouldn't reach here
				row += " "
				continue
			}

			// Use true color (RGB) format: \033[38;2;R;G;Bm
			row += fmt.Sprintf("\033[38;2;%d;%d;%dm%s", r8, g8, b8, char)
		}
		row += "\033[0m\n" // Reset color at end of line
		output += row
	}

	return output
}

// PlayAnimation displays the animation in an infinite loop
// Returns when context is cancelled
func (a *PixelBlockAnimator) PlayAnimation(duration time.Duration) {
	if len(a.frames) == 0 {
		return
	}

	startTime := time.Now()
	frameIndex := 0

	for time.Since(startTime) < duration {
		// Clear screen and home cursor
		fmt.Fprint(os.Stderr, "\033[H\033[2J")

		// Render and display frame
		output := a.RenderFrameToBlocks(frameIndex)
		fmt.Fprint(os.Stderr, output)
		fmt.Fprint(os.Stderr, "\n")
		os.Stderr.Sync()

		// Sleep for this frame's delay
		time.Sleep(a.delays[frameIndex])

		// Move to next frame
		frameIndex = (frameIndex + 1) % len(a.frames)
	}
}

// PlayAnimationInfinite displays the animation in an infinite loop (no time limit)
func (a *PixelBlockAnimator) PlayAnimationInfinite() {
	if len(a.frames) == 0 {
		return
	}

	frameIndex := 0

	for {
		// Clear screen and home cursor: ESC H (home) + ESC 2J (clear)
		fmt.Fprint(os.Stderr, "\033[H\033[2J")

		// Render and display frame
		output := a.RenderFrameToBlocks(frameIndex)
		fmt.Fprint(os.Stderr, output)
		fmt.Fprint(os.Stderr, "\n")
		os.Stderr.Sync()

		// Sleep for this frame's delay
		time.Sleep(a.delays[frameIndex])

		// Move to next frame
		frameIndex = (frameIndex + 1) % len(a.frames)
	}
}
