#!/bin/bash
# Test script to verify new default behavior

echo "Testing agent command with no arguments (should start UI):"
echo "Command: ./ledit agent"
echo "Expected: Should start interactive TUI mode"
echo ""

echo "Testing agent command with arguments (should run directly):" 
echo "Command: ./ledit agent \"test message\""
echo "Expected: Should run in direct mode without UI"
echo ""

echo "Testing code command with no arguments (should start UI):"
echo "Command: ./ledit code"  
echo "Expected: Should start interactive TUI mode"
echo ""

echo "Testing code command with arguments (should run directly):"
echo "Command: ./ledit code \"generate a hello world function\""
echo "Expected: Should run in direct mode without UI"

