/*
 * term_exec.c - Native PTY support for ledit Android terminal emulator
 * 
 * Provides:
 * - openpty() - Create PTY master/slave pair
 * - forkpty() - Fork with PTY as controlling terminal
 * - Signal handling for process groups
 * - Terminal attribute configuration
 * 
 * This library bridges Android's Bionic libc with Java/Kotlin
 * via JNI to provide proper PTY support for shell sessions.
 */

#define LOG_TAG "TermExec"
#include <android/log.h>
#include <jni.h>

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <fcntl.h>
#include <errno.h>
#include <termios.h>
#include <signal.h>
#include <poll.h>
#include <sys/ioctl.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <pty.h>
#include <util.h>

#define LOGI(...) __android_log_print(ANDROID_LOG_INFO, LOG_TAG, __VA_ARGS__)
#define LOGE(...) __android_log_print(ANDROID_LOG_ERROR, LOG_TAG, __VA_ARGS__)
#define LOGD(...) __android_log_print(ANDROID_LOG_DEBUG, LOG_TAG, __VA_ARGS__)

/* Global references for callbacks */
static JavaVM *g_jvm = NULL;
static jobject g_callbackObj = NULL;

/*
 * Initialize the JNI environment
 */
static jint JNI_OnLoad(JavaVM *vm, void *reserved) {
    g_jvm = vm;
    LOGI("TermExec native library loaded");
    return JNI_VERSION_1_6;
}

/*
 * Store callback object for process events
 */
static void storeCallback(JNIEnv *env, jobject callbackObj) {
    if (g_callbackObj) {
        (*env)->DeleteGlobalRef(env, g_callbackObj);
    }
    g_callbackObj = (*env)->NewGlobalRef(env, callbackObj);
}

/*
 * Get JNI environment for current thread
 */
static JNIEnv *getJNIEnv() {
    if (!g_jvm) return NULL;
    
    JNIEnv *env;
    int status = (*g_jvm)->GetEnv(g_jvm, (void**)&env, JNI_VERSION_1_6);
    if (status == JNI_EDETACHED) {
        (*g_jvm)->AttachCurrentThread(g_jvm, &env, NULL);
    }
    return env;
}

/*
 * Notify callback of process started
 */
static void notifyProcessStarted(JNIEnv *env, int pid) {
    if (!g_callbackObj) return;
    
    jclass cls = (*env)->GetObjectClass(env, g_callbackObj);
    jmethodID method = (*env)->GetMethodID(env, cls, "onProcessStarted", "(I)V");
    if (method) {
        (*env)->CallVoidMethod(env, g_callbackObj, method, (jint)pid);
    }
}

/*
 * Notify callback of process exited
 */
static void notifyProcessExited(JNIEnv *env, int exitCode) {
    if (!g_callbackObj) return;
    
    jclass cls = (*env)->GetObjectClass(env, g_callbackObj);
    jmethodID method = (*env)->GetMethodID(env, cls, "onProcessExited", "(I)V");
    if (method) {
        (*env)->CallVoidMethod(env, g_callbackObj, method, (jint)exitCode);
    }
}

/*
 * Notify callback of error
 */
static void notifyError(JNIEnv *env, const char *message) {
    if (!g_callbackObj) return;
    
    jclass cls = (*env)->GetObjectClass(env, g_callbackObj);
    jmethodID method = (*env)->GetMethodID(env, cls, "onError", "(Ljava/lang/String;)V");
    if (method) {
        jstring msg = (*env)->NewStringUTF(env, message);
        (*env)->CallVoidMethod(env, g_callbackObj, method, msg);
        (*env)->DeleteLocalRef(env, msg);
    }
}

/*
 * JNI: Open PTY master/slave pair
 * 
 * Returns array of [master_fd, slave_fd] or null on failure
 */
static jintArray Java_com_ledit_android_pty_PTYSession_openPty(JNIEnv *env, jobject thiz) {
    int master_fd, slave_fd;
    char slave_name[64];
    
    if (openpty(&master_fd, &slave_fd, slave_name, NULL, NULL) != 0) {
        LOGE("openpty failed: %s", strerror(errno));
        return NULL;
    }
    
    LOGD("Opened PTY: master=%d, slave=%d (%s)", master_fd, slave_fd, slave_name);
    
    jintArray result = (*env)->NewIntArray(env, 2);
    jint arr[2] = {master_fd, slave_fd};
    (*env)->SetIntArrayRegion(env, result, 0, 2, arr);
    
    return result;
}

/*
 * JNI: Fork process with PTY as controlling terminal
 * 
 * args:
 *   master_fd: PTY master file descriptor
 *   slave_fd: PTY slave file descriptor  
 *   envp: environment variables array
 *   dir: working directory
 *   shell: shell to execute
 *   argv: command arguments
 * 
 * Returns child PID or -1 on failure
 */
static jint Java_com_ledit_android_pty_PTYSession_forkPty(
        JNIEnv *env, jobject thiz,
        jint master_fd, jint slave_fd,
        jobjectArray envp, jstring dir, jstring shell, jobjectArray argv) {
    
    pid_t pid = forkpty(&master_fd, NULL, NULL, NULL);
    
    if (pid < 0) {
        LOGE("forkpty failed: %s", strerror(errno));
        return -1;
    }
    
    if (pid == 0) {
        /* Child process - execute shell */
        
        /* Get shell path */
        const char *shell_str = (*env)->GetStringUTFChars(env, shell, NULL);
        
        /* Get working directory */
        const char *dir_str = NULL;
        if (dir) {
            dir_str = (*env)->GetStringUTFChars(env, dir, NULL);
            if (dir_str && chdir(dir_str) != 0) {
                LOGE("chdir failed: %s", strerror(errno));
            }
        }
        
        /* Set environment */
        if (envp) {
            jsize envc = (*env)->GetArrayLength(env, envp);
            for (jsize i = 0; i < envc; i++) {
                jstring e = (jstring)(*env)->GetObjectArrayElement(env, envp, i);
                const char *ev = (*env)->GetStringUTFChars(env, e, NULL);
                putenv((char*)ev);
                (*env)->ReleaseStringUTFChars(env, e, ev);
            }
        }
        
        /* Build argument vector */
        jsize argc = 0;
        if (argv) {
            argc = (*env)->GetArrayLength(env, argv);
        }
        
        char **args = NULL;
        if (argc > 0) {
            args = malloc((argc + 1) * sizeof(char*));
            for (jsize i = 0; i < argc; i++) {
                jstring a = (jstring)(*env)->GetObjectArrayElement(env, argv, i);
                args[i] = (char*)(*env)->GetStringUTFChars(env, a, NULL);
            }
            args[argc] = NULL;
        }
        
        /* Execute shell */
        execv(shell_str, args ? args : (char*[]){(char*)shell_str, NULL});
        
        /* If we get here, exec failed */
        LOGE("exec failed: %s", strerror(errno));
        _exit(127);
    }
    
    /* Parent process */
    LOGD("Forked process with PID: %d", pid);
    
    /* Notify callback */
    notifyProcessStarted(env, pid);
    
    return (jint)pid;
}

/*
 * JNI: Set terminal attributes
 */
static jboolean Java_com_ledit_android_pty_PTYSession_setTerminalAttrs(
        JNIEnv *env, jobject thiz,
        jint fd, jint input_speed, jint output_speed,
        jint cflag, jint lflag, jint iflag, jint oflag) {
    
    struct termios tios;
    
    if (tcgetattr(fd, &tios) != 0) {
        LOGE("tcgetattr failed: %s", strerror(errno));
        return JNI_FALSE;
    }
    
    /* Set input/output speeds */
    tios.c_ispeed = (speed_t)input_speed;
    tios.c_ospeed = (speed_t)output_speed;
    
    /* Set control modes */
    tios.c_cflag = (tcflag_t)cflag;
    
    /* Set local modes */
    tios.c_lflag = (tcflag_t)lflag;
    
    /* Set input modes */
    tios.c_iflag = (tcflag_t)iflag;
    
    /* Set output modes */
    tios.c_oflag = (tcflag_t)oflag;
    
    if (tcsetattr(fd, TCSANOW, &tios) != 0) {
        LOGE("tcsetattr failed: %s", strerror(errno));
        return JNI_FALSE;
    }
    
    return JNI_TRUE;
}

/*
 * JNI: Set window size
 */
static jboolean Java_com_ledit_android_pty_PTYSession_setWindowSize(
        JNIEnv *env, jobject thiz,
        jint fd, jint cols, jint rows, jint xpix, jint ypix) {
    
    struct winsize ws;
    ws.ws_col = (unsigned short)cols;
    ws.ws_row = (unsigned short)rows;
    ws.ws_xpixel = (unsigned short)xpix;
    ws.ws_ypixel = (unsigned short)ypix;
    
    if (ioctl(fd, TIOCSWINSZ, &ws) != 0) {
        LOGE("TIOCSWINSZ failed: %s", strerror(errno));
        return JNI_FALSE;
    }
    
    return JNI_TRUE;
}

/*
 * JNI: Grant PTY slave permissions
 */
static jint Java_com_ledit_android_pty_PTYSession_grantPty(JNIEnv *env, jobject thiz, jint fd) {
    return grantpt(fd);
}

/*
 * JNI: Unlock PTY slave
 */
static jint Java_com_ledit_android_pty_PTYSession_unlockPty(JNIEnv *env, jobject thiz, jint fd) {
    return unlockpt(fd);
}

/*
 * JNI: Get PTY slave name
 */
static jstring Java_com_ledit_android_pty_PTYSession_getPtyName(JNIEnv *env, jobject thiz, jint fd) {
    char name[64];
    if (ptsname_r(fd, name, sizeof(name)) != 0) {
        LOGE("ptsname_r failed: %s", strerror(errno));
        return NULL;
    }
    return (*env)->NewStringUTF(env, name);
}

/*
 * JNI: Send signal to process group
 * 
 * Uses kill(-pid, sig) to send signal to entire process group
 */
static jboolean Java_com_ledit_android_pty_PTYSession_sendSignalToProcessGroup(
        JNIEnv *env, jobject thiz, jint pid, jint sig) {
    
    /* Negative PID means send to process group */
    if (kill(-(pid_t)pid, sig) != 0) {
        LOGE("kill(-%d, %d) failed: %s", pid, sig, strerror(errno));
        return JNI_FALSE;
    }
    
    LOGD("Sent signal %d to process group %d", sig, pid);
    return JNI_TRUE;
}

/*
 * JNI: Send signal to process
 */
static jboolean Java_com_ledit_android_pty_PTYSession_sendSignal(
        JNIEnv *env, jobject thiz, jint pid, jint sig) {
    
    if (kill((pid_t)pid, sig) != 0) {
        LOGE("kill(%d, %d) failed: %s", pid, sig, strerror(errno));
        return JNI_FALSE;
    }
    
    LOGD("Sent signal %d to process %d", sig, pid);
    return JNI_TRUE;
}

/*
 * JNI: Wait for process to exit
 * 
 * Returns exit code or -1 on error
 */
static jint Java_com_ledit_android_pty_PTYSession_waitForProcess(
        JNIEnv *env, jobject thiz, jint pid) {
    
    int status;
    pid_t result = waitpid((pid_t)pid, &status, 0);
    
    if (result < 0) {
        LOGE("waitpid failed: %s", strerror(errno));
        return -1;
    }
    
    if (WIFEXITED(status)) {
        int exitCode = WEXITSTATUS(status);
        LOGD("Process %d exited with code %d", pid, exitCode);
        notifyProcessExited(env, exitCode);
        return exitCode;
    }
    
    if (WIFSIGNALED(status)) {
        int sig = WTERMSIG(status);
        LOGD("Process %d killed by signal %d", pid, sig);
        notifyProcessExited(env, 128 + sig);
        return 128 + sig;
    }
    
    return -1;
}

/*
 * JNI: Set callback for process events
 */
static void Java_com_ledit_android_pty_PTYSession_setCallback(
        JNIEnv *env, jobject thiz, jobject callbackObj) {
    storeCallback(env, callbackObj);
}

/*
 * JNI: Close file descriptor
 */
static jint Java_com_ledit_android_pty_PTYSession_closeFd(JNIEnv *env, jobject thiz, jint fd) {
    return close(fd);
}

/*
 * JNI: Write to file descriptor
 */
static jint Java_com_ledit_android_pty_PTYSession_writeFd(
        JNIEnv *env, jobject thiz, jint fd, jbyteArray data) {
    
    jbyte *buf = (*env)->GetByteArrayElements(env, data, NULL);
    jsize len = (*env)->GetArrayLength(env, data);
    
    ssize_t written = write(fd, buf, len);
    
    (*env)->ReleaseByteArrayElements(env, data, buf, 0);
    
    return (jint)written;
}

/*
 * JNI: Read from file descriptor
 */
static jint Java_com_ledit_android_pty_PTYSession_readFd(
        JNIEnv *env, jobject thiz, jint fd, jbyteArray data) {
    
    jbyte *buf = (*env)->GetByteArrayElements(env, data, NULL);
    jsize len = (*env)->GetArrayLength(env, data);
    
    ssize_t nread = read(fd, buf, len);
    
    (*env)->ReleaseByteArrayElements(env, data, buf, 0);
    
    return (jint)nread;
}

/*
 * JNI: Check if descriptor has data ready
 */
static jboolean Java_com_ledit_android_pty_PTYSession_isDataAvailable(
        JNIEnv *env, jobject thiz, jint fd, jint timeout_ms) {
    
    struct pollfd pfd;
    pfd.fd = fd;
    pfd.events = POLLIN;
    pfd.revents = 0;
    
    int ret = poll(&pfd, 1, timeout_ms);
    
    if (ret > 0 && (pfd.revents & POLLIN)) {
        return JNI_TRUE;
    }
    
    return JNI_FALSE;
}

/* JNI method table */
static JNINativeMethod g_methods[] = {
    {"nativeOpenPty", "()[I", (void*)Java_com_ledit_android_pty_PTYSession_openPty},
    {"nativeForkPty", "(I[I[Ljava/lang/String;Ljava/lang/String;Ljava/lang/String;[Ljava/lang/String;)I", (void*)Java_com_ledit_android_pty_PTYSession_forkPty},
    {"nativeSetTerminalAttrs", "(IIIII)Z", (void*)Java_com_ledit_android_pty_PTYSession_setTerminalAttrs},
    {"nativeSetWindowSize", "(IIII)Z", (void*)Java_com_ledit_android_pty_PTYSession_setWindowSize},
    {"nativeGrantPty", "(I)I", (void*)Java_com_ledit_android_pty_PTYSession_grantPty},
    {"nativeUnlockPty", "(I)I", (void*)Java_com_ledit_android_pty_PTYSession_unlockPty},
    {"nativeGetPtyName", "(I)Ljava/lang/String;", (void*)Java_com_ledit_android_pty_PTYSession_getPtyName},
    {"nativeSendSignalToProcessGroup", "(II)Z", (void*)Java_com_ledit_android_pty_PTYSession_sendSignalToProcessGroup},
    {"nativeSendSignal", "(II)Z", (void*)Java_com_ledit_android_pty_PTYSession_sendSignal},
    {"nativeWaitForProcess", "(I)I", (void*)Java_com_ledit_android_pty_PTYSession_waitForProcess},
    {"nativeSetCallback", "(Lcom/ledit/android/pty/PTYCallback;)V", (void*)Java_com_ledit_android_pty_PTYSession_setCallback},
    {"nativeCloseFd", "(I)I", (void*)Java_com_ledit_android_pty_PTYSession_closeFd},
    {"nativeWriteFd", "(I[B)I", (void*)Java_com_ledit_android_pty_PTYSession_writeFd},
    {"nativeReadFd", "(I[B)I", (void*)Java_com_ledit_android_pty_PTYSession_readFd},
    {"nativeIsDataAvailable", "(II)Z", (void*)Java_com_ledit_android_pty_PTYSession_isDataAvailable},
};

/*
 * Register native methods for PTYSession
 */
__attribute__((visibility("default")))
jint Java_com_ledit_android_pty_PTYSession_registerNatives(JNIEnv *env, jclass cls) {
    return (*env)->RegisterNatives(env, cls, g_methods, sizeof(g_methods) / sizeof(g_methods[0]));
}