.PHONY: build install clean help test dev

# Default target
help:
	@echo "Celeste CLI Build Commands"
	@echo "=========================="
	@echo "  make build        - Build Celeste binary in current directory"
	@echo "  make install      - Build and install to ~/.local/bin/Celeste"
	@echo "  make dev          - Build, install, and test in PATH"
	@echo "  make clean        - Remove local binary"
	@echo "  make test         - Run installed binary test"
	@echo "  make help         - Show this help message"

# Build the binary
build:
	@echo "🔨 Building Celeste..."
	@go build -o Celeste main.go scaffolding.go animation.go ui.go assets.go interactive.go terminal_display.go ascii_art.go
	@echo "✅ Build complete: ./Celeste"

# Build and install to PATH
install: build
	@echo "📦 Installing to PATH..."
	@cp Celeste ~/.local/bin/Celeste
	@chmod +x ~/.local/bin/Celeste
	@echo "✅ Celeste installed to ~/.local/bin/Celeste"

# Development workflow: build, install, and test
dev: install
	@echo "🎯 Testing installed binary..."
	@Celeste --version
	@echo "✨ Ready for development!"

# Clean up local binary
clean:
	@echo "🧹 Cleaning up..."
	@rm -f Celeste
	@echo "✅ Cleanup complete"

# Test the installed binary
test:
	@echo "🧪 Testing Celeste binary..."
	@which Celeste > /dev/null && echo "✅ Celeste found in PATH" || echo "❌ Celeste not found in PATH"
	@Celeste --version 2>/dev/null && echo "✅ Version check passed" || echo "⚠️  Version check failed"
