package internal

import (
	"encoding/binary"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// Touch input device configuration.
// Defaults match the official Raspberry Pi 7" touchscreen at 1920x1080.
// Override with VEGAS_TOUCH_DEVICE env var (set to "none" to disable).
const (
	defaultTouchDevice = "/dev/input/event1"
	touchMaxX          = 1920.0
	touchMaxY          = 1080.0
)

// Linux input event constants
const (
	evKey  = 0x01
	evAbs  = 0x03
	absX   = 0x00
	absY   = 0x01
	absMtX = 0x35 // ABS_MT_POSITION_X (multitouch)
	absMtY = 0x36 // ABS_MT_POSITION_Y (multitouch)
	btnTouch = 330
)

// inputEvent matches the Linux kernel input_event struct on 64-bit systems.
// Layout: timeval (16 bytes) + type (2) + code (2) + value (4) = 24 bytes.
type inputEvent struct {
	TimeSec  int64
	TimeUsec int64
	Type     uint16
	Code     uint16
	Value    int32
}

// rawTouchMsg carries raw touch coordinates from the hardware listener.
// Converted to tea.MouseMsg in App.Update() using current terminal dimensions.
type rawTouchMsg struct {
	X int
	Y int
}

// StartTouchListener reads raw input events from the touchscreen device
// and sends rawTouchMsg via p.Send() on each tap. Call as a goroutine
// before p.Run(). Exits silently if the device is unavailable (non-Pi).
func StartTouchListener(p *tea.Program) {
	device := os.Getenv("VEGAS_TOUCH_DEVICE")
	if device == "none" {
		return
	}
	if device == "" {
		device = defaultTouchDevice
	}

	f, err := os.Open(device)
	if err != nil {
		// Device not found — not running on Pi or no permissions.
		// This is expected on dev machines. Silently return.
		return
	}
	defer f.Close()

	var event inputEvent
	var currentX, currentY int32

	for {
		err := binary.Read(f, binary.LittleEndian, &event)
		if err != nil {
			return
		}

		// Track touch coordinates (both single-touch and multitouch protocols)
		if event.Type == evAbs {
			switch event.Code {
			case absX, absMtX:
				currentX = event.Value
			case absY, absMtY:
				currentY = event.Value
			}
		}

		// BTN_TOUCH value=0 (finger lift) = completed tap.
		// Fire on lift, not down — gives the user time to adjust
		// finger position for precise targeting (standard mobile UX).
		if event.Type == evKey && event.Code == btnTouch && event.Value == 0 {
			p.Send(rawTouchMsg{X: int(currentX), Y: int(currentY)})
		}
	}
}
