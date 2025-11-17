# Celeste CLI Build Workflow Guide

## Quick Start

After making code changes, update the binary in your PATH with:

```bash
make install
```

That's it! Your binary in `~/.local/bin/Celeste` is now updated.

## Available Commands

| Command | Purpose |
|---------|---------|
| `make help` | Show all available commands |
| `make build` | Build Celeste binary locally (creates `./Celeste`) |
| `make install` | Build and install to `~/.local/bin/Celeste` ⭐ Recommended for development |
| `make dev` | Build, install, and test (full workflow with verification) |
| `make clean` | Remove local binary (`./Celeste`) |
| `make test` | Verify installed binary works correctly |

## Typical Development Workflow

```bash
# Edit your code
nano main.go

# Build and install to PATH in one command
make install

# Test the updated binary
Celeste --version

# Or test interactively
Celeste -i
```

## One-Liner Workflow

For rapid development iterations:

```bash
make install && Celeste --version
```

## Full Development Cycle (with verification)

```bash
make dev
```

This will:
1. 🔨 Build the binary
2. 📦 Install to `~/.local/bin/Celeste`
3. 🎯 Test to verify it works
4. ✨ Show success confirmation

## Shell Integration (Optional but Recommended)

Add this function to your `~/.bashrc` or `~/.zshrc` for a global shortcut:

```bash
# Celeste CLI development function
function cbuild() {
    cd ~/Desktop/celesteCLI
    make install
    if [ $? -eq 0 ]; then
        echo "✨ Build successful!"
        Celeste --version
    else
        echo "❌ Build failed!"
        return 1
    fi
}
```

Then reload your shell:
```bash
source ~/.bashrc  # or source ~/.zshrc
```

Now you can rebuild from anywhere:
```bash
cbuild
```

## What Gets Built

The Makefile compiles all Go source files:
- `main.go` - Core CLI application
- `scaffolding.go` - Prompt templates
- `animation.go` - Corruption animations
- `ui.go` - UI system
- `assets.go` - Embedded GIF assets
- `interactive.go` - Interactive mode
- `terminal_display.go` - Terminal capability detection
- `ascii_art.go` - ASCII art functions

Into a single executable: `~/.local/bin/Celeste`

## Troubleshooting

### Binary not updating in PATH
```bash
# Verify PATH includes ~/.local/bin
echo $PATH | grep ".local/bin"

# If not, add to ~/.bashrc or ~/.zshrc:
export PATH="$HOME/.local/bin:$PATH"
```

### "permission denied" when running Celeste
```bash
chmod +x ~/.local/bin/Celeste
```

### Build errors
```bash
# Ensure all Go files are present
ls *.go

# Tidy dependencies
go mod tidy

# Try building again
make build
```

### Want to test without installing?
```bash
make build
./Celeste --version
```

## Performance Tips

The Makefile is optimized for quick rebuilds. Go incremental compilation means:
- Rebuilding only changed files
- Typical rebuild time: 1-3 seconds
- Full clean build: 3-5 seconds

## Common Aliases

Add these to your shell profile for even faster workflows:

```bash
# Quick build and install
alias cbuild='make install'

# Build and test
alias ctest='make dev'

# Build with cleanup
alias cbuildclean='make clean && make install'

# View help
alias chelp='make help'
```

## FAQ

**Q: Do I need to be in the celesteCLI directory?**
A: Yes, the Makefile must be run from the project root directory.

**Q: Will this replace the old binary if it's running?**
A: On Unix-like systems (macOS, Linux), yes. The new binary will replace the old one even while it's running, though the old process will continue.

**Q: Can I use this with an IDE?**
A: Yes! Configure your IDE to run `make install` as a build/run step.

**Q: What if the build fails?**
A: The Makefile will stop and show the error. Fix the issue and run `make install` again.

**Q: Do I need to run `go mod tidy` first?**
A: No, the Makefile handles everything. Just run `make install`.

## Next Steps

After building:
```bash
# View version info
Celeste --version

# Get help
Celeste --help

# Try interactive mode (with Kusanagi animation)
Celeste -i

# Generate content
Celeste --format short --platform twitter --topic "gaming"
```

---

**Remember:** After ANY code change, just run `make install` and you're done! 🚀
