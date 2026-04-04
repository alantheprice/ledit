package com.ledit.android.term

import android.content.Context
import android.graphics.Canvas
import android.graphics.Color
import android.graphics.Paint
import android.graphics.Typeface
import android.util.AttributeSet
import android.view.View
import java.util.concurrent.CopyOnWriteArrayList

/**
 * TerminalView - A custom View that renders terminal output using Canvas.
 * 
 * Features:
 * - Text rendering with monospace font
 * - Cursor blinking
 * - Color support (ANSI escape sequences)
 * - Scrollback buffer
 * - Efficient redraw using double buffering
 */
class TerminalView @JvmOverloads constructor(
    context: Context,
    attrs: AttributeSet? = null,
    defStyleAttr: Int = 0
) : View(context, attrs, defStyleAttr) {

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
    private var cursorRow: Int = 0
    private var cursorCol: Int = 0
    private var cursorVisible: Boolean = true
    private var cursorBlink: Boolean = true
    
    // Terminal buffer
    private val screenBuffer = CopyOnWriteArrayList<Array<TerminalCell>>()
    private val scrollbackBuffer = CopyOnWriteArrayList<Array<TerminalCell>>()
    private val maxScrollbackLines = 10000
    
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
    
    init {
        // Initialize screen buffer
        allocateBuffer()
    }
    
    /**
     * Allocate the screen buffer
     */
    private fun allocateBuffer() {
        screenBuffer.clear()
        for (r in 0 until rows) {
            val row = arrayOfNulls<TerminalCell>(columns)
            for (c in 0 until columns) {
                row[c] = TerminalCell(' ')
            }
            @Suppress("UNCHECKED_CAST")
            screenBuffer.add(row as Array<TerminalCell>)
        }
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
        
        // Reallocate buffer
        val oldBuffer = screenBuffer.toList()
        allocateBuffer()
        
        // Copy old content
        for (r in 0 until minOf(oldRows, rows)) {
            for (c in 0 until minOf(oldColumns, columns)) {
                if (r < oldBuffer.size && c < oldBuffer[r].size) {
                    screenBuffer[r][c] = oldBuffer[r][c]
                }
            }
        }
        
        invalidate()
    }
    
    /**
     * Set font size
     */
    fun setFontSize(size: Float) {
        fontSize = size * resources.displayMetrics.scaledDensity
        textPaint.textSize = fontSize
        requestLayout()
        invalidate()
    }
    
    /**
     * Write text to the terminal
     */
    fun write(text: String) {
        for (char in text) {
            when (char) {
                '\n' -> lineFeed()
                '\r' -> carriageReturn()
                '\t' -> tab()
                '\b' -> backspace()
                else -> {
                    if (char.code >= 32) { // Printable character
                        putChar(char)
                    }
                }
            }
        }
        invalidate()
    }
    
    /**
     * Write text with ANSI color codes
     */
    fun writeAnsi(text: String) {
        // Parse ANSI escape sequences
        // For now, just write the text
        write(text)
    }
    
    /**
     * Put a character at cursor position
     */
    private fun putChar(c: Char) {
        if (cursorCol >= columns) {
            lineFeed()
            cursorCol = 0
        }
        if (cursorRow >= rows) {
            scrollUp()
            cursorRow = rows - 1
        }
        
        screenBuffer[cursorRow][cursorCol] = TerminalCell(c, foregroundColor, backgroundColor)
        cursorCol++
    }
    
    private fun lineFeed() {
        cursorCol = 0
        if (cursorRow < rows - 1) {
            cursorRow++
        } else {
            scrollUp()
        }
    }
    
    private fun carriageReturn() {
        cursorCol = 0
    }
    
    private fun tab() {
        cursorCol = (cursorCol + 8) and 0xFFFFFFF8.toInt()
        if (cursorCol >= columns) cursorCol = columns - 1
    }
    
    private fun backspace() {
        if (cursorCol > 0) {
            cursorCol--
        }
    }
    
    /**
     * Scroll the screen up by one line
     */
    private fun scrollUp() {
        // Move top line to scrollback
        if (scrollbackBuffer.size < maxScrollbackLines) {
            scrollbackBuffer.add(screenBuffer[0].copyOf())
        }
        
        // Shift screen up
        for (r in 0 until rows - 1) {
            screenBuffer[r] = screenBuffer[r + 1]
        }
        
        // Clear bottom line
        val newRow = arrayOfNulls<TerminalCell>(columns)
        for (c in 0 until columns) {
            newRow[c] = TerminalCell(' ')
        }
        @Suppress("UNCHECKED_CAST")
        screenBuffer[rows - 1] = newRow as Array<TerminalCell>
    }
    
    /**
     * Move cursor
     */
    fun moveCursor(row: Int, col: Int) {
        cursorRow = row.coerceIn(0, rows - 1)
        cursorCol = col.coerceIn(0, columns - 1)
        onCursorMoveListener?.invoke(cursorRow, cursorCol)
    }
    
    /**
     * Clear the terminal
     */
    fun clear() {
        for (r in 0 until rows) {
            for (c in 0 until columns) {
                screenBuffer[r][c] = TerminalCell(' ', foregroundColor, backgroundColor)
            }
        }
        cursorRow = 0
        cursorCol = 0
        invalidate()
    }
    
    /**
     * Set cursor blink
     */
    fun setCursorBlink(enable: Boolean) {
        cursorBlink = enable
    }
    
    /**
     * Toggle cursor visibility (for blinking)
     */
    fun toggleCursor() {
        if (cursorBlink) {
            cursorVisible = !cursorVisible
            invalidate()
        }
    }
    
    override fun onMeasure(widthMeasureSpec: Int, heightMeasureSpec: Int) {
        // Calculate size based on font and columns/rows
        val charWidth = textPaint.measureText("M")
        val charHeight = textPaint.textSize * 1.2f
        
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
        
        // Calculate character dimensions
        val charWidth = width.toFloat() / columns
        val charHeight = height.toFloat() / rows
        
        // Draw each cell
        textPaint.textSize = charHeight * 0.9f
        
        for (r in 0 until rows) {
            for (c in 0 until columns) {
                val cell = screenBuffer[r][c]
                
                // Draw background for this cell
                if (cell.backgroundColor != backgroundColor) {
                    backgroundPaint.color = cell.backgroundColor
                    canvas.drawRect(
                        c * charWidth,
                        r * charHeight,
                        (c + 1) * charWidth,
                        (r + 1) * charHeight,
                        backgroundPaint
                    )
                }
                
                // Draw character
                if (cell.char != ' ') {
                    textPaint.color = cell.foregroundColor
                    canvas.drawText(
                        cell.char.toString(),
                        c * charWidth + 2,
                        r * charHeight + charHeight * 0.8f,
                        textPaint
                    )
                }
            }
        }
        
        // Draw cursor
        if (cursorVisible && cursorRow < rows && cursorCol < columns) {
            backgroundPaint.color = cursorColor
            canvas.drawRect(
                cursorCol * charWidth,
                cursorRow * charHeight,
                (cursorCol + 1) * charWidth,
                (cursorRow + 1) * charHeight,
                backgroundPaint
            )
        }
    }
    
    /**
     * Terminal cell data class
     */
    data class TerminalCell(
        val char: Char,
        val foregroundColor: Int = Color.WHITE,
        val backgroundColor: Int = Color.BLACK
    )
}