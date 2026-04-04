# 06-background-service - Specification Document

## Component Overview

**Purpose**: Implement an Android foreground service that keeps the ledit agent running in the background, allowing users to execute long-running agent tasks while the app is in the background or the process is killed.

**Files**:
- `SPEC.md` — This specification document
- `TODOS.md` — Implementation todo items

---

## Why Foreground Service is Needed

### User Experience Requirements

1. **Long-running Tasks**: Ledit agent tasks (coding, debugging, researching) can take significant time. Users should not need to keep the app in the foreground.

2. **Process Survival**: Android aggressively kills background processes. A regular background service would be terminated, causing:
   - Incomplete agent tasks
   - Lost progress
   - Poor user experience

3. **Background Execution**: Android 8+ (Oreo) imposes background execution limits. Only foreground services with persistent notifications are guaranteed to run.

4. **Battery Optimization**: Users may enable battery optimization. Foreground services request exemption from some restrictions.

### Technical Requirements

1. **Persistent Notification**: Required by Android to indicate the service is running
2. **Restart on Process Death**: Service should restart automatically if killed
3. **IPC Communication**: UI needs to communicate with the service (send tasks, receive status)
4. **State Persistence**: Current task state must survive process death

---

## Service Implementation Details

### Service Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Android Application                      │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────┐         ┌─────────────────────────────┐  │
│  │   Main UI    │◄───────►│   LeditAgentService         │  │
│  │  (Activity)  │  AIDL   │   (Foreground Service)     │  │
│  │              │         │                             │  │
│  │ - Task Input │         │ - Persistent notification  │  │
│  │ - Status     │         │ - Agent process management │  │
│  │ - Logs       │         │ - Task queue               │  │
│  └──────────────┘         │ - IPC bindings             │  │
│                           └─────────────────────────────┘  │
│                                      │                     │
│                           ┌──────────┴──────────┐         │
│                           │   Go Core (gRPC)   │         │
│                           │   (via gomobile)   │         │
│                           └────────────────────┘         │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Service Class Structure

```kotlin
// LeditAgentService.kt
class LeditAgentService : Service() {
    // Lifecycle management
    // Notification handling
    // Agent process management
    // IPC handlers
}
```

### Key Implementation Components

1. **Service Declaration** (`AndroidManifest.xml`):
   ```xml
   <service
       android:name=".service.LeditAgentService"
       android:enabled="true"
       android:exported="false"
       android:foregroundServiceType="dataSync"
       android:stopWithTask="false" />
   ```

2. **Foreground Service Type**: Use `dataSync` (Android 14+) for long-running data operations

3. **Service Lifecycle**:
   - `onCreate()`: Initialize agent, load saved state
   - `onStartCommand()`: Handle start commands, return `START_STICKY`
   - `onDestroy()`: Save state, clean up resources

4. **START_STICKY**: Return this to restart service if killed by system

---

## Notification Requirements

### Foreground Notification Specifications

1. **Notification Channel** (Android 8+):
   - Channel ID: `ledit_agent_channel`
   - Channel Name: "Agent Service"
   - Importance: `IMPORTANCE_LOW` (doesn't interrupt user)
   - Description: "Keeps the ledit agent running"

2. **Notification Content**:
   - **Title**: "Ledit Agent Running"
   - **Text**: Current task description or "Idle"
   - **Icon**: App icon (must be visible)
   - **Ongoing**: `true` (cannot be dismissed while service runs)

3. **Notification Actions** (optional):
   - "Stop": Stop the service and agent
   - "Open": Bring app to foreground

### Notification Updates

- Update notification text when task status changes
- Show progress indicator for long operations
- Clear notification when service stops

---

## Process Death Handling

### Survival Strategies

1. **`START_STICKY` Return Value**:
   - Android restarts service with `null` intent if killed
   - Use: `return START_STICKY`

2. **Process Death Detection**:
   - Track service process ID
   - Monitor for unexpected termination
   - Implement watchdog to restart if needed

3. **State Persistence**:
   - Save current task state to disk (JSON/file)
   - Load state on service restart
   - Store in app-internal storage: `files/agent_state.json`

4. **Wake Locks**:
   - Use partial wake lock if CPU-intensive tasks
   - Acquire on task start, release on completion
   - Request `WAKE_LOCK` permission

### State File Format

```json
{
  "currentTask": {
    "id": "task-123",
    "description": "Implement login feature",
    "status": "in_progress",
    "progress": 45,
    "startedAt": "2024-01-15T10:30:00Z"
  },
  "taskHistory": [...],
  "agentPid": 12345,
  "lastUpdated": "2024-01-15T10:35:00Z"
}
```

### Restart Implementation

```kotlin
override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
    if (intent?.action == ACTION_RESTART) {
        // Called after process death - restore state
        restoreState()
    }
    return START_STICKY
}
```

---

## IPC with the Service

### AIDL Interface

Define AIDL for bidirectional communication:

```aidl
// IAgentService.aidl
interface IAgentService {
    // Send task to agent
    void submitTask(String taskDescription);
    
    // Get current status
    AgentStatus getStatus();
    
    // Get task history
    List<TaskInfo> getTaskHistory();
    
    // Cancel current task
    void cancelCurrentTask();
    
    // Register callback for updates
    void registerCallback(IAgentCallback callback);
    
    // Stop service
    void stopService();
}

interface IAgentCallback {
    void onStatusChanged(AgentStatus status);
    void onTaskProgress(int progress, String message);
    void onTaskComplete(String result);
    void onTaskError(String error);
}
```

### IPC Usage from UI

```kotlin
// In Activity
private val serviceConnection = object : ServiceConnection {
    override fun onServiceConnected(name: ComponentName?, binder: IBinder?) {
        agentService = IAgentService.Stub.asInterface(binder)
    }
    
    override fun onServiceDisconnected(name: ComponentName?) {
        agentService = null
    }
}

// Submit task
agentService?.submitTask("Implement user authentication")

// Receive updates via callback
agentService?.registerCallback(object : IAgentCallback.Stub() {
    override fun onTaskProgress(progress: Int, message: String) {
        // Update UI
    }
})
```

### Alternative: Use Bound Service Pattern

- Service binds to Activity on UI creation
- Unbinds when Activity destroyed
- Maintains connection for IPC
- Handles configuration changes (rotation)

---

## Binding to Main Activity

### Service Binding Strategy

1. **Start Foreground First**:
   ```kotlin
   val intent = Intent(this, LeditAgentService::class.java)
   startForegroundService(intent)
   ```

2. **Then Bind**:
   ```kotlin
   bindService(
       Intent(this, LeditAgentService::class.java),
       serviceConnection,
       Context.BIND_AUTO_CREATE
   )
   ```

3. **Handle Both Scenarios**:
   - App in foreground: Bound service provides IPC
   - App in background: Service continues running
   - Process death: Service restarts, state restored

---

## Dependencies & Permissions

### Required Permissions

```xml
<!-- AndroidManifest.xml -->
<uses-permission android:name="android.permission.FOREGROUND_SERVICE" />
<uses-permission android:name="android.permission.FOREGROUND_SERVICE_DATA_SYNC" />
<uses-permission android:name="android.permission.POST_NOTIFICATIONS" />
<uses-permission android:name="android.permission.WAKE_LOCK" />
<uses-permission android:name="android.permission.REQUEST_IGNORE_BATTERY_OPTIMIZATIONS" />
```

### Permission Requests

1. **Notification Permission** (Android 13+):
   ```kotlin
   if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
       requestPermission(Manifest.permission.POST_NOTIFICATIONS)
   }
   ```

2. **Battery Optimization** (optional):
   ```kotlin
   val intent = Intent(Settings.ACTION_REQUEST_IGNORE_BATTERY_OPTIMIZATIONS)
   intent.data = Uri.parse("package:$packageName")
   startActivity(intent)
   ```

### Dependencies

- AndroidX Core KTX
- AndroidX Lifecycle (ViewModel, LiveData)
- Material Components (for notification icon)

---

## Success Criteria

### Functional Requirements

- [ ] Service starts as foreground service with persistent notification
- [ ] Service survives process death and restarts automatically
- [ ] IPC communication works between Activity and Service
- [ ] Task state persists across process death
- [ ] Tasks can be submitted from UI and executed in background
- [ ] Task progress updates are delivered to UI via callbacks
- [ ] Service can be stopped cleanly via UI or notification action

### Technical Requirements

- [ ] Notification channel created (Android 8+)
- [ ] Notification permission requested (Android 13+)
- [ ] `START_STICKY` returned from `onStartCommand()`
- [ ] State saved to disk on changes
- [ ] State restored on service restart
- [ ] Wake lock acquired during CPU-intensive tasks
- [ ] Proper cleanup in `onDestroy()`

### Test Scenarios

1. **Basic Functionality**:
   - Start service → notification appears
   - Submit task → task executes
   - View progress → updates received

2. **Background Survival**:
   - Start task, put app in background → task continues
   - Kill app process → service restarts
   - Task state restored → continues from where left off

3. **UI Integration**:
   - Start service, open app → binds successfully
   - Submit task from UI → executes in service
   - Receive callbacks → UI updates correctly

---

## Implementation Notes

### Go Integration

The foreground service will invoke the Go core (compiled via gomobile) to execute agent tasks. The service acts as the bridge between Android UI and Go-based ledit core.

### Error Handling

- Handle Go core crashes gracefully
- Implement timeout for tasks
- Provide user feedback on failures

### Logging

- Use Android Log with tag `LeditAgent`
- Log service lifecycle events
- Log task submissions and completions
