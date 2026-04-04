package com.ledit.android.util

import android.content.Context

/**
 * ShellBundle - High-level utility for managing bundled shell tools (toybox) in the APK.
 *
 * This class provides a convenient wrapper around [ShellBundleManager] for common use cases.
 * It handles:
 * - Extracting toybox binary from assets to app's private directory
 * - Setting executable permissions
 * - Providing paths to bundled commands
 * - Executing shell commands
 *
 * ## Usage
 *
 * Initialize at app startup:
 * ```
 * ShellBundle.initialize(context)
 * ```
 *
 * Execute commands:
 * ```
 * val result = ShellBundle.execute(context, "ls", "-la")
 * if (result.isSuccess) {
 *     println(result.stdout)
 * }
 * ```
 *
 * Get the shell to use for terminal sessions:
 * ```
 * val shell = ShellBundle.getShell(context)
 * ```
 *
 * @see ShellBundleManager
 */
object ShellBundle {

    /**
     * Initialize the shell bundle - extract binary if needed
     * @return true if initialization succeeded
     */
    fun initialize(context: Context): Boolean {
        return ShellBundleManager.initialize(context)
    }

    /**
     * Initialize with download fallback if assets not available
     * @param context Application context
     * @param forceDownload If true, skip assets and download directly
     * @return true if initialization succeeded
     */
    fun initializeWithFallback(context: Context, forceDownload: Boolean = false): Boolean {
        return ShellBundleManager.initializeWithFallback(context, forceDownload)
    }

    /**
     * Get the path to the toybox binary
     * @return Absolute path or null if not available
     */
    fun getBinaryPath(): String? = ShellBundleManager.getBinaryPath()

    /**
     * Check if toybox is available
     * @return true if shell bundle is available and working
     */
    fun isAvailable(): Boolean = ShellBundleManager.isAvailable()

    /**
     * Execute a command using the bundled shell
     * @param context Application context
     * @param command Command string to execute
     * @return ExecutionResult with exit code, stdout, and stderr
     */
    fun execute(context: Context, command: String): ExecutionResult {
        return ShellBundleManager.execute(context, command)
    }

    /**
     * Execute a command with arguments
     * @param context Application context
     * @param args Command arguments
     * @return ExecutionResult with exit code, stdout, and stderr
     */
    fun execute(context: Context, vararg args: String): ExecutionResult {
        return ShellBundleManager.execute(context, *args)
    }

    /**
     * Execute a command with stdin input
     * @param context Application context
     * @param stdin Input to provide to stdin (can be null)
     * @param args Command arguments
     * @return ExecutionResult with exit code, stdout, and stderr
     */
    fun executeWithStdin(context: Context, stdin: String?, vararg args: String): ExecutionResult {
        return ShellBundleManager.executeWithStdin(context, stdin, *args)
    }

    /**
     * Create an interactive process for streaming I/O
     * @param context Application context
     * @param args Command arguments
     * @return ProcessHandle for interacting with the process, or null if failed
     */
    fun createInteractiveProcess(context: Context, vararg args: String): ProcessHandle? {
        return ShellBundleManager.createInteractiveProcess(context, *args)
    }

    /**
     * List available commands in toybox
     * @param context Application context
     * @return List of command names
     */
    fun listCommands(context: Context): List<String> {
        return ShellBundleManager.listCommands(context)
    }

    /**
     * Check if a specific command is available
     * @param context Application context
     * @param command Command name to check
     * @return true if command is available
     */
    fun hasCommand(context: Context, command: String): Boolean {
        return ShellBundleManager.hasCommand(context, command)
    }

    /**
     * Get the version of toybox
     * @param context Application context
     * @return Version string or null if not available
     */
    fun getVersion(context: Context): String? {
        return ShellBundleManager.getVersion(context)
    }

    /**
     * Get the shell to use for terminal sessions
     * Returns either bundled toybox or system shell
     * @param context Application context
     * @return Path to shell executable
     */
    fun getShell(context: Context): String {
        return ShellBundleManager.getShellCommand(context)
    }

    /**
     * Set custom download URL for shell bundle
     * @param url Base URL for downloading toybox binaries
     */
    fun setDownloadUrl(url: String) {
        ShellBundleManager.setDownloadUrl(url)
    }

    /**
     * Get the target architecture for the current device
     * @return Architecture string (arm64, arm, x86_64, x86)
     */
    fun getTargetArchitecture(): String {
        return ShellBundleManager.getTargetArchitecture()
    }
}
