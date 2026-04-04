# 06-background-service - Todo Items

## Status Legend
- `[x]` = Done
- `[ ]` = Pending

## Todo Items

### Phase 1: Core Service

- [x] **T001** Create Service class skeleton extending Service
  - Completion: LeditAgentService.kt created with onCreate, onStartCommand, onDestroy
  - Location: `app/src/main/kotlin/com/ledit/android/service/LeditAgentService.kt`

- [x] **T002** Register Service in AndroidManifest.xml
  - Completion: Service declared with foregroundServiceType and stopWithTask=false
  - Location: `app/src/main/AndroidManifest.xml`

### Phase 2: Notifications

- [x] **T003** Implement foreground notification
  - Completion: Notification channel created, notification shows "Ledit Agent Running"
  - Location: `LeditAgentService.kt` createNotification()

- [x] **T004** Add notification actions (Stop, Open)
  - Completion: PendingIntent actions allow user to stop service or open app

### Phase 3: State & IPC

- [x] **T005** Implement AIDL IPC interface
  - Completion: ILeditAgentService.aidl defines isRunning(), getCurrentTask(), submitTask(), getLastResult()
  - Location: `app/src/main/aidl/com/ledit/android/service/ILeditAgentService.aidl`

- [x] **T006** Implement Service Binder
  - Completion: LeditAgentBinder.kt provides IPC access to service
  - Location: `app/src/main/kotlin/com/ledit/android/service/LeditAgentBinder.kt`

- [x] **T007** Enable service binding in onBind()
  - Completion: Service returns binder for IPC communication

### Phase 4: Task Execution

- [x] **T008** Add Wake Lock support
  - Completion: PowerManager.WakeLock acquired in onCreate, released in onDestroy
  - Location: `LeditAgentService.kt` acquireWakeLock(), releaseWakeLock()

- [x] **T009** Implement task queue and executeTask()
  - Completion: submitTask() method available, placeholder for Go agent integration

### Phase 5: UI Integration

- [x] **T010** Add service start/stop from Activity
  - Completion: LeditAgentService.start(), stop(), executeTask() companion methods

- [x] **T011** Integrate with MainActivity notification permission
  - Completion: MainActivity requests POST_NOTIFICATIONS permission

### Phase 6: Testing

- [ ] **T012** Test service starts and shows notification
- [ ] **T013** Test service restarts after process death (START_STICKY)
- [ ] **T014** Test task submission from UI
- [ ] **T015** Test background execution continues

### Phase 7: Finalization

- [ ] **T016** Document API in code comments
- [ ] **T017** Final verification against SPEC.md success criteria

---

## Summary

| Status | Count |
|--------|-------|
| Completed | 11 |
| Pending | 6 |
| **Total** | **17** |

---

*Generated: 2025-04-04*
*Component: 06-background-service*
*Updated: 2026-04-04 - Core service implementation complete*