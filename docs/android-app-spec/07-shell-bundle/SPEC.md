# 07 Shell Bundle Component Specification

## Overview

The `07-shell-bundle` component covers bundling a shell utility suite (toybox or busybox) into an Android application for use on unrooted devices. This provides a more complete shell experience than the limited `/system/bin/sh` available on stock Android.

## Purpose

On unrooted Android devices, the system shell (`/system/bin/sh`) is severely limited:
- Missing common POSIX utilities (ls, cat, grep, tar, etc.)
- Incompatible behavior with standard Linux commands
- Restricted functionality for scripting and automation

Bundling a full-featured shell suite solves these limitations by providing:
- Complete set of standard Unix utilities
- Consistent behavior across Android versions
- Self-contained binaries that don't require root access

## Shell Options

### Toybox

- **License**: 0-clause BSD (public domain equivalent)
- **Size**: Smaller footprint (~300KB for full suite)
- **Status**: Default choice for most Android builds
- **Maintenance**: Actively maintained, used in Android AOSP since 2016
- **Commands**: ~200+ utilities included

### Busybox

- **License**: GPLv2
- **Size**: Very small single binary (~1-2MB)
- **Status**: Historical standard, still widely used
- **Maintenance**: Active development continues
- **Commands**: ~400+ applets, configurable at build time

### Shim Layer (toybox-bb shim)

- **Purpose**: Provides compatibility layer between toybox and busybox commands
- **Use case**: Scripts expecting specific busybox applet names
- **Implementation**: Symlinks or wrapper scripts

**Recommendation**: Use **toybox** for its permissive licensing and integration with Android AOSP.

## Bundling into APK

### Build Process

1. **Cross-compile** toybox/busybox for Android target architecture
   - Supported architectures: arm, arm64, x86, x86_64
   - Use Android NDK with appropriate toolchain

2. **Configuration**
   - Select desired utilities/applets
   - Configure static vs dynamic linking (static preferred for portability)
   - Set appropriate optimization flags

3. **APK Integration**
   - Place compiled binary in `assets/` or `lib/` directory
   - Native libraries: `lib/<abi>/libname.so`
   - Executable assets require extraction to filesystem at runtime

### File Structure Example

```
app/
├── src/main/
│   ├── assets/
│   │   └── toybox          # Pre-compiled toybox binary
│   └── jni/
│       └── ...            # Optional: NDK build for custom compilation
```

## Executing Bundled Binaries

### Runtime Extraction

Binary must be extracted from APK assets to filesystem before execution:

```java
// Android/Java Example
public class ShellExtractor {
    public static File extractAsset(Context context, String assetName) throws IOException {
        File outFile = new File(context.getFilesDir(), assetName);
        if (!outFile.exists()) {
            try (InputStream is = context.getAssets().open(assetName);
                 OutputStream os = new FileOutputStream(outFile)) {
                byte[] buffer = new byte[4096];
                int bytesRead;
                while ((bytesRead = is.read(buffer)) != -1) {
                    os.write(buffer, 0, bytesRead);
                }
            }
            outFile.setExecutable(true);
        }
        return outFile;
    }
}
```

### Execution Methods

1. **ProcessBuilder** (Java/Kotlin)
   ```java
   ProcessBuilder pb = new ProcessBuilder(toyboxPath, "ls", "-la");
   pb.directoryworkingDir);
   pb.start();
   ```

2. **Runtime.exec()** (Legacy)
   ```java
   Runtime.getRuntime().exec(toyboxPath + " ls -la");
   ```

3. **Native JNI Execution**
   - For tighter integration, use JNI to spawn processes
   - Better control over stdin/stdout/stderr

### Path Configuration

- Store in app-private directory: `context.getFilesDir()`
- Optional: Add to PATH via environment modification
- Consider `bindMount` or `link` for common command names

## Licensing Considerations

### Toybox: 0-clause BSD

- **Commercial use**: Allowed
- **Distribution**: Allowed with attribution
- **Source code**: Not required to distribute
- **Warranty**: None (AS-IS)
- **Implication**: No copyleft restrictions, fully compatible with proprietary apps

### Busybox: GPLv2

- **Commercial use**: Allowed with obligations
- **Distribution**: Must provide source or offer to provide
- **Modification**: Must document changes
- **Implication**: May require source disclosure for linked applications

**Recommendation**: Use **toybox** to avoid GPLv2 obligations and simplify licensing compliance.

## Success Criteria

1. **Build Success**
   - [ ] Toybox compiles for all target Android architectures (arm, arm64, x86, x86_64)
   - [ ] Binary runs on unrooted Android without crashes

2. **Functionality**
   - [ ] Core utilities work: ls, cat, grep, find, tar, gzip, wc, etc.
   - [ ] Shell scripts execute correctly with bundled shell
   - [ ] No dependency on system utilities that may be missing

3. **Integration**
   - [ ] Binary extracts from APK assets successfully
   - [ ] Executable permissions set correctly
   - [ ] Process execution handles stdin/stdout/stderr properly

4. **Performance**
   - [ ] Binary size reasonable (< 500KB for toybox)
   - [ ] Execution time acceptable for interactive use

5. **Legal**
   - [ ] Licensing allows intended distribution model
   - [ ] All dependencies also have compatible licenses