# 01-go-mobile Component Specification

## Component Overview

The `01-go-mobile` component handles the compilation of the Go `ledit` code into an Android library (AAR) using `gomobile bind`. This enables the core Go-based ledit editor functionality to be embedded within an Android application, providing a native Go implementation of text editing capabilities accessible from Java/Kotlin code.

## Technical Requirements

### Go Constraints

- **Go Version**: Go 1.21 or later required
  - gomobile requires modern Go features and toolchain support
- **No CGO**: The Go code must be completely CGO-free
  - No C dependencies or C interoperability permitted
  - All external system calls must be replaced with pure Go alternatives
  - This constraint ensures cross-platform compilation without native toolchain dependencies
- **Pure Go Dependencies**: All third-party packages must be pure Go implementations

### Android Requirements

- **Android SDK**: API level 21 (Android 5.0) minimum target
- **NDK**: Not required for gomobile bind (pure Go compilation)
- **Output Format**: Android AAR (Android Archive) library

### Build Requirements

- gomobile toolchain installed and in PATH
- Android SDK with build-tools and platform packages
- Go 1.21+ with module support enabled

## Dependencies

### Required Tools

| Tool | Version | Purpose |
|------|---------|---------|
| `go` | 1.21+ | Go toolchain for compilation |
| `gomobile` | latest | Generate Android bindings |
| `Android SDK` | API 21+ | Android build environment |
| `Android Build Tools` | latest | AAR generation |

### Go Module Dependencies

```
github.com/gomarkdown/markdown  - Pure Go markdown parsing
```

All dependencies must be verified as CGO-free before inclusion.

## Implementation Details

### Package Structure

```
ledit/
├── go.mod                    # Module definition
├── go.sum                    # Dependency checksums
├── bind/
│   └── main.go              # gomobile bind entry point
├── editor/
│   ├── editor.go           # Core editor interface
│   ├── buffer.go            # Buffer management
│   └── cursor.go            # Cursor operations
├── commands/
│   └── command.go           # Command pattern implementation
├── modes/
│   ├── mode.go              # Mode interface
│   ├── insert.go            # Insert mode
│   └── normal.go            # Normal mode (vi-style)
├── state/
│   └── state.go             # Editor state management
└── mobile/
    └── mobile.go            # Android export interface
```

### Bind Command

The gomobile bind command generates Java bindings from Go code:

```bash
gomobile bind -target=android \
  -javapkg=com.ledit.editor \
  -out=ledit.aar \
  ./bind
```

**Parameters**:
- `-target=android`: Compile for Android platform
- `-javapkg=com.ledit.editor`: Java package namespace for generated classes
- `-out=ledit.aar`: Output archive filename
- `./bind`: Package containing @export annotated types

### Java Package Naming

**Package**: `com.ledit.editor`

Generated Java classes will reside in this namespace. The following classes will be generated:

| Go Type | Java Class | Description |
|---------|------------|-------------|
| `Editor` | `Editor` | Main editor interface |
| `Buffer` | `Buffer` | Text buffer representation |
| `EditorState` | `EditorState` | Editor state object |

### Mobile Export Interface

The Go code must expose a public interface annotated for gomobile export:

```go
// Editor is the main editor interface exposed to Android
type Editor interface {
    // Insert text at cursor position
    Insert(text string)
    
    // Delete character at cursor
    Delete()
    
    // Move cursor
    Move(direction string)
    
    // Get current buffer content
    GetText() string
    
    // Get cursor position
    GetCursor() int
}
```

### Build Output

- **Primary Artifact**: `ledit.aar` - Android library archive
- **Secondary Artifact**: `ledit-sources.jar` - Source JAR (optional)

## Success Criteria

### Build Success

- [ ] `go build` succeeds without CGO dependencies
- [ ] `gomobile bind` generates valid AAR file
- [ ] AAR contains compiled Go code (lib/ folder with `.so` files for supported ABIs)
- [ ] AAR contains Java bindings (classes.jar)
- [ ] Generated JAR compiles without errors in Android project

### Functional Verification

- [ ] Editor interface methods callable from Java/Kotlin
- [ ] Text insertion works correctly
- [ ] Cursor positioning works correctly
- [ ] Buffer state persists across method calls

### Code Quality

- [ ] No CGO imports in any Go source files
- [ ] All public APIs have Go doc comments
- [ ] All exported types have @export annotations for gomobile
- [ ] Module path correctly set to `github.com/ledit/ledit`

### Compatibility

- [ ] Builds successfully on Linux (primary development platform)
- [ ] Output AAR compatible with Android API 21+
- [ ] Supports arm64-v8a, armeabi-v7a, x86, x86_64 ABIs