#!/bin/bash
# Build script for ledit Android app
# Run this on a machine with proper Android SDK and NDK installed

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Ledit Android Build Script ===${NC}"

# Check prerequisites
check_prereq() {
    if ! command -v go &> /dev/null; then
        echo -e "${RED}Error: Go is not installed${NC}"
        exit 1
    fi
    if ! command -v java &> /dev/null; then
        echo -e "${RED}Error: Java is not installed${NC}"
        exit 1
    fi
    if [ -z "$ANDROID_HOME" ]; then
        echo -e "${RED}Error: ANDROID_HOME is not set${NC}"
        exit 1
    fi
    echo -e "${GREEN}Prerequisites OK${NC}"
}

# Install gomobile if needed
install_gomobile() {
    if ! command -v gomobile &> /dev/null; then
        echo -e "${YELLOW}Installing gomobile...${NC}"
        go install golang.org/x/mobile/cmd/gomobile@latest
        go get golang.org/x/mobile/cmd/gomobile@latest
    fi
}

# Build Go AAR
build_go_aar() {
    echo -e "${GREEN}Building Go AAR...${NC}"
    
    mkdir -p app/libs
    
    gomobile bind -target=android \
        -androidapi=24 \
        -javapkg=com.ledit.editor \
        -out=app/libs/ledit.aar \
        ./bind
    
    echo -e "${GREEN}Go AAR built successfully${NC}"
}

# Build native PTY library
build_native_lib() {
    echo -e "${GREEN}Building native PTY library...${NC}"
    
    if [ -z "$ANDROID_NDK_HOME" ]; then
        echo -e "${YELLOW}Warning: ANDROID_NDK_HOME not set, skipping native build${NC}"
        return
    fi
    
    cd app/src/main/jni
    $ANDROID_NDK_HOME/ndk-build
    cd ../../../../../
    
    # Copy to libs directory
    mkdir -p app/libs
    for arch in armeabi-v7a arm64-v8a x86 x86_64; do
        if [ -d "app/src/main/libs/$arch" ]; then
            mkdir -p "app/libs/$arch"
            cp app/src/main/libs/$arch/*.so "app/libs/$arch/" 2>/dev/null || true
        fi
    done
    
    echo -e "${GREEN}Native library built${NC}"
}

# Build Android APK
build_apk() {
    echo -e "${GREEN}Building APK...${NC}"
    
    ./gradlew assembleDebug
    
    echo -e "${GREEN}APK built successfully${NC}"
}

# Main
check_prereq
install_gomobile
build_go_aar
build_native_lib
build_apk

echo -e "${GREEN}=== Build Complete ===${NC}"
echo "APK location: app/build/outputs/apk/debug/app-debug.apk"