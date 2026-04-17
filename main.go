package main

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ── Color palette ────────────────────────────────────────────────────────────
var (
	colOrange   = lipgloss.Color("#F4810A")
	colAmber    = lipgloss.Color("#FFBE3B")
	colMuted    = lipgloss.Color("#9E7D5A")
	colBgDark   = lipgloss.Color("#1A0F00")
	colSubtle   = lipgloss.Color("#3D3D3D")
	colResponse = lipgloss.Color("#E8D5B0")
)

// ── Pre-allocated styles (avoids per-frame allocation) ──────────────────────
var (
	userLabelStyle = lipgloss.NewStyle().Foreground(colAmber).Bold(true)
	userTextStyle  = lipgloss.NewStyle().Foreground(colAmber)
	botLabelStyle  = lipgloss.NewStyle().Foreground(colOrange).Bold(true)
	placeholderStyle = lipgloss.NewStyle().Foreground(colSubtle).Italic(true)
	promptStyle    = lipgloss.NewStyle().Foreground(colOrange).Bold(true)
	cursorStyle    = lipgloss.NewStyle().Foreground(colAmber)
	inputTextStyle = lipgloss.NewStyle().Foreground(colAmber)
	placeholderInputStyle = lipgloss.NewStyle().Foreground(colSubtle).Italic(true)
	hintStyle      = lipgloss.NewStyle().Foreground(colSubtle)
)

// ── ASCII header art ─────────────────────────────────────────────────────────
var fmergeArtLines = []string{
	`███████╗███╗   ███╗███████╗██████╗  ██████╗ ███████╗`,
	`██╔════╝████╗ ████║██╔════╝██╔══██╗██╔════╝ ██╔════╝`,
	`█████╗  ██╔████╔██║█████╗  ██████╔╝██║  ███╗█████╗  `,
	`██╔══╝  ██║╚██╔╝██║██╔══╝  ██╔══██╗██║   ██║██╔══╝  `,
	`██║     ██║ ╚═╝ ██║███████╗██║  ██║╚██████╔╝███████╗`,
	`╚═╝     ╚═╝     ╚═╝╚══════╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝`,
}

// Minimum width to show the full ASCII art (art is ~52 chars + padding)
const minArtWidth = 58

// ── Domain types ─────────────────────────────────────────────────────────────
type msgRole int

const (
	roleUser msgRole = iota
	roleAssistant
)

type chatMsg struct {
	role    msgRole
	content string
}

type model struct {
	width        int
	height       int
	input        string
	messages     []chatMsg
	scrollOffset int

	// Cached rendered lines — invalidated when messages change.
	cachedLines     []string
	cachedMsgCount  int
	cachedInnerW    int
}

// ── Bubbletea interface ──────────────────────────────────────────────────────
func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Width changed → cached renders are stale.
		m.cachedMsgCount = 0

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "enter":
			text := strings.TrimSpace(m.input)
			if text != "" {
				m.messages = append(m.messages, chatMsg{role: roleUser, content: m.input})
				// Placeholder response — swap for real backend later.
				m.messages = append(m.messages, chatMsg{
					role:    roleAssistant,
					content: "Processing: \"" + text + "\"\n\nFMerge response will appear here once the backend is connected.",
				})
				m.input = ""
				m.scrollOffset = 0
			}

		case "up":
			m.scrollOffset += 3

		case "pageup":
			// Half-viewport jump feels natural
			step := m.chatAreaHeight() / 2
			if step < 1 {
				step = 1
			}
			m.scrollOffset += step

		case "down":
			m.scrollOffset -= 3

		case "pagedown":
			step := m.chatAreaHeight() / 2
			if step < 1 {
				step = 1
			}
			m.scrollOffset -= step

		case "home":
			// Jump to oldest messages
			m.scrollOffset = len(m.cachedLines)

		case "end":
			// Jump to newest messages
			m.scrollOffset = 0

		case "backspace", "ctrl+h":
			if len(m.input) > 0 {
				runes := []rune(m.input)
				m.input = string(runes[:len(runes)-1])
			}

		case "ctrl+w":
			trimmed := strings.TrimRight(m.input, " ")
			if idx := strings.LastIndex(trimmed, " "); idx >= 0 {
				m.input = m.input[:idx+1]
			} else {
				m.input = ""
			}

		default:
			if k := msg.Key(); k.Text != "" {
				m.input += k.Text
			}
		}

	case tea.MouseWheelMsg:
		// Use the typed Button field — no string conversion needed.
		if msg.Button == tea.MouseWheelUp {
			m.scrollOffset += 3
		} else if msg.Button == tea.MouseWheelDown {
			m.scrollOffset -= 3
		}
	}

	// ── Clamp scroll here (in Update) so it actually persists ───────────
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}

	// ── Rebuild rendered-line cache if content or width changed ─────────
	innerW := m.width - 6
	if innerW < 10 {
		innerW = 10
	}
	if len(m.messages) != m.cachedMsgCount || innerW != m.cachedInnerW {
		m.rebuildLineCache(innerW)
	}

	return m, nil
}

func (m model) View() tea.View {
	v := tea.NewView(m.renderChat())
	// Alt screen gives us a clean full-window canvas that handles resize
	// properly — no stale frames left behind.
	v.AltScreen = true
	// Enable mouse cell motion so scroll wheel actions report cleanly
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// chatAreaHeight returns the number of terminal lines available for messages.
func (m model) chatAreaHeight() int {
	w := m.width
	if w < 20 {
		w = 80
	}
	h := m.height
	if h < 6 {
		h = 24
	}
	headerH := lipgloss.Height(m.renderHeader(w))
	const inputH = 3
	avail := h - headerH - inputH
	if avail < 1 {
		avail = 1
	}
	return avail
}

// rebuildLineCache renders all messages into flat lines and caches the result.
// Called from Update() only when the message list or terminal width changes.
func (m *model) rebuildLineCache(innerW int) {
	responseBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colOrange).
		Padding(0, 1).
		Width(innerW).
		Foreground(colResponse)

	lines := make([]string, 0, len(m.messages)*4)
	for _, cm := range m.messages {
		switch cm.role {
		case roleUser:
			lines = append(lines,
				"  "+userLabelStyle.Render("You"),
				"  "+userTextStyle.Render(cm.content),
				"",
			)
		case roleAssistant:
			box := responseBoxStyle.Render(cm.content)
			lines = append(lines, "  "+botLabelStyle.Render("FMerge"))
			for _, l := range strings.Split(box, "\n") {
				lines = append(lines, "  "+l)
			}
			lines = append(lines, "")
		}
	}

	m.cachedLines = lines
	m.cachedMsgCount = len(m.messages)
	m.cachedInnerW = innerW
}

// ── Chat renderer ────────────────────────────────────────────────────────────
func (m model) renderChat() string {
	w := m.width
	if w < 20 {
		w = 80
	}
	h := m.height
	if h < 6 {
		h = 24
	}

	divStyle := lipgloss.NewStyle().Foreground(colOrange)

	// ── Header ───────────────────────────────────────────────────────────────
	header := m.renderHeader(w)
	headerH := lipgloss.Height(header)

	// ── Input bar (3 lines) ──────────────────────────────────────────────────
	divLine := divStyle.Render(strings.Repeat("━", w))
	cursor := cursorStyle.Render("█")
	var inputContent string
	if m.input == "" {
		inputContent = placeholderInputStyle.Render("Type your message...") + cursor
	} else {
		inputContent = inputTextStyle.Render(m.input) + cursor
	}
	prompt := promptStyle.Render(" ▸ ")
	hintText := hintStyle.Render("  Enter to send · Ctrl+C to quit")
	inputBar := divLine + "\n" + prompt + inputContent + "\n" + hintText
	const inputH = 3

	// ── Messages ─────────────────────────────────────────────────────────────
	innerW := w - 6
	if innerW < 10 {
		innerW = 10
	}

	var msgLines []string

	if len(m.messages) == 0 {
		msgLines = []string{
			"",
			"  " + placeholderStyle.Render("No messages yet. Type something below and press Enter ↵"),
		}
	} else {
		msgLines = m.cachedLines
	}

	// Available height for messages area
	availH := h - headerH - inputH
	if availH < 1 {
		availH = 1
	}

	totalLines := len(msgLines)
	maxScroll := totalLines - availH
	if maxScroll < 0 {
		maxScroll = 0
	}

	scrollOff := m.scrollOffset
	if scrollOff > maxScroll {
		scrollOff = maxScroll
	}
	if scrollOff < 0 {
		scrollOff = 0
	}

	// Calculate slice bounds
	startIndex := totalLines - availH - scrollOff
	if startIndex < 0 {
		startIndex = 0
	}
	endIndex := startIndex + availH
	if endIndex > totalLines {
		endIndex = totalLines
	}

	visible := msgLines[startIndex:endIndex]

	// Pad with blank lines at the bottom to fill exactly availH lines.
	// This ensures total output is always exactly h lines (alt screen needs this).
	padded := make([]string, availH)
	copy(padded, visible)
	// remaining entries are already "" (zero value)

	messagesArea := strings.Join(padded, "\n")

	// ── Assemble ─────────────────────────────────────────────────────────────
	return header + "\n" + messagesArea + "\n" + inputBar
}

// renderHeader builds the header — full art for wide terminals, compact for narrow.
func (m model) renderHeader(w int) string {
	divStyle := lipgloss.NewStyle().Foreground(colOrange)
	divider := divStyle.Render(strings.Repeat("━", w))

	tagText := lipgloss.NewStyle().
		Foreground(colOrange).
		Background(colBgDark).
		Bold(true).
		Italic(true).
		Render("Fast. Flexible. File Merging, Redefined.")
	
	verText := lipgloss.NewStyle().
		Foreground(colMuted).
		Background(colBgDark).
		Render("v0.1.0")
	
	gap := lipgloss.NewStyle().Background(colBgDark).Render("  ")
	combinedText := lipgloss.JoinHorizontal(lipgloss.Center, tagText, gap, verText)

	if w < minArtWidth {
		// ── Compact header for narrow terminals ──────────────────────────
		brand := lipgloss.NewStyle().
			Foreground(colAmber).
			Background(colBgDark).
			Bold(true).
			Render("⬡ FMERGE")

		headerBlock := lipgloss.NewStyle().
			Background(colBgDark).
			Width(w).
			Align(lipgloss.Center).
			PaddingTop(1).
			PaddingBottom(1).
			Render(lipgloss.JoinVertical(lipgloss.Center, brand, verText))

		return headerBlock + "\n" + divider
	}

	// ── Full art header ──────────────────────────────────────────────────
	fmergeArt := strings.Join(fmergeArtLines, "\n")
	artStyled := lipgloss.NewStyle().
		Foreground(colAmber).
		Background(colBgDark).
		Bold(true).
		Render(fmergeArt)
	
	artWidth := lipgloss.Width(artStyled)
	
	// Pad the combined text to exactly match the art width.
	// This prevents JoinVertical from adding unstyled spaces!
	combinedTextPadded := lipgloss.NewStyle().
		Background(colBgDark).
		Width(artWidth).
		Align(lipgloss.Center).
		Render(combinedText)

	innerBlock := lipgloss.JoinVertical(lipgloss.Center,
		artStyled,
		combinedTextPadded,
	)

	headerBlock := lipgloss.NewStyle().
		Background(colBgDark).
		Width(w).
		Align(lipgloss.Center).
		PaddingTop(1).
		PaddingBottom(1).
		Render(innerBlock)

	return headerBlock + "\n" + divider
}

// ── Entry point ──────────────────────────────────────────────────────────────
func main() {
	p := tea.NewProgram(model{})
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
