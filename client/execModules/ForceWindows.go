package execModules

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

var (
	forceDll                   = syscall.NewLazyDLL("forcefg.dll")
	procForceForegroundByTitle = forceDll.NewProc("ForceForegroundByTitle")
)

// 调用 DLL
func callForceFG(title string) (int, error) {
	if err := forceDll.Load(); err != nil {
		return 0, err
	}
	u16, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return 0, err
	}
	r1, _, _ := procForceForegroundByTitle.Call(uintptr(unsafe.Pointer(u16)))
	return int(r1), nil
}

// 监控并切换前台
func FocusWindows(gameTitle string) {
	fmt.Println("[Client-WatchDog] 启动自动焦点监控（DLL 强制前台）...")

	for {
		status, err := callForceFG(gameTitle) // 直接传 "SCUM  "
		if err != nil {
			fmt.Printf("[Client-WatchDog] DLL 调用出错: %v\n", err)
		} else {
			switch status {
			case 0:
				fmt.Printf("[Client-WatchDog] DLL 结果: 未找到窗口 (0) => '%s'\n", gameTitle)
			case 1:
				fmt.Printf("[Client-WatchDog] DLL 结果: 找到可见但未成为前台 (1) => '%s'\n", gameTitle)
			case 2:
				fmt.Printf("[Client-WatchDog] DLL 结果: 已成功切换为前台 (2) => '%s'\n", gameTitle)
			default:
				fmt.Printf("[Client-WatchDog] DLL 返回未知状态: %d => '%s'\n", status, gameTitle)
			}
		}

		time.Sleep(5 * time.Second)
	}
}
