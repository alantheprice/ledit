# 04-WebUI Integration Specification

## 1. Component Overview

This component handles the integration of the ledit WebUI into an Android WebView, enabling the rich text editor to run as an embedded web application within the native Android app.

### 1.1 Purpose

The WebUI integration allows users to access ledit's full-featured editor interface directly within the Android application, providing a native-like experience while leveraging the existing web-based frontend.

### 1.2 Architecture Summary

```
┌─────────────────────────────────────────────────────────────┐
│                      Android App                             │
│  ┌─────────────────┐    ┌─────────────────────────────────┐ │
│  │  MainActivity   │───▶│      WebView Component          │ │
│  │  (Native UI)    │    │  ┌───────────────────────────┐  │ │
│  │                 │    │  │   WebView                 │  │ │
│  │                 │    │  │   (Embedded Browser)      │  │ │
│  │                 │    │  │   ┌───────────────────┐  │  │ │
│  │                 │    │  │   │ ledit WebUI       │  │  │ │
│  │                 │    │  │   │ (localhost:54000) │  │  │ │
│  └─────────────────┘    │   │   └───────────────────┘  │  │ │
│                         │   └───────────────────────────┘  │ │
│  ┌─────────────────┐    └─────────────────────────────────┘ │
│  │  LedisService   │                                         │
│  │  (Background)   │    JavaScript Interface                  │
│  └─────────────────┘    (Android ←→ WebView Bridge)          │
└─────────────────────────────────────────────────────────────┘
```

### 1.3 Key Requirements

- Embed WebView that loads ledit WebUI from local server
- Establish bidirectional communication between Android and WebUI
- Run ledit server on localhost:54000 within the app
- Configure WebView for optimal editor performance
- Handle permissions and security appropriately

---

## 2. WebView Integration Approach

### 2.1 Loading the WebUI

The WebView loads the ledit WebUI from the local HTTP server running at `http://localhost:54000`.

```java
// Primary URL loading
webView.loadUrl("http://localhost:54000");
```

### 2.2 Integration Points

| Component | Responsibility |
|-----------|-----------------|
| `MainActivity` | Hosts the WebView, manages lifecycle |
| `WebViewFragment` | Encapsulates WebView logic and UI |
| `LedisService` | Manages the embedded HTTP server |
| `JsInterface` | Handles JavaScript ↔ Android communication |

### 2.3 Fragment Architecture

The WebView is encapsulated in `WebViewFragment` for modularity:

```
WebViewFragment
├── onCreateView()
│   └── Inflates layout with WebView
├── onViewCreated()
│   └── Configures WebView settings
├── onStart()
│   └── Starts LedisService if needed
├── onStop()
│   └── Stops service when backgrounded
└── getWebView()
    └── Exposes WebView for testing
```

### 2.4 Layout Structure

```xml
<?xml version="1.0" encoding="utf-8"?>
<FrameLayout
    xmlns:android="http://schemas.android.com/apk/res/android"
    android:layout_width="match_parent"
    android:layout_height="match_parent">

    <WebView
        android:id="@+id/webView"
        android:layout_width="match_parent"
        android:layout_height="match_parent" />

    <ProgressBar
        android:id="@+id/progressBar"
        android:layout_width="wrap_content"
        android:layout_height="wrap_content"
        android:layout_gravity="center"
        android:visibility="gone" />

</FrameLayout>
```

---

## 3. JavaScript Interface

### 3.1 Interface Design

The JavaScript interface enables the WebUI running in WebView to communicate with the Android native layer for file operations, settings, and other native functionality.

### 3.2 Interface Class

```java
public class JsInterface {
    private final Activity activity;
    private final JsCallback callback;

    public JsInterface(Activity activity, JsCallback callback) {
        this.activity = activity;
        this.callback = callback;
    }

    // File Operations
    @JavascriptInterface
    public void saveFile(String content, String filename) { }

    @JavascriptInterface
    public void openFile(String filename) { }

    @JavascriptInterface
    public String[] listFiles() { }

    // Settings
    @JavascriptInterface
    public void setSetting(String key, String value) { }

    @JavascriptInterface
    public String getSetting(String key) { }

    // App Control
    @JavascriptInterface
    public void closeApp() { }

    @JavascriptInterface
    public String getAppVersion() { }
}
```

### 3.3 Interface Registration

```java
webView.addJavascriptInterface(new JsInterface(this, callback), "leditAndroid");
```

The interface is exposed to JavaScript as `window.leditAndroid`.

### 3.4 JavaScript Usage

```javascript
// Save a file from WebUI
window.leditAndroid.saveFile(JSON.stringify(documentState), 'notes.json');

// Retrieve a setting
const theme = window.leditAndroid.getSetting('theme');

// List available files
const files = JSON.parse(window.leditAndroid.listFiles());
```

### 3.5 Communication Flow

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   WebUI (JS)    │────▶│  WebView Bridge │────▶│  JsInterface    │
│                 │     │  (addJavascript │     │  (@Javascript   │
│  window.ledit   │     │   Interface)    │     │   Interface)    │
│  Android.save() │     │                 │     │                 │
└─────────────────┘     └─────────────────┘     └─────────────────┘
         │                                              │
         │              ┌─────────────────┐             │
         │              │  JsCallback     │             │
         │              │  (Interface)    │             │
         │              └────────┬────────┘             │
         │                       │                      │
         ▼                       ▼                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Native Android Layer                         │
│  - File I/O (via Storage Access Framework)                      │
│  - SharedPreferences for settings                                │
│  - Activity lifecycle management                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 3.6 Callback Interface

```java
public interface JsCallback {
    void onFileSaved(String filename);
    void onFileOpened(String content, String filename);
    void onError(String message);
}
```

### 3.7 Return Values from Android to WebView

For async operations, Android can call back into JavaScript:

```java
// From JsInterface or native code
webView.post(() -> {
    webView.evaluateJavascript(
        "leditAndroid.onFileLoaded('" + escapeJs(content) + "')", 
        null
    );
});
```

---

## 4. Ledis Server Integration

### 4.1 Server Configuration

The ledit server runs as an embedded HTTP server within the Android app process.

| Property | Value |
|----------|-------|
| Host | `127.0.0.1` (localhost only) |
| Port | `54000` |
| Protocol | HTTP |
| Bind | Loopback interface only |

### 4.2 Server Implementation

The server runs as a background service (`LedisService`) that:

1. Starts when the app enters foreground
2. Binds to localhost only (not exposed to network)
3. Serves the WebUI assets from app resources
4. Stops when the app moves to background

```java
public class LedisService extends Service {
    private NanoHttpd server;
    
    @Override
    public int onStartCommand(Intent intent, int flags, int startId) {
        server = new LedisHttpServer(54000);
        try {
            server.start();
        } catch (IOException e) {
            Log.e(TAG, "Failed to start server", e);
        }
        return START_STICKY;
    }
    
    @Override
    public void onDestroy() {
        if (server != null) {
            server.stop();
        }
        super.onDestroy();
    }
}
```

### 4.3 Server URL Handling

The WebView is configured to treat `localhost:54000` as a trusted origin:

```java
WebViewDatabase.getInstance(this).setShouldInterpolateTouchUrl(true);
// Or configure via ChromeClient for older Android versions
```

### 4.4 Service Lifecycle

```
App Launch
    │
    ▼
MainActivity.onCreate()
    │
    ▼
WebViewFragment.onStart()
    │
    ▼
LedisService.start() ──▶ Server starts on localhost:54000
    │
    ▼
WebView.loadUrl("http://localhost:54000")
    │
    ▼
WebUI renders ✓
```

```
App Backgrounded / Closed
    │
    ▼
WebViewFragment.onStop()
    │
    ▼
LedisService.stop() ──▶ Server stops
    │
    ▼
Resources released
```

### 4.5 Connection Verification

Before loading, verify the server is ready:

```java
private boolean isServerRunning() {
    try {
        HttpClient client = new HttpClient("127.0.0.1", 54000);
        return client.connect();
    } catch (IOException e) {
        return false;
    }
}
```

---

## 5. WebView Settings and Permissions

### 5.1 WebView Configuration

```java
private void configureWebView() {
    WebSettings settings = webView.getSettings();
    
    // JavaScript is required for WebUI functionality
    settings.setJavaScriptEnabled(true);
    
    // Allow file access within the app
    settings.setAllowFileAccess(true);
    
    // Allow content URIs
    settings.setAllowContentAccess(true);
    
    // Enable DOM storage for WebUI state
    settings.setDomStorageEnabled(true);
    
    // Enable database storage
    settings.setDatabaseEnabled(true);
    
    // Enable cache
    settings.setCacheMode(WebSettings.LOAD_DEFAULT);
    
    // Allow viewport scaling
    settings.setUseWideViewPort(true);
    settings.setLoadWithOverviewMode(true);
    
    // Allow zoom controls
    settings.setBuiltInZoomControls(true);
    settings.setDisplayZoomControls(false);
    
    // Enable JavaScript to access Android interface
    settings.setAllowFileAccessFromFileURLs(true);
    settings.setAllowUniversalAccessFromFileURLs(true);
    
    // Mixed content (if needed)
    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP) {
        settings.setMixedContentMode(WebSettings.MIXED_CONTENT_COMPATIBILITY_MODE);
    }
    
    // Enable hardware acceleration
    webView.setLayerType(View.LAYER_TYPE_HARDWARE, null);
}
```

### 5.2 Security Settings

```java
// Disable debug for production
WebView.setWebContentsDebuggingEnabled(false);

// Handle SSL errors appropriately
webView.setWebViewClient(new WebViewClient() {
    @Override
    public void onReceivedSslError(WebView view, SslErrorHandler handler, SslError error) {
        // For localhost development, allow
        // For production: handler.cancel()
        handler.proceed();
    }
});
```

### 5.3 Required Permissions

**AndroidManifest.xml:**

```xml
<!-- Internet permission for local server (loopback) -->
<uses-permission android:name="android.permission.INTERNET" />
<uses-permission android:name="android.permission.ACCESS_NETWORK_STATE" />

<!-- Optional: For file picker integration -->
<uses-permission android:name="android.permission.READ_EXTERNAL_STORAGE" />
<uses-permission android:name="android.permission.WRITE_EXTERNAL_STORAGE" />
```

### 5.4 WebChromeClient Configuration

```java
webView.setWebChromeClient(new WebChromeClient() {
    @Override
    public void onProgressChanged(WebView view, int newProgress) {
        if (newProgress < 100) {
            progressBar.setVisibility(View.VISIBLE);
        } else {
            progressBar.setVisibility(View.GONE);
        }
    }
    
    @Override
    public void onReceivedTitle(WebView view, String title) {
        // Update activity title if needed
    }
});
```

### 5.5 WebViewClient Configuration

```java
webView.setWebViewClient(new WebViewClient() {
    @Override
    public void onPageStarted(WebView view, String url, Bitmap favicon) {
        progressBar.setVisibility(View.VISIBLE);
    }
    
    @Override
    public void onPageFinished(WebView view, String url) {
        progressBar.setVisibility(View.GONE);
    }
    
    @Override
    public boolean shouldOverrideUrlLoading(WebView view, String url) {
        // Handle navigation within WebUI
        if (url.startsWith("http://localhost:54000")) {
            return false; // Let WebView handle
        }
        // Open external links in browser
        Intent intent = new Intent(Intent.ACTION_VIEW, Uri.parse(url));
        startActivity(intent);
        return true;
    }
});
```

### 5.6 ClearData on Exit

```java
@Override
protected void onDestroy() {
    // Clean up WebView
    webView.clearCache(true);
    webView.clearHistory();
    webView.clearFormData();
    
    // Remove JavaScript interface
    webView.removeJavascriptInterface("leditAndroid");
    
    super.onDestroy();
}
```

---

## 6. Success Criteria

### 6.1 Functional Requirements

| ID | Requirement | Verification |
|----|-------------|--------------|
| F1 | WebView loads ledit WebUI from localhost:54000 | Visual verification that editor renders |
| F2 | JavaScript interface is accessible from WebUI | Test calling `window.leditAndroid.getAppVersion()` |
| F3 | Bidirectional communication works | Save file from WebUI, verify native callback |
| F4 | WebView handles editor interactions | Type, format, save content via WebUI |
| F5 | Service starts/stops with activity lifecycle | Background app, verify server stops |

### 6.2 Performance Requirements

| ID | Requirement | Target |
|----|-------------|--------|
| P1 | WebUI load time | < 3 seconds on warm start |
| P2 | WebUI render time | < 500ms after server response |
| P3 | JavaScript ↔ Android call latency | < 100ms |
| P4 | Memory usage | < 100MB additional |

### 6.3 Security Requirements

| ID | Requirement | Verification |
|----|-------------|--------------|
| S1 | Server only binds to localhost | No external network access |
| S2 | JavaScript interface is namespaced | Uses `leditAndroid` namespace |
| S3 | No remote code execution vulnerabilities | Security review |
| S4 | SSL/TLS for any future remote access | Production requirement |

### 6.4 Compatibility Requirements

| ID | Requirement | Target |
|----|-------------|--------|
| C1 | Minimum Android API | API 21 (Android 5.0) |
| C2 | Target Android API | API 34 (Android 14) |
| C3 | WebView compatibility | Chrome WebView on all devices |
| C4 | Orientation handling | Support portrait and landscape |

### 6.5 Test Scenarios

1. **Cold Start**: Launch app → verify WebUI loads within 3 seconds
2. **Hot Restart**: Return to app → verify server re-starts and WebUI re-connects
3. **Background/Foreground**: Background app → server stops → foreground → server starts
4. **JavaScript Call**: Call `window.leditAndroid.getAppVersion()` → verify returns version
5. **File Save**: Save file from WebUI → verify file appears in app storage
6. **Offline**: Disable network → verify WebUI still functions (local server)
7. **Memory**: Open WebUI, perform operations → verify memory stays under limit
8. **Error Recovery**: Server fails to start → verify graceful error message

### 6.6 Visual Checkpoints

- [ ] WebView displays ledit editor interface
- [ ] Editor accepts text input
- [ ] Formatting buttons function
- [ ] File save/open dialogs integrate with Android
- [ ] Loading indicator shows during page load
- [ ] Error states display appropriate messages

---

## 7. Implementation Notes

### 7.1 File Structure

```
app/src/main/java/com/ledit/android/
├── ui/
│   ├── WebViewFragment.java
│   └── layout/
│       └── fragment_webview.xml
├── service/
│   └── LedisService.java
├── bridge/
│   ├── JsInterface.java
│   └── JsCallback.java
└── MainActivity.java

app/src/main/assets/
└── webui/ (embedded web assets if not served from server)
```

### 7.2 Dependencies

- NanoHttpd (embedded HTTP server)
- AndroidX WebView
- AndroidX Lifecycle components

### 7.3 Future Considerations

- Consider implementing WebView2 for newer Android versions
- Add support for push notifications via WebUI
- Implement proper state preservation across configuration changes