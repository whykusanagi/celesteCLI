package main

import (
	"fmt"
	"os"
)

// DisplayGIFAnimated displays pre-rendered ASCII art of the GIF
// No external dependencies required - uses pre-rendered ASCII art
func DisplayGIFAnimated(gifData []byte, assetType AssetType) error {
	displayPreRenderedASCII(assetType)
	return nil
}

// DisplayGIFStatic displays just the first frame using pre-rendered ASCII
func DisplayGIFStatic(gifData []byte, assetType AssetType) error {
	displayPreRenderedASCII(assetType)
	return nil
}

// displayPreRenderedASCII displays the pre-rendered ASCII art for an asset
func displayPreRenderedASCII(assetType AssetType) {
	switch assetType {
	case PixelWink:
		fmt.Fprintf(os.Stderr, "%s\n", PixelWinkASCII)
	case Kusanagi:
		fmt.Fprintf(os.Stderr, "%s\n", KusanagiASCII)
	}
}

// DisplayAssetOptimal displays the asset using pre-rendered ASCII art
func DisplayAssetOptimal(assetType AssetType) error {
	displayPreRenderedASCII(assetType)
	return nil
}

// TerminalInfo returns information about the terminal and ASCII art display
func TerminalInfo() string {
	return `Terminal Capabilities:
  ✓ Pre-rendered ASCII art (no external dependencies)
  ✓ Zero dependencies - all assets embedded at build time

  Celeste uses beautiful pre-rendered ASCII art created with chafa.
  No runtime dependencies required - everything is compiled in!
`
}
