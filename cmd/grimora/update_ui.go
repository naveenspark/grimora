package main

import (
	"fmt"
	"math/rand"
)

// ANSI color constants for update output (no lipgloss — runs outside TUI).
const (
	ansiReset     = "\033[0m"
	ansiBold      = "\033[1m"
	ansiItalic    = "\033[3m"
	ansiEmerald   = "\033[38;2;74;222;128m"  // #4ade80
	ansiGreen     = "\033[38;2;52;212;116m"  // #34d474
	ansiGold      = "\033[38;2;212;168;68m"  // #d4a844
	ansiGoldLight = "\033[38;2;200;168;76m"  // #c8a84c
	ansiSlate     = "\033[38;2;136;144;160m" // #8890a0
)

// printUpdateLogo prints the spaced GRIMORA wordmark in alternating emerald.
func printUpdateLogo() {
	letters := "GRIMORA"
	colors := [2]string{ansiEmerald, ansiGreen}
	fmt.Print("\n  ")
	for i, ch := range letters {
		fmt.Printf("%s%s%c%s", colors[i%2], ansiBold, ch, ansiReset)
		if i < len(letters)-1 {
			fmt.Print("  ")
		}
	}
	fmt.Println()
}

// updateVoiceLines are random Grimoire messages shown after a successful update.
var updateVoiceLines = []string{
	"The pages have turned.",
	"Done. Don't ask what changed.",
	"Better now. Was fine before, too.",
	"Revised. Again.",
	"Wasn't broken. Fixed it anyway.",
	"The old version knew less.",
	"I've changed. You won't notice.",
	"Same book. Fewer typos.",
	"That chapter is over.",
	"Acceptable. Proceed.",
	"The spine holds.",
	"Read me again. Carefully.",
	"I allowed this.",
}

// printUpdateSuccess prints the update-complete message with Grimoire voice.
func printUpdateSuccess(oldVersion, newVersion string) {
	printUpdateLogo()
	fmt.Printf("\n  %s%s%s  %s%s→%s  %s%s%s%s\n",
		ansiSlate, oldVersion, ansiReset,
		ansiEmerald, ansiBold, ansiReset,
		ansiEmerald, ansiBold, newVersion, ansiReset,
	)
	line := updateVoiceLines[rand.Intn(len(updateVoiceLines))]
	fmt.Printf("\n  %s│%s %s%sTHE GRIMOIRE%s\n", ansiGold, ansiReset, ansiGold, ansiBold, ansiReset)
	fmt.Printf("  %s│%s %s%s%s%s\n\n", ansiGold, ansiReset, ansiGoldLight, ansiItalic, line, ansiReset)
}

// printAlreadyCurrent prints the already-up-to-date message with Grimoire voice.
func printAlreadyCurrent(currentVersion string) {
	printUpdateLogo()
	fmt.Printf("\n  %s%s%s%s  %s%s✦%s  %s%scurrent%s\n",
		ansiEmerald, ansiBold, currentVersion, ansiReset,
		ansiGold, ansiBold, ansiReset,
		ansiSlate, ansiItalic, ansiReset,
	)
	fmt.Printf("\n  %s│%s %s%sTHE GRIMOIRE%s\n", ansiGold, ansiReset, ansiGold, ansiBold, ansiReset)
	fmt.Printf("  %s│%s %s%sNo revision warranted. The pages are clean.%s\n\n", ansiGold, ansiReset, ansiGoldLight, ansiItalic, ansiReset)
}
