# NDK Android Makefile for libtermexec - Native PTY support for ledit
# Builds a shared library that provides PTY functionality to the Android app

LOCAL_PATH := $(call my-dir)

# Clear variables
include $(CLEAR_VARS)

# Library name
LOCAL_MODULE := termexec

# Source files
LOCAL_SRC_FILES := term_exec.c

# Compiler flags
LOCAL_CFLAGS := -Wall -Wextra -Werror -O2 -fPIC

# Use Bionic libc
LOCAL_LDLIBS := -llog -lutils

# Generate export symbols
LOCAL_EXPORT_C_INCLUDES := $(LOCAL_PATH)

# Build as shared library
include $(BUILD_SHARED_LIBRARY)