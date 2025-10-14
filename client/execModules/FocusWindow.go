package execModules

import (
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// 引用 user32.dll
var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procGetWindowTextW      = user32.NewProc("GetWindowTextW")
	procFindWindowW         = user32.NewProc("FindWindowW")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
)

// 获取当前活动窗口标题
func getActiveWindowTitle() string {
	hWnd, _, _ := procGetForegroundWindow.Call()
	if hWnd == 0 {
		return ""
	}

	buf := make([]uint16, 256)
	procGetWindowTextW.Call(hWnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf)
}

// 查找指定窗口句柄
func findWindow(title string) uintptr {
	hWnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(title))))
	return hWnd
}

// 激活指定窗口
func activateWindow(hWnd uintptr) bool {
	ret, _, _ := procSetForegroundWindow.Call(hWnd)
	return ret != 0
}

func FocusWindows(gameTitle string) {

	fmt.Println("启动自动焦点监控程序...")

	for {
		active := getActiveWindowTitle()
		if active == "" {
			time.Sleep(1 * time.Second)
			continue
		}

		if !strings.Contains(strings.ToLower(active), strings.ToLower(gameTitle)) {
			fmt.Printf("当前焦点不在游戏窗口（当前: %s），尝试切换...\n", active)
			hWnd := findWindow(gameTitle)
			if hWnd != 0 {
				ok := activateWindow(hWnd)
				if ok {
					fmt.Printf("已切换焦点到游戏窗口: %s\n", gameTitle)
				} else {
					fmt.Println("⚠️  无法切换焦点，请以管理员权限运行。")
				}
			} else {
				fmt.Println("⚠️  未找到游戏窗口。请确认窗口标题正确。")
			}
		} else {
			fmt.Println("✅ 焦点在游戏窗口中。")
		}

		time.Sleep(1 * time.Second)
	}
}
