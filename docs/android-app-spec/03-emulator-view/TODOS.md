# 03-emulator-view TODO

Phases and actionable items for terminal rendering and VT-100 escape sequence parsing.

## Phase 1: Core VT-100 Parser

- [ ] **Implement VT-100 escape sequence parser** — Parse CSI, OSC, ESC sequences into structured commands (cursor movements, colors, erases). Completion: Parser handles all standard VT-100/VT-220 sequences without errors.

- [ ] **Create ANSI color/state mapper** — Map ANSI SGR codes to terminal state (foreground, background, bold, italic, underline). Completion: All 16 basic + 256 extended color codes map correctly.

- [ ] **Build character cell data structure** — Define Cell struct with char, fg/bg color, attributes (bold, italic, underline, blink, inverse). Completion: Cell struct supports all required attributes.

- [ ] **Implement screen buffer** — Create 2D array of Cell objects representing terminal screen. Completion: Buffer supports resize, clear, scroll.

## Phase 2: Terminal State Management

- [ ] **Implement cursor position tracking** — Track cursor row, column with bounds checking. Completion: Cursor moves correctly with all movement commands.

- [ ] **Add scroll region support** — Handle top/bottom scroll margins for partial screen scrolling. Completion: Scroll regions work correctly with apps like vim.

- [ ] **Implement line wrapping** — Toggle auto-wrap mode, handle wrap at column 80/132. Completion: Wrapped lines display correctly.

- [ ] **Build input state machine** — Parse partial escape sequences for input (e.g., arrow keys as CSI sequences). Completion: All standard arrow/modifier keys recognized.

## Phase 3: Rendering Engine

- [ ] **Create Canvas-based renderer** — Render screen buffer to Android Canvas with monospace font. Completion: All characters render at correct positions.

- [ ] **Implement efficient redraw** — Track dirty regions to minimize full redraws. Completion: Partial updates work, no flicker.

- [ ] **Add Unicode/ligature support** — Handle wide characters (CJK), emoji rendering. Completion: Wide chars take 2 columns, emoji renders correctly.

- [ ] **Implement cursor blink** — Configurable cursor blink, block/underline/caret styles. Completion: Cursor blinks at default 500ms interval.

## Phase 4: Integration & Features

- [ ] **Connect PTY input/output** — Wire parser to pseudo-terminal for bidirectional data flow. Completion: Shell commands produce correct terminal output.

- [ ] **Add touch selection** — Implement text selection via touch drag. Completion: Selected text copied to clipboard.

- [ ] **Implement scrollback buffer** — Store history lines beyond visible screen. Completion: Scrollback configurable, scroll works with touch.

- [ ] **Add mouse support** — Parse mouse escape sequences (X10, UTF-8, SGR). Completion: Mouse clicks report to application.

## Phase 5: Polish & Performance

- [ ] **Optimize for low-latency rendering** — Target <16ms frame time for 60fps. Completion: Smooth scrolling without lag.

- [ ] **Add accessibility support** — Screen reader announcements for terminal content. Completion: TalkBack reads selected line.

- [ ] **Implement font sizing** — Dynamic font scaling, maintain aspect ratio. Completion: Font size adjustable via pinch-zoom.

- [ ] **Add themes** — Support light/dark terminal themes + custom color schemes. Completion: At least 3 built-in themes available.
