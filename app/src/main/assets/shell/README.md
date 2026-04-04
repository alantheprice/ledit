# Shell Bundle Assets

This directory contains shell binaries for the LeDit Android app.

## Toybox Binary

The app uses [toybox](https://landley.net/toybox/) as the bundled shell utility suite.
Toybox is licensed under 0-clause BSD (public domain equivalent).

## Required Binaries

To enable the shell bundle functionality, add statically-linked toybox binaries
for the following architectures:

| Architecture | Asset Filename     | ABI                |
|--------------|-------------------|-------------------|
| arm64        | toybox-arm64      | arm64-v8a         |
| arm          | toybox-arm        | armeabi-v7a       |
| x86_64       | toybox-x86_64     | x86_64            |
| x86          | toybox-x86        | x86               |

## Build Instructions

### Using Toybox's Official Build

```bash
# Clone toybox repository
git clone https://github.com/landley/toybox.git
cd toybox

# For arm64 (Android)
make android_defconfig
make CROSS_COMPILE=aarch64-linux-gnu- -j$(nproc)

# The resulting binary is located at:
# ./toybox
```

### Using the provided build script

A build script is available in the project's `scripts/` directory that automates
cross-compilation for all supported architectures.

## Binary Requirements

- **Statically linked**: No dynamic library dependencies
- **Position independent**: Should work with ASLR
- **Minimal configuration**: Include essential utilities (ls, cat, grep, tar, etc.)

## Usage

The binary is extracted at runtime to the app's private directory:
`/data/data/com.ledit.android/files/bin/toybox`

This happens automatically via `ShellBundleManager` when the app initializes.
