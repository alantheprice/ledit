# 04-WebUI Integration - TODOs

> Embedding ledit WebUI in Android WebView and JavaScript interface communication.

## Phase 1: Project Setup & WebView Integration

### Todo Items

- [ ] **T001** Create `WebViewFragment` class extending `Fragment`
  - *Status*: Pending
  - *Completion*: Fragment class created with `onCreateView()`, `onViewCreated()`, `onStart()`, `onStop()` lifecycle methods
  - *Location*: `app/src/main/java/com/ledit/android/ui/WebViewFragment.java`

- [ ] **T002** Create WebView layout XML (`fragment_webview.xml`)
  - *Status*: Pending
  - *Completion*: Layout contains WebView and ProgressBar with proper IDs and visibility settings
  - *Location*: `app/src/main/res/layout/fragment_webview.xml`

- [ ] **T003** Add WebView to `MainActivity` layout
  - *Status*: Pending
  - *Completion*: MainActivity hosts WebViewFragment or embeds WebView directly

- [ ] **T004** Configure WebView to load ledit WebUI from localhost:54000
  - *Status*: Pending
  - *Completion*: `webView.loadUrl("http://localhost:54000")` called in onStart() lifecycle

- [ ] **T005** Add required permissions to AndroidManifest.xml
  - *Status*: Pending
  - *Completion*: INTERNET, ACCESS_NETWORK_STATE permissions added

---

## Phase 2: JavaScript Bridge Implementation

### Todo Items

- [ ] **T010** Create `JsCallback` interface
  - *Status*: Pending
  - *Completion*: Interface defines `onFileSaved()`, `onFileOpened()`, `onError()` callback methods
  - *Location*: `app/src/main/java/com/ledit/android/bridge/JsCallback.java`

- [ ] **T011** Create `JsInterface` class with @JavascriptInterface methods
  - *Status*: Pending
  - *Completion*: Class implements: `saveFile()`, `openFile()`, `listFiles()`, `setSetting()`, `getSetting()`, `closeApp()`, `getAppVersion()`
  - *Location*: `app/src/main/java/com/ledit/android/bridge/JsInterface.java`

- [ ] **T012** Register JsInterface with WebView
  - *Status*: Pending
  - *Completion*: `webView.addJavascriptInterface(new JsInterface(this, callback), "leditAndroid")` called after WebView initialization

- [ ] **T013** Implement callback mechanism for async operations
  - *Status*: Pending
  - *Completion*: `webView.post(() -> webView.evaluateJavascript())` pattern implemented for returning data to WebUI

- [ ] **T014** Create JavaScript test harness for interface verification
  - *Status*: Pending
  - *Completion*: Console test: `window.leditAndroid.getAppVersion()` returns app version string

---

## Phase 3: Ledis Server Integration

### Todo Items

- [ ] **T020** Create `LedisService` class extending `Service`
  - *Status*: Pending
  - *Completion*: Service class with onStartCommand() and onDestroy() lifecycle methods
  - *Location*: `app/src/main/java/com/ledit/android/service/LedisService.java`

- [ ] **T021** Integrate NanoHttpd for embedded HTTP server
  - *Status*: Pending
  - *Completion*: NanoHttpd dependency added to build.gradle, custom server class created

- [ ] **T022** Configure server to bind to localhost:54000
  - *Status*: Pending
  - *Completion*: Server binds to 127.0.0.1:54000, not exposed to external network

- [ ] **T023** Implement service lifecycle tied to WebViewFragment
  - *Status*: Pending
  - *Completion*: Service starts in onStart(), stops in onStop()

- [ ] **T024** Implement server connection verification
  - *Status*: Pending
  - *Completion*: `isServerRunning()` method verifies server responds before WebView loads URL

---

## Phase 4: WebView Configuration & Security

### Todo Items

- [ ] **T030** Configure WebViewSettings for editor performance
  - *Status*: Pending
  - *Completion*: JavaScript enabled, DOM storage enabled, database enabled, cache enabled, viewport settings configured

- [ ] **T031** Configure WebViewClient
  - *Status*: Pending
  - *Completion*: onPageStarted() shows progress, onPageFinished() hides progress, shouldOverrideUrlLoading() handles navigation

- [ ] **T032** Configure WebChromeClient
  - *Status*: Pending
  - *Completion*: onProgressChanged() updates loading indicator, onReceivedTitle() updates activity title

- [ ] **T033** Handle SSL errors appropriately
  - *Status*: Pending
  - *Completion*: SslErrorHandler.proceed() called for localhost development

- [ ] **T034** Disable WebView debug in production
  - *Status*: Pending
  - *Completion*: `WebView.setWebContentsDebuggingEnabled(false)` in production builds

- [ ] **T035** Implement cleanup in onDestroy()
  - *Status*: Pending
  - *Completion*: clearCache(), clearHistory(), clearFormData(), removeJavascriptInterface() called

---

## Phase 5: Testing & Verification

### Todo Items

- [ ] **T040** Verify WebView loads WebUI from localhost:54000
  - *Status*: Pending
  - *Completion*: Visual verification - ledit editor interface renders in WebView

- [ ] **T041** Verify JavaScript interface accessible
  - *Status*: Pending
  - *Completion*: `window.leditAndroid.getAppVersion()` returns version string in console

- [ ] **T042** Verify bidirectional communication
  - *Status*: Pending
  - *Completion*: Save file from WebUI triggers native callback, data flows back to JavaScript

- [ ] **T043** Verify service lifecycle
  - *Status*: Pending
  - *Completion*: Background app → server stops; foreground → server restarts

- [ ] **T044** Performance testing - WebUI load time < 3 seconds
  - *Status*: Pending
  - *Completion*: Cold start test passes < 3s target

- [ ] **T045** Performance testing - JS ↔ Android call latency < 100ms
  - *Status*: Pending
  - *Completion*: Timing measurement passes < 100ms target

- [ ] **T046** Memory usage verification < 100MB
  - *Status*: Pending
  - *Completion*: Memory profiler shows < 100MB additional memory used

- [ ] **T047** Offline functionality test
  - *Status*: Pending
  - *Completion*: WebUI functions with network disabled (local server)

- [ ] **T048** Error recovery test - server fails to start
  - *Status*: Pending
  - *Completion*: Graceful error message displayed when server unavailable

- [ ] **T049** Visual checkpoints verification
  - [ ] WebView displays ledit editor interface
  - [ ] Editor accepts text input
  - [ ] Formatting buttons function
  - [ ] File save/open integrates with Android
  - [ ] Loading indicator shows during page load
  - [ ] Error states display appropriate messages

---

## Phase 6: Compatibility & Polish

### Todo Items

- [ ] **T050** Verify minimum API 21 (Android 5.0) compatibility
  - *Status*: Pending
  - *Completion*: App runs on API 21 device/emulator

- [ ] **T051** Verify target API 34 (Android 14) compatibility
  - *Status*: Pending
  - *Completion*: App compiles and runs on API 34 device/emulator

- [ ] **T052** Test orientation handling - portrait and landscape
  - *Status*: Pending
  - *Completion*: WebView and editor function correctly in both orientations

- [ ] **T053** Chrome WebView compatibility across devices
  - *Status*: Pending
  - *Completion*: Functions on different device manufacturers

---

## Summary

| Status | Count |
|--------|-------|
| Pending | 28 |
| In Progress | 0 |
| Completed | 0 |
| **Total** | **28** |

---

*Generated: 2025-04-04*
*Component: 04-webui-integration*