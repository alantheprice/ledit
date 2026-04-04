package com.ledit.android.service

/**
 * AIDL interface for LeditAgentService IPC.
 * Defines the contract for communication between Activities and the Service.
 */
interface ILeditAgentService {
    /**
     * Check if the agent service is running
     */
    boolean isRunning();
    
    /**
     * Get the current task description
     */
    String getCurrentTask();
    
    /**
     * Submit a new task to be executed
     * @param taskDescription Description of the task to run
     */
    void submitTask(String taskDescription);
    
    /**
     * Get the last task result
     */
    String getLastResult();
}