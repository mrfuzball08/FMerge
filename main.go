package main

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ── Color palette variables ──────────────────────────────────────────────────
// Defined in palette.go

// ── Pre-allocated styles (avoids per-frame allocation) ──────────────────────
var (
	userLabelStyle        lipgloss.Style
	userTextStyle         lipgloss.Style
	botLabelStyle         lipgloss.Style
	placeholderStyle      lipgloss.Style
	promptStyle           lipgloss.Style
	cursorStyle           lipgloss.Style
	inputTextStyle        lipgloss.Style
	placeholderInputStyle lipgloss.Style
	hintStyle             lipgloss.Style
	divStyle              lipgloss.Style
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

	// Per-message render cache — avoids re-running Lip Gloss on old messages.
	renderedLines []string
	renderedWidth int
}

type model struct {
	width        int
	height       int
	input        string
	messages     []chatMsg
	scrollOffset int

	// History navigation state.
	// historyIndex is the index into the user-message history we're browsing;
	// -1 means we're at the live draft. inputDraft saves whatever the user
	// had typed before pressing Up.
	historyIndex int
	inputDraft   string

	// Files loaded via /read commands
	loadedFiles []loadedFile

	// Cached rendered lines — invalidated when messages change.
	cachedLines    []string
	cachedMsgCount int
	cachedInnerW   int

	// Cached header — rebuilt only on window resize.
	headerStr    string
	headerHeight int
}

// ── Bubbletea interface ──────────────────────────────────────────────────────
func (m model) Init() tea.Cmd {
	return nil
}

// userMessages returns the content of every roleUser message in order.
// Called only during Up/Down navigation — O(n) but n is small and the
// result is never stored, so there's zero per-frame allocation.
func userMessages(msgs []chatMsg) []string {
	out := make([]string, 0, len(msgs))
	for _, m := range msgs {
		if m.role == roleUser {
			out = append(out, m.content)
		}
	}
	return out
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Rebuild header cache on resize.
		m.headerStr = m.renderHeader(msg.Width)
		m.headerHeight = lipgloss.Height(m.headerStr)
		// Width changed → cached renders are stale.
		m.cachedMsgCount = 0

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "enter":
			text := strings.TrimSpace(m.input)
			if text != "" {
				m.input = ""
				m.inputDraft = ""
				m.historyIndex = -1
				m.scrollOffset = 0

				if strings.HasPrefix(text, "/") {
					parts := strings.SplitN(text, " ", 2)
					switch parts[0] {
										case "/read":
						if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
							m.messages = append(m.messages, chatMsg{
								role:    roleAssistant,
								content: "Usage: /read <filepath>",
							})
						} else {
							path := strings.TrimSpace(parts[1])
							m.messages = append(m.messages, chatMsg{
								role:    roleUser,
								content: "/read " + path,
							})
							m.messages = append(m.messages, chatMsg{
								role:    roleAssistant,
								content: fmt.Sprintf("Reading %s...", path),
							})
							cmds = append(cmds, readFilesCmd([]string{path}))
						}
					case "/theme":
						if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
							m.messages = append(m.messages, chatMsg{
								role:    roleUser,
								content: "/theme",
							})
							m.messages = append(m.messages, chatMsg{
								role:    roleAssistant,
								content: "Usage: /theme <theme-name>\nType /theme-list to see available themes.",
							})
						} else {
							themeName := strings.TrimSpace(parts[1])
							m.messages = append(m.messages, chatMsg{
								role:    roleUser,
								content: "/theme " + themeName,
							})
							t, err := LoadThemeByName(themeName)
							if err != nil {
								m.messages = append(m.messages, chatMsg{
									role:    roleAssistant,
									content: fmt.Sprintf("Error loading theme: %v", err),
								})
							} else {
								_ = SaveActiveTheme(themeName)
								// Force invalidation of render cache
								m.cachedMsgCount = 0
								for i := range m.messages {
									m.messages[i].renderedWidth = 0
								}
								// Force header rebuild
								m.headerStr = m.renderHeader(m.width)
								m.headerHeight = lipgloss.Height(m.headerStr)
								
								m.messages = append(m.messages, chatMsg{
									role:    roleAssistant,
									content: fmt.Sprintf("Theme successfully changed to '%s'!", t.Name),
								})
							}
						}
					case "/theme-list":
						m.messages = append(m.messages, chatMsg{
							role:    roleUser,
							content: "/theme-list",
						})
						list, err := LoadThemesList()
						if err != nil {
							m.messages = append(m.messages, chatMsg{
								role:    roleAssistant,
								content: fmt.Sprintf("Error loading theme list: %v", err),
							})
						} else {
							var sb strings.Builder
							sb.WriteString("Available themes:\n\n")
							for _, name := range list {
								sb.WriteString(fmt.Sprintf(" • %s\n", name))
							}
							sb.WriteString(" • custom (configured via color_config.json)\n\n")
							sb.WriteString("Apply with: /theme <name>")
							m.messages = append(m.messages, chatMsg{
								role:    roleAssistant,
								content: sb.String(),
							})
						}
					default:
						m.messages = append(m.messages, chatMsg{
							role:    roleAssistant,
							content: fmt.Sprintf("Unknown command: %s\nType /help for available commands.", parts[0]),
						})
					}
				} else {
					m.messages = append(m.messages, chatMsg{role: roleUser, content: text})
					// Placeholder response — swap for real backend later.
					m.messages = append(m.messages, chatMsg{
						role:    roleAssistant,
						content: "Processing: \"" + text + "\"\n\nFMerge response will appear here once the backend is connected.",
					})
				}
			}

		case "up":
			// Navigate backward through sent-message history.
			history := userMessages(m.messages)
			if len(history) == 0 {
				break
			}
			if m.historyIndex == -1 {
				// Save whatever the user has typed so far.
				m.inputDraft = m.input
				m.historyIndex = len(history) - 1
			} else if m.historyIndex > 0 {
				m.historyIndex--
			}
			m.input = history[m.historyIndex]

		case "down":
			// Navigate forward through sent-message history.
			if m.historyIndex == -1 {
				break // already at live draft, nothing to do
			}
			history := userMessages(m.messages)
			if m.historyIndex < len(history)-1 {
				m.historyIndex++
				m.input = history[m.historyIndex]
			} else {
				// Reached the end — restore the live draft.
				m.historyIndex = -1
				m.input = m.inputDraft
			}

		case "ctrl+up":
			m.scrollOffset += 3

		case "ctrl+down":
			m.scrollOffset -= 3

		case "pageup":
			// Half-viewport jump feels natural
			step := m.chatAreaHeight() / 2
			if step < 1 {
				step = 1
			}
			m.scrollOffset += step

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

	case tea.PasteMsg:
		// Bracketed paste: the terminal wraps Ctrl+V content in escape
		// sequences that Bubble Tea decodes into this message. Append
		// directly to the input buffer, stripping any embedded newlines
		// so a multi-line paste becomes a single input line.
		pasted := strings.ReplaceAll(msg.String(), "\n", " ")
		pasted = strings.ReplaceAll(pasted, "\r", "")
		if pasted != "" {
			m.input += pasted
		}

	case filesLoadedMsg:
		if msg.err != nil {
			m.messages = append(m.messages, chatMsg{
				role:    roleAssistant,
				content: "Error loading file: " + msg.err.Error(),
			})
			break
		}
		m.loadedFiles = append(m.loadedFiles, msg.files...)
		for _, f := range msg.files {
			content := fmt.Sprintf("Loaded %s (%d lines)\n\n%s", f.path, len(f.lines), strings.Join(f.lines, "\n"))
			m.messages = append(m.messages, chatMsg{
				role:    roleAssistant,
				content: content,
			})
		}

	case tea.MouseWheelMsg:
		// Use the typed Button field — no string conversion needed.
		if msg.Button == tea.MouseWheelUp {
			m.scrollOffset += 3
		} else if msg.Button == tea.MouseWheelDown {
			m.scrollOffset -= 3
		}
	}

	// ── Rebuild rendered-line cache if content or width changed ─────────
	innerW := m.width - 6
	if innerW < 10 {
		innerW = 10
	}
	if len(m.messages) != m.cachedMsgCount || innerW != m.cachedInnerW {
		m.rebuildLineCache(innerW)
	}

	// ── Clamp scroll AFTER cache rebuild so we use fresh line counts ────
	// Both lower AND upper bound — prevents infinite dead-scroll accumulation.
	maxScroll := len(m.cachedLines) - m.chatAreaHeight()
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scrollOffset > maxScroll {
		m.scrollOffset = maxScroll
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}

	if len(cmds) > 0 {
		return m, tea.Batch(cmds...)
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
	// Request keyboard enhancements so the terminal distinguishes
	// modifier+special-key combos (e.g. shift+up vs plain up).
	v.KeyboardEnhancements = tea.KeyboardEnhancements{}
	return v
}

// chatAreaHeight returns the number of terminal lines available for messages.
// Uses the cached header height to avoid re-rendering the header.
func (m model) chatAreaHeight() int {
	h := m.height
	if h < 6 {
		h = 24
	}
	headerH := m.headerHeight
	if headerH == 0 {
		// Fallback before first WindowSizeMsg.
		headerH = 10
	}
	const inputH = 3
	avail := h - headerH - inputH
	if avail < 1 {
		avail = 1
	}
	return avail
}

// rebuildLineCache renders all messages into flat lines and caches the result.
// Called from Update() only when the message list or terminal width changes.
// Uses a per-message render cache: only messages that haven't been rendered
// at the current width are processed through Lip Gloss, making the cost of
// adding a new message O(1) instead of O(N).
func (m *model) rebuildLineCache(innerW int) {
	responseBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colPrimary).
		Padding(0, 1).
		Width(innerW).
		Foreground(colResponse)

	lines := make([]string, 0, len(m.messages)*4)
	for i := range m.messages {
		cm := &m.messages[i]
		// Only re-render if the cached width doesn't match or cache is empty.
		if cm.renderedWidth != innerW || len(cm.renderedLines) == 0 {
			var msgLines []string
			switch cm.role {
			case roleUser:
				msgLines = []string{
					"  " + userLabelStyle.Render("You"),
					"  " + userTextStyle.Render(cm.content),
					"",
				}
			case roleAssistant:
				box := responseBoxStyle.Render(cm.content)
				split := strings.Split(box, "\n")
				msgLines = make([]string, 0, len(split)+2)
				msgLines = append(msgLines, "  "+botLabelStyle.Render("FMerge"))
				for _, l := range split {
					msgLines = append(msgLines, "  "+l)
				}
				msgLines = append(msgLines, "")
			}
			cm.renderedLines = msgLines
			cm.renderedWidth = innerW
		}
		lines = append(lines, cm.renderedLines...)
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

	// ── Header (use cache — rebuilt only on resize) ─────────────────────────
	header := m.headerStr
	headerH := m.headerHeight
	if header == "" {
		// Fallback before first WindowSizeMsg.
		header = m.renderHeader(w)
		headerH = lipgloss.Height(header)
	}

	// ── Input bar (3 lines) ──────────────────────────────────────────────────
	divLine := divStyle.Render(strings.Repeat("━", w))
	cursor := cursorStyle.Render("█")
	var inputContent string
	if m.input == "" {
		inputContent = cursor + placeholderInputStyle.Render("Type your message...")
	} else {
		inputContent = inputTextStyle.Render(m.input) + cursor
	}
	prompt := promptStyle.Render(" ▸ ")
	hintText := hintStyle.Render("  Enter · ↑↓ history · Ctrl+↑↓ scroll · Ctrl+V paste · Ctrl+C quit")
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
	divider := divStyle.Render(strings.Repeat("━", w))

	tagText := lipgloss.NewStyle().
		Foreground(colPrimary).
		Background(colBgDark).
		Bold(true).
		Italic(true).
		Render("True TUI for true mergers")

	verText := lipgloss.NewStyle().
		Foreground(colMuted).
		Background(colBgDark).
		Render("v0.1.0")

	gap := lipgloss.NewStyle().Background(colBgDark).Render("  ")
	combinedText := lipgloss.JoinHorizontal(lipgloss.Center, tagText, gap, verText)

	if w < minArtWidth {
		// ── Compact header for narrow terminals ──────────────────────────
		// We set an explicit identical width and alignment on both elements
		// to prevent JoinVertical from generating unstyled padding space tiles.
		brand := lipgloss.NewStyle().
			Foreground(colSecondary).
			Background(colBgDark).
			Bold(true).
			Width(12).
			Align(lipgloss.Center).
			Render("⬡ FMERGE")

		verTextPadded := lipgloss.NewStyle().
			Foreground(colMuted).
			Background(colBgDark).
			Width(12).
			Align(lipgloss.Center).
			Render("v0.1.0")

		headerBlock := lipgloss.NewStyle().
			Background(colBgDark).
			Width(w).
			Align(lipgloss.Center).
			PaddingTop(1).
			PaddingBottom(1).
			Render(lipgloss.JoinVertical(lipgloss.Center, brand, verTextPadded))

		return headerBlock + "\n" + divider
	}

	// ── Full art header ──────────────────────────────────────────────────
	fmergeArt := strings.Join(fmergeArtLines, "\n")
	artStyled := lipgloss.NewStyle().
		Foreground(colSecondary).
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
	if err := LoadColorPalette(); err != nil {
		fmt.Fprintln(os.Stderr, "warning: could not load color palette:", err)
	}
	p := tea.NewProgram(model{historyIndex: -1})
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
