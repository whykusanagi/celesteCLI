.PHONY: build install clean help test dev

# Default target
help:
	@echo "Celeste CLI Build Commands"
	@echo "=========================="
	@echo "  make build        - Build celeste binary in current directory"
	@echo "  make install      - Build and install to ~/.local/bin/celeste"
	@echo "  make dev          - Build, install, and test in PATH"
	@echo "  make clean        - Remove local binary"
	@echo "  make test         - Run installed binary test"
	@echo "  make help         - Show this help message"

# Build the binary
build:
	@echo "🔨 Building Celeste..."
	@cd cmd/celeste && go build -o ../../celeste .
	@echo "✅ Build complete: ./celeste"

# Build and install to PATH
install: build
	@echo "📦 Installing to PATH..."
	@cp celeste ~/.local/bin/celeste
	@chmod +x ~/.local/bin/celeste
	@echo "✅ celeste installed to ~/.local/bin/celeste"

# Development workflow: build, install, and test
dev: install
	@echo "🎯 Testing installed binary..."
	@celeste --version
	@echo "✨ Ready for development!"

# Clean up local binary
clean:
	@echo "🧹 Cleaning up..."
	@rm -f celeste
	@echo "✅ Cleanup complete"

# Test the installed binary
test:
	@echo "🧪 Testing celeste binary..."
	@which celeste > /dev/null && echo "✅ celeste found in PATH" || echo "❌ celeste not found in PATH"
	@celeste --version 2>/dev/null && echo "✅ Version check passed" || echo "⚠️  Version check failed"
