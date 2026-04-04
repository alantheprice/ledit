# Ledit Android App - Specification Document

## Overview

This document specifies the requirements, architecture, and implementation plan for converting the `ledit` project into a native Android application with full terminal emulation capabilities.

## Project Context

**Current State:**
- `ledit` is a Go-based CLI tool with a WebUI (runs at http://localhost:54000)
- Currently runs on Linux/macOS/Windows and in Termux on Android
- Features: coding agent, 10 specialized personas, multi-provider LLM support, file operations, terminal integration

**Goal:**
- Native Android app (APK) with embedded terminal + full ledit functionality
- NOT Termux — a standalone app with terminal built-in
- Users can run ledit agent tasks AND have a working terminal shell

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    Ledit Android App                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐    ┌──────────────────────────────────────┐ │
│  │   Android    │    │         Ledit Core (Go)              │ │
│  │     UI       │◄──►│  - Agent orchestration               │ │
│  │ (Kotlin)     │    │  - LLM providers                    │ │
│  │              │    │  - Tool execution                  │ │
│  │ - WebView    │    │  - Subagent system                  │ │
│  │ - Terminal   │    │  - Provider catalog                │ │
│  │ - Navigation │    │                                      │ │
│  └──────┬───────┘    └──────────────┬───────────────────────┘ │
│         │                           │                          │
│         │                     ┌─────┴─────┐                   │
│         │                     │  Go       │                   │
│         │                     │  Mobile   │                   │
│         │                     │  Bindings │                   │
│         │                     └───────────┘                   │
│         │                           │                          │
│  ┌──────┴───────┐    ┌─────────────┴───────────────────────┐  │
│  │ Android      │    │     Android Native Libraries       │  │
│  │ Terminal     │    │  - libtermexec (PTY)               │  │
│  │ Component    │    │  - emulatorview (rendering)       │  │
│  │              │    │  - AOSP terminal code              │  │
│  └──────────────┘    └────────────────────────────────────┘  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Component Overview

| Component | Files | Description |
|-----------|-------|-------------|
| **01-go-mobile** | `SPEC.md`, `TODOS.md` | Compile Go code to Android library via gomobile |
| **02-terminal-pty** | `SPEC.md`, `TODOS.md` | PTY subsystem + process execution |
| **03-emulator-view** | `SPEC.md`, `TODOS.md` | Terminal rendering with VT-100 parsing |
| **04-webui-integration** | `SPEC.md`, `TODOS.md` | Embed ledit WebUI in Android WebView |
| **05-android-shell** | `SPEC.md`, `TODOS.md` | Android UI wrapper, navigation, settings |
| **06-background-service** | `SPEC.md`, `TODOS.md` | Foreground service for persistent agent |
| **07-shell-bundle** | `SPEC.md`, `TODOS.md` | Bundled shell (toybox/busybox) for unrooted |

---

## Key Technical Decisions

1. **Hybrid Architecture**: Go core with Kotlin/Java UI — not full rewrite
2. **gomobile for Go bindings**: Compile ledit to `.aar` library
3. **jackpal/Android-Terminal-Emulator**: Terminal libraries (Apache 2.0)
4. **WebView for WebUI**: Host ledit WebUI, not reimplement
5. **Foreground Service**: Keep agent running in background

---

## Dependencies & Licensing

### Open Source Libraries
- **jackpal/Android-Terminal-Emulator** — Apache 2.0
- **master-hax/aosp-terminal** — Apache 2.0
- **termux/termux-app** components — GPL/Apache 2.0
- **gomobile** — BSD-style

### Android Requirements
- minSdkVersion: 24 (Android 7.0)
- targetSdkVersion: 34 (Android 14)
- NDK for native code compilation

---

## Success Criteria

- [ ] APK builds and installs on Android device
- [ ] Terminal emulator works (can run shell commands)
- [ ] Ledit WebUI accessible via WebView
- [ ] Ledit agent can execute tasks
- [ ] Background service keeps agent alive
- [ ] App survives process death

---

## Related Documents

- [01-go-mobile/SPEC.md](./01-go-mobile/SPEC.md)
- [01-go-mobile/TODOS.md](./01-go-mobile/TODOS.md)
- [02-terminal-pty/SPEC.md](./02-terminal-pty/SPEC.md)
- [02-terminal-pty/TODOS.md](./02-terminal-pty/TODOS.md)
- [03-emulator-view/SPEC.md](./03-emulator-view/SPEC.md)
- [03-emulator-view/TODOS.md](./03-emulator-view/TODOS.md)
- [04-webui-integration/SPEC.md](./04-webui-integration/SPEC.md)
- [04-webui-integration/TODOS.md](./04-webui-integration/TODOS.md)
- [05-android-shell/SPEC.md](./05-android-shell/SPEC.md)
- [05-android-shell/TODOS.md](./05-android-shell/TODOS.md)
- [06-background-service/SPEC.md](./06-background-service/SPEC.md)
- [06-background-service/TODOS.md](./06-background-service/TODOS.md)
- [07-shell-bundle/SPEC.md](./07-shell-bundle/SPEC.md)
- [07-shell-bundle/TODOS.md](./07-shell-bundle/TODOS.md)