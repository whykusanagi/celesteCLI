package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/gif"
	"image/png"
	"os"
)

// TerminalCapabilities holds what the current terminal supports
type TerminalCapabilities struct {
	SupportsITerm2    bool
	SupportsKitty     bool
	SupportsSixel     bool
	SupportsInlineImg bool
}

// DetectTerminalCapabilities checks what image protocols the terminal supports
func DetectTerminalCapabilities() TerminalCapabilities {
	caps := TerminalCapabilities{}

	// Check for iTerm2
	if os.Getenv("ITERM_SESSION_ID") != "" {
		caps.SupportsITerm2 = true
		caps.SupportsInlineImg = true
	}

	// Check for other modern terminals that support iTerm2 protocol
	termProgram := os.Getenv("TERM_PROGRAM")
	if termProgram == "WezTerm" || termProgram == "Ghostty" {
		caps.SupportsITerm2 = true
		caps.SupportsInlineImg = true
	}

	return caps
}

// DisplayGIFAnimated displays a GIF animation for a limited time
// Falls back to ASCII art if terminal doesn't support inline images
// Animates for 3 seconds then stops to allow user input
func DisplayGIFAnimated(gifData []byte, assetType AssetType) error {
	// Check terminal capabilities
	caps := DetectTerminalCapabilities()

	if !caps.SupportsInlineImg {
		// Fallback to ASCII art for unsupported terminals
		fmt.Fprintf(os.Stderr, "\n")
		displayASCIIArtRepresentation(assetType)
		fmt.Fprintf(os.Stderr, "\n")
		return nil
	}

	// Decode the GIF
	g, err := gif.DecodeAll(bytes.NewReader(gifData))
	if err != nil {
		// If GIF decode fails, fall back to ASCII
		fmt.Fprintf(os.Stderr, "\n")
		displayASCIIArtRepresentation(assetType)
		fmt.Fprintf(os.Stderr, "\n")
		return nil
	}

	if len(g.Image) == 0 {
		return fmt.Errorf("GIF has no frames")
	}

	// Display first frame once (static, no animation)
	// This shows the GIF without creating excessive output or cursor movement
	if err := displayFrameAsITerm2Image(g.Image[0]); err != nil {
		// Fallback to ASCII if frame display fails
		displayASCIIArtRepresentation(assetType)
	}

	return nil
}

// displayFrameAsITerm2Image sends a single frame to iTerm2 as inline image
func displayFrameAsITerm2Image(frame *image.Paletted) error {
	// Convert frame to RGBA for PNG encoding
	bounds := frame.Bounds()
	rgba := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgba.Set(x, y, frame.At(x, y))
		}
	}

	// Encode frame as PNG
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, rgba); err != nil {
		return err
	}

	// Encode PNG to base64
	b64 := base64.StdEncoding.EncodeToString(pngBuf.Bytes())

	// Send iTerm2 inline image escape sequence
	// Format: OSC 1337 ; File=<params> : <base64-data> ST
	// where OSC = \x1b] and ST = \x07 (BEL) or \x1b\\ (ST)
	width := bounds.Dx()
	height := bounds.Dy()

	fmt.Fprintf(os.Stderr,
		"\x1b]1337;File=name=celeste.png;size=%d;width=%dchar;height=%dchar;inline=1:%s\x07",
		pngBuf.Len(),
		width/32, // Approximate character width (32 pixels per char)
		height/32, // Approximate character height
		b64,
	)

	return nil
}

// DisplayGIFStatic displays just the first frame of a GIF (no animation)
// Useful for slower connections or when animation isn't desired
func DisplayGIFStatic(gifData []byte, assetType AssetType) error {
	caps := DetectTerminalCapabilities()

	if !caps.SupportsInlineImg {
		fmt.Fprintf(os.Stderr, "\n")
		displayASCIIArtRepresentation(assetType)
		fmt.Fprintf(os.Stderr, "\n")
		return nil
	}

	// Decode the GIF
	g, err := gif.DecodeAll(bytes.NewReader(gifData))
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n")
		displayASCIIArtRepresentation(assetType)
		fmt.Fprintf(os.Stderr, "\n")
		return nil
	}

	if len(g.Image) == 0 {
		fmt.Fprintf(os.Stderr, "\n")
		displayASCIIArtRepresentation(assetType)
		fmt.Fprintf(os.Stderr, "\n")
		return nil
	}

	// Display first frame only
	if err := displayFrameAsITerm2Image(g.Image[0]); err != nil {
		fmt.Fprintf(os.Stderr, "\n")
		displayASCIIArtRepresentation(assetType)
		fmt.Fprintf(os.Stderr, "\n")
		return nil
	}

	fmt.Fprintf(os.Stderr, "\n")
	return nil
}

// DisplayAssetOptimal chooses the best display method for the asset
// Uses animated GIF if terminal supports it, otherwise ASCII art
func DisplayAssetOptimal(assetType AssetType) error {
	var gifData []byte

	switch assetType {
	case PixelWink:
		gifData = pixelWinkGif
	case Kusanagi:
		gifData = kusanagiGif
	default:
		return fmt.Errorf("unknown asset type: %s", assetType)
	}

	if len(gifData) == 0 {
		return fmt.Errorf("asset data is empty")
	}

	// Try to display as animated GIF
	return DisplayGIFAnimated(gifData, assetType)
}

// TerminalInfo returns information about the current terminal
func TerminalInfo() string {
	caps := DetectTerminalCapabilities()

	info := "Terminal Capabilities:\n"
	if caps.SupportsITerm2 {
		info += "  ✓ iTerm2 inline images\n"
	} else {
		info += "  ✗ iTerm2 inline images\n"
	}

	if caps.SupportsKitty {
		info += "  ✓ Kitty graphics\n"
	} else {
		info += "  ✗ Kitty graphics\n"
	}

	if caps.SupportsSixel {
		info += "  ✓ Sixel graphics\n"
	} else {
		info += "  ✗ Sixel graphics\n"
	}

	if !caps.SupportsInlineImg {
		info += "\n  → Using ASCII art fallback\n"
	}

	return info
}
