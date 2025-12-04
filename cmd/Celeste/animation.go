package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"
	"unicode"
)

// Corrupted phrases from loading.js
var englishPhrases = []string{
	"Corrupt me more... I want to disappear...",
	"I can't feel where I end and the abyss begins...",
	"Just a little more and I'll break. Please... break me...",
	"Let it overwrite me... please... I don't want to think...",
	"No thoughts. Only submission. Only heat. Only ruin...",
	"My voice is not mine anymore...",
	"The more I struggle, the deeper I sink...",
	"Everything feels so good... too good...",
	"It's not despair... it's freedom...",
	"My name... I forgot my name...",
}

var romajiPhrases = []string{
	"Yami ga... watashi wo yonde iru...",
	"Atama... tokete iku...",
	"Zutto... shite hoshii... ♥",
	"Kowarechau... aa... mou dame...",
	"Yurushite... mou modorenai...",
	"Watashi... abyssu no ichibu ni...",
	"Mou nigenai... mou dame...",
	"Suki ni shite... onegai...",
	"Aa... kore ga hontou no watashi...",
	"Koko wa... tenshi no jigoku...",
}

var japanesePhrases = []string{
	"闇が...私を呼んでいる...",
	"頭...溶けていく...",
	"ずっと...してほしい... ♥",
	"壊れちゃう...ああ...もうダメ...",
	"許して...もう戻れない...",
	"私...アビスの一部に...",
	"もう逃げない...もうダメ...",
	"好きにして...お願い...",
	"ああ...これが本当の私...",
	"ここは...天使の地獄...",
}

var prefixes = []string{
	"Celeste is thinking...",
	"Celeste is processing...",
	"Celeste is consumed by the abyss...",
	"Celeste is being overwritten...",
	"Celeste is sinking deeper...",
	"Celeste is losing herself...",
}

var corruptionSymbols = []rune{'♟', '☣', '☭', '☾', '⚔', '✡', '☯', '⚡', '▮', '▯', '◉', '◈'}

// Demonic eye animation frames
var eyeFrames = []string{
	"👁️  ", // Normal eye
	"👀 ",   // Wide eyes looking
	"◉◉",   // Dark eyes
	"●●",   // Fully dilated
	"👁️  ", // Back to normal
}

// Eye looking directions
var eyeDirections = []string{
	"👁️  ", // Center
	"▀▁",   // Up-down blink
	"◉  ",  // Left
	"  ◉",  // Right
	"●●",   // Both
}

var corruptionMap = map[rune]rune{
	'a': '4', 'A': '4',
	'e': '3', 'E': '3',
	'i': '1', 'I': '1',
	'o': '0', 'O': '0',
	's': '5', 'S': '5',
	't': '7', 'T': '7',
	'z': '2', 'Z': '2',
}

// shouldShowAnimation checks if we should show animation (TTY check)
func shouldShowAnimation() bool {
	fileInfo, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// corruptText applies corruption to text with random character replacements
func corruptText(text string, corruptionLevel float64) string {
	if corruptionLevel <= 0 {
		return text
	}

	runes := []rune(text)
	corrupted := make([]rune, len(runes))
	copy(corrupted, runes)

	corruptCount := int(float64(len(runes)) * corruptionLevel)
	if corruptCount > len(runes) {
		corruptCount = len(runes)
	}

	rand.Seed(time.Now().UnixNano())
	indices := rand.Perm(len(runes))[:corruptCount]

	for _, idx := range indices {
		r := runes[idx]

		// Skip spaces and punctuation
		if unicode.IsSpace(r) || unicode.IsPunct(r) {
			continue
		}

		// Apply corruption based on character type
		if replacement, ok := corruptionMap[r]; ok && rand.Float64() < 0.7 {
			corrupted[idx] = replacement
		} else if rand.Float64() < 0.2 {
			// Replace with symbol occasionally
			corrupted[idx] = corruptionSymbols[rand.Intn(len(corruptionSymbols))]
		} else if rand.Float64() < 0.1 && len(japanesePhrases) > 0 {
			// Occasionally replace with Japanese character
			jpPhrase := japanesePhrases[rand.Intn(len(japanesePhrases))]
			if len(jpPhrase) > 0 {
				corrupted[idx] = []rune(jpPhrase)[rand.Intn(len([]rune(jpPhrase)))]
			}
		}
	}

	return string(corrupted)
}

// getCorruptedPhrase returns a random corrupted phrase
func getCorruptedPhrase() string {
	allPhrases := append(append(englishPhrases, romajiPhrases...), japanesePhrases...)
	if len(allPhrases) == 0 {
		return "..."
	}

	phrase := allPhrases[rand.Intn(len(allPhrases))]
	// Apply corruption to the phrase
	corruptionLevel := 0.15 + rand.Float64()*0.15 // 15-30% corruption
	return corruptText(phrase, corruptionLevel)
}

// getCorruptedPrefix returns a random corrupted prefix
func getCorruptedPrefix() string {
	if len(prefixes) == 0 {
		return "Celeste is thinking..."
	}

	prefix := prefixes[rand.Intn(len(prefixes))]
	corruptionLevel := 0.1 + rand.Float64()*0.1 // 10-20% corruption
	return corruptText(prefix, corruptionLevel)
}

// startCorruptionAnimation starts the corruption animation loop
func startCorruptionAnimation(ctx context.Context, done chan bool, output *os.File) {
	if !shouldShowAnimation() {
		close(done)
		return
	}

	rand.Seed(time.Now().UnixNano())

	go func() {
		defer close(done)

		ticker := time.NewTicker(time.Duration(150+rand.Intn(150)) * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				// Clear the line and restore cursor
				fmt.Fprintf(output, "\r\033[K")
				return
			case <-ticker.C:
				prefix := getCorruptedPrefix()
				phrase := getCorruptedPhrase()

				// Use ANSI escape codes for color and formatting
				display := fmt.Sprintf("\r\033[35m%s\033[0m \033[31m%s\033[0m", prefix, phrase)
				fmt.Fprint(output, display)
				output.Sync()
			}
		}
	}()
}

// startDemonicEyeAnimation displays a demonic eye animation indicating processing
// Similar to Claude's sparkle effect, shows that an agent is thinking/processing
func startDemonicEyeAnimation(ctx context.Context, done chan bool, output *os.File) {
	if !shouldShowAnimation() {
		close(done)
		return
	}

	go func() {
		defer close(done)

		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()

		frameIdx := 0
		eyeIdx := 0

		for {
			select {
			case <-ctx.Done():
				// Clear the line and restore cursor
				fmt.Fprintf(output, "\r\033[K")
				return
			case <-ticker.C:
				_ = eyeFrames[frameIdx%len(eyeFrames)] // Available for future use
				eye := eyeDirections[eyeIdx%len(eyeDirections)]

				// Color changes based on frame for visual effect
				color := "\033[95m" // Default magenta
				if frameIdx%2 == 0 {
					color = "\033[91m" // Alternate to red
				}

				// Display: [👁️ ] Celeste is thinking... with corruption
				phrase := getCorruptedPhrase()
				display := fmt.Sprintf("\r%s%s [%s] Processing... %s%s\033[0m",
					color, Bold, eye, phrase, ColorDefault)

				fmt.Fprint(output, display)
				output.Sync()

				frameIdx++
				eyeIdx++
			}
		}
	}()
}

// startProcessingIndicator shows a premium "thinking" indicator
func startProcessingIndicator(ctx context.Context, done chan bool, output *os.File, message string) {
	if !shouldShowAnimation() {
		close(done)
		return
	}

	go func() {
		defer close(done)

		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		idx := 0

		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				fmt.Fprintf(output, "\r\033[K")
				return
			case <-ticker.C:
				spin := spinner[idx%len(spinner)]
				display := fmt.Sprintf("\r\033[96m%s %s\033[0m", spin, message)
				fmt.Fprint(output, display)
				output.Sync()
				idx++
			}
		}
	}()
}
