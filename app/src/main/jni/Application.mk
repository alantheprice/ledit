# NDK Application Makefile for ledit Android PTY implementation
# Specifies the ABI targets for the native library

# Target ABI: armeabi-v7a, arm64-v8a, x86, x86_64
# Using all to support broadest range of devices
APP_ABI := armeabi-v7a arm64-v8a x86 x86_64

# Use the default NDK level
# APP_PLATFORM := android-21

# C++ standard library
APP_STL := c++_shared

# Build type: debug or release
APP_BUILD_MODE := release