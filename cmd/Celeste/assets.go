package main

import (
	_ "embed"
	"encoding/base64"
	"fmt"
	"os"
)

// Embed pixel art asset directly into the binary for portability
//
//go:embed assets/kusanagi_4x.gif
var celesteGif []byte

// AssetType defines different asset types available
type AssetType string

const (
	Celeste AssetType = "celeste"
)

// DisplayPixelArt displays a pixel art asset in the terminal
// This is a simple implementation - can be enhanced with sixel support for better terminals
func DisplayPixelArt(assetType AssetType) error {
	var gifData []byte

	switch assetType {
	case Celeste:
		gifData = celesteGif
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
	case Celeste:
		// Celeste pixel art representation
		fmt.Fprintf(os.Stderr, "    ┌─────────────────────┐\n")
		fmt.Fprintf(os.Stderr, "    │   ✨ Celeste ✨     │\n")
		fmt.Fprintf(os.Stderr, "    │    c0rrupt3d...    │\n")
		fmt.Fprintf(os.Stderr, "    │   深淵への堕落...    │\n")
		fmt.Fprintf(os.Stderr, "    └─────────────────────┘\n")
	}
}

// GetAssetBase64 returns the base64 encoded version of an asset
// Useful for embedding in JSON or API payloads
func GetAssetBase64(assetType AssetType) (string, error) {
	var gifData []byte

	switch assetType {
	case Celeste:
		gifData = celesteGif
	default:
		return "", fmt.Errorf("unknown asset type: %s", assetType)
	}

	if len(gifData) == 0 {
		return "", fmt.Errorf("asset data is empty")
	}

	return base64.StdEncoding.EncodeToString(gifData), nil
}

// GetCelesteGif returns the raw GIF data for the Celeste pixel art
func GetCelesteGif() []byte {
	return celesteGif
}

// ListAvailableAssets returns a list of available embedded assets
func ListAvailableAssets() []string {
	return []string{
		"celeste - Celeste pixel art (364x560)",
	}
}
