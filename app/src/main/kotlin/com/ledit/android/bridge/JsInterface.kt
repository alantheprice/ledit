package com.ledit.android.bridge

import android.app.Activity
import android.content.Context
import android.content.SharedPreferences
import android.webkit.JavascriptInterface
import android.webkit.WebView
import com.ledit.android.BuildConfig
import java.io.File

/**
 * JavaScript interface for WebView to communicate with Android native layer.
 * Exposed to JavaScript as `window.leditAndroid`.
 */
class JsInterface(
    private val activity: Activity,
    private val callback: JsCallback?
) {

    companion object {
        const val INTERFACE_NAME = "leditAndroid"
        private const val PREFS_NAME = "ledit_prefs"
    }

    private val prefs: SharedPreferences by lazy {
        activity.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
    }

    private val appFilesDir: File by lazy {
        activity.filesDir
    }

    /**
     * Save content to a file.
     * @param content The content to save
     * @param filename The name of the file to save
     */
    @JavascriptInterface
    fun saveFile(content: String, filename: String) {
        if (!isValidFilename(filename)) {
            callback?.onError("Invalid filename")
            return
        }
        try {
            val file = File(appFilesDir, filename)
            file.writeText(content)
            callback?.onFileSaved(filename)
        } catch (e: Exception) {
            callback?.onError("Failed to save file")
        }
    }

    /**
     * Open and read a file.
     * @param filename The name of the file to open
     */
    @JavascriptInterface
    fun openFile(filename: String) {
        if (!isValidFilename(filename)) {
            callback?.onError("Invalid filename")
            return
        }
        try {
            val file = File(appFilesDir, filename)
            if (file.exists()) {
                val content = file.readText()
                callback?.onFileOpened(content, filename)
            } else {
                callback?.onError("File not found: $filename")
            }
        } catch (e: Exception) {
            callback?.onError("Failed to open file")
        }
    }

    /**
     * List all files in the app's storage directory.
     * @return JSON array of filenames
     */
    @JavascriptInterface
    fun listFiles(): String {
        return try {
            val files = appFilesDir.listFiles()?.map { it.name } ?: emptyList()
            files.joinToString(",", "[", "]") { "\"$it\"" }
        } catch (e: Exception) {
            callback?.onError("Failed to list files")
            "[]"
        }
    }

    /**
     * Set a setting value.
     * @param key The setting key
     * @param value The setting value
     */
    @JavascriptInterface
    fun setSetting(key: String, value: String) {
        // Validate key - only allow safe characters
        if (!key.matches(Regex("^[a-zA-Z0-9_]+$"))) {
            callback?.onError("Invalid setting key")
            return
        }
        prefs.edit().putString(key, value).apply()
    }

    /**
     * Get a setting value.
     * @param key The setting key
     * @return The setting value or empty string if not found
     */
    @JavascriptInterface
    fun getSetting(key: String): String {
        return prefs.getString(key, "") ?: ""
    }

    /**
     * Close the application.
     */
    @JavascriptInterface
    fun closeApp() {
        activity.finish()
    }

    /**
     * Get the application version.
     * @return The app version string
     */
    @JavascriptInterface
    fun getAppVersion(): String {
        return try {
            activity.packageManager.getPackageInfo(activity.packageName, 0).versionName ?: "1.0"
        } catch (e: Exception) {
            BuildConfig.VERSION_NAME
        }
    }

    /**
     * Delete a file.
     * @param filename The name of the file to delete
     * @return true if deletion was successful
     */
    @JavascriptInterface
    fun deleteFile(filename: String): Boolean {
        if (!isValidFilename(filename)) {
            callback?.onError("Invalid filename")
            return false
        }
        return try {
            val file = File(appFilesDir, filename)
            val deleted = file.delete()
            if (!deleted) {
                callback?.onError("Failed to delete file: $filename")
            }
            deleted
        } catch (e: Exception) {
            callback?.onError("Failed to delete file")
            false
        }
    }

    /**
     * Check if a file exists.
     * @param filename The name of the file to check
     * @return true if the file exists
     */
    @JavascriptInterface
    fun fileExists(filename: String): Boolean {
        if (!isValidFilename(filename)) {
            return false
        }
        return File(appFilesDir, filename).exists()
    }

    /**
     * Get the files directory path.
     * @return The absolute path to the files directory
     */
    @JavascriptInterface
    fun getFilesDir(): String {
        return appFilesDir.absolutePath
    }

    /**
     * Validate filename to prevent path traversal attacks.
     * @param filename The filename to validate
     * @return true if valid, false otherwise
     */
    private fun isValidFilename(filename: String): Boolean {
        if (filename.isEmpty() || filename.length > 255) return false
        if (filename.contains("..") || filename.startsWith("/")) return false
        if (filename.contains("\\")) return false // Windows paths
        // Check for null bytes and other dangerous characters
        if (filename.contains("\u0000")) return false
        // Only allow alphanumeric, dash, underscore, dot
        return filename.matches(Regex("^[a-zA-Z0-9._-]+$"))
    }
}

/**
 * Helper object for evaluating JavaScript callbacks in WebView.
 */
object JsEvaluator {
        /**
         * Evaluate JavaScript in a WebView safely.
         * @param webView The WebView to evaluate in
         * @param script The JavaScript code to execute
         */
        fun evaluate(webView: WebView?, script: String) {
            webView?.post {
                webView.evaluateJavascript(script, null)
            }
        }

        /**
         * Escape a string for use in JavaScript.
         * @param input The input string
         * @return The escaped string
         */
        fun escapeJs(input: String): String {
            return input
                .replace("\\", "\\\\")
                .replace("'", "\\'")
                .replace("\n", "\\n")
                .replace("\r", "\\r")
                .replace("\t", "\\t")
        }
    }