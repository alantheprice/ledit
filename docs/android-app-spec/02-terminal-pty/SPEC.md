# Component: 02-terminal-pty

## Overview

The PTY (pseudo-terminal) subsystem handles spawning shell processes and managing bidirectional I/O between the Android terminal emulator and the shell. It creates virtual terminal pairs using Android's `/dev/ptmx` device, allowing the app to communicate with child processes like a real terminal.

## Purpose

- Spawn shell processes (typically `/system/bin/sh` or custom shells)
- Create PTY pairs (master/slave) for terminal communication
- Handle foreground/background process groups
- Support job control (Ctrl+C, Ctrl+Z, etc.)
- Manage process lifecycle (start, wait, signal, exit)

## Technical Requirements

### How PTY Works on Android

1. **Master Endpoint**: Open `/dev/ptmx` (terminal multiplexer) to get a master file descriptor
2. **Slave Endpoint**: The kernel creates a corresponding `/dev/pts/N` slave device
3. **Process Fork**: Fork the child process with its stdin/stdout/stderr connected to the slave PTY
4. **I/O Communication**: All reads/writes to master FD go to child's terminal

### Key System Calls (via native lib)

- `open("/dev/ptmx", O_RDWR)` — Open master PTY
- `ptsname()` — Get slave device name from master FD
- `fork()` — Create subprocess with PTY as controlling terminal
- `execvp()` — Execute shell command in child
- `waitpid()` — Wait for process exit
- `kill()` — Send signals (SIGINT, SIGKILL, etc.)

### Android-Specific Considerations

- Requires `android.os.ParcelFileDescriptor` wrapping for Java interop
- `/dev/ptmx` exists on all Android devices (Linux kernel requirement)
- Must run `start()` on background thread (blocked on main thread)
- Signal handling uses negative PID for process groups

## Library Choice

### libtermexec (jackpal/Android-Terminal-Emulator)

**Source**: https://github.com/jackpal/Android-Terminal-Emulator/tree/master/libtermexec

**Why libtermexec**:
- Battle-tested (used by Termux, TermOne Plus, and many terminal apps)
- Handles the complex JNI/native bridge for PTY creation
- Provides clean Java API with ParcelFileDescriptor integration
- Well-maintained native library (`libjackpal-termexec2.so`)

**Core API**:
```java
TermExec exec = new TermExec("sh", "-c", "your command");
exec.environment().put("TERM", "xterm-256color");
int pid = exec.start(ptmxFd);
TermExec.sendSignal(pid, Signal.SIGINT); // Ctrl+C equivalent
int exitCode = TermExec.waitFor(pid);
```

**Alternative Considered**: Manual JNI implementation
- Rejected: Too complex, error-prone, no benefit over existing library

## Implementation Approach

### Phase 1: Native Library Setup
1. Create Android NDK module for `libjackpal-termexec2.so`
2. Implement C code for PTY creation using `forkpty()` or manual `openpty()` + `fork()`
3. Build as shared library (.so)

### Phase 2: Java API Layer
1. Create wrapper class mirroring TermExec API
2. Handle ParcelFileDescriptor lifecycle
3. Add environment variable management
4. Implement async execution support

### Phase 3: Process Management
1. Signal sending (SIGINT, SIGKILL, SIGTSTP, etc.)
2. Exit code retrieval
3. Process group management

### Architecture

```
┌─────────────────────────────────────────┐
│           Shell Process                │
│  (/system/bin/sh -c "...")              │
└──────────────┬──────────────────────────┘
               │ /dev/pts/N (slave PTY)
               ▼
┌─────────────────────────────────────────┐
│          PTY Subsystem                  │
│  libjackpal-termexec2.so               │
│  - openpty() / forkpty()               │
│  - waitpid() / kill()                 │
└──────────────┬──────────────────────────┘
               │ /dev/ptmx (master PTY)
               │ ParcelFileDescriptor
               ▼
┌─────────────────────────────────────────┐
│       PTYSessionManager (Java)           │
│       - startProcess()                 │
│       - write()/read()                │
│       - signal()/waitFor()           │
└─────────────────────────────────────────┘
```

## Dependencies

### Internal Dependencies
- `01-terminal-core` — Core terminal infrastructure
- `00-shared` — Common utilities, logging

### External Dependencies
- **Android System**: `ParcelFileDescriptor`, `Binder`, `Looper`
- **Native Library**: C/C++ NDK toolchain
- **System Calls**: `/dev/ptmx`, `forkpty()`, `ptsname()`

### Gradle Dependencies
```groovy
implementation 'com.jakewh:libtermexec:??'  // Or build from source
```

## Success Criteria

### Functional Criteria
- [ ] Shell process spawns successfully with PTY
- [ ] Bidirectional I/O works (input reaches shell, output reaches app)
- [ ] Signals deliver correctly (SIGINT = Ctrl+C, etc.)
- [ ] Exit codes returned properly
- [ ] Environment variables propagate to subprocess

### Performance Criteria
- [ ] Input latency < 10ms
- [ ] No blocking on main thread
- [ ] Process startup < 100ms

### Robustness Criteria
- [ ] Handles process crashes gracefully
- [ ] Proper cleanup on abnormal termination
- [ ] No resource leaks (file descriptors)
- [ ] Works across Android versions (API 21+)

### Test Scenarios
1. **Basic**: Run `echo hello`, verify output
2. **Interactive**: Run `cat`, send input, verify echo
3. **Signal**: Run `sleep 100`, send SIGINT, verify termination
4. **Environment**: Set `FOO=bar`, verify in subprocess
5. **Exit Code**: Run `exit 42`, verify return value