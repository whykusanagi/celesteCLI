package main

import (
	_ "embed"
	"encoding/base64"
	"fmt"
	"os"
	"time"
)

// Embed pixel art assets directly into the binary for portability
//
//go:embed assets/pixel_wink_full.gif
var pixelWinkGif []byte

//go:embed assets/kusanagi_4x.gif
var kusanagiGif []byte

// AssetType defines different asset types available
type AssetType string

const (
	PixelWink  AssetType = "pixel_wink"
	Kusanagi   AssetType = "kusanagi"
)

// DisplayPixelArt displays a pixel art asset in the terminal
// This is a simple implementation - can be enhanced with sixel support for better terminals
func DisplayPixelArt(assetType AssetType) error {
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

	// For now, display a simple text representation
	// In the future, this could use sixel or iTerm2 inline images
	displayASCIIArtRepresentation(assetType)
	return nil
}

// displayASCIIArtRepresentation shows an ASCII art version of the pixel art with animation effect
func displayASCIIArtRepresentation(assetType AssetType) {
	switch assetType {
	case PixelWink:
		// Animated wink - eye blink effect (3 frames)
		frames := []string{
			"(╯°□°)╯︵ ┻━┻", // Normal
			"(╯ °  °)╯︵ ┻━┻", // Wink/closed
			"(╯°□°)╯︵ ┻━┻", // Back to normal
		}

		// Show animation
		for _, frame := range frames {
			fmt.Fprintf(os.Stderr, "    ┌─────────────────┐\n")
			fmt.Fprintf(os.Stderr, "    │   ✨ Celeste ✨  │\n")
			fmt.Fprintf(os.Stderr, "    │  %s  │\n", frame)
			fmt.Fprintf(os.Stderr, "    └─────────────────┘\n")
			time.Sleep(400 * time.Millisecond)
			// Clear previous frame by moving cursor up
			fmt.Fprintf(os.Stderr, "\033[4A\033[K\033[K\033[K\033[K")
		}

		// Show final frame
		fmt.Fprintf(os.Stderr, "    ┌─────────────────┐\n")
		fmt.Fprintf(os.Stderr, "    │   ✨ Celeste ✨  │\n")
		fmt.Fprintf(os.Stderr, "    │  (╯°□°)╯︵ ┻━┻  │\n")
		fmt.Fprintf(os.Stderr, "    └─────────────────┘\n")

	case Kusanagi:
		// Kusanagi/abyss themed art with pulsing corruption
		frames := []string{
			"c0rrupt3d...",
			"c0rrupt1ng...",
			"c0rrupt3d...",
		}
		japaneseFades := []string{
			"深淵への堕落...",
			"深淵...消失...",
			"深淵への堕落...",
		}

		for i, frame := range frames {
			fmt.Fprintf(os.Stderr, "    ┌─────────────────────┐\n")
			fmt.Fprintf(os.Stderr, "    │  🌑 Kusanagi Abyss 🌑│\n")
			fmt.Fprintf(os.Stderr, "    │    %s    │\n", frame)
			fmt.Fprintf(os.Stderr, "    │   %s   │\n", japaneseFades[i])
			fmt.Fprintf(os.Stderr, "    └─────────────────────┘\n")
			time.Sleep(350 * time.Millisecond)
			// Clear previous frame
			fmt.Fprintf(os.Stderr, "\033[4A\033[K\033[K\033[K\033[K")
		}

		// Show final frame
		fmt.Fprintf(os.Stderr, "    ┌─────────────────────┐\n")
		fmt.Fprintf(os.Stderr, "    │  🌑 Kusanagi Abyss 🌑│\n")
		fmt.Fprintf(os.Stderr, "    │    c0rrupt3d...    │\n")
		fmt.Fprintf(os.Stderr, "    │   深淵への堕落...    │\n")
		fmt.Fprintf(os.Stderr, "    └─────────────────────┘\n")
	}

	// Small pause after animation
	time.Sleep(200 * time.Millisecond)
}

// GetAssetBase64 returns the base64 encoded version of an asset
// Useful for embedding in JSON or API payloads
func GetAssetBase64(assetType AssetType) (string, error) {
	var gifData []byte

	switch assetType {
	case PixelWink:
		gifData = pixelWinkGif
	case Kusanagi:
		gifData = kusanagiGif
	default:
		return "", fmt.Errorf("unknown asset type: %s", assetType)
	}

	if len(gifData) == 0 {
		return "", fmt.Errorf("asset data is empty")
	}

	return base64.StdEncoding.EncodeToString(gifData), nil
}

// ListAvailableAssets returns a list of available embedded assets
func ListAvailableAssets() []string {
	return []string{
		"pixel_wink - Celeste pixel art winking",
		"kusanagi - Kusanagi/abyss themed artwork",
	}
}
