#!/bin/bash

# Test script to validate Celeste animation
# This script runs the Celeste CLI and captures animation frames

echo "=== Celeste Animation Test ==="
echo ""
echo "Testing prompt animation..."
echo ""

# Create a test input file
cat > /tmp/test_input.txt <<'INPUT'
test message
exit
INPUT

# Create dummy config files to prevent startup errors
mkdir -p /root/.celeste
cat > /root/.celeste/personality.yml <<'CONFIG'
name: Celeste
personality: mysterious
CONFIG

cat > /root/.celesteAI <<'CONFIG'
CELESTE_API_KEY=dummy_key_for_testing
CELESTE_API_ENDPOINT=http://localhost:8000
CONFIG

# Run Celeste in interactive mode and capture all output with timing info
# Force a pseudo-terminal to enable animations
script -q -c "timeout 3 ./Celeste --interactive < /tmp/test_input.txt" /tmp/test_output.log || true

# Display the raw output
echo ""
echo "=== Raw Output ==="
cat /tmp/test_output.log

echo ""
echo ""
echo "=== Test Analysis ==="
echo ""
echo "Looking for animation indicators..."
echo ""

# Check for ANSI clear line escape sequence
if grep -q $'\033\[2K' /tmp/test_output.log; then
    echo "✓ FOUND: ANSI clear line sequence (\033[2K) - Animation should work!"
else
    echo "✗ MISSING: ANSI clear line sequence - Animation might not work"
fi

# Check for color codes
if grep -q $'\033\[38;5' /tmp/test_output.log; then
    echo "✓ FOUND: Color codes present"
else
    echo "✗ MISSING: Color codes"
fi

# Check for multiple frames on same line (bad - no clearing)
FRAME_COUNT=$(grep -o $'\033\[38;5;[0-9]\+m' /tmp/test_output.log | wc -l)
echo "✓ Found $FRAME_COUNT color code changes"

if [ "$FRAME_COUNT" -ge 4 ]; then
    echo "✓ Multiple animation frames detected"
fi

echo ""
echo "=== Expected Output ==="
echo "The output should show:"
echo "1. Kusanagi ASCII art (colored pixel art)"
echo "2. Animated prompt with \\033[2K (clear line) and \\033[38;5;NNNm (colors)"
echo "3. Processing messages with animations"
echo "4. PixelWink ASCII art on exit (colored pixel art)"
echo ""
echo "Raw animation sequence should look like:"
echo "  ...\\033[2K\\r\\033[38;5;213m🎀 Tell Celeste...\\033[0m..."
echo "  ...\\033[2K\\r\\033[38;5;177m🎀 Tell Celeste...\\033[0m..."
echo "NOT like:"
echo "  ...\\033[38;5;213m...Tell Celeste...\\033[0m\\033[38;5;177m...Tell Celeste...\\033[0m..."
