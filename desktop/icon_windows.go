//go:build windows

package desktop

import (
	_ "embed"
	"encoding/binary"
	"unsafe"
)

//go:embed fluenta.ico
var embeddedFluentAICO []byte

var (
	procCreateIconFromResourceEx = user32.NewProc("CreateIconFromResourceEx")
	procDestroyIcon              = user32.NewProc("DestroyIcon")
	procDrawIconEx               = user32.NewProc("DrawIconEx")
)

const (
	wmSetIcon   = 0x0080
	iconSmallID = 0
	iconBigID   = 1
	diNormal    = 0x0003
)

// loadFluentAIcons decodes crisp 32x32 (large) and 16x16 (small) per-pixel alpha HICONs
// from the embedded FluentA ICO, falling back to PE resources / idiApplication if needed.
func loadFluentAIcons(hInst uintptr) (largeIcon, smallIcon uintptr) {
	if data := findICOFrame(embeddedFluentAICO, 32); len(data) > 0 {
		h, _, _ := procCreateIconFromResourceEx.Call(
			uintptr(unsafe.Pointer(&data[0])),
			uintptr(len(data)),
			1,
			0x00030000,
			32,
			32,
			0,
		)
		largeIcon = h
	}
	if data := findICOFrame(embeddedFluentAICO, 16); len(data) > 0 {
		h, _, _ := procCreateIconFromResourceEx.Call(
			uintptr(unsafe.Pointer(&data[0])),
			uintptr(len(data)),
			1,
			0x00030000,
			16,
			16,
			0,
		)
		smallIcon = h
	}
	if largeIcon == 0 {
		if h, _, _ := procLoadIconW.Call(hInst, 1); h != 0 {
			largeIcon = h
		} else {
			largeIcon, _, _ = procLoadIconW.Call(0, idiApplication)
		}
	}
	if smallIcon == 0 {
		smallIcon = largeIcon
	}
	return largeIcon, smallIcon
}

func destroyIcon(hIcon uintptr) {
	if hIcon != 0 {
		_, _, _ = procDestroyIcon.Call(hIcon)
	}
}

func drawIcon16(hdc, hIcon uintptr, x, y int32) {
	if hdc == 0 || hIcon == 0 {
		return
	}
	_, _, _ = procDrawIconEx.Call(hdc, uintptr(x), uintptr(y), hIcon, 16, 16, 0, 0, diNormal)
}

func findICOFrame(ico []byte, targetSize int) []byte {
	if len(ico) < 6 {
		return nil
	}
	if binary.LittleEndian.Uint16(ico[0:2]) != 0 || binary.LittleEndian.Uint16(ico[2:4]) != 1 {
		return nil
	}
	count := int(binary.LittleEndian.Uint16(ico[4:6]))
	if count <= 0 || len(ico) < 6+count*16 {
		return nil
	}

	var best []byte
	bestDiff := 1 << 30
	for i := 0; i < count; i++ {
		entry := ico[6+i*16 : 6+(i+1)*16]
		w := int(entry[0])
		if w == 0 {
			w = 256
		}
		sz := int(binary.LittleEndian.Uint32(entry[8:12]))
		off := int(binary.LittleEndian.Uint32(entry[12:16]))
		if off < 0 || sz <= 0 || off+sz > len(ico) {
			continue
		}
		diff := w - targetSize
		if diff < 0 {
			diff = -diff
		}
		if diff < bestDiff {
			bestDiff = diff
			best = ico[off : off+sz]
		}
	}
	return best
}

