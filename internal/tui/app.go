package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/naveenspark/grimora/internal/browser"
	"github.com/naveenspark/grimora/pkg/client"
	"github.com/naveenspark/grimora/pkg/domain"
)

type view int

const (
	viewHome view = iota
	viewHall
	viewGrimoire
	viewYou
	viewCreate
)

// meLoadedMsg carries the result of GetMe + ForgeStats.
type meLoadedMsg struct {
	me    *domain.Magician
	stats *domain.ForgeStats
	err   error
}

// showPeekMsg triggers the peek overlay for a magician.
type showPeekMsg struct {
	login string
}

// App is the root Bubbletea model.
type App struct {
	client     *client.Client
	view       view
	home       homeModel
	hall       hallModel
	grimoire   grimoireModel
	you        youModel
	create     createModel
	peek       peekModel
	peekOpen   bool
	helpOpen   bool
	helpCursor int
	me         *domain.Magician
	stats      *domain.ForgeStats
	width      int
	height     int
	frame      int // logo shimmer animation frame
}

// NewApp creates a new TUI application.
func NewApp(c *client.Client) App {
	return App{
		client:   c,
		home:     newHomeModel(c),
		hall:     newHallModel(c),
		grimoire: newGrimoireModel(c),
		you:      newYouModel(c),
		create:   newCreateModel(c),
		peek:     newPeekModel(c),
	}
}

func (a App) Init() tea.Cmd {
	return tea.Batch(a.home.Init(), shimmerTickCmd(), a.loadMe())
}

func (a App) loadMe() tea.Cmd {
	c := a.client
	return func() tea.Msg {
		me, err := c.GetMe(context.Background())
		if err != nil {
			return meLoadedMsg{err: err}
		}
		stats, err := c.GetForgeStats(context.Background())
		if err != nil {
			stats = nil
		}
		return meLoadedMsg{me: me, stats: stats}
	}
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		// Chrome: header(2) + tabs(1) + input(1) + help(1) = 5 lines
		bodyHeight := msg.Height - 5
		bodyMsg := tea.WindowSizeMsg{Width: msg.Width, Height: bodyHeight}
		a.home, _ = a.home.Update(bodyMsg)
		a.hall, _ = a.hall.Update(bodyMsg)
		a.grimoire, _ = a.grimoire.Update(bodyMsg)
		a.you, _ = a.you.Update(bodyMsg)
		a.peek, _ = a.peek.Update(bodyMsg)
		a.create, _ = a.create.Update(bodyMsg)

	case shimmerTickMsg:
		a.frame++
		return a, shimmerTickCmd()

	case meLoadedMsg:
		if msg.err == nil && msg.me != nil {
			a.me = msg.me
			a.stats = msg.stats
		}
		// Propagate to sub-models that need user identity
		a.you, _ = a.you.Update(msg)
		a.hall, _ = a.hall.Update(msg)
		return a, nil

	case showPeekMsg:
		a.peekOpen = true
		a.peek = newPeekModel(a.client)
		return a, a.peek.load(msg.login)

	case tea.KeyMsg:
		// Help overlay captures all keys when open
		if a.helpOpen {
			switch msg.String() {
			case "h", "esc":
				a.helpOpen = false
			case "q", "ctrl+c":
				return a, tea.Quit
			case "j", "down":
				if a.helpCursor < len(helpItems)-1 {
					a.helpCursor++
				}
			case "k", "up":
				if a.helpCursor > 0 {
					a.helpCursor--
				}
			case "enter":
				item := helpItems[a.helpCursor]
				if item.url != "" {
					browser.Open(item.url) //nolint:errcheck // best-effort browser open
				}
			}
			return a, nil
		}

		// Peek overlay captures all keys when open
		if a.peekOpen {
			var cmd tea.Cmd
			a.peek, cmd = a.peek.Update(msg)
			if a.peek.closed {
				a.peekOpen = false
			}
			return a, cmd
		}

		// Global keys (only when not editing)
		if !a.isEditing() {
			switch msg.String() {
			case "h":
				a.helpOpen = true
				a.helpCursor = 0
				return a, nil
			case "q", "ctrl+c":
				return a, tea.Quit
			case "1":
				if a.view != viewHome {
					a.view = viewHome
					return a, a.home.Init()
				}
				return a, nil
			case "2":
				if a.view != viewHall {
					a.view = viewHall
					return a, a.hall.Init()
				}
				return a, nil
			case "3":
				if a.view != viewGrimoire {
					a.view = viewGrimoire
					return a, a.grimoire.Init()
				}
				return a, nil
			case "4":
				if a.view != viewYou {
					a.view = viewYou
					return a, a.you.Init()
				}
				return a, nil
			case "n":
				if a.view != viewCreate {
					a.view = viewCreate
					return a, nil
				}
			case "esc":
				if a.view == viewCreate {
					a.view = viewHome
					return a, a.home.Init()
				}
			}
		} else if msg.String() == "esc" && a.view == viewCreate {
			a.view = viewHome
			return a, a.home.Init()
		}
	}

	// Route peek messages when overlay is open
	if a.peekOpen {
		var cmd tea.Cmd
		a.peek, cmd = a.peek.Update(msg)
		if a.peek.closed {
			a.peekOpen = false
		}
		return a, cmd
	}

	var cmd tea.Cmd
	switch a.view {
	case viewHome:
		a.home, cmd = a.home.Update(msg)
	case viewHall:
		a.hall, cmd = a.hall.Update(msg)
	case viewGrimoire:
		a.grimoire, cmd = a.grimoire.Update(msg)
	case viewYou:
		a.you, cmd = a.you.Update(msg)
	case viewCreate:
		a.create, cmd = a.create.Update(msg)
	}

	return a, cmd
}

func (a App) isEditing() bool {
	switch a.view {
	case viewGrimoire:
		return a.grimoire.editing
	case viewCreate:
		return true
	case viewHall:
		return a.hall.inputFocused
	case viewYou:
		return a.you.wsState != wsNormal
	}
	return false
}

func (a App) View() string {
	// Header: centered shimmer logo
	logo := renderShimmerLogo(a.frame)

	// Stats line below logo
	statsLine := ""
	if a.me != nil {
		parts := []string{}
		if a.me.CardNumber > 0 {
			parts = append(parts, fmt.Sprintf("#%d", a.me.CardNumber))
		}
		if a.stats != nil {
			parts = append(parts, fmt.Sprintf("%d forged", a.stats.SpellsForged))
			parts = append(parts, fmt.Sprintf("%.0f%% accepted", a.stats.AcceptanceRate*100))
		}
		if a.me.GuildID != "" {
			parts = append(parts, GuildStyle(a.me.GuildID).Render(a.me.GuildID))
		}
		if len(parts) > 0 {
			statsLine = metaStyle.Render(strings.Join(parts, " . "))
		}
	}

	// Center the logo within terminal width
	logoWidth := lipgloss.Width(logo)
	logoPad := (a.width - logoWidth) / 2
	if logoPad < 0 {
		logoPad = 0
	}
	header := strings.Repeat(" ", logoPad) + logo

	if statsLine != "" {
		statsWidth := lipgloss.Width(statsLine)
		statsPad := (a.width - statsWidth) / 2
		if statsPad < 0 {
			statsPad = 0
		}
		header += "\n" + strings.Repeat(" ", statsPad) + statsLine
	} else {
		header += "\n"
	}

	// Tab bar: 1 Home  2 Hall  3 Grimoire  4 You
	type tabEntry struct {
		key  string
		name string
		v    view
	}
	tabs := []tabEntry{
		{"1", "Home", viewHome},
		{"2", "Hall", viewHall},
		{"3", "Grimoire", viewGrimoire},
		{"4", "You", viewYou},
	}

	// Tab bar: 4 equal-width columns spread across terminal (matches mockup flex:1)
	colWidth := a.width / len(tabs)
	var tabBar strings.Builder
	for _, t := range tabs {
		var label string
		if t.v == a.view {
			label = accentStyle.Render(t.key) + " " + selectedStyle.Underline(true).Render(t.name)
		} else {
			label = metaStyle.Render(t.key) + " " + dimStyle.Render(t.name)
		}
		// Hall tab: presence badge
		if t.v == viewHall && a.hall.presenceCount > 0 {
			label += " " + presenceDotStyle.Render("●") + dimStyle.Render(fmt.Sprintf("%d", a.hall.presenceCount))
		}
		// Center label within its column
		labelWidth := lipgloss.Width(label)
		leftPad := (colWidth - labelWidth) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		rightPad := colWidth - labelWidth - leftPad
		if rightPad < 0 {
			rightPad = 0
		}
		tabBar.WriteString(strings.Repeat(" ", leftPad) + label + strings.Repeat(" ", rightPad))
	}
	centeredTabs := tabBar.String()

	// Body
	var body string
	var help string
	switch a.view {
	case viewHome:
		body = a.home.View()
		help = " " + helpEntry("1-4", "tabs") + "  " + helpEntry("/", "search") + "  " + helpEntry("n", "forge") + "  " + helpEntry("enter", "ask") + "  " + helpEntry("h", "help") + "  " + helpEntry("q", "quit")
	case viewHall:
		body = a.hall.View()
		if a.hall.inputFocused {
			help = " " + helpEntry("1-4", "tabs") + "  " + helpEntry("enter", "send") + "  " + helpEntry("esc", "nav")
		} else {
			help = " " + helpEntry("1-4", "tabs") + "  " + helpEntry("j/k", "scroll") + "  " + helpEntry("enter", "type") + "  " + helpEntry("h", "help") + "  " + helpEntry("q", "quit")
		}
	case viewGrimoire:
		body = a.grimoire.View()
		if a.grimoire.detail {
			help = " " + helpEntry("1-4", "tabs") + "  " + helpEntry("u", "upvote") + "  " + helpEntry("c", "copy") + "  " + helpEntry("s", "save") + "  " + helpEntry("p", "peek") + "  " + helpEntry("esc", "back")
		} else {
			help = " " + helpEntry("1-4", "tabs") + "  " + helpEntry("j/k", "nav") + "  " + helpEntry("/", "search") + "  " + helpEntry("w", "toggle") + "  " + helpEntry("t", "tag") + "  " + helpEntry("s", "sort") + "  " + helpEntry("n", "forge") + "  " + helpEntry("h", "help") + "  " + helpEntry("q", "quit")
		}
	case viewYou:
		body = a.you.View()
		help = " " + helpEntry("1-4", "tabs") + "  " + a.you.helpKeys()
	case viewCreate:
		body = a.create.View()
		help = " " + helpEntry("tab", "next") + "  " + helpEntry("h/l", "tag") + "  " + helpEntry("ctrl+s", "submit") + "  " + helpEntry("esc", "cancel")
	}

	// Peek overlay
	if a.peekOpen {
		body = a.peek.View()
		help = " " + helpEntry("f", "follow") + "  " + helpEntry("esc", "close")
	}

	// Help overlay
	if a.helpOpen {
		body = helpView(a.helpCursor)
		help = " " + helpEntry("j/k", "nav") + "  " + helpEntry("enter", "open") + "  " + helpEntry("esc", "close")
	}

	// Input bar: persistent on Home (placeholder), Hall has its own inside body
	var inputBar string
	switch a.view {
	case viewHome:
		inputBar = " " + inputPromptStyle.Render("> ") + inputPlaceholderStyle.Render("ask the grimoire...")
	default:
		inputBar = ""
	}
	if a.peekOpen {
		inputBar = ""
	}

	// Chrome budget: header(2) + tabs(1) + input(1) + help(1) = 5 lines + body
	chrome := 5
	body = strings.TrimRight(truncateToHeight(body, a.height-chrome), "\n")

	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s", header, centeredTabs, body, inputBar, help)
}
