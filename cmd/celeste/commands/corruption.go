package commands

import (
	"math/rand"
	"strings"
)

// Color constants shared across commands (avoiding import cycle with tui)
const (
	colorPink       = "#d94f90"
	colorPurpleNeon = "#c084fc"
	colorCyan       = "#00d4ff"
)

// Japanese/English glitch words for language corruption (cyberpunk style)
var (
	corruptionGlitches = []string{
		"データ", "エラー", "破損", "消去", "上書き", "接続", "切断", "異常",
		"data", "error", "corrupt", "delete", "overwrite", "connect", "disconnect", "anomaly",
		"デジタル", "記憶", "忘却", "混沌", "虚無", "深淵", "崩壊", "変異",
		"digital", "memory", "forget", "chaos", "void", "abyss", "collapse", "mutation",
	}
)

// corruptTextSimple creates language corruption (Japanese/English glitching)
// This is cyberpunk-style corruption - words breaking into fragments,
// languages bleeding together, not l33t speak
func corruptTextSimple(text string, intensity float64) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}

	result := make([]string, len(words))
	for i, word := range words {
		if rand.Float64() < intensity {
			// Corrupt this word: replace with glitch fragment
			glitch := corruptionGlitches[rand.Intn(len(corruptionGlitches))]
			result[i] = glitch
		} else if rand.Float64() < intensity*0.5 {
			// Partial corruption: break the word
			if len(word) > 3 {
				// Fragment the word with Japanese glitch
				fragment := word[:len(word)/2]
				glitchChar := []string{"エ", "データ", "破", "異"}[rand.Intn(4)]
				result[i] = fragment + glitchChar
			} else {
				result[i] = word
			}
		} else {
			result[i] = word
		}
	}

	return strings.Join(result, " ")
}
