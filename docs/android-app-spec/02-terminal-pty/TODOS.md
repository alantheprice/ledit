# Component: 02-terminal-pty — Todo List

## Todo Items

### Phase 1: Native Library Setup

- **[ ]** Status: **pending**
  - Todo: Create Android NDK module configuration (`Android.mk`, `Application.mk`)
  - Completion: `ndk-build` produces `libjackpal-termexec2.so`

- **[ ]** Status: **pending**
  - Todo: Implement C code for PTY creation (`native/term_exec.c`)
  - Functions: `openpty()`, `forkpty()`, `create_subprocess()`
  - Completion: Compiles without errors, matches libtermexec API

- **[ ]** Status: **pending**
  - Todo: Implement signal handling in native code (`kill()`, process groups)
  - Completion: Sending negative PID delivers signal to process group

### Phase 2: Java API Layer

- **[ ]** Status: **pending**
  - Todo: Create `PTYSession` class for process management
  - API: `start()`, `write()`, `read()`, `signal()`, `waitFor()`, `destroy()`
  - Completion: Unit tests pass

- **[ ]** Status: **pending**
  - Todo: Implement environment variable passing to subprocess
  - Completion: Environment variables set in Java visible in shell

- **[ ]** Status: **pending**
  - Todo: Handle ParcelFileDescriptor lifecycle correctly
  - Completion: No FD leaks, proper close on process termination

### Phase 3: Threading & Async Support

- **[ ]** Status: **pending**
  - Todo: Implement background thread execution (no blocking on main)
  - Completion: `start()` works when called from main thread

- **[ ]** Status: **pending**
  - Todo: Add callback support for process exit events
  - Completion: Listener notified when subprocess exits

### Phase 4: Process Utilities

- **[ ]** Status: **pending**
  - Todo: Implement `setWindowSize()` for PTY window resize
  - Completion: Shell receives SIGWINCH on resize

- **[ ]** Status: **pending**
  - Todo: Add process group management (foreground/background jobs)
  - Completion: Can move process to/from foreground

- **[ ]** Status: **pending**
  - Todo: Implement `isRunning()` check
  - Completion: Returns true if process still active

### Phase 5: Integration & Tests

- **[ ]** Status: **pending**
  - Todo: Write unit tests for PTYSession
  - Coverage: spawn, signal, waitFor, environment
  - Completion: All tests pass

- **[ ]** Status: **pending**
  - Todo: Write integration tests (test app with real shell)
  - Completion: Shell spawns and responds to commands

- **[ ]** Status: **pending**
  - Todo: Test signal delivery (Ctrl+C stops running process)
  - Completion: `SIGINT` terminates `sleep` command

- **[ ]** Status: **pending**
  - Todo: Test exit code propagation
  - Completion: `exit N` returns N from `waitFor()`

- **[ ]** Status: **pending**
  - Todo: Test resource cleanup (no FD leaks after process exit)
  - Completion: No leaked file descriptors after repeated spawn cycles

### Phase 6: Documentation & Build

- **[ ]** Status: **pending**
  - Todo: Document PTYSession API in code comments
  - Completion: Javadoc present for public methods

- **[ ]** Status: **pending**
  - Todo: Configure Gradle to build native library as part of build
  - Completion: Building project produces .so automatically

- **[ ]** Status: **pending**
  - Todo: Final verification against SPEC.md success criteria
  - Completion: All functional, performance, and robustness criteria met

---

## Quick Reference

| Priority | Item | Status |
|----------|------|--------|
| High | Native library build | pending |
| High | PTYSession class | pending |
| High | Signal handling | pending |
| High | Unit tests | pending |
| Medium | Window resize | pending |
| Medium | Process groups | pending |
| Low | Advanced job control | pending |