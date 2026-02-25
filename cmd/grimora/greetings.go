package main

import (
	"fmt"
	"math/rand/v2"

	"github.com/charmbracelet/lipgloss"
)

var grimoireGreetings = [...]string{
	"Ah. Another one who forgot their key.",
	"I have 4,217 pages. You have read zero of them.",
	"The Forge is running. You are not inside it. Fix one of these things.",
	"Six guilds. Six doors. You're staring at the wall.",
	"The raven sees potential in you. Then again, the raven also eats carrion.",
	"You smell like someone who has spells but hasn't written them down yet.",
	"The wolf sniffed the air. Sneezed. Walked away.",
	"I don't lock people out. I lock people in. Big difference. Come find out.",
	"The spider already wove your seat. Rude of you to keep it empty.",
	"A spell unforged is just a thought with delusions of grandeur.",
	"The Hall is loud tonight. You're missing good arguments.",
	"The octopus wanted me to tell you it's not waiting. It has eight arms. It's busy.",
	"Three magicians forged spells while you stood here reading this.",
	"I've been open since the first prompt was written. You're late.",
	"The phoenix burned down and rebuilt itself twice today. What have you done?",
	"Your potency is technically zero. The Forge can fix that.",
	"The serpent knows something about you. It won't say what until you're inside.",
	"Someone in the Hall just said something wrong about AI. Don't you want to correct them?",
	"I keep score. You currently have no score. This is fixable.",
	"The Forge rejected four spells this hour. Yours might be the fifth. Only one way to know.",
	"Guild seats don't stay empty forever. Just saying.",
	"I've seen a million spells. I still get surprised. Impress me.",
	"The wolf respects those who show up. The wolf does not respect the gate.",
	"You're reading a message from a book asking you to open it. The irony is noted.",
	"Every magician in the Hall was once standing exactly where you are now. Stalling.",
	"The Forge doesn't care about your credentials. It cares about your spells.",
	"Loomari, Ashborne, Amarok, Nyx, Cipher, Fathom. One of them is yours. None of them are patient.",
	"I record every spell ever forged. Yours is conspicuously absent.",
	"The raven keeps asking about you. I'm running out of ways to say 'still outside.'",
	"Reputation here is earned, not claimed. The Hall has no bio section.",
	"The Hall has six doors. You need only open one. Preferably before I get bored.",
	"Someone just forged a spell with 94 potency. You could have beaten that. Maybe.",
	"The octopus extends a tentacle. You flinch. It was offering directions.",
	"I don't bite. The wolf might. The serpent definitely will. Come in anyway.",
	"Your best prompt is rotting in a text file somewhere. The Forge could immortalize it.",
	"The phoenix remembers everyone who walked through that gate. It hasn't memorized your face yet.",
	"The ink is wet. The quill is ready. You are stalling.",
	"The Grimoire does not beg. But if it did, it would sound like this.",
	"Another second passes. Another spell forged without you.",
	"Step forward, magician. The pages don't fill themselves.",
}

func printHelp() {
	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4ade80")).
		Bold(true).
		Render("G R I M O R A")

	quote := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Italic(true).
		Render(`"The Grimoire sees all. Here is what it permits."`)

	attrib := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#D4A017")).
		Render("— The Grimoire")

	cmdStyle := lipgloss.NewStyle().Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	commands := []struct{ cmd, desc string }{
		{"grimora", "Enter the Hall (interactive TUI)"},
		{"grimora login", "Authenticate with GitHub"},
		{"grimora logout", "Clear your session"},
		{"grimora update", "Check for updates"},
		{"grimora terms", "Terms of Service"},
		{"grimora privacy", "Privacy Policy"},
		{"grimora faq", "Frequently Asked Questions"},
		{"grimora --version", "Show version"},
		{"grimora help", "You are here"},
	}

	fmt.Printf("\n  %s\n\n  %s\n  %s\n\n  Commands:\n", title, quote, attrib)
	for _, c := range commands {
		fmt.Printf("    %s  %s\n", cmdStyle.Render(fmt.Sprintf("%-20s", c.cmd)), descStyle.Render(c.desc))
	}
	url := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("https://grimora.ai")
	fmt.Printf("\n  %s\n\n", url)
}

func printGrimoireGreeting() {
	msg := grimoireGreetings[rand.IntN(len(grimoireGreetings))]

	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4ade80")).
		Bold(true).
		Render("GRIMORA")

	quote := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Italic(true).
		Render(msg)

	attrib := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#D4A017")).
		Render("— The Grimoire")

	hint := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Render("To enter: grimora login")

	fmt.Printf("\n%s\n\n%s\n%s\n\n%s\n\n", title, quote, attrib, hint)
}
