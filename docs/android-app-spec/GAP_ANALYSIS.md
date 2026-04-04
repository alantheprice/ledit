# Android App Implementation Gap Analysis

## Executive Summary

The Android app has a **foundation** but requires significant additional work to become a **functional APK**. This document identifies what's implemented vs. what's needed.

---

## Component Status

| Component | Status | Implementation | Gaps |
|-----------|--------|----------------|------|
| **01-go-mobile** | ⚠️ Partial | Has mobile/bind packages | No AAR built, not referenced in Gradle |
| **02-terminal-pty** | ⚠️ Partial | Has JNI C code, PTYSession | Native lib not compiled, not integrated |
| **03-emulator-view** | ⚠️ Partial | Has parser/renderer/state | Not wired to PTY, mock data only |
| **04-webui-integration** | ✅ Good | WebUIFragment, JsInterface | Working but needs WebUI server |
| **05-android-shell** | ⚠️ Partial | UI components exist | Fragment navigation issues |
| **06-background-service** | ⚠️ Partial | Service skeleton | Not connected to agent logic |
| **07-shell-bundle** | ⚠️ Partial | Manager code | No toybox binary included |

---

## Critical Gaps

### 1. No Built AAR (01-go-mobile)

**Problem**: The Go code exists but:
- `gomobile bind` has never been run
- No `ledit.aar` exists
- `app/build.gradle` doesn't reference any AAR

**What's needed**:
```gradle
dependencies {
    implementation files('libs/ledit.aar')
}
```

**Action**: Run `gomobile bind -target=android -javapkg=com.ledit.editor -out=app/libs/ledit.aar ./bind`

---

### 2. Native Library Not Built (02-terminal-pty)

**Problem**: 
- JNI C code exists (`term_exec.c`)
- But NDK build never run
- No `.so` file created

**What's needed**:
- Run `ndk-build` in `app/src/main/jni/`
- Copy `.so` to proper location
- Load library in PTYSession

**Action**: Add to `app/build.gradle`:
```gradle
sourceSets {
    main {
        jniLibs.srcDirs = ['libs']
    }
}
```

---

### 3. Terminal Not Connected to PTY

**Problem**: 
- `TerminalView` exists with VT-100 parser
- `PTYSession` exists for process management
- **But they aren't wired together**

**What's needed**:
- TerminalFragment should connect PTYSession's stdout → TerminalView's input
- TerminalView's keystrokes → PTYSession's stdin
- Currently uses mock/placeholder data

**Action**: Update `TerminalFragment.kt` to wire PTY → TerminalView

---

### 4. No HTTP Server for WebUI

**Problem**:
- `LedisService` is a placeholder HTTP server
- WebUI needs to serve files at `http://localhost:54000`
- Currently no real server implementation

**What's needed**:
- Implement proper HTTP server in `LedisService`
- Serve static WebUI files
- Handle WebSocket for real-time communication

**Action**: Implement NanoHttpd handler in `LedisService`

---

### 5. Background Service Not Connected

**Problem**:
- `LeditAgentService` exists with AIDL
- But doesn't actually run ledit agent
- No connection to Go core

**What's needed**:
- Load Go library (AAR)
- Initialize agent
- Handle task queue

**Action**: Integrate AAR and call Go functions from Service

---

### 6. Missing Build Configuration

**What's missing in `app/build.gradle`**:
- NDK configuration for native library
- AAR dependency reference
- ProGuard rules for release
- BuildConfig fields
- Version info

---

### 7. Resource Files Incomplete

**Missing**:
- `ic_launcher.png` (drawable XML is placeholder)
- App icon backgrounds
- Splash screen
- Notification icons

---

## Detailed Gap Matrix

### By File/Component

| File | Current State | Required |
|------|---------------|----------|
| `app/libs/ledit.aar` | ❌ Missing | Run gomobile bind |
| `app/libs/arm64-v8a/libtermexec.so` | ❌ Missing | Run ndk-build |
| `PTYSession.kt` | Has native method stubs | Actually load .so |
| `TerminalFragment.kt` | Mock PTY | Wire to real PTYSession |
| `LedisService.kt` | Empty skeleton | Implement HTTP server |
| `LeditAgentService.kt` | Empty skeleton | Call Go agent |
| `MainActivity.kt` | Basic navigation | Fix fragment handling |
| `WebUIFragment.kt` | Basic WebView | Handle JS callbacks properly |

### By Functionality

| Feature | Implemented | Notes |
|---------|------------|-------|
| PTY creation | Partial | JNI code exists, not compiled |
| Shell execution | Partial | Uses ProcessBuilder fallback |
| VT-100 parsing | ✅ Good | Full parser implemented |
| Terminal rendering | ✅ Good | Canvas renderer works |
| WebView WebUI | ✅ Good | JS bridge works |
| Settings persistence | ✅ Good | SharedPreferences |
| Notifications | ✅ Good | Foreground service |
| AIDL IPC | ✅ Good | Interface defined |

---

## Priority Fixes

### P0 - Must Fix for APK to Work

1. **Build Go AAR** - Without this, no Go functionality works
2. **Build Native .so** - Terminal needs PTY library
3. **Wire PTY → Terminal** - Not connected currently

### P1 - Should Fix for App Functionality

4. **Implement HTTP Server** - WebUI won't load
5. **Connect Agent Service** - Background tasks won't work
6. **Fix Fragment Navigation** - UI doesn't switch properly

### P2 - Nice to Have

7. **Add Real Icons** - Placeholder XML only
8. **ProGuard Config** - For release builds
9. **Build Variants** - Debug vs Release

---

## Build Verification Checklist

When building on a proper machine, verify:

- [ ] `gomobile bind` produces `ledit.aar`
- [ ] `ndk-build` produces `libjackpal-termexec2.so`
- [ ] AAR added to `app/libs/`
- [ ] `.so` added to `app/libs/arm64-v8a/`
- [ ] Gradle sync finds both
- [ ] `./gradlew assembleDebug` completes
- [ ] APK contains `.so` files
- [ ] APK installs on device

---

## Notes for Build Machine

The build requires:
1. Go 1.21+ with gomobile installed
2. Android NDK (for PTY .so)
3. Android SDK with build-tools 34
4. Java 17

Run these commands on build machine:

```bash
# 1. Build Go AAR
cd /path/to/ledit
go install golang.org/x/mobile/cmd/gomobile@latest
gomobile bind -target=android -javapkg=com.ledit.editor -out=app/libs/ledit.aar ./bind

# 2. Build native lib
cd app/src/main/jni
$NDK_ROOT/ndk-build

# 3. Build APK
cd /path/to/ledit
./gradlew assembleDebug
```

---

*Generated: 2026-04-04*
*Status: GAP ANALYSIS - Requires significant work*