package main

import (
	"os"
	"strconv"
	"syscall"
	"unsafe"
)

// ─── Terminal ──────────────────────────────────────────────────────────────

// getTerminalWidth returns the terminal width in columns.
// Priority: COLUMNS env var → ioctl TIOCGWINSZ (stdout, stderr, /dev/tty) → 80 default.
func getTerminalWidth() int {
	// Try COLUMNS env var first
	if s := os.Getenv("COLUMNS"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			return v
		}
	}

	type winsize struct {
		Row    uint16
		Col    uint16
		Xpixel uint16
		Ypixel uint16
	}
	var ws winsize

	// Try stdout, then stderr (stdout may be piped when Claude Code captures output)
	for _, fd := range []uintptr{uintptr(syscall.Stdout), uintptr(syscall.Stderr)} {
		_, _, errno := syscall.Syscall(
			syscall.SYS_IOCTL,
			fd,
			uintptr(syscall.TIOCGWINSZ),
			uintptr(unsafe.Pointer(&ws)),
		)
		if errno == 0 && ws.Col > 0 {
			return int(ws.Col)
		}
	}

	// Try /dev/tty as last resort (works even when both stdout and stderr are piped)
	if f, err := os.Open("/dev/tty"); err == nil {
		defer f.Close()
		_, _, errno := syscall.Syscall(
			syscall.SYS_IOCTL,
			f.Fd(),
			uintptr(syscall.TIOCGWINSZ),
			uintptr(unsafe.Pointer(&ws)),
		)
		if errno == 0 && ws.Col > 0 {
			return int(ws.Col)
		}
	}

	return 80
}
