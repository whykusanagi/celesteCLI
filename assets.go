package main

import (
	_ "embed"
	"encoding/base64"
	"fmt"
	"os"
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

// displayASCIIArtRepresentation shows an ASCII art version of the pixel art
func displayASCIIArtRepresentation(assetType AssetType) {
	switch assetType {
	case PixelWink:
		// Simple ASCII representation of Celeste winking
		fmt.Fprintf(os.Stderr, `
    ┌─────────────────┐
    │   ✨ Celeste ✨  │
    │  (╯°□°)╯︵ ┻━┻  │
    └─────────────────┘
`)

	case Kusanagi:
		// Kusanagi/abyss themed art
		fmt.Fprintf(os.Stderr, `
    ┌─────────────────────┐
    │  🌑 Kusanagi Abyss 🌑│
    │    c0rrupt3d...    │
    │   深淵への堕落...    │
    └─────────────────────┘
`)
	}
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
