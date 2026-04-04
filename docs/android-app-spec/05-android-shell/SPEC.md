# Component: 05-android-shell

## Overview

The **05-android-shell** component provides the outer Android UI wrapper that holds the terminal emulator and WebView components together. It serves as the application's structural backbone, managing the main activity lifecycle, fragment-based navigation, user settings persistence, and build configuration.

## Purpose

- **Application Shell**: Create the top-level Android application structure that orchestrates all components
- **Navigation**: Implement smooth transitions between terminal view, WebUI, and settings screens
- **Settings**: Provide user-configurable preferences (theme, font size, shell path, etc.)
- **Build Configuration**: Configure Gradle and AndroidManifest for a production-ready APK

## Relationship to Other Components

```
┌─────────────────────────────────────────────────────────────────┐
│                      05-android-shell                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │ MainActivity│  │  Navigation │  │   Settings (Prefs)     │  │
│  │ (Shell)     │  │  (Fragments)│  │   (SharedPreferences)   │  │
│  └──────┬──────┘  └──────┬──────┘  └───────────┬─────────────┘  │
│         │                │                     │                │
│         ▼                ▼                     ▼                │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              Integrates with:                            │    │
│  │  - 02-terminal-pty (PTYSession)                         │    │
│  │  - 03-emulator-view (TerminalView)                      │    │
│  │  - 04-webui-integration (WebView)                       │    │
│  │  - 06-background-service (Service binding)            │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

## App Structure

### Activities

| Activity | Purpose |
|----------|---------|
| `MainActivity` | Primary activity hosting the terminal and WebView; handles navigation |
| `SettingsActivity` | Standalone activity for app preferences (optional, can be fragment) |

### Fragments

| Fragment | Purpose |
|----------|---------|
| `TerminalFragment` | Hosts the terminal emulator view |
| `WebUIFragment` | Hosts the ledit WebUI WebView |
| `SettingsFragment` | App configuration and preferences |

### Navigation

- **Bottom Navigation Bar**: Switch between Terminal, WebUI, Settings
- **Fragment-based**: Single Activity architecture with fragment transactions
- **Back Stack**: Proper back navigation handling

## UI Components Needed

### Main Layout Components

1. **CoordinatorLayout** — Root container for app bar and content
2. **BottomNavigationView** — Tab-based navigation
3. **FragmentContainerView** — Fragment destination
4. **Toolbar** — App bar with title and overflow menu

### Terminal View Layout

```xml
<FrameLayout>
  <TerminalView android:id="@+id/terminalView" />
</FrameLayout>
```

### WebView Layout

```xml
<WebView android:id="@+id/webView" />
```

### Settings Layout

- PreferenceFragmentCompat with XML preferences

## Settings and Preferences

### Stored Preferences

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `theme` | string | "system" | Theme: "light", "dark", "system" |
| `font_size` | int | 12 | Terminal font size in sp |
| `font_family` | string | "monospace" | Terminal font family |
| `shell_path` | string | "/system/bin/sh" | Default shell executable |
| `scrollback_lines` | int | 10000 | Scrollback buffer size |
| `keep_screen_on` | boolean | false | Prevent screen timeout |
| `vibrate_on_keypress` | boolean | false | Haptic feedback |
| `webui_url` | string | "http://localhost:54000" | WebUI server URL |
| `background_service` | boolean | true | Run agent in background |

### Data Storage

- **SharedPreferences**: Use `PreferenceManager.getDefaultSharedPreferences()`
- **Preferences**: Use AndroidX Preference library for type-safe access

## Permissions Handling

### Required Permissions

```xml
<uses-permission android:name="android.permission.INTERNET" />
<uses-permission android:name="android.permission.FOREGROUND_SERVICE" />
<uses-permission android:name="android.permission.FOREGROUND_SERVICE_SPECIAL_USE" />
<uses-permission android:name="android.permission.POST_NOTIFICATIONS" />
<uses-permission android:name="android.permission.VIBRATE" />
<uses-permission android:name="android.permission.WAKE_LOCK" />
```

### Permission Rationale

| Permission | Reason |
|------------|--------|
| `INTERNET` | Connect to WebUI server, download resources |
| `FOREGROUND_SERVICE` | Keep ledit agent running in background |
| `POST_NOTIFICATIONS` | Show notification when service is running (Android 13+) |
| `VIBRATE` | Haptic feedback on keypress |
| `WAKE_LOCK` | Prevent device sleep during terminal session |

### Runtime Permissions

- `POST_NOTIFICATIONS`: Request at runtime for Android 13+
- Other permissions declared in manifest (no runtime needed)

## Build Configuration

### Gradle (build.gradle)

```groovy
plugins {
    id 'com.android.application'
    id 'org.jetbrains.kotlin.android'
}

android {
    namespace 'com.ledit.android'
    compileSdk 34
    
    defaultConfig {
        applicationId "com.ledit.android"
        minSdk 24
        targetSdk 34
        versionCode 1
        versionName "1.0.0"
    }
    
    buildTypes {
        release {
            minifyEnabled true
            proguardFiles getDefaultProguardFile('proguard-android-optimize.txt'), 'proguard-rules.pro'
        }
        debug {
            debuggable true
        }
    }
    
    compileOptions {
        sourceCompatibility JavaVersion.VERSION_17
        targetCompatibility JavaVersion.VERSION_17
    }
    
    kotlinOptions {
        jvmTarget = '17'
    }
}

dependencies {
    implementation 'androidx.core:core-ktx:1.12.0'
    implementation 'androidx.appcompat:appcompat:1.6.1'
    implementation 'com.google.android.material:material:1.11.0'
    implementation 'androidx.preference:preference-ktx:1.2.1'
    implementation 'androidx.fragment:fragment-ktx:1.6.2'
    implementation 'androidx.navigation:navigation-fragment-ktx:2.7.6'
    
    // Terminal components
    implementation project(':emulator-view')
    
    // Go bindings (from 01-go-mobile)
    implementation project(':go-mobile-bindings')
}
```

### AndroidManifest.xml

```xml
<?xml version="1.0" encoding="utf-8"?>
<manifest xmlns:android="http://schemas.android.com/apk/res/android"
    xmlns:tools="http://schemas.android.com/tools">
    
    <uses-permission android:name="android.permission.INTERNET" />
    <uses-permission android:name="android.permission.FOREGROUND_SERVICE" />
    <uses-permission android:name="android.permission.FOREGROUND_SERVICE_SPECIAL_USE" />
    <uses-permission android:name="android.permission.POST_NOTIFICATIONS" />
    <uses-permission android:name="android.permission.VIBRATE" />
    <uses-permission android:name="android.permission.WAKE_LOCK" />
    
    <application
        android:allowBackup="true"
        android:icon="@mipmap/ic_launcher"
        android:label="@string/app_name"
        android:supportsRtl="true"
        android:theme="@style/Theme.Ledit"
        android:usesCleartextTraffic="true"
        tools:targetApi="34">
        
        <activity
            android:name=".MainActivity"
            android:exported="true"
            android:launchMode="singleTask"
            android:configChanges="orientation|screenSize|keyboardHidden">
            <intent-filter>
                <action android:name="android.intent.action.MAIN" />
                <category android:name="android.intent.category.LAUNCHER" />
            </intent-filter>
        </activity>
        
        <service
            android:name=".service.LeditAgentService"
            android:foregroundServiceType="specialUse"
            android:exported="false" />
        
    </application>
</manifest>
```

### ProGuard Rules

```proguard
# Keep terminal classes
-keep class com.ledit.terminal.** { *; }

# Keep Go bindings
-keep class com.ledit.gobindings.** { *; }

# Kotlin serialization
-keepattributes *Annotation*, InnerClasses
-dontnote kotlinx.serialization.AnnotationsKt

# Keep Parcelable implementations
-keepclassmembers class * implements android.os.Parcelable {
    static ** CREATOR;
}
```

## Key Implementation Classes

### MainActivity

```kotlin
class MainActivity : AppCompatActivity() {
    // Navigation setup
    // Fragment transaction handling
    // Service binding for background agent
    // Permission handling
    
    fun navigateToTerminal()
    fun navigateToWebUI()
    fun navigateToSettings()
}
```

### PreferencesManager

```kotlin
object PreferencesManager {
    fun getTheme(): ThemeMode
    fun getFontSize(): Int
    fun getShellPath(): String
    // ...
}
```

### NavigationController

```kotlin
class NavigationController(private val navController: NavController) {
    fun showTerminal()
    fun showWebUI()
    fun showSettings()
}
```

## Success Criteria

### Functional Criteria

- [ ] App launches and displays terminal view by default
- [ ] Bottom navigation switches between Terminal, WebUI, Settings
- [ ] Settings persist across app restarts
- [ ] Theme changes apply immediately
- [ ] Back button handling works correctly (exit confirmation if terminal active)

### Build Criteria

- [ ] Gradle build produces debug APK
- [ ] Release build produces signed APK
- [ ] No lint errors or warnings
- [ ] APK installs and runs on Android 7.0+ (API 24)

### Integration Criteria

- [ ] Terminal view renders correctly with PTY connection
- [ ] WebView loads WebUI at configured URL
- [ ] Settings changes reflect in terminal/WebUI components
- [ ] Background service starts/stops based on preference

### UX Criteria

- [ ] Smooth fragment transitions (no flicker)
- [ ] Proper loading states during WebView navigation
- [ ] Accessible: content descriptions on interactive elements
- [ ] Keyboard navigation works