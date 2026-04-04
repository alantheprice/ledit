// Package editor provides the core text editor functionality.
package editor

import "strings"

// Editor represents a text editor with cursor management.
type Editor struct {
	buffer []rune
	cursor int
}

// New creates a new Editor instance.
func New() *Editor {
	return &Editor{
		buffer: []rune{},
		cursor: 0,
	}
}

// Insert inserts text at the current cursor position.
func (e *Editor) Insert(text string) {
	if text == "" {
		return
	}
	// Insert at cursor position
	before := e.buffer[:e.cursor]
	after := e.buffer[e.cursor:]
	newChars := []rune(text)
	e.buffer = append(before, append(newChars, after...)...)
	e.cursor += len(newChars)
}

// Delete deletes the character at the current cursor position.
func (e *Editor) Delete() {
	if e.cursor >= len(e.buffer) {
		return
	}
	e.buffer = append(e.buffer[:e.cursor], e.buffer[e.cursor+1:]...)
}

// Move moves the cursor in the specified direction.
// Valid directions: "left", "right", "start", "end"
func (e *Editor) Move(direction string) {
	switch direction {
	case "left":
		if e.cursor > 0 {
			e.cursor--
		}
	case "right":
		if e.cursor < len(e.buffer) {
			e.cursor++
		}
	case "start":
		e.cursor = 0
	case "end":
		e.cursor = len(e.buffer)
	}
}

// GetText returns the current buffer content.
func (e *Editor) GetText() string {
	return string(e.buffer)
}

// GetCursor returns the current cursor position.
func (e *Editor) GetCursor() int {
	return e.cursor
}

// GetBuffer returns the current buffer as runes.
func (e *Editor) GetBuffer() []rune {
	return e.buffer
}

// SetText sets the buffer content.
func (e *Editor) SetText(text string) {
	e.buffer = []rune(text)
	if e.cursor > len(e.buffer) {
		e.cursor = len(e.buffer)
	}
}

// LineCount returns the number of lines in the buffer.
func (e *Editor) LineCount() int {
	return strings.Count(string(e.buffer), "\n") + 1
}

