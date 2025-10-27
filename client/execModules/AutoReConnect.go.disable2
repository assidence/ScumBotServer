//go:build windows

package execModules

import (
	"fmt"
	"image"
	"log"
	"syscall"
	"time"
	"unsafe"

	"gocv.io/x/gocv"
)

var (
	user32dll1                 = syscall.NewLazyDLL("user32.dll") // 避免重复定义
	gdi32dll                   = syscall.NewLazyDLL("gdi32.dll")
	procFindWindowW            = user32dll1.NewProc("FindWindowW")
	procSetForeground          = user32dll1.NewProc("SetForegroundWindow")
	procMoveWindow             = user32dll1.NewProc("MoveWindow")
	procGetClientRect          = user32dll1.NewProc("GetClientRect")
	procGetDC                  = user32dll1.NewProc("GetDC")
	procReleaseDC              = user32dll1.NewProc("ReleaseDC")
	procCreateCompatibleDC     = gdi32dll.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap = gdi32dll.NewProc("CreateCompatibleBitmap")
	procSelectObject           = gdi32dll.NewProc("SelectObject")
	procBitBlt                 = gdi32dll.NewProc("BitBlt")
	procDeleteDC               = gdi32dll.NewProc("DeleteDC")
	procDeleteObject           = gdi32dll.NewProc("DeleteObject")
	procGetDIBits              = gdi32dll.NewProc("GetDIBits")
	procSendInputNew           = user32dll1.NewProc("SendInput")
	procGetSystemMetrics       = user32dll1.NewProc("GetSystemMetrics")
)

type RECT struct {
	Left, Top, Right, Bottom int32
}

type BITMAPINFOHEADER struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

// 发送键盘按键
func keyTap(keyCode uint16) {
	var input [1]struct {
		Type uint32
		Ki   struct {
			WVk         uint16
			WScan       uint16
			DwFlags     uint32
			Time        uint32
			DwExtraInfo uintptr
		}
	}

	input[0].Type = 1 // INPUT_KEYBOARD
	input[0].Ki.WVk = keyCode
	procSendInputNew.Call(
		uintptr(1),
		uintptr(unsafe.Pointer(&input[0])),
		unsafe.Sizeof(input[0]),
	)
}

// 鼠标点击
func clickAt(x, y int) {
	var input [3]struct {
		Type uint32
		Mi   struct {
			Dx          int32
			Dy          int32
			MouseData   uint32
			DwFlags     uint32
			Time        uint32
			DwExtraInfo uintptr
		}
	}

	input[0].Type = 0 // INPUT_MOUSE
	input[0].Mi.Dx = int32(x * 65535 / 1920)
	input[0].Mi.Dy = int32(y * 65535 / 1080)
	input[0].Mi.DwFlags = 0x8000 | 0x0001 // MOUSEEVENTF_ABSOLUTE | MOUSEEVENTF_MOVE

	input[1].Type = 0
	input[1].Mi.DwFlags = 0x0002 // LEFTDOWN

	input[2].Type = 0
	input[2].Mi.DwFlags = 0x0004 // LEFTUP

	procSendInputNew.Call(uintptr(3), uintptr(unsafe.Pointer(&input[0])), unsafe.Sizeof(input[0]))
}

func utf16Ptr(s string) *uint16 {
	ptr, _ := syscall.UTF16PtrFromString(s)
	return ptr
}

func findWindow(title string) (uintptr, error) {
	hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(utf16Ptr(title))))
	if hwnd == 0 {
		return 0, fmt.Errorf("未找到窗口: %s", title)
	}
	return hwnd, nil
}

func setForeground(hwnd uintptr) {
	procSetForeground.Call(hwnd)
}

func moveWindow(hwnd uintptr, x, y, w, h int) {
	procMoveWindow.Call(hwnd, uintptr(x), uintptr(y), uintptr(w), uintptr(h), 1)
}

func getClientRect(hwnd uintptr) (RECT, error) {
	var rect RECT
	ret, _, _ := procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
	if ret == 0 {
		return rect, fmt.Errorf("GetClientRect 失败")
	}
	return rect, nil
}

// 获取屏幕分辨率
func getScreenSize() (int, int) {
	cx, _, _ := procGetSystemMetrics.Call(0) // SM_CXSCREEN
	cy, _, _ := procGetSystemMetrics.Call(1) // SM_CYSCREEN
	return int(cx), int(cy)
}

// 截取窗口
func captureWindow(hwnd uintptr) (gocv.Mat, error) {
	rect, err := getClientRect(hwnd)
	if err != nil {
		return gocv.NewMat(), err
	}
	width := int(rect.Right - rect.Left)
	height := int(rect.Bottom - rect.Top)
	if width <= 0 || height <= 0 {
		return gocv.NewMat(), fmt.Errorf("窗口无效大小")
	}

	hDC, _, _ := procGetDC.Call(hwnd)
	if hDC == 0 {
		return gocv.NewMat(), fmt.Errorf("GetDC失败")
	}
	defer procReleaseDC.Call(hwnd, hDC)

	memDC, _, _ := procCreateCompatibleDC.Call(hDC)
	defer procDeleteDC.Call(memDC)

	hBmp, _, _ := procCreateCompatibleBitmap.Call(hDC, uintptr(width), uintptr(height))
	defer procDeleteObject.Call(hBmp)

	procSelectObject.Call(memDC, hBmp)
	procBitBlt.Call(memDC, 0, 0, uintptr(width), uintptr(height), hDC, 0, 0, 0x00CC0020)

	bi := BITMAPINFOHEADER{
		Size:     uint32(unsafe.Sizeof(BITMAPINFOHEADER{})),
		Width:    int32(width),
		Height:   -int32(height),
		Planes:   1,
		BitCount: 32,
	}

	img := gocv.NewMatWithSize(height, width, gocv.MatTypeCV8UC4)
	data, err := img.DataPtrUint8()
	if err != nil {
		return gocv.NewMat(), fmt.Errorf("获取数据指针失败: %v", err)
	}
	procGetDIBits.Call(memDC, hBmp, 0, uintptr(height),
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(unsafe.Pointer(&bi)), 0)

	return img, nil
}

// 模板匹配
func matchTemplate(screenshot gocv.Mat, templatePath string) (image.Point, bool) {
	template := gocv.IMRead(templatePath, gocv.IMReadColor)
	if template.Empty() {
		return image.Point{}, false
	}
	defer template.Close()

	result := gocv.NewMat()
	defer result.Close()

	gocv.MatchTemplate(screenshot, template, &result, gocv.TmCcoeffNormed, gocv.NewMat())
	_, maxVal, _, maxLoc := gocv.MinMaxLoc(result)
	if maxVal > 0.9 {
		return maxLoc, true
	}
	return image.Point{}, false
}

// 计算区域平均 BGR
func meanBGR(region gocv.Mat) (r, g, b float64) {
	if region.Empty() {
		return 0, 0, 0
	}
	data, err := region.DataPtrUint8()
	if err != nil {
		return 0, 0, 0
	}
	channels := region.Channels()
	rows := region.Rows()
	cols := region.Cols()

	var rSum, gSum, bSum float64
	for i := 0; i < rows*cols; i++ {
		idx := i * channels
		bSum += float64(data[idx])
		gSum += float64(data[idx+1])
		rSum += float64(data[idx+2])
	}
	total := float64(rows * cols)
	return rSum / total, gSum / total, bSum / total
}

func AutoReConnect() {
	title := "SCUM  " // 两个空格
	okBtn := "./png/ok.png"
	continueBtn := "./png/continue.png"

	hwnd, err := findWindow(title)
	if err != nil {
		log.Fatal(err)
	}

	// 右上角固定窗口
	screenW, _ := getScreenSize()
	winW, winH := 1920, 1080
	x := screenW - winW
	y := 0
	moveWindow(hwnd, x, y, winW, winH)
	setForeground(hwnd)

	for {
		img, err := captureWindow(hwnd)
		if err != nil {
			log.Println("截图失败:", err)
			time.Sleep(time.Second)
			continue
		}

		// OK 按钮
		if p, found := matchTemplate(img, okBtn); found {
			log.Println("点击 OK 按钮")
			clickAt(p.X+10, p.Y+10)
		}

		// 继续游戏
		if p, found := matchTemplate(img, continueBtn); found {
			log.Println("点击继续游戏")
			clickAt(p.X+10, p.Y+10)
		}

		// 检查聊天框
		chatRect := image.Rect(30, 30, 330, 180)
		chatRegion := img.Region(chatRect)
		r, g, b := meanBGR(chatRegion)
		chatRegion.Close()

		if r+g+b < 30 {
			log.Println("聊天框未打开 → 按 T")
			keyTap(0x54) // T
		} else if b < r*1.4 || b < g*1.4 {
			log.Println("聊天框未蓝色 → 按 Tab")
			for b < r*1.4 || b < g*1.4 {
				keyTap(0x09) // Tab
				time.Sleep(200 * time.Millisecond)
				chatRegion := img.Region(chatRect)
				r, g, b = meanBGR(chatRegion)
				chatRegion.Close()
			}
		}

		img.Close()
		time.Sleep(500 * time.Millisecond)
	}
}
