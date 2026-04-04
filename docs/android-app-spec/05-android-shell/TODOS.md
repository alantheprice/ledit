# Component: 05-android-shell — Todo List

## Todo Items

### Phase 1: Project Setup & Gradle Configuration

- **[ ]** Status: **pending**
  - Todo: Create Gradle wrapper and initial build.gradle files
  - Completion: `gradle wrapper` runs successfully, `./gradlew assembleDebug` executes

- **[ ]** Status: **pending**
  - Todo: Configure build.gradle with Android SDK versions (compileSdk 34, minSdk 24, targetSdk 34)
  - Completion: Gradle sync completes without errors

- **[ ]** Status: **pending**
  - Todo: Add AndroidX dependencies (core-ktx, appcompat, material, fragment, navigation, preference)
  - Completion: All dependencies resolve from Maven Central/Google repositories

- **[ ]** Status: **pending**
  - Todo: Create AndroidManifest.xml with required permissions and activity declarations
  - Completion: Manifest validates, app installs with all permissions granted

### Phase 2: MainActivity & Navigation Structure

- **[ ]** Status: **pending**
  - Todo: Create MainActivity with single-activity architecture
  - Completion: Activity launches without crashes, handles onCreate/onDestroy lifecycle

- **[ ]** Status: **pending**
  - Todo: Implement bottom navigation with BottomNavigationView
  - Completion: Navigation bar displays with 3 tabs (Terminal, WebUI, Settings), clicking switches fragments

- **[ ]** Status: **pending**
  - Todo: Create empty fragment placeholders (TerminalFragment, WebUIFragment, SettingsFragment)
  - Completion: Fragments instantiate and display basic text content

- **[ ]** Status: **pending**
  - Todo: Add Navigation component with NavGraph for fragment destinations
  - Completion: NavController manages fragment transactions, back stack works correctly

### Phase 3: UI Layouts & Resources

- **[ ]** Status: **pending**
  - Todo: Create main activity layout with CoordinatorLayout and BottomNavigationView
  - Completion: Layout renders correctly on device/emulator

- **[ ]** Status: **pending**
  - Todo: Create fragment layouts (terminal_fragment.xml, webui_fragment.xml, settings_fragment.xml)
  - Completion: Each fragment shows its layout when navigated to

- **[ ]** Status: **pending**
  - Todo: Add string resources for app name, navigation labels, menu items
  - Completion: All UI text displays correctly (no hardcoded strings visible)

- **[ ]** Status: **pending**
  - Todo: Add theme/colors.xml and styles.xml for Material Design
  - Completion: App uses Material theme, supports light/dark mode

### Phase 4: Settings & Preferences

- **[ ]** Status: **pending**
  - Todo: Create SharedPreferences wrapper (PreferencesManager object)
  - Completion: Preferences read/write correctly, persist across app restarts

- **[ ]** Status: **pending**
  - Todo: Create preference XML (preferences.xml) with all app settings
  - Settings: theme, font_size, font_family, shell_path, scrollback_lines, keep_screen_on, vibrate_on_keypress, webui_url, background_service
  - Completion: PreferenceScreen displays all settings with correct types

- **[ ]** Status: **pending**
  - Todo: Implement SettingsFragment extending PreferenceFragmentCompat
  - Completion: Settings fragment loads preferences, changes save automatically

- **[ ]** Status: **pending**
  - Todo: Add theme switching logic (light/dark/system)
  - Completion: Theme changes apply immediately when preference is modified

### Phase 5: Terminal Integration

- **[ ]** Status: **pending**
  - Todo: Create TerminalFragment that hosts TerminalView from 03-emulator-view
  - Completion: Terminal view renders and accepts input

- **[ ]** Status: **pending**
  - Todo: Connect PTYSession to TerminalView (read/write streams)
  - Completion: Text typed in terminal appears in PTY, shell output displays in terminal

- **[ ]** Status: **pending**
  - Todo: Apply font size preference to TerminalView
  - Completion: Changing font_size preference updates terminal text size

- **[ ]** Status: **pending**
  - Todo: Implement keep_screen_on preference
  - Completion: Screen stays on when enabled, turns off normally when disabled

### Phase 6: WebUI Integration

- **[ ]** Status: **pending**
  - Todo: Create WebUIFragment that hosts WebView
  - Completion: WebView component exists in fragment

- **[ ]** Status: **pending**
  - Todo: Configure WebView with JavaScript enabled, DOM storage
  - Completion: WebView can load web pages, JavaScript executes

- **[ ]** Status: **pending**
  - Todo: Load WebUI URL from preferences (default: http://localhost:54000)
  - Completion: WebView navigates to configured URL on fragment display

- **[ ]** Status: **pending**
  - Todo: Handle WebView navigation errors (show error state)
  - Completion: Network errors display user-friendly message

### Phase 7: Service Integration (06-background-service)

- **[ ]** Status: **pending**
  - Todo: Create service binding in MainActivity
  - Completion: Can bind/unbind to LeditAgentService

- **[ ]** Status: **pending**
  - Todo: Request POST_NOTIFICATIONS permission at runtime (Android 13+)
  - Completion: Permission prompt appears, user choice handled correctly

- **[ ]** Status: **pending**
  - Todo: Start/stop background service based on preference
  - Completion: Service runs when preference enabled, stops when disabled

### Phase 8: Build & Packaging

- **[ ]** Status: **pending**
  - Todo: Run `gradlew assembleDebug` to build debug APK
  - Completion: APK generated in app/build/outputs/apk/debug/

- **[ ]** Status: **pending**
  - Todo: Configure ProGuard rules for release build
  - Completion: Release APK builds without errors, classes preserved correctly

- **[ ]** Status: **pending**
  - Todo: Test APK installs on Android 7.0+ device/emulator
  - Completion: App installs, launches, all features work

### Phase 9: Polish & UX

- **[ ]** Status: **pending**
  - Todo: Add loading/progress indicators for WebView navigation
  - Completion: ProgressBar shows during page load

- **[ ]** Status: **pending**
  - Todo: Add exit confirmation dialog when pressing back on terminal
  - Completion: Dialog appears with "Exit app?" options

- **[ ]** Status: **pending**
  - Todo: Add content descriptions for accessibility
  - Completion: TalkBack reads navigation elements correctly

- **[ ]** Status: **pending**
  - Todo: Verify keyboard navigation works (tab through elements)
  - Completion: All interactive elements focusable via keyboard

---

## Quick Reference

| Priority | Item | Status |
|----------|------|--------|
| High | Gradle build setup | pending |
| High | MainActivity & navigation | pending |
| High | Settings implementation | pending |
| High | Terminal fragment integration | pending |
| High | WebView fragment integration | pending |
| Medium | Service binding | pending |
| Medium | ProGuard & release build | pending |
| Low | Loading indicators | pending |
| Low | Accessibility | pending |
| Low | Keyboard navigation | pending |