package com.ledit.android.service

import android.os.Binder

/**
 * Binder for LeditAgentService IPC.
 * Allows Activities to communicate with the foreground service.
 */
class LeditAgentBinder(val service: LeditAgentService) : Binder() {
    
    /**
     * Get the service instance
     */
    fun getService(): LeditAgentService = service
    
    /**
     * Check if agent is running
     */
    fun isRunning(): Boolean = service.isAgentRunning()
    
    /**
     * Get current task description
     */
    fun getCurrentTask(): String = service.getCurrentTask()
    
    /**
     * Submit a new task
     */
    fun submitTask(task: String) {
        service.submitTask(task)
    }
}