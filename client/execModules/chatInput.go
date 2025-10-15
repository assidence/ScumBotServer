package execModules

import (
	"fmt"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

var (
	user32        = syscall.NewLazyDLL("user32.dll")
	procSendInput = user32.NewProc("SendInput")
)

const (
	INPUT_KEYBOARD    = 1
	KEYEVENTF_KEYUP   = 0x0002
	KEYEVENTF_UNICODE = 0x0004
)

// KEYBDINPUT layout for SendInput
type KEYBDINPUT struct {
	WVk         uint16
	WScan       uint16
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
	_           [8]byte // <-- 手动补齐对齐填充
}

type INPUT struct {
	Type uint32
	_    uint32 // padding
	Ki   KEYBDINPUT
}

// sendInputs - wrapper to call SendInput with a slice of INPUT
func sendInputs(inputs []INPUT) (uint32, error) {
	if len(inputs) == 0 {
		return 0, nil
	}
	//fmt.Printf("[Debug] Sizeof INPUT = %d bytes\n", unsafe.Sizeof(inputs[0]))
	n, _, err := procSendInput.Call(
		uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		uintptr(unsafe.Sizeof(inputs[0])),
	)
	return uint32(n), err
}

// helper: press a virtual-key (down then up) with small delay
func pressKeyVk(vk uint16, downDelay time.Duration) error {
	inDown := INPUT{
		Type: INPUT_KEYBOARD,
		Ki: KEYBDINPUT{
			WVk: vk,
		},
	}
	inUp := INPUT{
		Type: INPUT_KEYBOARD,
		Ki: KEYBDINPUT{
			WVk:     vk,
			DwFlags: KEYEVENTF_KEYUP,
		},
	}
	if _, err := sendInputs([]INPUT{inDown}); err != nil && err != syscall.Errno(0) {
		return err
	}
	time.Sleep(downDelay)
	if _, err := sendInputs([]INPUT{inUp}); err != nil && err != syscall.Errno(0) {
		return err
	}
	return nil
}

// helper: key down / up (separate) for modifiers like Ctrl
func keyDownVk(vk uint16) error {
	in := INPUT{
		Type: INPUT_KEYBOARD,
		Ki: KEYBDINPUT{
			WVk: vk,
		},
	}
	_, err := sendInputs([]INPUT{in})
	if err != nil && err != syscall.Errno(0) {
		return err
	}
	return nil
}
func keyUpVk(vk uint16) error {
	in := INPUT{
		Type: INPUT_KEYBOARD,
		Ki: KEYBDINPUT{
			WVk:     vk,
			DwFlags: KEYEVENTF_KEYUP,
		},
	}
	_, err := sendInputs([]INPUT{in})
	if err != nil && err != syscall.Errno(0) {
		return err
	}
	return nil
}

// helper: send a Unicode code unit (use KEYEVENTF_UNICODE). We send as UTF-16 code units.
func sendUnicodeRune(r rune) error {
	// encode rune to UTF-16 (could be surrogate pair)
	utf := utf16.Encode([]rune{r})
	for _, cu := range utf {
		down := INPUT{
			Type: INPUT_KEYBOARD,
			Ki: KEYBDINPUT{
				WVk:     0,
				WScan:   cu,
				DwFlags: KEYEVENTF_UNICODE,
			},
		}
		up := INPUT{
			Type: INPUT_KEYBOARD,
			Ki: KEYBDINPUT{
				WVk:     0,
				WScan:   cu,
				DwFlags: KEYEVENTF_UNICODE | KEYEVENTF_KEYUP,
			},
		}
		if _, err := sendInputs([]INPUT{down}); err != nil && err != syscall.Errno(0) {
			return err
		}
		// small gap
		time.Sleep(8 * time.Millisecond)
		if _, err := sendInputs([]INPUT{up}); err != nil && err != syscall.Errno(0) {
			return err
		}
		time.Sleep(5 * time.Millisecond)
	}
	return nil
}

// SendChatMessage will: press 't', Ctrl+A, send message (unicode), Enter.
// gameTitle param is optional for logging only (can be empty).
// This function is synchronous and returns after sending.
func SendChatMessage(message string) error {
	// small guard
	if message == "" {
		return fmt.Errorf("empty message")
	}

	// Step 1: press 't' to open chat
	// Virtual-key code for 'T' is 0x54
	/*if err := pressKeyVk(0x54, 40*time.Millisecond); err != nil {
		return err
	}
	*/
	// give the game a moment to open chat box
	time.Sleep(120 * time.Millisecond)

	// Step 2: Ctrl + A (select all)
	// VK_CONTROL = 0x11, 'A' = 0x41
	fmt.Println("1")
	if err := keyDownVk(0x11); err != nil {
		return err
	}
	fmt.Println("2")
	// press 'A' down/up
	if err := pressKeyVk(0x41, 30*time.Millisecond); err != nil {
		_ = keyUpVk(0x11)
		return err
	}
	fmt.Println("3")
	// release Ctrl
	if err := keyUpVk(0x11); err != nil {
		return err
	}
	time.Sleep(60 * time.Millisecond)

	// Step 3: send the new message as Unicode (this will replace selected text)
	for _, r := range message {
		if err := sendUnicodeRune(r); err != nil {
			return err
		}
	}
	time.Sleep(40 * time.Millisecond)

	// Step 4: press Enter (VK_RETURN = 0x0D)
	if err := pressKeyVk(0x0D, 30*time.Millisecond); err != nil {
		return err
	}

	return nil
}
