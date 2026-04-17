# ⬡ FMERGE

**FMerge** is a modern, chat-based Terminal User Interface (TUI) designed for file merging tasks. It features a responsive design, rich aesthetics, and a "chat-first" workflow.

## Overview

FMerge delivers a premium terminal experience with a focus on speed and clarity. By bypassing traditional menu structures, it places the user directly into an interactive chat environment optimized for complex merging workflows.

## Features

- **Direct Boot**: Bypasses splash screens to drop users directly into the active chat interface.
- **Responsive Header**:
    - High-fidelity ASCII art on wide terminals.
    - Minimalist "⬡ FMERGE" adaptive layout for narrow screens (< 58 chars).
- **Premium Aesthetics**: Uses a curated workspace palette (`#F4810A` Orange, `#FFBE3B` Amber) with a deep, focused background (`#1A0F00`).
- **Smooth Performance**:
    - **Cached Rendering**: Message lines are cached and only rebuilt on content or size changes, ensuring smooth scrolling.
    - **Optimized UI**: Pre-allocated style tokens and typed event handling for zero-overhead interactions.
- **Interactive Layout**:
    - Supports full mouse wheel scrolling.
    - Alt-screen mode for a clean, flicker-free experience.
    - Pinned input bar with real-time feedback.

## Key Bindings

| Key | Action |
| --- | --- |
| `Enter` | Send message |
| `Up` / `Down` | Scroll message history (3 lines) |
| `PgUp` / `PgDn` | Scroll half-viewport |
| `Home` / `End` | Jump to start / end of history |
| `Ctrl+W` | Delete word |
| `Ctrl+C` | Quit |
| `Mouse Wheel` | Smooth scrolling |

## Tech Stack

- **Language**: Go
- **UI Framework**: [Bubble Tea (v2)](https://github.com/charmbracelet/bubbletea)
- **Styling**: [Lip Gloss (v2)](https://github.com/charmbracelet/lipgloss)

## Development

### Prerequisites

- Go 1.25.8 or later

### Build & Run

```bash
# Build the binary
go build -o merger main.go

# Run the application
./merger
```
