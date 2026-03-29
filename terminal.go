package main

import (
	"os"
	"strconv"
	"syscall"
	"unsafe"
)

// ─── Terminal ──────────────────────────────────────────────────────────────

// getTerminalWidth returns the terminal width in columns.
// Priority: COLUMNS env var → ioctl TIOCGWINSZ → 120 default.
func getTerminalWidth() int {
	// Try COLUMNS env var first
	if s := os.Getenv("COLUMNS"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			return v
		}
	}

	// Try ioctl TIOCGWINSZ on stdout
	type winsize struct {
		Row    uint16
		Col    uint16
		Xpixel uint16
		Ypixel uint16
	}
	var ws winsize
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(syscall.Stdout),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&ws)),
	)
	if errno == 0 && ws.Col > 0 {
		return int(ws.Col)
	}

	return 120
}
