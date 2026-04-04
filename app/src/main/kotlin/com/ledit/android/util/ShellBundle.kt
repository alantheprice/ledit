package com.ledit.android.util

import android.content.Context
import android.util.Log
import java.io.File
import java.io.FileOutputStream
import java.io.InputStream

/**
 * ShellBundle - Utility for managing bundled shell tools (toybox) in the APK.
 * 
 * This class handles:
 * - Extracting toybox binary from assets to app's private directory
 * - Setting executable permissions
 * - Providing paths to bundled commands
 */
object ShellBundle {

    private const val TAG = "ShellBundle"
    
    // Binary filename in assets
    private const val TOYBOX_ASSET = "toybox"
    
    // Directory for extracted binaries
    private const val BIN_DIR = "bin"
    
    // Cached binary path
    private var binaryPath: String? = null
    
    /**
     * Initialize the shell bundle - extract binary if needed
     */
    fun initialize(context: Context): Boolean {
        if (binaryPath != null && File(binaryPath!!).exists()) {
            return true // Already initialized
        }
        
        return try {
            extractBinary(context)
        } catch (e: Exception) {
            Log.e(TAG, "Failed to initialize shell bundle", e)
            false
        }
    }
    
    /**
     * Extract toybox binary from assets to private directory
     */
    private fun extractBinary(context: Context): Boolean {
        val binDir = File(context.filesDir, BIN_DIR)
        if (!binDir.exists()) {
            binDir.mkdirs()
        }
        
        val binaryFile = File(binDir, TOYBOX_ASSET)
        
        // Check if already extracted
        if (binaryFile.exists()) {
            binaryPath = binaryFile.absolutePath
            return setExecutable(binaryFile)
        }
        
        // Try to extract from assets
        return try {
            val assets = context.assets
            val inputStream: InputStream = try {
                assets.open("$TOYBOX_ASSET")
            } catch (e: Exception) {
                Log.w(TAG, "No toybox binary in assets, using system shell")
                return false
            }
            
            inputStream.use { input ->
                FileOutputStream(binaryFile).use { output ->
                    input.copyTo(output)
                }
            }
            
            binaryPath = binaryFile.absolutePath
            setExecutable(binaryFile)
            
            Log.d(TAG, "Extracted toybox to: $binaryPath")
            true
        } catch (e: Exception) {
            Log.e(TAG, "Failed to extract toybox", e)
            false
        }
    }
    
    /**
     * Set executable permission on the binary
     */
    private fun setExecutable(path: String): Boolean {
        return try {
            val file = File(path)
            val success = file.setExecutable(true)
            if (success) {
                Log.d(TAG, "Set executable: $path")
            } else {
                Log.w(TAG, "Failed to set executable: $path")
            }
            success
        } catch (e: Exception) {
            Log.e(TAG, "Error setting executable", e)
            false
        }
    }
    
    /**
     * Get the path to the toybox binary
     */
    fun getBinaryPath(): String? = binaryPath
    
    /**
     * Check if toybox is available
     */
    fun isAvailable(): Boolean {
        return binaryPath != null && File(binaryPath!!).exists()
    }
    
    /**
     * Execute a command using the bundled shell
     * @return Pair of (exitCode, output)
     */
    fun execute(context: Context, command: String): Pair<Int, String> {
        if (!initialize(context)) {
            return Pair(-1, "Shell bundle not available")
        }
        
        val binary = binaryPath ?: return Pair(-1, "No binary")
        
        return try {
            val process = ProcessBuilder(binary, command)
                .redirectErrorStream(true)
                .start()
            
            val output = process.inputStream.bufferedReader().readText()
            val exitCode = process.waitFor()
            
            Pair(exitCode, output)
        } catch (e: Exception) {
            Pair(-1, "Execution failed: ${e.message}")
        }
    }
    
    /**
     * Execute a command with arguments
     */
    fun execute(context: Context, vararg args: String): Pair<Int, String> {
        if (!initialize(context)) {
            return Pair(-1, "Shell bundle not available")
        }
        
        val binary = binaryPath ?: return Pair(-1, "No binary")
        
        return try {
            val process = ProcessBuilder(binary, *args)
                .redirectErrorStream(true)
                .start()
            
            val output = process.inputStream.bufferedReader().readText()
            val exitCode = process.waitFor()
            
            Pair(exitCode, output)
        } catch (e: Exception) {
            Pair(-1, "Execution failed: ${e.message}")
        }
    }
    
    /**
     * List available commands in toybox
     */
    fun listCommands(context: Context): List<String> {
        val result = execute(context, "--list")
        if (result.first == 0) {
            return result.second.lines().map { it.trim() }.filter { it.isNotEmpty() }
        }
        return emptyList()
    }
    
    /**
     * Get the shell to use for terminal
     * Returns either bundled toybox or system shell
     */
    fun getShell(context: Context): String {
        return if (isAvailable()) {
            binaryPath!!
        } else {
            "/system/bin/sh"
        }
    }
}