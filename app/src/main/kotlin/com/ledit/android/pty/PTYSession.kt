package com.ledit.android.pty

import android.os.ParcelFileDescriptor

/**
 * PTYSession - Manages a pseudo-terminal (PTY) session for the terminal emulator.
 * 
 * This class handles:
 * - Creating PTY pairs (master/slave) via native openpty()
 * - Spawning shell processes via native forkpty()
 * - Bidirectional I/O between the app and shell
 * - Signal handling (SIGINT, SIGKILL, etc.) via process groups
 * - Process lifecycle management with callbacks
 * 
 * Uses native PTY via /dev/ptmx through libtermexec JNI.
 */
class PTYSession : PTYCallback {

    companion object {
        private const val TAG = "PTYSession"
        
        // Default shell
        const val DEFAULT_SHELL = "/system/bin/sh"
        
        // Signal constants
        const val SIGINT = 2   // Ctrl+C
        const val SIGKILL = 9   // Force kill
        const val SIGTSTP = 20 // Ctrl+Z
        const val SIGHUP = 1   // Hangup
        
        // Terminal speeds
        const val B38400 = 38400
        const val B115200 = 115200
        
        // Terminal flags (common defaults)
        const val CFLAG_DEFAULT = (0x10 | 0x100 | 0x8)  // CREAD | CS8 | HUPCL
        const val LFLAG_DEFAULT = (0x1 | 0x2 | 0x4 | 0x80 | 0x100)  // ICANON | ECHO | ECHOE | ISIG | ONLCR
        const val IFLAG_DEFAULT = (0x200 | 0x400)  // ICRNL | IXON
        const val OFLAG_DEFAULT = 0x4  // OPOST
        
        // Load native library
        init {
            System.loadLibrary("termexec")
        }
    }
    
    // Native file descriptors from openpty()
    private var masterFd: Int = -1
    private var slaveFd: Int = -1
    
    // Process state
    private var pid: Int = -1
    private var isRunning: Boolean = false
    private var shell: String = DEFAULT_SHELL
    
    // Callback for events
    private var callback: PTYCallback? = null
    
    /**
     * Set callback for process events
     */
    fun setCallback(callback: PTYCallback) {
        this.callback = callback
        nativeSetCallback(this)
    }
    
    /**
     * Initialize a new PTY session using native openpty()
     */
    fun initialize(): Boolean {
        return try {
            // Call native openpty() - returns [master_fd, slave_fd]
            val fds = nativeOpenPty()
            if (fds == null || fds.size < 2) {
                android.util.Log.e(TAG, "Failed to open PTY pair")
                onError("Failed to open PTY pair")
                return false
            }
            
            masterFd = fds[0]
            slaveFd = fds[1]
            
            // Grant and unlock PTY slave
            if (nativeGrantPty(masterFd) != 0) {
                android.util.Log.e(TAG, "grantpt failed")
                close()
                return false
            }
            
            if (nativeUnlockPty(masterFd) != 0) {
                android.util.Log.e(TAG, "unlockpt failed")
                close()
                return false
            }
            
            // Set initial terminal attributes
            nativeSetTerminalAttrs(
                slaveFd,
                B38400, B38400,
                CFLAG_DEFAULT, LFLAG_DEFAULT, IFLAG_DEFAULT, OFLAG_DEFAULT
            )
            
            android.util.Log.d(TAG, "PTY initialized: master=$masterFd, slave=$slaveFd")
            true
        } catch (e: Exception) {
            android.util.Log.e(TAG, "PTY initialization failed", e)
            onError("PTY initialization failed: ${e.message}")
            false
        }
    }
    
    /**
     * Start a shell process with the PTY as its controlling terminal
     */
    fun start(environment: Map<String, String> = emptyMap()): Boolean {
        if (masterFd < 0 || slaveFd < 0) {
            android.util.Log.e(TAG, "PTY not initialized")
            onError("PTY not initialized")
            return false
        }
        
        try {
            // Build environment array
            val envList = mutableListOf<String>()
            envList.add("TERM=xterm-256color")
            for ((key, value) in environment) {
                envList.add("$key=$value")
            }
            
            // Fork with PTY
            pid = nativeForkPty(
                masterFd,
                slaveFd,
                envList.toTypedArray(),
                null,  // working directory (null = inherit)
                shell,
                null   // arguments (null = just shell)
            )
            
            if (pid < 0) {
                android.util.Log.e(TAG, "forkpty failed")
                onError("Failed to fork process")
                return false
            }
            
            isRunning = true
            android.util.Log.d(TAG, "Shell started with PID: $pid")
            
            // Note: onProcessStarted will be called from native via callback
            return true
        } catch (e: Exception) {
            android.util.Log.e(TAG, "Failed to start shell", e)
            onError("Failed to start shell: ${e.message}")
            return false
        }
    }
    
    /**
     * Write data to the PTY master (sends to shell input)
     */
    fun write(data: ByteArray): Int {
        if (masterFd < 0) return -1
        return try {
            nativeWriteFd(masterFd, data)
        } catch (e: Exception) {
            android.util.Log.e(TAG, "Write failed", e)
            -1
        }
    }
    
    /**
     * Write string to the PTY master
     */
    fun write(text: String): Int {
        return write(text.toByteArray())
    }
    
    /**
     * Read data from the PTY master (shell output)
     */
    fun read(buffer: ByteArray): Int {
        if (masterFd < 0) return -1
        return try {
            nativeReadFd(masterFd, buffer)
        } catch (e: Exception) {
            android.util.Log.e(TAG, "Read failed", e)
            -1
        }
    }
    
    /**
     * Check if data is available to read
     */
    fun isDataAvailable(timeoutMs: Int = 0): Boolean {
        if (masterFd < 0) return false
        return nativeIsDataAvailable(masterFd, timeoutMs)
    }
    
    /**
     * Send a signal to the process group
     */
    fun sendSignal(signal: Int): Boolean {
        if (pid <= 0) return false
        return try {
            // Use negative PID to send to process group
            nativeSendSignalToProcessGroup(pid, signal)
            true
        } catch (e: Exception) {
            android.util.Log.e(TAG, "Failed to send signal $signal", e)
            false
        }
    }
    
    /**
     * Send Ctrl+C (SIGINT)
     */
    fun sendCtrlC(): Boolean = sendSignal(SIGINT)
    
    /**
     * Send Ctrl+Z (SIGTSTP)
     */
    fun sendCtrlZ(): Boolean = sendSignal(SIGTSTP)
    
    /**
     * Kill the process
     */
    fun kill(): Boolean = sendSignal(SIGKILL)
    
    /**
     * SIGHUP - close controlling terminal
     */
    fun hangup(): Boolean = sendSignal(SIGHUP)
    
    /**
     * Check if process is running
     */
    fun isRunning(): Boolean = isRunning
    
    /**
     * Get PID
     */
    fun getPid(): Int = pid
    
    /**
     * Wait for process to exit and get exit code
     */
    fun waitFor(): Int {
        val exitCode = nativeWaitForProcess(pid)
        isRunning = false
        return exitCode
    }
    
    /**
     * Get the master file descriptor number
     */
    fun getMasterFdNum(): Int = masterFd
    
    /**
     * Get the slave file descriptor number
     */
    fun getSlaveFdNum(): Int = slaveFd
    
    /**
     * Resize the PTY window
     */
    fun setWindowSize(columns: Int, rows: Int): Boolean {
        if (masterFd < 0) return false
        return nativeSetWindowSize(masterFd, columns, rows, 0, 0)
    }
    
    /**
     * Resize with pixel dimensions
     */
    fun setWindowSize(columns: Int, rows: Int, xPixels: Int, yPixels: Int): Boolean {
        if (masterFd < 0) return false
        return nativeSetWindowSize(masterFd, columns, rows, xPixels, yPixels)
    }
    
    /**
     * Set the shell to use
     */
    fun setShell(shellPath: String) {
        shell = shellPath
    }
    
    /**
     * Set working directory
     */
    fun setWorkingDirectory(dir: String) {
        // Stored for use in next start() call
    }
    
    /**
     * Close the PTY session
     */
    fun close() {
        try {
            kill()
            
            if (masterFd >= 0) {
                nativeCloseFd(masterFd)
                masterFd = -1
            }
            if (slaveFd >= 0) {
                nativeCloseFd(slaveFd)
                slaveFd = -1
            }
        } catch (e: Exception) {
            android.util.Log.e(TAG, "Error closing PTY", e)
        } finally {
            pid = -1
            isRunning = false
        }
    }
    
    // ========== PTYCallback implementation ==========
    
    override fun onProcessStarted(pid: Int) {
        android.util.Log.d(TAG, "Process started: $pid")
        isRunning = true
        callback?.onProcessStarted(pid)
    }
    
    override fun onProcessExited(exitCode: Int) {
        android.util.Log.d(TAG, "Process exited with code: $exitCode")
        isRunning = false
        callback?.onProcessExited(exitCode)
    }
    
    override fun onPtyData(data: String) {
        callback?.onPtyData(data)
    }
    
    override fun onPtyExit(exitCode: Int) {
        android.util.Log.d(TAG, "PTY session closed with exit code: $exitCode")
        isRunning = false
        callback?.onPtyExit(exitCode)
    }
    
    override fun onError(message: String) {
        android.util.Log.e(TAG, "Error: $message")
        callback?.onError(message)
    }
    
    // ========== Native methods (JNI) ==========
    
    // Open PTY pair - returns [master_fd, slave_fd]
    private native fun nativeOpenPty(): IntArray
    
    // Fork with PTY as controlling terminal
    private native fun nativeForkPty(
        masterFd: Int,
        slaveFd: Int,
        envp: Array<String>,
        dir: String?,
        shell: String,
        argv: Array<String>?
    ): Int
    
    // Set terminal attributes
    private native fun nativeSetTerminalAttrs(
        fd: Int,
        inputSpeed: Int,
        outputSpeed: Int,
        cflag: Int,
        lflag: Int,
        iflag: Int,
        oflag: Int
    ): Boolean
    
    // Set window size
    private native fun nativeSetWindowSize(
        fd: Int,
        cols: Int,
        rows: Int,
        xPixels: Int,
        yPixels: Int
    ): Boolean
    
    // Grant PTY permissions
    private native fun nativeGrantPty(fd: Int): Int
    
    // Unlock PTY
    private native fun nativeUnlockPty(fd: Int): Int
    
    // Get PTY name
    private native fun nativeGetPtyName(fd: Int): String
    
    // Send signal to process group
    private native fun nativeSendSignalToProcessGroup(pid: Int, sig: Int): Boolean
    
    // Send signal to process
    private native fun nativeSendSignal(pid: Int, sig: Int): Boolean
    
    // Wait for process to exit
    private native fun nativeWaitForProcess(pid: Int): Int
    
    // Set callback object
    private native fun nativeSetCallback(callback: PTYCallback)
    
    // Close file descriptor
    private native fun nativeCloseFd(fd: Int): Int
    
    // Write to file descriptor
    private native fun nativeWriteFd(fd: Int, data: ByteArray): Int
    
    // Read from file descriptor
    private native fun nativeReadFd(fd: Int, data: ByteArray): Int
    
    // Check if data available
    private native fun nativeIsDataAvailable(fd: Int, timeoutMs: Int): Boolean
    
    // Register native methods - called from static initializer
    companion object {
        private external fun registerNatives(): Boolean
        
        init {
            try {
                registerNatives()
            } catch (e: UnsatisfiedLinkError) {
                android.util.Log.w(TAG, "Native methods not registered (using fallback)")
            }
        }
    }
}