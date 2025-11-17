package main

import (
	"fmt"
	"os"
)

// AssetType defines different asset types available
type AssetType string

const (
	PixelWink  AssetType = "pixel_wink"
	Kusanagi   AssetType = "kusanagi"
)

// DisplayPixelArt displays a pixel art asset in the terminal
// This is a simple implementation - can be enhanced with sixel support for better terminals
func DisplayPixelArt(assetType AssetType) error {
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
		fmt.Fprintf(os.Stderr, "    ┌─────────────────┐\n")
		fmt.Fprintf(os.Stderr, "    │   ✨ Celeste ✨  │\n")
		fmt.Fprintf(os.Stderr, "    │  (╯°□°)╯︵ ┻━┻  │\n")
		fmt.Fprintf(os.Stderr, "    └─────────────────┘\n")

	case Kusanagi:
		// Kusanagi/abyss themed art
		fmt.Fprintf(os.Stderr, "    ┌─────────────────────┐\n")
		fmt.Fprintf(os.Stderr, "    │  🌑 Kusanagi Abyss 🌑│\n")
		fmt.Fprintf(os.Stderr, "    │    c0rrupt3d...    │\n")
		fmt.Fprintf(os.Stderr, "    │   深淵への堕落...    │\n")
		fmt.Fprintf(os.Stderr, "    └─────────────────────┘\n")
	}
}

// GetAssetBase64 returns the base64 encoded version of an asset
// Useful for embedding in JSON or API payloads
func GetAssetBase64(assetType AssetType) (string, error) {
	switch assetType {
	case Kusanagi:
		// Return the pre-encoded base64 directly from embedded assets
		return KusanagiGIFBase64, nil
	case PixelWink:
		// PixelWink is no longer embedded (was removed due to animation artifacts)
		return "", fmt.Errorf("PixelWink asset is no longer available")
	default:
		return "", fmt.Errorf("unknown asset type: %s", assetType)
	}
}

// ListAvailableAssets returns a list of available embedded assets
func ListAvailableAssets() []string {
	return []string{
		"pixel_wink - Celeste pixel art winking",
		"kusanagi - Kusanagi/abyss themed artwork",
	}
}
