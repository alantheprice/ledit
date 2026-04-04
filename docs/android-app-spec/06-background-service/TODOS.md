# 06-background-service - Implementation Todos

## Overview

Implementation todo items for the foreground service component that keeps the ledit agent running in the background.

**Status Legend**:
- `pending` - Not started
- `in_progress` - Currently being implemented
- `completed` - Implemented and verified

---

## Service Implementation

### TODO-01: Create Service Class Skeleton

**Description**: Create the basic `LeditAgentService` class with lifecycle methods

**Status**: `pending`

**Completion Criteria**:
- [ ] Class extends `Service` and implements required methods
- [ ] `onCreate()` initializes basic state
- [ ] `onStartCommand()` returns `START_STICKY`
- [ ] `onDestroy()` handles cleanup
- [ ] [ ] Build compiles without errors

**Files to Create**:
- `app/src/main/kotlin/com/ledit/android/service/LeditAgentService.kt`

---

### TODO-02: Register Service in Manifest

**Description**: Add service declaration to AndroidManifest.xml with correct attributes

**Status**: `pending`

**Completion Criteria**:
- [ ] Service declared with correct name
- [ ] `android:foregroundServiceType="dataSync"` set
- [ ] `android:stopWithTask="false"` set
- [ ] Permission added: `FOREGROUND_SERVICE`
- [ ] Permission added: `FOREGROUND_SERVICE_DATA_SYNC`

**Files to Modify**:
- `app/src/main/AndroidManifest.xml`

---

### TODO-03: Implement Foreground Notification

**Description**: Create notification channel and show persistent foreground notification

**Status**: `pending`

**Completion Criteria**:
- [ ] Notification channel created (Android 8+)
- [ ] Notification permission requested (Android 13+)
- [ ] Foreground notification displayed on service start
- [ ] Notification text updates with task status
- [ ] Notification action to stop service works
- [ ] Notification cleared on service stop

**Files to Modify**:
- `app/src/main/kotlin/com/ledit/android/service/LeditAgentService.kt`

---

### TODO-04: Implement State Persistence

**Description**: Save and restore agent state across process death

**Status**: `pending`

**Completion Criteria**:
- [ ] State data class defined with JSON serialization
- [ ] State saved to file on task changes
- [ ] State loaded from file on service restart
- [ ] `agent_state.json` stored in app internal storage
- [ ] State correctly indicates task progress after restart

**Files to Create/Modify**:
- `app/src/main/kotlin/com/ledit/android/service/AgentState.kt`
- `app/src/main/kotlin/com/ledit/android/service/LeditAgentService.kt`

---

### TODO-05: Implement AIDL IPC Interface

**Description**: Create AIDL interface for bidirectional communication between UI and service

**Status**: `pending`

**Completion Criteria**:
- [ ] `IAgentService.aidl` defines submitTask, getStatus, cancelTask methods
- [ ] `IAgentCallback.aidl` defines progress/callback methods
- [ ] `AgentStatus.aidl` parcelable data class defined
- [ ] Service implements AIDL interface
- [ ] AIDL generates Java stubs without errors

**Files to Create**:
- `app/src/main/aidl/com/ledit/android/service/IAgentService.aidl`
- `app/src/main/aidl/com/ledit/android/service/IAgentCallback.aidl`
- `app/src/main/aidl/com/ledit/android/service/AgentStatus.aidl`

---

### TODO-06: Implement Service Binder

**Description**: Implement binder to allow Activity to bind and communicate with service

**Status**: `pending`

**Completion Criteria**:
- [ ] Service returns binder from `onBind()`
- [ ] Binder implements IAgentService interface
- [ ] `ServiceConnection` in Activity connects successfully
- [ ] Methods callable from Activity to Service
- [ ] Callbacks delivered from Service to Activity

**Files to Modify**:
- `app/src/main/kotlin/com/ledit/android/service/LeditAgentService.kt`
- `app/src/main/kotlin/com/ledit/android/ui/MainActivity.kt`

---

### TODO-07: Add Wake Lock Support

**Description**: Add wake lock to keep CPU alive during intensive tasks

**Status**: `pending`

**Completion Criteria**:
- [ ] `WAKE_LOCK` permission added to manifest
- [ ] Partial wake lock acquired on task start
- [ ] Wake lock released on task completion/error
- [ ] Wake lock released in onDestroy

**Files to Modify**:
- `app/src/main/AndroidManifest.xml`
- `app/src/main/kotlin/com/ledit/android/service/LeditAgentService.kt`

---

### TODO-08: Implement Task Queue and Execution

**Description**: Implement task submission, queueing, and execution via Go core

**Status**: `pending`

**Completion Criteria**:
- [ ] Tasks are queued if one is already running
- [ ] Go core invoked to execute agent task
- [ ] Progress callbacks delivered to UI
- [ ] Task completion/error results delivered
- [ ] Task history maintained

**Files to Modify**:
- `app/src/main/kotlin/com/ledit/android/service/LeditAgentService.kt`
- Integration with Go core from 01-go-mobile

---

### TODO-09: Integrate with UI

**Description**: Connect UI components to service for task submission and status display

**Status**: `pending`

**Completion Criteria**:
- [ ] Service starts when user initiates task
- [ ] Task input field submits to service
- [ ] Status display shows current task progress
- [ ] Task history visible in UI
- [ ] Stop/cancel functionality works
- [ ] App survives process death (service restarts, binds)

**Files to Modify**:
- `app/src/main/kotlin/com/ledit/android/ui/MainActivity.kt`
- `app/src/main/kotlin/com/ledit/android/ui/AgentViewModel.kt`

---

## Testing

### TODO-10: Basic Functionality Tests

**Description**: Test basic service lifecycle and IPC

**Status**: `pending`

**Completion Criteria**:
- [ ] Test: Start service → notification appears
- [ ] Test: Submit task → task executes
- [ ] Test: View progress → updates received
- [ ] Test: Complete task → result delivered

---

### TODO-11: Process Death Survival Tests

**Description**: Test service survives process death

**Status**: `pending`

**Completion Criteria**:
- [ ] Test: Start task, kill app process → service restarts
- [ ] Test: State restored after restart
- [ ] Test: Task continues from where it left off
- [ ] Test: Notification updates after restart

---

### TODO-12: Background Execution Tests

**Description**: Test service works when app in background

**Status**: `pending`

**Completion Criteria**:
- [ ] Test: Start task, press home → task continues
- [ ] Test: Notification shows correct status
- [ ] Test: Return to app → status synced

---

## Cleanup

### TODO-13: Code Review and Cleanup

**Description**: Review implementation and clean up

**Status**: `pending`

**Completion Criteria**:
- [ ] Code follows Android/Kotlin conventions
- [ ] Error handling in place
- [ ] Logging implemented
- [ ] No memory leaks
- [ ] Proper resource cleanup
- [ ] Final build compiles and runs

---

## Implementation Order

```
Phase 1: Core Service
├── TODO-01: Create Service Class Skeleton
├── TODO-02: Register Service in Manifest

Phase 2: Notifications
└── TODO-03: Implement Foreground Notification

Phase 3: State & IPC
├── TODO-04: Implement State Persistence
├── TODO-05: Implement AIDL IPC Interface
└── TODO-06: Implement Service Binder

Phase 4: Task Execution
├── TODO-07: Add Wake Lock Support
└── TODO-08: Implement Task Queue and Execution

Phase 5: UI Integration
└── TODO-09: Integrate with UI

Phase 6: Testing
├── TODO-10: Basic Functionality Tests
├── TODO-11: Process Death Survival Tests
└── TODO-12: Background Execution Tests

Phase 7: Finalization
└── TODO-13: Code Review and Cleanup
```

---

## Dependencies

- **01-go-mobile**: Go core library must be built first
- **05-android-shell**: UI components needed for TODO-09
- **AndroidX Lifecycle**: For ViewModel and LiveData
- **AndroidX Core KTX**: For Kotlin extensions
