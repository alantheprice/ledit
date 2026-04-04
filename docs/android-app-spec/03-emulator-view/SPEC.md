# 03-emulator-view Component Specification

## Component Overview

The `emulator-view` component is responsible for terminal rendering and visual display within the Android terminal emulator application. It handles the parsing and interpretation of VT-100/VT-220 escape sequences and renders terminal output to a Canvas-based view. This component serves as the visual layer between the terminal buffer and the user, providing real-time text rendering with support for colors, cursor positioning, and various text attributes.

The component draws heavily from the implementation patterns established in the Android-Terminal-Emulator project by jackpal, adapting the core terminal emulation logic for use within the broader application's architecture.

---

## Terminal Emulation Requirements

### Supported Terminal Standards

The component shall support:

- **VT-100**: Full compatibility with the DEC VT-100 terminal standard, including all standard control sequences and character sets
- **VT-220**: Extended support for the VT-220 standard, including the 8-bit character set and additional control functions
- **ANSI**: Compatibility with ANSI X3.64 escape sequences for broad compatibility with modern applications

### Character Set Support

- ASCII (7-bit)
- ISO-8859-1 (Latin-1)
- UTF-8 encoding for Unicode text

### Screen Dimensions

- Configurable rows and columns (default: 80x24)
- Support for variable font sizes
- Dynamic resize handling with terminal notification

---

## Rendering Approach

### Canvas-Based Rendering

The emulator-view uses a Canvas-based rendering architecture:

1. **SurfaceView**: Utilizes SurfaceView for efficient hardware-accelerated rendering
2. **Double Buffering**: Implements double buffering to prevent screen tearing and ensure smooth updates
3. **Bitmapped Font Rendering**: Characters are rendered as bitmap glyphs drawn directly to the Canvas
4. **Tile-based Updates**: Only repaints changed screen regions to optimize performance

### Rendering Pipeline

```
Terminal Buffer → Emulator View → Canvas → Display
     (data)      (processing)   (draw)  (output)
```

1. Terminal buffer updates with new characters/attributes
2. Emulator-view receives update notification
3. Changed regions are identified
4. Bitmap glyphs are drawn to Canvas for affected characters
5. Canvas content is displayed via SurfaceView

### Text Rendering

- Monospace font rendering (configurable font family)
- Bold text simulation via doubled glyph rendering
- Italic text simulation where supported by font
- Blinking cursor support

---

## Dependencies and Libraries

### External Libraries

The component references the terminal emulation implementation from the Android-Terminal-Emulator project (jackpal/Android-Terminal-Emulator):

**Key classes adapted from the source:**

- `TerminalEmulator` - Core VT-100/VT-220 emulation engine
- `TerminalBuffer` - Screen buffer management
- `TranscriptScreen` - Scrollback buffer handling
- `Screen` - Active screen display management
- `EscapeSequences` - Constants for escape sequence definitions

### Internal Dependencies

- Input method integration for keyboard input
- Session management for terminal process communication
- Settings provider for terminal configuration

---

## Escape Sequence Handling

### Control Sequences (CSI - Control Sequence Introducer)

The component shall parse and execute CSI sequences with the following format: `CSI Pm;...Pm fn`

| Sequence | Function | Implementation |
|----------|----------|----------------|
| CSI A | Cursor Up (CUU) | Move cursor up n rows |
| CSI B | Cursor Down (CUD) | Move cursor down n rows |
| CSI C | Cursor Forward (CUF) | Move cursor right n columns |
| CSI D | Cursor Back (CUB) | Move cursor left n columns |
| CSI E | Cursor Next Line (CNL) | Move cursor to beginning of n lines down |
| CSI F | Cursor Previous Line (CPL) | Move cursor to beginning of n lines up |
| CSI G | Cursor Horizontal Absolute (CHA) | Move cursor to column n |
| CSI H / CSI f | Cursor Position (CUP) | Move cursor to row n, column m |
| CSI J | Erase in Display (ED) | Clear screen portions |
| CSI K | Erase in Line (EL) | Clear line portions |
| CSI L | Insert Lines (IL) | Insert n blank lines |
| CSI M | Delete Lines (DL) | Delete n lines |
| CSI P | Delete Characters (DCH) | Delete n characters |
| CSI @ | Insert Characters (ICH) | Insert n blank characters |
| CSI S | Scroll Up (SU) | Scroll screen up n lines |
| CSI T | Scroll Down (SD) | Scroll screen down n lines |
| CSI X | Erase Characters (ECH) | Erase n characters |

### Select Graphic Rendition (SGR)

The component shall handle SGR sequences: `CSI Pm m`

| Parameter | Effect |
|-----------|--------|
| 0 | Reset / Normal |
| 1 | Bold |
| 2 | Dim |
| 3 | Italic |
| 4 | Underline |
| 5 | Blink (slow) |
| 7 | Inverse |
| 9 | Crossed out |
| 10-19 | Font selection |
| 22 | Normal intensity |
| 23 | Not italic |
| 24 | Not underlined |
| 25 | Not blinking |
| 27 | Not inverse |
| 30-37 | Foreground color |
| 38 | Extended foreground color |
| 39 | Default foreground color |
| 40-47 | Background color |
| 48 | Extended background color |
| 49 | Default background color |
| 90-97 | Bright foreground (8-color mode) |
| 100-107 | Bright background (8-color mode) |

### Color Handling

- Standard 16-color mode (8 colors + bright variants)
- 256-color mode (8-bit color)
- True color mode (24-bit RGB) where supported by renderer
- Color palette customization

### Additional Escape Sequences

- **Device Status Report (DSR)**: Respond to status queries
- **Terminal Attributes (DECSC)**: Report terminal state
- **Soft Terminal Reset (DECSTR)**: Reset terminal to default state
- **Character Set Designation (SCS)**: Handle G0, G1, G2, G3 character sets

### Sequence Parser Requirements

- Incremental parsing (handle partial sequences)
- Error recovery for malformed sequences
- Timeout handling for incomplete sequences
- Support for 7-bit and 8-bit sequence formats

---

## Success Criteria

### Functional Requirements

1. **Text Rendering**: Accurate rendering of all printable ASCII and extended characters
2. **Color Support**: Proper foreground and background color application for all SGR codes
3. **Cursor Positioning**: Correct cursor placement for all CUP sequences
4. **Screen Clearing**: Proper clearing behavior for all ED and EL variants
5. **Line Editing**: Correct insert/delete character and line operations
6. **Scrolling**: Smooth scrolling for scrollback buffer and screen edge handling

### Performance Requirements

1. **Responsiveness**: UI remains responsive (60 FPS target) during rapid terminal output
2. **Memory**: Efficient memory usage for scrollback buffer (configurable, default 10,000 lines)
3. **Latency**: Minimal input-to-display latency (< 50ms target)

### Compatibility Requirements

1. **Shell Compatibility**: Works with common shells (bash, zsh, fish)
2. **Application Compatibility**: Handles output from common CLI applications (vim, less, tmux)
3. **UTF-8 Handling**: Proper handling of multi-byte UTF-8 sequences

### Visual Quality Requirements

1. **Font Rendering**: Crisp, readable text at all supported font sizes
2. **Color Accuracy**: Accurate color reproduction matching ANSI specifications
3. **Cursor Visibility**: Clear, visible cursor with configurable style (block, underline, bar)
4. **Selection**: Proper text selection highlighting and copy functionality

### Integration Requirements

1. **Input Integration**: Properly routes keyboard input to terminal
2. **Session Management**: Handles session attach/detach gracefully
3. **Configuration**: Responds to terminal configuration changes (colors, fonts, size)
4. **Lifecycle**: Proper handling of pause/resume and configuration changes

---

## Implementation Notes

- The component shall expose a clean API for session binding and terminal input/output
- All terminal rendering logic shall be isolated in the emulator-view module
- Configuration changes shall be observable via appropriate change listeners
- The component shall support both hardware and software keyboard input methods