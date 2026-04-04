// Package mobile provides the gomobile export interface for the ledit editor.
// This package exposes the core editor functionality to Android via gomobile bind.
package mobile

import (
	"github.com/alantheprice/ledit/pkg/editor"
)

// Editor is the main editor interface exposed to Android.
//export Editor
type Editor struct {
	impl *editor.Editor
}

// NewEditor creates a new Editor instance.
//export NewEditor
func NewEditor() *Editor {
	return &Editor{
		impl: editor.New(),
	}
}

// Insert inserts text at the current cursor position.
//export Insert
func (e *Editor) Insert(text string) {
	e.impl.Insert(text)
}

// Delete deletes the character at the current cursor position.
//export Delete
func (e *Editor) Delete() {
	e.impl.Delete()
}

// Move moves the cursor in the specified direction.
//export Move
func (e *Editor) Move(direction string) {
	e.impl.Move(direction)
}

// GetText returns the current buffer content.
//export GetText
func (e *Editor) GetText() string {
	return e.impl.GetText()
}

// GetCursor returns the current cursor position.
//export GetCursor
func (e *Editor) GetCursor() int {
	return e.impl.GetCursor()
}

