package com.ledit.android.term

import android.content.Context
import android.graphics.Canvas
import android.graphics.Color
import android.graphics.Paint
import android.graphics.Typeface
import android.os.Handler
import android.os.Looper
import android.util.AttributeSet
import android.view.GestureDetector
import android.view.KeyEvent
import android.view.MotionEvent
import android.view.ScaleGestureDetector
import android.view.View
import android.view.inputmethod.BaseInputConnection
import android.view.inputmethod.EditorInfo
import android.view.inputmethod.InputMethodManager
import com.ledit.android.pty.PTYCallback
import com.ledit.android.pty.PTYSession
import java.util.concurrent.CopyOnWriteArrayList

/**
 * TerminalView - A custom View that renders terminal output and handles input.
 * 
 * Integrates:
 * - EscapeParser for VT-100 sequence parsing
 * - ScreenBuffer for screen content and scrollback
 * - TerminalState for cursor and terminal state
 * - TerminalRenderer for Canvas-based rendering
 * 
 * Features:
 * - PTY session integration
 * - Touch selection
 * - Scrollback buffer navigation
 * - Mouse support basics
 * - Font sizing with pinch zoom
 * - Cursor blinking
 */
class TerminalView @JvmOverloads constructor(
    context: Context,
    attrs: AttributeSet? = null,
    defStyleAttr: Int = 0
) : View(context, attrs, defStyleAttr), EscapeParser.CommandListener, PTYCallback {

    // Terminal components
    private val screenBuffer: ScreenBuffer
    private val terminalState: TerminalState
    private val escapeParser: EscapeParser
    private val terminalRenderer: TerminalRenderer
    
    // PTY session
    private var ptySession: PTYSession? = null
    
    // Configuration
    var columns: Int = 80
        private set
    var rows: Int = 24
        private set
    
    // Fonts
    private var fontSize: Float = 12f * resources.displayMetrics.scaledDensity
    private var fontFamily: Typeface = Typeface.MONOSPACE
    
    // Colors
    var backgroundColor: Int = Color.BLACK
    var foregroundColor: Int = Color.WHITE
    var cursorColor: Int = Color.WHITE
    
    // Cursor state
    private var cursorBlink: Boolean = true
    private val cursorBlinkHandler = Handler(Looper.getMainLooper())
    private val cursorBlinkRunnable = object : Runnable {
        override fun run() {
            terminalRenderer.toggleCursor()
            cursorBlinkHandler.postDelayed(this, 500)
        }
    }
    
    // Paint objects
    private val textPaint = Paint().apply {
        isAntiAlias = true
        typeface = fontFamily
        textSize = fontSize
    }
    
    private val backgroundPaint = Paint().apply {
        style = Paint.Style.FILL
    }
    
    // Callbacks
    var onCursorMoveListener: ((row: Int, col: Int) -> Unit)? = null
    var onTerminalResizeListener: ((cols: Int, rows: Int) -> Unit)? = null
    var onTitleChangeListener: ((title: String) -> Unit)? = null
    
    // Input method
    private var inputMethodManager: InputMethodManager? = null
    private var inputConnection: TerminalInputConnection? = null
    
    // Gesture detectors
    private var scaleGestureDetector: ScaleGestureDetector? = null
    private var gestureDetector: GestureDetector? = null
    
    // Selection state
    private var isSelecting: Boolean = false
    private var selectionStart: Pair<Int, Int>? = null
    private var selectionEnd: Pair<Int, Int>? = null
    
    // Scrollback scroll position
    private var scrollbackScrollPosition: Int = 0
    private var isScrolledBack: Boolean = false
    
    // Mouse support
    private var mouseTrackingEnabled: Boolean = false
    private var mouseButton: Int = 0
    
    // Scroll handling
    private var scrollY: Float = 0f
    private val scrollVelocityThreshold = 300f
    
    // Character dimensions
    private var charWidth: Float = 0f
    private var charHeight: Float = 0f
    
    // Listeners for terminal events
    private val terminalListeners = CopyOnWriteArrayList<TerminalViewListener>()
    
    /**
     * Interface for terminal view events
     */
    interface TerminalViewListener {
        fun onTerminalOutput(text: String)
        fun onTerminalResize(cols: Int, rows: Int)
        fun onTerminalTitleChange(title: String)
    }
    
    init {
        // Initialize terminal components
        screenBuffer = ScreenBuffer(columns, rows, 10000)
        terminalState = TerminalState(columns, rows)
        escapeParser = EscapeParser().apply {
            listener = this@TerminalView
        }
        terminalRenderer = TerminalRenderer(context, attrs, defStyleAttr).apply {
            setTerminal(screenBuffer, terminalState)
            setTerminalColors(foregroundColor, backgroundColor)
        }
        
        // Initialize input method
        inputMethodManager = context.getSystemService(Context.INPUT_METHOD_SERVICE) as? InputMethodManager
        
        // Initialize gesture detectors
        scaleGestureDetector = ScaleGestureDetector(context, object : ScaleGestureDetector.SimpleOnScaleGestureListener() {
            override fun onScale(detector: ScaleGestureDetector): Boolean {
                val scaleFactor = detector.scaleFactor
                setFontSize(fontSize * scaleFactor)
                return true
            }
        })
        
        gestureDetector = GestureDetector(context, object : GestureDetector.SimpleOnGestureListener() {
            override fun onScroll(e1: MotionEvent?, e2: MotionEvent, distanceX: Float, distanceY: Float): Boolean {
                // Handle scroll for scrollback
                if (isScrolledBack) {
                    val buffer = screenBuffer
                    val maxScroll = buffer.scrollbackLines
                    scrollbackScrollPosition = (scrollbackScrollPosition + (distanceY / charHeight).toInt())
                        .coerceIn(0, maxScroll)
                    terminalRenderer.setScrollbackOffset(scrollbackScrollPosition)
                    return true
                }
                return false
            }
            
            override fun onFling(e1: MotionEvent?, e2: MotionEvent, velocityX: Float, velocityY: Float): Boolean {
                // Handle fling for scrollback
                if (kotlin.math.abs(velocityY) > scrollVelocityThreshold) {
                    val buffer = screenBuffer
                    val maxScroll = buffer.scrollbackLines
                    val scrollAmount = if (velocityY > 0) 10 else -10
                    scrollbackScrollPosition = (scrollbackScrollPosition + scrollAmount)
                        .coerceIn(0, maxScroll)
                    terminalRenderer.setScrollbackOffset(scrollbackScrollPosition)
                    return true
                }
                return false
            }
            
            override fun onLongPress(e: MotionEvent) {
                // Start selection on long press
                val (row, col) = getPositionFromTouch(e.x, e.y)
                selectionStart = Pair(row, col)
                selectionEnd = Pair(row, col)
                isSelecting = true
                updateSelection()
            }
        })
        
        // Calculate character dimensions
        updateCharDimensions()
        
        // Start cursor blink
        startCursorBlink()
    }
    
    /**
     * Attach a PTY session
     */
    fun setSession(session: PTYSession) {
        ptySession = session
        session.setCallback(this)
        
        // Notify of initial size
        onTerminalResize(columns, rows)
    }
    
    /**
     * Write data to the terminal (from PTY)
     */
    override fun onPtyData(data: String) {
        post {
            processInput(data)
        }
    }
    
    /**
     * Handle PTY exit
     */
    override fun onPtyExit(exitCode: Int) {
        post {
            // Notify listeners
            for (listener in terminalListeners) {
                // Could notify of terminal close
            }
        }
    }
    
    /**
     * Process input data (from PTY or directly)
     */
    fun processInput(data: String) {
        escapeParser.processString(data)
    }
    
    /**
     * Write text to the PTY (user input)
     */
    fun writeToPty(text: String) {
        ptySession?.write(text)
    }
    
    // EscapeParser.CommandListener implementation
    
    override fun onPrintable(char: Char) {
        handlePrintableChar(char)
    }
    
    override fun onControl(char: Char) {
        handleControlChar(char)
    }
    
    override fun onCSI(command: EscapeParser.CSICommand) {
        handleCSICommand(command)
    }
    
    override fun onOSC(command: EscapeParser.OSCCommand) {
        handleOSCCommand(command)
    }
    
    override fun onESC(command: EscapeParser.ESCCommand) {
        handleESCCommand(command)
    }
    
    override fun onESCBracket(command: String) {
        // Legacy handler - not used
    }
    
    /**
     * Handle printable character
     */
    private fun handlePrintableChar(char: Char) {
        val state = terminalState
        val buffer = screenBuffer
        
        // Handle wrap pending
        if (state.wrapPending) {
            state.carriageReturn()
            state.lineFeed()
            state.wrapPending = false
        }
        
        // Insert or overstrike
        if (state.insertMode) {
            buffer.insertCharacters(state.cursorRow, state.cursorCol, 1)
        }
        
        // Create cell with current attributes
        val cell = state.createCell(char)
        
        // Set character in buffer
        buffer.setChar(state.cursorRow, state.cursorCol, cell)
        
        // Advance cursor
        if (state.advanceCursor()) {
            // Wrap occurred
            state.carriageReturn()
            state.lineFeed()
        }
        
        invalidate()
    }
    
    /**
     * Handle control character
     */
    private fun handleControlChar(char: Char) {
        val state = terminalState
        val buffer = screenBuffer
        
        when (char.code) {
            0x07 -> { // BEL - could play sound
            }
            0x08 -> { // BS - Backspace
                state.backspace()
            }
            0x09 -> { // HT - Tab
                state.tab()
            }
            0x0A, 0x0B, 0x0C -> { // LF, VT, FF - Line Feed
                state.lineFeed()
                // Scroll if at bottom of scroll region
                if (state.cursorRow > state.scrollBottom) {
                    buffer.scrollUp(state.scrollTop, state.scrollBottom)
                    state.cursorRow = state.scrollBottom
                }
            }
            0x0D -> { // CR - Carriage Return
                state.carriageReturn()
            }
            else -> {
                // Other control characters - ignore
            }
        }
        
        invalidate()
    }
    
    /**
     * Handle CSI command
     */
    private fun handleCSICommand(command: EscapeParser.CSICommand) {
        val state = terminalState
        val buffer = screenBuffer
        val params = command.params
        
        when (command.type) {
            // Cursor movement
            EscapeParser.CSICommand.Type.CUU -> {
                state.cursorUp(command.getParam1())
            }
            EscapeParser.CSICommand.Type.CUD -> {
                state.cursorDown(command.getParam1())
            }
            EscapeParser.CSICommand.Type.CUF -> {
                state.cursorForward(command.getParam1())
            }
            EscapeParser.CSICommand.Type.CUB -> {
                state.cursorBack(command.getParam1())
            }
            EscapeParser.CSICommand.Type.CNL -> {
                state.cursorNextLine(command.getParam1())
            }
            EscapeParser.CSICommand.Type.CPL -> {
                state.cursorPreviousLine(command.getParam1())
            }
            EscapeParser.CSICommand.Type.CHA -> {
                state.cursorHorizontalAbsolute(command.getParam1())
            }
            EscapeParser.CSICommand.Type.CUP -> {
                state.cursorPosition(command.getParam1(), command.getParam2())
            }
            
            // Erase
            EscapeParser.CSICommand.Type.ED -> {
                val param = command.getParam1()
                when (param) {
                    0 -> buffer.clearScreenBelow(state.cursorRow, state.cursorCol)
                    1 -> buffer.clearScreenAbove(state.cursorRow, state.cursorCol)
                    2, 3 -> buffer.clear()
                }
            }
            EscapeParser.CSICommand.Type.EL -> {
                val param = command.getParam1()
                when (param) {
                    0 -> buffer.clearLineToEnd(state.cursorRow, state.cursorCol)
                    1 -> buffer.clearLineToBeginning(state.cursorRow, state.cursorCol)
                    2 -> buffer.clearLine(state.cursorRow)
                }
            }
            EscapeParser.CSICommand.Type.ECH -> {
                buffer.eraseCharacters(state.cursorRow, state.cursorCol, command.getParam1())
            }
            
            // Insert/Delete
            EscapeParser.CSICommand.Type.ICH -> {
                buffer.insertCharacters(state.cursorRow, state.cursorCol, command.getParam1())
            }
            EscapeParser.CSICommand.Type.DCH -> {
                buffer.deleteCharacters(state.cursorRow, state.cursorCol, command.getParam1())
            }
            EscapeParser.CSICommand.Type.IL -> {
                buffer.insertLine(state.cursorRow, command.getParam1(), state.scrollBottom)
            }
            EscapeParser.CSICommand.Type.DL -> {
                buffer.deleteLines(state.cursorRow, command.getParam1(), state.scrollBottom)
            }
            
            // Scrolling
            EscapeParser.CSICommand.Type.SU -> {
                repeat(command.getParam1()) {
                    buffer.scrollUp(state.scrollTop, state.scrollBottom)
                }
            }
            EscapeParser.CSICommand.Type.SD -> {
                repeat(command.getParam1()) {
                    buffer.scrollDown(state.scrollTop, state.scrollBottom)
                }
            }
            EscapeParser.CSICommand.Type.DECSTBM -> {
                val top = if (params.isEmpty() || params[0] == 0) 1 else params[0]
                val bottom = if (params.size < 2 || params[1] == 0) rows else params[1]
                state.setScrollRegion(top - 1, bottom - 1)
            }
            
            // SGR - Select Graphic Rendition
            EscapeParser.CSICommand.Type.SGR -> {
                handleSGR(params)
            }
            
            // Save/Restore cursor
            EscapeParser.CSICommand.Type.SCUSR -> {
                state.saveCursor()
            }
            EscapeParser.CSICommand.Type.RCUSR -> {
                state.restoreCursor()
            }
            
            // Mode changes
            EscapeParser.CSICommand.Type.SM -> {
                handleSetMode(command.prefix, params, true)
            }
            EscapeParser.CSICommand.Type.RM -> {
                handleSetMode(command.prefix, params, false)
            }
            
            else -> {
                // Unknown command - ignore
            }
        }
        
        invalidate()
    }
    
    /**
     * Handle SGR (Select Graphic Rendition) parameters
     */
    private fun handleSGR(params: IntArray) {
        val state = terminalState
        
        if (params.isEmpty()) {
            // Reset
            state.currentFgColor = Cell.COLOR_WHITE
            state.currentBgColor = Cell.COLOR_BLACK
            state.currentBold = false
            state.currentItalic = false
            state.currentUnderline = false
            state.currentBlink = false
            state.currentInverse = false
            state.currentDim = false
            return
        }
        
        var i = 0
        while (i < params.size) {
            val param = params[i]
            
            when (param) {
                0 -> { // Reset
                    state.currentFgColor = Cell.COLOR_WHITE
                    state.currentBgColor = Cell.COLOR_BLACK
                    state.currentBold = false
                    state.currentItalic = false
                    state.currentUnderline = false
                    state.currentBlink = false
                    state.currentInverse = false
                    state.currentDim = false
                }
                1 -> state.currentBold = true
                2 -> state.currentDim = true
                3 -> state.currentItalic = true
                4 -> state.currentUnderline = true
                5 -> state.currentBlink = true
                7 -> state.currentInverse = true
                22 -> { state.currentBold = false; state.currentDim = false }
                23 -> state.currentItalic = false
                24 -> state.currentUnderline = false
                25 -> state.currentBlink = false
                27 -> state.currentInverse = false
                
                // Foreground colors
                30 -> state.currentFgColor = Cell.COLOR_BLACK
                31 -> state.currentFgColor = Cell.COLOR_RED
                32 -> state.currentFgColor = Cell.COLOR_GREEN
                33 -> state.currentFgColor = Cell.COLOR_YELLOW
                34 -> state.currentFgColor = Cell.COLOR_BLUE
                35 -> state.currentFgColor = Cell.COLOR_MAGENTA
                36 -> state.currentFgColor = Cell.COLOR_CYAN
                37 -> state.currentFgColor = Cell.COLOR_WHITE
                39 -> state.currentFgColor = Cell.COLOR_WHITE
                
                // Background colors
                40 -> state.currentBgColor = Cell.COLOR_BLACK
                41 -> state.currentBgColor = Cell.COLOR_RED
                42 -> state.currentBgColor = Cell.COLOR_GREEN
                43 -> state.currentBgColor = Cell.COLOR_YELLOW
                44 -> state.currentBgColor = Cell.COLOR_BLUE
                45 -> state.currentBgColor = Cell.COLOR_MAGENTA
                46 -> state.currentBgColor = Cell.COLOR_CYAN
                47 -> state.currentBgColor = Cell.COLOR_WHITE
                49 -> state.currentBgColor = Cell.COLOR_BLACK
                
                // Bright foreground
                90 -> state.currentFgColor = Cell.COLOR_BRIGHT_BLACK
                91 -> state.currentFgColor = Cell.COLOR_BRIGHT_RED
                92 -> state.currentFgColor = Cell.COLOR_BRIGHT_GREEN
                93 -> state.currentFgColor = Cell.COLOR_BRIGHT_YELLOW
                94 -> state.currentFgColor = Cell.COLOR_BRIGHT_BLUE
                95 -> state.currentFgColor = Cell.COLOR_BRIGHT_MAGENTA
                96 -> state.currentFgColor = Cell.COLOR_BRIGHT_CYAN
                97 -> state.currentFgColor = Cell.COLOR_BRIGHT_WHITE
                
                // Bright background
                100 -> state.currentBgColor = Cell.COLOR_BRIGHT_BLACK
                101 -> state.currentBgColor = Cell.COLOR_BRIGHT_RED
                102 -> state.currentBgColor = Cell.COLOR_BRIGHT_GREEN
                103 -> state.currentBgColor = Cell.COLOR_BRIGHT_YELLOW
                104 -> state.currentBgColor = Cell.COLOR_BRIGHT_BLUE
                105 -> state.currentBgColor = Cell.COLOR_BRIGHT_MAGENTA
                106 -> state.currentBgColor = Cell.COLOR_BRIGHT_CYAN
                107 -> state.currentBgColor = Cell.COLOR_BRIGHT_WHITE
                
                // Extended colors: 38;5;n (fg) or 48;5;n (bg)
                38 -> {
                    if (i + 2 < params.size && params[i + 1] == 5) {
                        state.currentFgColor = params[i + 2]
                        i += 2
                    } else if (i + 4 < params.size && params[i + 1] == 2) {
                        // True color: 38;2;r;g;b
                        // For simplicity, we map to nearest 256 color
                        val r = params[i + 2]
                        val g = params[i + 3]
                        val b = params[i + 4]
                        state.currentFgColor = TerminalColors.toAnsiColor(Color.rgb(r, g, b))
                        i += 4
                    }
                }
                48 -> {
                    if (i + 2 < params.size && params[i + 1] == 5) {
                        state.currentBgColor = params[i + 2]
                        i += 2
                    } else if (i + 4 < params.size && params[i + 1] == 2) {
                        val r = params[i + 2]
                        val g = params[i + 3]
                        val b = params[i + 4]
                        state.currentBgColor = TerminalColors.toAnsiColor(Color.rgb(r, g, b))
                        i += 4
                    }
                }
                
                else -> {
                    // Unknown parameter - ignore
                }
            }
            i++
        }
    }
    
    /**
     * Handle set/reset mode
     */
    private fun handleSetMode(prefix: String, params: IntArray, set: Boolean) {
        val state = terminalState
        
        for (param in params) {
            when (param) {
                1 -> {} // DECCKM - Cursor keys mode (ignore)
                2 -> {} // DECANM - ANSI mode (ignore)
                3 -> {} // DECCOLM - Column mode (ignore)
                4 -> {} // DECSCLM - Smooth scroll (ignore)
                5 -> state.cursorVisible = set // DECSCNM - Screen mode (invert)
                6 -> state.originMode = set // DECOM - Origin mode
                7 -> state.autoWrap = set // DECAWM - Auto wrap
                8 -> {} // DECARM - Auto repeat (ignore)
                9 -> {} // DECINLM - Interlace (ignore)
                12 -> {} // Cursor blink (ignore)
                25 -> state.cursorVisible = set // DECTCEM - Cursor visible
                30 -> {} // Show/hide scrollbar (ignore)
                34 -> {} // Cursor style (ignore)
                1000, 1002, 1003 -> {
                    // Mouse tracking
                    mouseTrackingEnabled = set
                }
                1049 -> {} // Alternate screen buffer (ignore)
                2004 -> {} // Bracketed paste mode (ignore)
                else -> {
                    // Unknown mode - ignore
                }
            }
        }
    }
    
    /**
     * Handle OSC command (title, colors)
     */
    private fun handleOSCCommand(command: EscapeParser.OSCCommand) {
        when (command.type) {
            EscapeParser.OSCCommand.Type.SET_TITLE,
            EscapeParser.OSCCommand.Type.SET_WINDOW_TITLE,
            EscapeParser.OSCCommand.Type.SET_ICON -> {
                onTitleChangeListener?.invoke(command.data)
            }
            else -> {
                // Other OSC commands - ignore
            }
        }
    }
    
    /**
     * Handle ESC command
     */
    private fun handleESCCommand(command: EscapeParser.ESCCommand) {
        val state = terminalState
        val buffer = screenBuffer
        
        when (command.type) {
            EscapeParser.ESCCommand.Type.DECSC -> {
                state.saveCursor()
            }
            EscapeParser.ESCCommand.Type.DECRC -> {
                state.restoreCursor()
            }
            EscapeParser.ESCCommand.Type.IND -> {
                state.lineFeed()
                if (state.cursorRow > state.scrollBottom) {
                    buffer.scrollUp(state.scrollTop, state.scrollBottom)
                    state.cursorRow = state.scrollBottom
                }
            }
            EscapeParser.ESCCommand.Type.NEL -> {
                state.carriageReturn()
                state.lineFeed()
            }
            EscapeParser.ESCCommand.Type.RI -> {
                // Reverse Index - move cursor up, scroll down if at top
                if (state.cursorRow > state.scrollTop) {
                    state.cursorRow--
                } else {
                    buffer.scrollDown(state.scrollTop, state.scrollBottom)
                }
            }
            EscapeParser.ESCCommand.Type.DECSTR -> {
                // Soft terminal reset
                state.reset()
                buffer.clear()
            }
            else -> {
                // Other ESC commands - ignore
            }
        }
        
        invalidate()
    }
    
    /**
     * Resize the terminal
     */
    fun resize(newColumns: Int, newRows: Int) {
        if (newColumns <= 0 || newRows <= 0) return
        
        val oldColumns = columns
        val oldRows = rows
        
        columns = newColumns
        rows = newRows
        
        screenBuffer.resize(newColumns, newRows)
        terminalState.resize(newColumns, newRows)
        
        updateCharDimensions()
        requestLayout()
        
        // Notify PTY of resize
        ptySession?.resize(columns, rows)
        onTerminalResizeListener?.invoke(columns, rows)
        
        invalidate()
    }
    
    /**
     * Set font size
     */
    fun setFontSize(size: Float) {
        fontSize = size * resources.displayMetrics.scaledDensity
        textPaint.textSize = fontSize
        updateCharDimensions()
        requestLayout()
        invalidate()
    }
    
    /**
     * Update character dimensions
     */
    private fun updateCharDimensions() {
        charWidth = textPaint.measureText("M")
        val fontMetrics = textPaint.fontMetrics
        charHeight = (fontMetrics.descent - fontMetrics.ascent) * 1.2f
    }
    
    /**
     * Start cursor blink
     */
    private fun startCursorBlink() {
        cursorBlinkHandler.postDelayed(cursorBlinkRunnable, 500)
    }
    
    /**
     * Stop cursor blink
     */
    private fun stopCursorBlink() {
        cursorBlinkHandler.removeCallbacks(cursorBlinkRunnable)
    }
    
    /**
     * Set cursor blink
     */
    fun setCursorBlink(enable: Boolean) {
        cursorBlink = enable
        if (enable) {
            startCursorBlink()
        } else {
            stopCursorBlink()
            terminalRenderer.showCursor()
        }
    }
    
    /**
     * Clear the terminal
     */
    fun clear() {
        screenBuffer.clear()
        terminalState.reset()
        invalidate()
    }
    
    /**
     * Write text to the terminal
     */
    fun write(text: String) {
        processInput(text)
    }
    
    override fun onMeasure(widthMeasureSpec: Int, heightMeasureSpec: Int) {
        val width = (charWidth * columns).toInt()
        val height = (charHeight * rows).toInt()
        
        setMeasuredDimension(
            resolveSize(width, widthMeasureSpec),
            resolveSize(height, heightMeasureSpec)
        )
    }
    
    override fun onDraw(canvas: Canvas) {
        // Draw background
        backgroundPaint.color = backgroundColor
        canvas.drawRect(0f, 0f, width.toFloat(), height.toFloat(), backgroundPaint)
        
        // Draw terminal using the renderer logic (inlined for efficiency)
        drawTerminal(canvas)
    }
    
    /**
     * Draw terminal content
     */
    private fun drawTerminal(canvas: Canvas) {
        val buffer = screenBuffer
        val state = terminalState
        
        val cw = width.toFloat() / columns
        val ch = height.toFloat() / rows
        
        // Draw each cell
        textPaint.textSize = ch * 0.85f
        
        for (r in 0 until rows) {
            val row = buffer.getRow(r) ?: continue
            for (c in 0 until columns) {
                val cell = row[c]
                drawCell(canvas, cell, r, c, cw, ch)
            }
        }
        
        // Draw cursor
        if (state.cursorVisible && terminalRenderer.let { it.terminalState?.cursorVisible != false }) {
            drawCursor(canvas, state, cw, ch)
        }
        
        // Draw selection
        if (selectionStart != null && selectionEnd != null) {
            drawSelection(canvas, cw, ch)
        }
    }
    
    /**
     * Draw a single cell
     */
    private fun drawCell(canvas: Canvas, cell: Cell, row: Int, col: Int, cw: Float, ch: Float) {
        val x = col * cw
        val y = row * ch
        
        // Get effective colors
        val fgColor = if (cell.inverse) {
            TerminalColors.getColor(cell.backgroundColor)
        } else {
            TerminalColors.getColor(cell.foregroundColor)
        }
        val bgColor = if (cell.inverse) {
            TerminalColors.getColor(cell.foregroundColor)
        } else {
            TerminalColors.getColor(cell.backgroundColor)
        }
        
        // Draw background
        if (bgColor != backgroundColor) {
            backgroundPaint.color = bgColor
            canvas.drawRect(x, y, x + cw, y + ch, backgroundPaint)
        }
        
        // Draw character
        if (cell.char != ' ' && !cell.hidden) {
            textPaint.color = fgColor
            
            // Bold simulation
            if (cell.bold) {
                textPaint.typeface = Typeface.create(fontFamily, Typeface.BOLD)
            } else {
                textPaint.typeface = fontFamily
            }
            
            // Underline
            textPaint.isUnderlineText = cell.underline
            
            canvas.drawText(cell.char.toString(), x + cw * 0.1f, y + ch * 0.8f, textPaint)
            
            textPaint.isUnderlineText = false
            textPaint.typeface = fontFamily
        }
    }
    
    /**
     * Draw cursor
     */
    private fun drawCursor(canvas: Canvas, state: TerminalState, cw: Float, ch: Float) {
        val cursorStyle = terminalRenderer.cursorStyle
        val x = state.cursorCol * cw
        val y = state.cursorRow * ch
        
        when (cursorStyle) {
            TerminalRenderer.CursorStyle.BLOCK -> {
                backgroundPaint.color = cursorColor
                canvas.drawRect(x, y, x + cw, y + ch, backgroundPaint)
                
                // Draw inverted character
                val cell = screenBuffer.getCell(state.cursorRow, state.cursorCol)
                if (cell != null) {
                    textPaint.color = if (cell.inverse) {
                        TerminalColors.getColor(cell.foregroundColor)
                    } else {
                        TerminalColors.getColor(cell.backgroundColor)
                    }
                    canvas.drawText(cell.char.toString(), x + cw * 0.1f, y + ch * 0.8f, textPaint)
                }
            }
            TerminalRenderer.CursorStyle.UNDERLINE -> {
                backgroundPaint.color = cursorColor
                canvas.drawRect(x, y + ch * 0.85f, x + cw, y + ch, backgroundPaint)
            }
            TerminalRenderer.CursorStyle.CARET -> {
                backgroundPaint.color = cursorColor
                canvas.drawRect(x, y, x + cw * 0.2f, y + ch, backgroundPaint)
            }
        }
    }
    
    /**
     * Draw selection
     */
    private fun drawSelection(canvas: Canvas, cw: Float, ch: Float) {
        val start = selectionStart ?: return
        val end = selectionEnd ?: return
        
        val (startRow, endRow) = if (start.first <= end.first) {
            Pair(start.first, end.first)
        } else {
            Pair(end.first, start.first)
        }
        
        val (startCol, endCol) = if (start.second <= end.second) {
            Pair(start.second, end.second)
        } else {
            Pair(end.second, start.second)
        }
        
        backgroundPaint.color = Color.argb(100, 100, 100, 255)
        
        if (startRow == endRow) {
            canvas.drawRect(
                startCol * cw, startRow * ch,
                (endCol + 1) * cw, (startRow + 1) * ch,
                backgroundPaint
            )
        } else {
            // First line
            canvas.drawRect(startCol * cw, startRow * ch, columns * cw, (startRow + 1) * ch, backgroundPaint)
            // Middle lines
            for (r in (startRow + 1) until endRow) {
                canvas.drawRect(0f, r * ch, columns * cw, (r + 1) * ch, backgroundPaint)
            }
            // Last line
            canvas.drawRect(0f, endRow * ch, (endCol + 1) * cw, (endRow + 1) * ch, backgroundPaint)
        }
    }
    
    override fun onTouchEvent(event: MotionEvent): Boolean {
        scaleGestureDetector?.onTouchEvent(event)
        gestureDetector?.onTouchEvent(event)
        
        when (event.action) {
            MotionEvent.ACTION_DOWN -> {
                if (!isScrolledBack) {
                    val (row, col) = getPositionFromTouch(event.x, event.y)
                    selectionStart = Pair(row, col)
                    selectionEnd = Pair(row, col)
                    isSelecting = true
                }
                return true
            }
            MotionEvent.ACTION_MOVE -> {
                if (isSelecting) {
                    val (row, col) = getPositionFromTouch(event.x, event.y)
                    selectionEnd = Pair(row, col)
                    invalidate()
                }
                return true
            }
            MotionEvent.ACTION_UP -> {
                if (isSelecting) {
                    isSelecting = false
                    // Could copy selection to clipboard
                }
                return true
            }
        }
        return super.onTouchEvent(event)
    }
    
    /**
     * Get terminal position from touch coordinates
     */
    private fun getPositionFromTouch(x: Float, y: Float): Pair<Int, Int> {
        val col = (x / charWidth).toInt().coerceIn(0, columns - 1)
        val row = (y / charHeight).toInt().coerceIn(0, rows - 1)
        return Pair(row + scrollbackScrollPosition, col)
    }
    
    /**
     * Update selection display
     */
    private fun updateSelection() {
        invalidate()
    }
    
    /**
     * Get selected text
     */
    fun getSelectedText(): String {
        val start = selectionStart ?: return ""
        val end = selectionEnd ?: return ""
        
        // Normalize selection
        val (startRow, endRow) = if (start.first <= end.first) {
            Pair(start.first, end.first)
        } else {
            Pair(end.first, start.first)
        }
        
        val (startCol, endCol) = if (start.second <= end.second) {
            Pair(start.second, end.second)
        } else {
            Pair(end.second, start.second)
        }
        
        return screenBuffer.getText(startRow, startCol, endRow, endCol)
    }
    
    /**
     * Clear selection
     */
    fun clearSelection() {
        selectionStart = null
        selectionEnd = null
        isSelecting = false
        invalidate()
    }
    
    /**
     * Scroll back to see scrollback
     */
    fun scrollBack(lines: Int) {
        val maxScroll = screenBuffer.scrollbackLines
        scrollbackScrollPosition = (scrollbackScrollPosition + lines).coerceIn(0, maxScroll)
        isScrolledBack = scrollbackScrollPosition > 0
        terminalRenderer.setScrollbackOffset(scrollbackScrollPosition)
        invalidate()
    }
    
    /**
     * Scroll to end (latest screen content)
     */
    fun scrollToEnd() {
        scrollbackScrollPosition = 0
        isScrolledBack = false
        terminalRenderer.setScrollbackOffset(0)
        invalidate()
    }
    
    /**
     * Check if scrolled back
     */
    fun isScrolledBack(): Boolean = isScrolledBack
    
    /**
     * Handle mouse event (basic mouse support)
     */
    fun handleMouseEvent(event: MotionEvent, button: Int) {
        if (!mouseTrackingEnabled) return
        
        val (row, col) = getPositionFromTouch(event.x, event.y)
        mouseButton = button
        
        // Send mouse escape sequence
        // Format: CSI M Cb Cx Cy
        // Cb = button (0-3), Cx/Cy = coordinates (0-based, +32)
        val cb = button and 0x03
        val cx = (col + 32).coerceIn(32, 127)
        val cy = (row + 32).coerceIn(32, 127)
        
        val sequence = "\u001B[M${cb.toChar()}${cx.toChar()}${cy.toChar()}"
        writeToPty(sequence)
    }
    
    /**
     * Get screen buffer (for external access)
     */
    fun getScreenBuffer(): ScreenBuffer = screenBuffer
    
    /**
     * Get terminal state (for external access)
     */
    fun getTerminalState(): TerminalState = terminalState
    
    /**
     * Add terminal listener
     */
    fun addTerminalListener(listener: TerminalViewListener) {
        terminalListeners.add(listener)
    }
    
    /**
     * Remove terminal listener
     */
    fun removeTerminalListener(listener: TerminalViewListener) {
        terminalListeners.remove(listener)
    }
    
    override fun onDetachedFromWindow() {
        super.onDetachedFromWindow()
        stopCursorBlink()
    }
    
    /**
     * Terminal input connection for soft keyboard
     */
    inner class TerminalInputConnection(targetView: View, var isEditor: Boolean) : BaseInputConnection(targetView, isEditor) {
        
        override fun commitText(text: CharSequence, cursorPosition: Int): Boolean {
            writeToPty(text.toString())
            return true
        }
        
        override fun sendKeyEvent(event: KeyEvent): Boolean {
            if (event.action == KeyEvent.ACTION_DOWN) {
                when (event.keyCode) {
                    KeyEvent.KEYCODE_ENTER -> {
                        writeToPty("\r")
                        return true
                    }
                    KeyEvent.KEYCODE_DEL -> {
                        writeToPty("\u007F")
                        return true
                    }
                    KeyEvent.KEYCODE_TAB -> {
                        writeToPty("\t")
                        return true
                    }
                    KeyEvent.KEYCODE_BACK -> {
                        writeToPty("\u001B") // ESC
                        return true
                    }
                    else -> {
                        // Handle other keys via key code
                        val chars = event.characters
                        if (chars != null && chars.isNotEmpty()) {
                            writeToPty(chars)
                            return true
                        }
                    }
                }
            }
            return super.sendKeyEvent(event)
        }
        
        override fun getExtractedText(request: ExtractedTextRequest?): ExtractedText? {
            return null
        }
        
        override fun getSelectedText(flags: Int): CharSequence? {
            return getSelectedText()
        }
    }
    
    // Input method handling
    
    override fun onCheckIsTextEditor(): Boolean = true
    
    override fun onCreateInputConnection(outAttrs: EditorInfo): android.view.inputmethod.InputConnection {
        outAttrs.inputType = android.text.InputType.TYPE_NULL
        outAttrs.imeOptions = EditorInfo.IME_FLAG_NO_EXTRACT_UI
        inputConnection = TerminalInputConnection(this, true)
        return inputConnection!!
    }
    
    /**
     * Show soft keyboard
     */
    fun showSoftKeyboard() {
        inputMethodManager?.showSoftInput(this, InputMethodManager.SHOW_IMPLICIT)
    }
    
    /**
     * Hide soft keyboard
     */
    fun hideSoftKeyboard() {
        inputMethodManager?.hideSoftInputFromWindow(windowToken, 0)
    }
}
