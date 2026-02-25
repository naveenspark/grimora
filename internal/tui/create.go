package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/naveenspark/grimora/pkg/client"
	"github.com/naveenspark/grimora/pkg/domain"
)

type createField int

const (
	fieldText createField = iota
	fieldTag
	fieldModel
	fieldContext
	numFields

	defaultModel = "claude-opus-4"
)

type createModel struct {
	client    *client.Client
	fields    [numFields]string
	focus     createField
	err       error
	statusMsg string
	submitted bool
}

type spellCreatedMsg struct {
	spell *domain.Spell
	err   error
}

func newCreateModel(c *client.Client) createModel {
	m := createModel{client: c}
	m.fields[fieldModel] = defaultModel
	return m
}

func (m createModel) Init() tea.Cmd {
	return nil
}

func (m createModel) Update(msg tea.Msg) (createModel, tea.Cmd) {
	switch msg := msg.(type) {
	case spellCreatedMsg:
		m.submitted = false
		if msg.err != nil {
			m.err = msg.err
			m.statusMsg = "failed to create spell"
		} else {
			m.statusMsg = fmt.Sprintf("spell created: %s", msg.spell.ID.String()[:8])
			// Reset form
			m.fields = [numFields]string{}
			m.fields[fieldModel] = defaultModel
			m.focus = fieldText
		}
		return m, nil

	case tea.KeyMsg:
		return m.updateKeys(msg)
	}
	return m, nil
}

func (m createModel) updateKeys(msg tea.KeyMsg) (createModel, tea.Cmd) {
	m.statusMsg = ""
	m.err = nil

	switch msg.String() {
	case "ctrl+s":
		return m.submit()
	case "tab", "down":
		m.focus = (m.focus + 1) % numFields
	case "shift+tab", "up":
		m.focus = (m.focus - 1 + numFields) % numFields
	case "backspace":
		f := &m.fields[m.focus]
		*f = editRune(*f, "backspace")
	case "enter":
		if m.focus == fieldText {
			m.fields[fieldText] += "\n"
		} else {
			m.focus = (m.focus + 1) % numFields
		}
	default:
		key := msg.String()
		if m.focus == fieldTag {
			// Cycle through tags with h/l
			if key == "h" || key == "l" {
				tags := domain.ValidTags
				current := m.fields[fieldTag]
				idx := 0
				for i, t := range tags {
					if t == current {
						idx = i
						break
					}
				}
				if key == "l" {
					idx = (idx + 1) % len(tags)
				} else {
					idx = (idx - 1 + len(tags)) % len(tags)
				}
				m.fields[fieldTag] = tags[idx]
				return m, nil
			}
		}
		if len(key) == 1 {
			m.fields[m.focus] += key
		}
	}
	return m, nil
}

func (m createModel) submit() (createModel, tea.Cmd) {
	text := strings.TrimSpace(m.fields[fieldText])
	tag := m.fields[fieldTag]

	if text == "" {
		m.statusMsg = "text is required"
		return m, nil
	}
	if tag == "" {
		m.statusMsg = "tag is required (use h/l to select)"
		return m, nil
	}
	if !domain.ValidTag(tag) {
		m.statusMsg = "invalid tag"
		return m, nil
	}

	m.submitted = true
	req := client.CreateSpellRequest{
		Text:    text,
		Tag:     tag,
		Model:   m.fields[fieldModel],
		Context: m.fields[fieldContext],
	}

	return m, func() tea.Msg {
		spell, err := m.client.CreateSpell(context.Background(), req)
		return spellCreatedMsg{spell: spell, err: err}
	}
}

func (m createModel) View() string {
	var b strings.Builder

	labels := [numFields]string{"text", "tag", "model", "context"}

	for i := createField(0); i < numFields; i++ {
		label := labels[i]
		value := m.fields[i]
		cursor := " "
		style := metaStyle
		if i == m.focus {
			cursor = ">"
			style = selectedStyle
		}

		if i == fieldTag {
			fmt.Fprintf(&b, "%s %s: %s  (h/l to cycle)\n",
				cursor, style.Render(label), TagStyle(value).Render(value))
		} else {
			displayValue := value
			if i == m.focus {
				displayValue += "█"
			}
			fmt.Fprintf(&b, "%s %s: %s\n", cursor, style.Render(label), displayValue)
		}
	}

	b.WriteString("\n")
	if m.submitted {
		b.WriteString(dimStyle.Render("creating..."))
	} else if m.statusMsg != "" {
		b.WriteString(upvoteStyle.Render(m.statusMsg))
	}

	return b.String()
}
