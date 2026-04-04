package com.ledit.android.ui

import android.annotation.SuppressLint
import android.content.ComponentName
import android.content.Context
import android.content.Intent
import android.content.ServiceConnection
import android.graphics.Bitmap
import android.os.Build
import android.os.Bundle
import android.os.IBinder
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.webkit.SslErrorHandler
import android.webkit.WebChromeClient
import android.webkit.WebResourceError
import android.webkit.WebResourceRequest
import android.webkit.WebSettings
import android.webkit.WebView
import android.webkit.WebViewClient
import android.widget.Button
import android.widget.LinearLayout
import android.widget.ProgressBar
import androidx.fragment.app.Fragment
import androidx.preference.PreferenceManager
import com.ledit.android.R
import com.ledit.android.bridge.JsCallback
import com.ledit.android.bridge.JsInterface
import com.ledit.android.service.LedisService

/**
 * WebUIFragment - Hosts the ledit WebUI in a WebView.
 * Loads the WebUI from the configured URL (default: http://localhost:54000)
 * and provides bidirectional communication with the Android native layer.
 */
class WebUIFragment : Fragment(), JsCallback {

    private var webView: WebView? = null
    private var progressBar: ProgressBar? = null
    private var errorView: LinearLayout? = null
    private var retryButton: Button? = null

    private var ledisService: LedisService? = null
    private var serviceBound = false

    companion object {
        private const val DEFAULT_WEBUI_URL = "http://localhost:54000"
        private const val SERVER_PORT = 54000
        private const val SERVER_HOST = "127.0.0.1"
    }

    private val serviceConnection = object : ServiceConnection {
        override fun onServiceConnected(name: ComponentName?, service: IBinder?) {
            // Service connected - server should be running
            serviceBound = true
        }

        override fun onServiceDisconnected(name: ComponentName?) {
            ledisService = null
            serviceBound = false
        }
    }

    @SuppressLint("SetJavaScriptEnabled")
    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? {
        return inflater.inflate(R.layout.fragment_webui, container, false)
    }

    @SuppressLint("SetJavaScriptEnabled")
    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)

        webView = view.findViewById(R.id.webView)
        progressBar = view.findViewById(R.id.progressBar)
        errorView = view.findViewById(R.id.errorView)
        retryButton = view.findViewById(R.id.retryButton)

        retryButton?.setOnClickListener {
            errorView?.visibility = View.GONE
            loadWebUI()
        }

        configureWebView()

        // Load WebUI URL from preferences
        loadWebUI()
    }

    @SuppressLint("SetJavaScriptEnabled")
    private fun configureWebView() {
        webView?.apply {
            settings.javaScriptEnabled = true
            settings.domStorageEnabled = true
            settings.allowFileAccess = true
            settings.allowContentAccess = true
            settings.loadWithOverviewMode = true
            settings.useWideViewPort = true
            settings.builtInZoomControls = true
            settings.displayZoomControls = false
            settings.databaseEnabled = true
            settings.setCacheMode(WebSettings.LOAD_DEFAULT)
            
            // Security: Don't allow file URL access to JS interface
            settings.allowFileAccessFromFileURLs = false
            settings.allowUniversalAccessFromFileURLs = false

            // Mixed content mode for Android 5.0+ (set to NEVER_ALLOW for security)
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP) {
                settings.mixedContentMode = WebSettings.MIXED_CONTENT_NEVER_ALLOW
            }

            // Enable hardware acceleration
            setLayerType(View.LAYER_TYPE_HARDWARE, null)

            // Add JavaScript interface
            addJavascriptInterface(JsInterface(requireActivity(), this@WebUIFragment), JsInterface.INTERFACE_NAME)

            webViewClient = object : WebViewClient() {
                override fun onPageStarted(view: WebView?, url: String?, favicon: Bitmap?) {
                    super.onPageStarted(view, url, favicon)
                    progressBar?.visibility = View.VISIBLE
                    errorView?.visibility = View.GONE
                }

                override fun onPageFinished(view: WebView?, url: String?) {
                    super.onPageFinished(view, url)
                    progressBar?.visibility = View.GONE
                }

                override fun onReceivedError(
                    view: WebView?,
                    request: WebResourceRequest?,
                    error: WebResourceError?
                ) {
                    super.onReceivedError(view, request, error)
                    // Only show error for main frame
                    if (request?.isForMainFrame == true) {
                        progressBar?.visibility = View.GONE
                        showError()
                    }
                }

                override fun onReceivedSslError(view: WebView?, handler: SslErrorHandler?, error: android.webkit.SslError?) {
                    // Only allow SSL for localhost development, cancel for all others
                    val url = error?.url ?: ""
                    if (url.startsWith("https://localhost") || url.startsWith("https://127.0.0.1")) {
                        handler?.proceed()
                    } else {
                        handler?.cancel()
                    }
                }

                override fun shouldOverrideUrlLoading(view: WebView?, request: WebResourceRequest?): Boolean {
                    val url = request?.url?.toString() ?: return false
                    // Handle navigation within WebUI
                    if (url.startsWith("http://localhost:$SERVER_PORT") || url.startsWith("http://127.0.0.1:$SERVER_PORT")) {
                        return false // Let WebView handle
                    }
                    // Open external links in browser
                    return false // For now, don't open external links
                }
            }

            webChromeClient = object : WebChromeClient() {
                override fun onProgressChanged(view: WebView?, newProgress: Int) {
                    if (newProgress < 100) {
                        progressBar?.visibility = View.VISIBLE
                    } else {
                        progressBar?.visibility = View.GONE
                    }
                }

                override fun onReceivedTitle(view: WebView?, title: String?) {
                    // Could update activity title if needed
                }
            }
        }
    }

    private fun showError() {
        errorView?.visibility = View.VISIBLE
    }

    private fun loadWebUI() {
        val prefs = PreferenceManager.getDefaultSharedPreferences(requireContext())
        val webUIUrl = prefs.getString("webui_url", DEFAULT_WEBUI_URL) ?: DEFAULT_WEBUI_URL

        webView?.loadUrl(webUIUrl)
    }

    override fun onStart() {
        super.onStart()
        // Start LedisService to serve WebUI
        startLedisService()
    }

    override fun onStop() {
        super.onStop()
        // Stop LedisService when backgrounded
        stopLedisService()
    }

    private fun startLedisService() {
        val intent = Intent(requireContext(), LedisService::class.java)
        requireContext().startService(intent)
        // Bind to service for lifecycle management
        requireContext().bindService(intent, serviceConnection, Context.BIND_AUTO_CREATE)
    }

    private fun stopLedisService() {
        if (serviceBound) {
            requireContext().unbindService(serviceConnection)
            serviceBound = false
        }
        // Stop the service to prevent it from running in background
        requireContext().stopService(Intent(requireContext(), LedisService::class.java))
    }

    override fun onResume() {
        super.onResume()
        webView?.onResume()
    }

    override fun onPause() {
        super.onPause()
        webView?.onPause()
    }

    override fun onDestroyView() {
        super.onDestroyView()
        // Clean up WebView
        webView?.apply {
            clearCache(true)
            clearHistory()
            clearFormData()
            removeJavascriptInterface(JsInterface.INTERFACE_NAME)
            destroy()
        }
        webView = null
        progressBar = null
        errorView = null
        retryButton = null
    }

    // JsCallback implementation
    override fun onFileSaved(filename: String) {
        // Could show toast or log
    }

    override fun onFileOpened(content: String, filename: String) {
        // Evaluate JavaScript callback in WebView
        webView?.let { wv ->
            val escapedContent = com.ledit.android.bridge.JsEvaluator.escapeJs(content)
            wv.post {
                wv.evaluateJavascript("if(window.leditAndroid && window.leditAndroid.onFileLoaded) { window.leditAndroid.onFileLoaded('$escapedContent', '$filename'); }", null)
            }
        }
    }

    override fun onError(message: String) {
        // Evaluate JavaScript error callback in WebView
        webView?.let { wv ->
            val escapedMessage = com.ledit.android.bridge.JsEvaluator.escapeJs(message)
            wv.post {
                wv.evaluateJavascript("if(window.leditAndroid && window.leditAndroid.onError) { window.leditAndroid.onError('$escapedMessage'); }", null)
            }
        }
    }

    /**
     * Get the WebView for testing purposes.
     */
    fun getWebView(): WebView? = webView
}