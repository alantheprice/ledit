package com.ledit.android.service

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Context
import android.content.Intent
import android.os.Build
import android.os.IBinder
import android.os.PowerManager
import android.util.Log
import androidx.core.app.NotificationCompat
import com.ledit.android.R
import com.ledit.android.ui.MainActivity

/**
 * LeditAgentService - Foreground service that keeps the ledit agent running in background.
 * 
 * This service:
 * - Runs as a foreground service with persistent notification
 * - Survives process death via START_STICKY
 * - Manages task execution
 * - Provides IPC for UI communication
 */
class LeditAgentService : Service() {

    companion object {
        private const val TAG = "LeditAgentService"
        
        // Notification channel and ID
        const val CHANNEL_ID = "ledit_agent_channel"
        const val NOTIFICATION_ID = 1
        
        // Actions
        const val ACTION_START = "com.ledit.android.action.START_AGENT"
        const val ACTION_STOP = "com.ledit.android.action.STOP_AGENT"
        const val ACTION_EXECUTE_TASK = "com.ledit.android.action.EXECUTE_TASK"
        
        // Extras
        const val EXTRA_TASK_DESCRIPTION = "task_description"
        
        // Wake lock tag
        const val WAKELOCK_TAG = "ledit:agent_wakelock"
    }

    private var wakeLock: PowerManager.WakeLock? = null
    private var notificationManager: NotificationManager? = null
    private var currentTask: String = "Idle"
    private var isRunning: Boolean = false
    private var lastResult: String = ""
    
    private val binder = LeditAgentBinder(this)

    override fun onCreate() {
        super.onCreate()
        Log.d(TAG, "Service created")
        
        notificationManager = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        createNotificationChannel()
        acquireWakeLock()
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        Log.d(TAG, "Service started with intent: ${intent?.action}")
        
        when (intent?.action) {
            ACTION_STOP -> {
                stopSelf()
                return START_NOT_STICKY
            }
            ACTION_EXECUTE_TASK -> {
                val taskDesc = intent.getStringExtra(EXTRA_TASK_DESCRIPTION) ?: "Running task"
                executeTask(taskDesc)
            }
        }
        
        // Start as foreground service
        startForeground(NOTIFICATION_ID, createNotification())
        isRunning = true
        
        // START_STICKY ensures service restarts if killed
        return START_STICKY
    }

    override fun onBind(intent: Intent?): IBinder? {
        return binder
    }

    override fun onDestroy() {
        Log.d(TAG, "Service destroyed")
        releaseWakeLock()
        isRunning = false
        super.onDestroy()
    }

    /**
     * Create notification channel for Android 8+
     */
    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                CHANNEL_ID,
                getString(R.string.notification_channel_name),
                NotificationManager.IMPORTANCE_LOW
            ).apply {
                description = getString(R.string.notification_channel_description)
                setShowBadge(false)
            }
            notificationManager?.createNotificationChannel(channel)
        }
    }

    /**
     * Create the foreground notification
     */
    private fun createNotification(): android.app.Notification {
        // Intent to open app when notification clicked
        val openIntent = Intent(this, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
        }
        val openPendingIntent = PendingIntent.getActivity(
            this, 0, openIntent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        // Stop action
        val stopIntent = Intent(this, LeditAgentService::class.java).apply {
            action = ACTION_STOP
        }
        val stopPendingIntent = PendingIntent.getService(
            this, 1, stopIntent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        return NotificationCompat.Builder(this, CHANNEL_ID)
            .setContentTitle(getString(R.string.notification_title))
            .setContentText(currentTask)
            .setSmallIcon(R.drawable.ic_launcher)
            .setContentIntent(openPendingIntent)
            .addAction(0, getString(R.string.notification_action_stop), stopPendingIntent)
            .setOngoing(true)
            .setPriority(NotificationCompat.PRIORITY_LOW)
            .setCategory(NotificationCompat.CATEGORY_SERVICE)
            .build()
    }

    /**
     * Update the notification with current task status
     */
    private fun updateNotification(taskDescription: String) {
        currentTask = taskDescription
        val notification = createNotification()
        notificationManager?.notify(NOTIFICATION_ID, notification)
    }

    /**
     * Execute a task (placeholder for agent integration)
     */
    private fun executeTask(taskDescription: String) {
        Log.d(TAG, "Executing task: $taskDescription")
        updateNotification(taskDescription)
        
        // TODO: Integrate with Go agent via gomobile
        // This is where you'd call the Go agent to execute tasks
    }

    /**
     * Acquire wake lock to keep CPU alive during background tasks
     */
    private fun acquireWakeLock() {
        val powerManager = getSystemService(Context.POWER_SERVICE) as PowerManager
        wakeLock = powerManager.newWakeLock(
            PowerManager.PARTIAL_WAKE_LOCK,
            WAKELOCK_TAG
        ).apply {
            acquire(10 * 60 * 1000L) // 10 minutes max
        }
        Log.d(TAG, "Wake lock acquired")
    }

    /**
     * Release wake lock
     */
    private fun releaseWakeLock() {
        wakeLock?.let {
            if (it.isHeld) {
                it.release()
                Log.d(TAG, "Wake lock released")
            }
        }
        wakeLock = null
    }

    /**
     * Check if service is running
     */
    fun isAgentRunning(): Boolean = isRunning

    /**
     * Get current task status
     */
    fun getCurrentTask(): String = currentTask
    
    /**
     * Submit a task to be executed
     */
    fun submitTask(task: String) {
        currentTask = task
        Log.d(TAG, "Task submitted: $task")
        updateNotification("Executing: $task")
        // TODO: Execute actual task via Go agent
    }
    
    /**
     * Get the last task result
     */
    fun getLastResult(): String = lastResult

    /**
     * Start the agent service from an Activity
     */
    companion object {
        fun start(context: Context) {
            val intent = Intent(context, LeditAgentService::class.java).apply {
                action = ACTION_START
            }
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                context.startForegroundService(intent)
            } else {
                context.startService(intent)
            }
        }

        /**
         * Schedule a task to be executed by the agent service
         */
        fun executeTask(context: Context, taskDescription: String) {
            val intent = Intent(context, LeditAgentService::class.java).apply {
                action = ACTION_EXECUTE_TASK
                putExtra(EXTRA_TASK_DESCRIPTION, taskDescription)
            }
            context.startService(intent)
        }

        /**
         * Stop the agent service
         */
        fun stop(context: Context) {
            val intent = Intent(context, LeditAgentService::class.java).apply {
                action = ACTION_STOP
            }
            context.startService(intent)
        }
    }
}