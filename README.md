# ⬡ FMERGE

**FMerge** is a modern, chat-based Terminal User Interface (TUI) designed for file merging tasks (not yet implemented, lol). It features a responsive design, rich aesthetics, and a "chat-first" interface with a command-driven workflow.

## Overview

FMerge delivers a premium terminal experience with a focus on speed and clarity. By bypassing traditional menu structures, it places the user directly into an interactive chat environment optimized for complex merging workflows.

## Features

- **Direct Boot**: Bypasses splash screens to drop users directly into the active chat interface.
- **Responsive Header**:
    - High-fidelity ASCII art on wide terminals.
    - Minimalist "⬡ FMERGE" adaptive layout for narrow screens (< 58 chars).
- **Premium Aesthetics**: Features a dynamic, JSON-driven color palette system. Comes with beautifully curated default themes (Classic Amber, Dracula, Nord, Sunset, Monochrome, and Gruvbox) which guarantee high-contrast legibility across all UI elements.
- **Dynamic Theme Engine**: Instantly switch themes on the fly using `/theme <name>` without restarting the application. The entire UI and message history repaints instantly.
- **Smooth Performance**:
    - **Per-Message Cache**: Individual message blocks are cached using Lip Gloss. Adding new messages is O(1) instead of re-rendering the entire history, eliminating lag on large files.
    - **Header Cache**: The complex ASCII header is cached on window resize to prevent 2× per-frame rendering overhead.
    - **Optimized UI**: Pre-allocated style tokens, correct scroll bound clamping, and typed event handling for zero-overhead interactions and perfectly snappy scrolling.
- **Interactive Layout**:
    - Supports full mouse wheel scrolling.
    - Alt-screen mode for a clean, flicker-free experience.
    - Pinned input bar with real-time feedback.
    - Flawless adaptive layouts without background rendering artifacts on narrow terminals.
- **Slash Commands**: 
    - `/read <path>`: Load markdown files on-the-fly.
    - `/theme <name>`: Instantly switch the active color palette.
    - `/theme-list`: Show all available themes.

## Key Bindings

| Key | Action |
| --- | --- |
| `Enter` | Send message |
| `Up` / `Down` | Navigate message history (shell-style) |
| `Ctrl+Up` / `Ctrl+Down` | Scroll message area (3 lines) **(Currently broken)** |
| `PgUp` / `PgDn` | Scroll half-viewport |
| `Home` / `End` | Jump to start / end of history |
| `Ctrl+V` | Paste from clipboard (bracketed paste — no extra tools needed) |
| `Ctrl+W` | Delete word |
| `Ctrl+C` | Quit |
| `Mouse Wheel` | Smooth scrolling |

## Tech Stack

- **Language**: Go
- **UI Framework**: [Bubble Tea (v2)](https://github.com/charmbracelet/bubbletea)
- **Styling**: [Lip Gloss (v2)](https://github.com/charmbracelet/lipgloss)

## Project Structure

```
FMerge/
├── main.go               # TUI entry point, Bubble Tea model, slash commands, rendering
├── files.go              # Async file reading command, path resolution, validation
├── md-merge.go           # Low-level markdown line reader
├── palette.go            # Dynamic color theme loader and Lip Gloss style manager
├── colorschemes.json     # (Generated) Available themes
├── color_config.json     # (Generated) Active theme selection and custom colors
├── local_docs/
│   └── CURRENT_STATE.md
├── README.md
├── go.mod
├── go.sum
└── .gitignore
```

## Development

### Prerequisites

- Go 1.25.8 or later

### Build & Run

```bash
# Build the binary
go build -o merger .

# Run the application
./merger
```
