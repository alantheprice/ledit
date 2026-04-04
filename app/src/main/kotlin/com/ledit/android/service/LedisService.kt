package com.ledit.android.service

import android.app.Service
import android.content.Context
import android.content.Intent
import android.os.IBinder
import android.util.Log
import fi.iki.elonen.NanoHTTPD
import java.io.IOException

/**
 * LedisService - Background service that runs an embedded HTTP server
 * to serve the ledit WebUI to the WebView.
 * 
 * The server binds to localhost only (127.0.0.1:54000) for security.
 */
class LedisService : Service() {

    companion object {
        private const val TAG = "LedisService"
        const val SERVER_PORT = 54000
        const val SERVER_HOST = "127.0.0.1"
    }

    private var server: LedisHttpServer? = null

    override fun onBind(intent: Intent?): IBinder? {
        return null
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        startServer()
        // START_STICKY ensures the service restarts if killed
        return START_STICKY
    }

    private fun startServer() {
        if (server == null) {
            try {
                server = LedisHttpServer(applicationContext, SERVER_PORT)
                server?.start()
                Log.d(TAG, "Server started on http://$SERVER_HOST:$SERVER_PORT")
            } catch (e: IOException) {
                Log.e(TAG, "Failed to start server", e)
            }
        }
    }

    private fun stopServer() {
        server?.stop()
        server = null
        Log.d(TAG, "Server stopped")
    }

    override fun onDestroy() {
        stopServer()
        super.onDestroy()
    }

    /**
     * Check if the server is running and responding.
     */
    fun isServerRunning(): Boolean {
        return server?.isAlive == true
    }

    /**
     * Custom HTTP server implementation using NanoHttpd.
     * Serves the WebUI from app assets or provides basic responses.
     */
    private class LedisHttpServer(
        private val context: Context,
        port: Int
    ) : NanoHTTPD(SERVER_HOST, port) {

        companion object {
            private const val TAG = "LedisHttpServer"
        }

        override fun serve(session: IHTTPSession): Response {
            val uri = session.uri
            Log.d(TAG, "Request: $uri")

            return try {
                when {
                    uri == "/" || uri.isEmpty() -> {
                        serveIndexHtml()
                    }
                    uri.startsWith("/api/") -> {
                        handleApiRequest(session)
                    }
                    else -> {
                        serveStaticFile(uri)
                    }
                }
            } catch (e: Exception) {
                Log.e(TAG, "Error serving request: $uri", e)
                newFixedLengthResponse(
                    HTTP_INTERNALERROR,
                    MIME_HTML,
                    "<html><body><h1>500 Internal Server Error</h1><p>${e.message}</p></body></html>"
                )
            }
        }

        private fun serveIndexHtml(): Response {
            // Try to load from assets first
            return try {
                val inputStream = context.assets.open("webui/index.html")
                val content = inputStream.bufferedReader().use { it.readText() }
                newFixedLengthResponse(HTTP_OK, MIME_HTML, content)
            } catch (e: IOException) {
                // Fall through to default response
                Log.d(TAG, "No webui/index.html in assets, using default")
                serveDefaultPage()
            }
        }

        private fun serveDefaultPage(): Response {
            val html = """
                <!DOCTYPE html>
                <html>
                <head>
                    <meta charset="UTF-8">
                    <meta name="viewport" content="width=device-width, initial-scale=1.0">
                    <title>Ledit WebUI Server</title>
                    <style>
                        body {
                            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
                            max-width: 600px;
                            margin: 50px auto;
                            padding: 20px;
                            background: #f5f5f5;
                        }
                        h1 { color: #333; }
                        .info {
                            background: white;
                            padding: 20px;
                            border-radius: 8px;
                            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
                        }
                        .status { color: #4CAF50; font-weight: bold; }
                    </style>
                </head>
                <body>
                    <h1>Ledit WebUI Server</h1>
                    <div class="info">
                        <p>Status: <span class="status">Running</span></p>
                        <p>Server is running on localhost:${SERVER_PORT}</p>
                        <p>Configure your WebUI to connect to this server.</p>
                    </div>
                </body>
                </html>
            """.trimIndent()

            return newFixedLengthResponse(HTTP_OK, MIME_HTML, html)
        }

        private fun serveStaticFile(uri: String): Response {
            // Remove leading slash and map to assets
            val assetPath = uri.removePrefix("/")
            
            // Validate path to prevent path traversal attacks
            if (!isSafeAssetPath(assetPath)) {
                Log.w(TAG, "Blocked potentially unsafe path: $assetPath")
                return notFound()
            }

            return try {
                val inputStream = context.assets.open(assetPath)
                val mimeType = getMimeType(assetPath)
                val content = inputStream.bufferedReader().use { it.readText() }
                newFixedLengthResponse(HTTP_OK, mimeType, content)
            } catch (e: IOException) {
                notFound()
            }
        }
        
        /**
         * Validate asset path to prevent path traversal attacks.
         */
        private fun isSafeAssetPath(path: String): Boolean {
            // Decode URL-encoded characters first
            val decoded = try {
                java.net.URLDecoder.decode(path, "UTF-8")
            } catch (e: Exception) {
                path
            }
            
            // Check for path traversal patterns
            if (decoded.contains("..")) return false
            if (decoded.startsWith("/")) return false
            // Block Windows path patterns
            if (decoded.contains(":") || decoded.contains("\\")) return false
            // Check for null bytes
            if (decoded.contains("\u0000")) return false
            
            return true
        }

        private fun handleApiRequest(session: IHTTPSession): Response {
            val uri = session.uri.removePrefix("/api/")
            val method = session.method

            Log.d(TAG, "API: $method /api/$uri")

            return when (uri) {
                "health" -> {
                    val json = """{"status": "ok", "server": "ledit", "port": $SERVER_PORT}"""
                    newFixedLengthResponse(HTTP_OK, "application/json", json)
                }
                "info" -> {
                    val json = """
                        {
                            "name": "ledit-server",
                            "version": "1.0.0",
                            "port": $SERVER_PORT
                        }
                    """.trimIndent()
                    newFixedLengthResponse(HTTP_OK, "application/json", json)
                }
                else -> {
                    newFixedLengthResponse(HTTP_NOTFOUND, "application/json", """{"error": "Not found"}""")
                }
            }
        }

        private fun notFound(): Response {
            return newFixedLengthResponse(
                HTTP_NOTFOUND,
                MIME_HTML,
                "<html><body><h1>404 Not Found</h1></body></html>"
            )
        }

        private fun getMimeType(path: String): String {
            return when {
                path.endsWith(".html") || path.endsWith(".htm") -> MIME_HTML
                path.endsWith(".css") -> "text/css"
                path.endsWith(".js") -> "application/javascript"
                path.endsWith(".json") -> "application/json"
                path.endsWith(".png") -> "image/png"
                path.endsWith(".jpg") || path.endsWith(".jpeg") -> "image/jpeg"
                path.endsWith(".svg") -> "image/svg+xml"
                path.endsWith(".ico") -> "image/x-icon"
                path.endsWith(".txt") -> "text/plain"
                else -> MIME_PLAINTEXT
            }
        }
    }
}