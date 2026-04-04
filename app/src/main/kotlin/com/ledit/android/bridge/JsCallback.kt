package com.ledit.android.bridge

/**
 * Callback interface for JavaScript to Android communication.
 * Provides async callback methods for file operations and errors.
 */
interface JsCallback {
    /**
     * Called when a file has been saved successfully.
     * @param filename The name of the file that was saved
     */
    fun onFileSaved(filename: String)

    /**
     * Called when a file has been opened successfully.
     * @param content The content of the file
     * @param filename The name of the file that was opened
     */
    fun onFileOpened(content: String, filename: String)

    /**
     * Called when an error occurs during an operation.
     * @param message The error message
     */
    fun onError(message: String)
}