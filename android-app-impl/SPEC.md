# Android App Implementation Script - Specification

## Overview

This document specifies the implementation script that iterates through the Android app spec TODOs and executes them.

## Purpose

Create a Python script that:
1. Reads SPEC.md and TODOS.md from each component folder in `docs/android-app-spec/`
2. Works through each component sequentially (in dependency order)
3. Tracks progress and status
4. Marks completed items in the TODOS.md files

## Script Location

`android-app-impl/run_impl.py` — Main implementation script

## Implementation Order

The script processes components in this order:

| # | Component | Folder | Purpose |
|---|-----------|--------|---------|
| 1 | go-mobile | 01-go-mobile | Compile Go ledit code to Android library |
| 2 | terminal-pty | 02-terminal-pty | PTY subsystem for terminal |
| 3 | emulator-view | 03-emulator-view | Terminal rendering |
| 4 | android-shell | 05-android-shell | App UI structure |
| 5 | webui-integration | 04-webui-integration | WebView + ledit server |
| 6 | background-service | 06-background-service | Foreground service |
| 7 | shell-bundle | 07-shell-bundle | Bundled shell |

Note: 04-webui-integration is done after 05-android-shell because the UI needs to exist first.

## Script Features

- **Progress tracking**: Maintains state of what's been completed
- **Component sequencing**: Follows dependency order
- **Status updates**: Marks completed items in TODOS.md
- **Logging**: Clear output of what's being worked on
- **Resume support**: Can continue from where left off

## Output

Script updates:
- `TODOS.md` files in each component folder (marks items as completed)
- Creates working directory for implementation files
- Logs progress to console

## Git Ignore

The `android-app-impl/` folder is added to `.gitignore` to exclude implementation artifacts from version control.