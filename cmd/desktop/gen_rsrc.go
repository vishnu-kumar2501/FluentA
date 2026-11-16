//go:build ignore

package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"unicode/utf16"
)

type frame struct {
	size int
	png  []byte
}

func utf16Bytes(s string) []byte {
	u := utf16.Encode([]rune(s + "\x00"))
	b := make([]byte, len(u)*2)
	for i, v := range u {
		binary.LittleEndian.PutUint16(b[i*2:], v)
	}
	return b
}

func pad4(b *bytes.Buffer) {
	for b.Len()%4 != 0 {
		b.WriteByte(0)
	}
}

func writeDirHeader(b *bytes.Buffer, numNamed, numID uint16) {
	_ = binary.Write(b, binary.LittleEndian, uint32(0)) // Characteristics
	_ = binary.Write(b, binary.LittleEndian, uint32(0)) // TimeDateStamp
	_ = binary.Write(b, binary.LittleEndian, uint16(0)) // MajorVersion
	_ = binary.Write(b, binary.LittleEndian, uint16(0)) // MinorVersion
	_ = binary.Write(b, binary.LittleEndian, numNamed)
	_ = binary.Write(b, binary.LittleEndian, numID)
}

func writeDirEntry(b *bytes.Buffer, id, offsetToData uint32, isSubdir bool) {
	if isSubdir {
		offsetToData |= 0x80000000
	}
	_ = binary.Write(b, binary.LittleEndian, id)
	_ = binary.Write(b, binary.LittleEndian, offsetToData)
}

func buildGroupIconDir(frames []frame) []byte {
	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.LittleEndian, uint16(0))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(len(frames)))
	for i, f := range frames {
		var w, h uint8
		if f.size < 256 {
			w = uint8(f.size)
			h = uint8(f.size)
		}
		buf.WriteByte(w)
		buf.WriteByte(h)
		buf.WriteByte(0)
		buf.WriteByte(0)
		_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
		_ = binary.Write(&buf, binary.LittleEndian, uint16(32))
		_ = binary.Write(&buf, binary.LittleEndian, uint32(len(f.png)))
		_ = binary.Write(&buf, binary.LittleEndian, uint16(i+1))
	}
	return buf.Bytes()
}

func buildString(key, val string) []byte {
	var buf bytes.Buffer
	kBytes := utf16Bytes(key)
	vBytes := utf16Bytes(val)
	valWChars := uint16(len(vBytes) / 2)

	_ = binary.Write(&buf, binary.LittleEndian, uint16(0)) // wLength placeholder
	_ = binary.Write(&buf, binary.LittleEndian, valWChars) // wValueLength in WCHARs
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1)) // wType (text)
	buf.Write(kBytes)
	pad4(&buf)
	buf.Write(vBytes)
	pad4(&buf)

	data := buf.Bytes()
	binary.LittleEndian.PutUint16(data[0:2], uint16(len(data)))
	return data
}

func buildVersionResource(props map[string]string) []byte {
	keys := []string{
		"Comments", "CompanyName", "FileDescription", "FileVersion",
		"InternalName", "LegalCopyright", "OriginalFilename", "ProductName", "ProductVersion",
	}
	var strEntries bytes.Buffer
	for _, k := range keys {
		if v, ok := props[k]; ok {
			strEntries.Write(buildString(k, v))
		}
	}

	var strTable bytes.Buffer
	_ = binary.Write(&strTable, binary.LittleEndian, uint16(0))
	_ = binary.Write(&strTable, binary.LittleEndian, uint16(0))
	_ = binary.Write(&strTable, binary.LittleEndian, uint16(1))
	strTable.Write(utf16Bytes("040904b0"))
	pad4(&strTable)
	strTable.Write(strEntries.Bytes())
	stData := strTable.Bytes()
	binary.LittleEndian.PutUint16(stData[0:2], uint16(len(stData)))

	var sfi bytes.Buffer
	_ = binary.Write(&sfi, binary.LittleEndian, uint16(0))
	_ = binary.Write(&sfi, binary.LittleEndian, uint16(0))
	_ = binary.Write(&sfi, binary.LittleEndian, uint16(1))
	sfi.Write(utf16Bytes("StringFileInfo"))
	pad4(&sfi)
	sfi.Write(stData)
	sfiData := sfi.Bytes()
	binary.LittleEndian.PutUint16(sfiData[0:2], uint16(len(sfiData)))

	var vfi bytes.Buffer
	_ = binary.Write(&vfi, binary.LittleEndian, uint16(0))
	_ = binary.Write(&vfi, binary.LittleEndian, uint16(0))
	_ = binary.Write(&vfi, binary.LittleEndian, uint16(1))
	vfi.Write(utf16Bytes("VarFileInfo"))
	pad4(&vfi)

	var transVar bytes.Buffer
	_ = binary.Write(&transVar, binary.LittleEndian, uint16(0))
	_ = binary.Write(&transVar, binary.LittleEndian, uint16(4))
	_ = binary.Write(&transVar, binary.LittleEndian, uint16(0))
	transVar.Write(utf16Bytes("Translation"))
	pad4(&transVar)
	_ = binary.Write(&transVar, binary.LittleEndian, uint16(0x0409))
	_ = binary.Write(&transVar, binary.LittleEndian, uint16(0x04b0))
	tvData := transVar.Bytes()
	binary.LittleEndian.PutUint16(tvData[0:2], uint16(len(tvData)))

	vfi.Write(tvData)
	vfiData := vfi.Bytes()
	binary.LittleEndian.PutUint16(vfiData[0:2], uint16(len(vfiData)))

	var vs bytes.Buffer
	_ = binary.Write(&vs, binary.LittleEndian, uint16(0))
	_ = binary.Write(&vs, binary.LittleEndian, uint16(52))
	_ = binary.Write(&vs, binary.LittleEndian, uint16(0))
	vs.Write(utf16Bytes("VS_VERSION_INFO"))
	pad4(&vs)

	_ = binary.Write(&vs, binary.LittleEndian, uint32(0xFEEF04BD)) // dwSignature
	_ = binary.Write(&vs, binary.LittleEndian, uint32(0x00010000)) // dwStrucVersion
	_ = binary.Write(&vs, binary.LittleEndian, uint32(0x00010000)) // dwFileVersionMS (1.0)
	_ = binary.Write(&vs, binary.LittleEndian, uint32(0x00000000)) // dwFileVersionLS (0.0)
	_ = binary.Write(&vs, binary.LittleEndian, uint32(0x00010000)) // dwProductVersionMS (1.0)
	_ = binary.Write(&vs, binary.LittleEndian, uint32(0x00000000)) // dwProductVersionLS (0.0)
	_ = binary.Write(&vs, binary.LittleEndian, uint32(0x0000003F)) // dwFileFlagsMask
	_ = binary.Write(&vs, binary.LittleEndian, uint32(0x00000000)) // dwFileFlags
	_ = binary.Write(&vs, binary.LittleEndian, uint32(0x00040004)) // dwFileOS = VOS_NT_WINDOWS32
	_ = binary.Write(&vs, binary.LittleEndian, uint32(0x00000001)) // dwFileType = VFT_APP
	_ = binary.Write(&vs, binary.LittleEndian, uint32(0x00000000)) // dwFileSubtype
	_ = binary.Write(&vs, binary.LittleEndian, uint32(0x00000000)) // dwFileDateMS
	_ = binary.Write(&vs, binary.LittleEndian, uint32(0x00000000)) // dwFileDateLS
	pad4(&vs)

	vs.Write(sfiData)
	pad4(&vs)
	vs.Write(vfiData)
	pad4(&vs)

	vsData := vs.Bytes()
	binary.LittleEndian.PutUint16(vsData[0:2], uint16(len(vsData)))
	return vsData
}

func buildManifestResource() []byte {
	manifest := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
  <assemblyIdentity
    version="1.0.0.0"
    processorArchitecture="amd64"
    name="FluentA.Desktop"
    type="win32"
  />
  <description>FluentA Desktop Assistant</description>
  <trustInfo xmlns="urn:schemas-microsoft-com:asm.v3">
    <security>
      <requestedPrivileges>
        <requestedExecutionLevel level="asInvoker" uiAccess="false"/>
      </requestedPrivileges>
    </security>
  </trustInfo>
  <compatibility xmlns="urn:schemas-microsoft-com:compatibility.v1">
    <application>
      <!-- Windows 10 and Windows 11 -->
      <supportedOS Id="{8e0f7a12-bfb3-4fe8-b9a5-48fd50a15a9a}"/>
      <!-- Windows 8.1 -->
      <supportedOS Id="{1f676c76-80e1-4239-95bb-83d0f6d0da78}"/>
      <!-- Windows 8 -->
      <supportedOS Id="{4a2f28e3-53b9-4441-ba9c-d69d4a4a6e38}"/>
      <!-- Windows 7 -->
      <supportedOS Id="{35138b9a-5d96-4fbd-8e2d-a2440225f93a}"/>
    </application>
  </compatibility>
  <application xmlns="urn:schemas-microsoft-com:asm.v3">
    <windowsSettings>
      <dpiAware xmlns="http://schemas.microsoft.com/SMI/2005/WindowsSettings">true/pm</dpiAware>
      <dpiAwareness xmlns="http://schemas.microsoft.com/SMI/2016/WindowsSettings">PerMonitorV2, PerMonitor</dpiAwareness>
    </windowsSettings>
  </application>
</assembly>`
	return []byte(manifest)
}

func readICO(icoPath string) ([]frame, error) {
	b, err := os.ReadFile(icoPath)
	if err != nil {
		return nil, err
	}
	if len(b) < 6 {
		return nil, fmt.Errorf("ico too small")
	}
	count := int(binary.LittleEndian.Uint16(b[4:6]))
	frames := make([]frame, 0, count)
	for i := 0; i < count; i++ {
		ent := b[6+i*16 : 6+(i+1)*16]
		w := int(ent[0])
		if w == 0 {
			w = 256
		}
		bytesInRes := binary.LittleEndian.Uint32(ent[8:12])
		imgOffset := binary.LittleEndian.Uint32(ent[12:16])
		pngData := b[imgOffset : imgOffset+bytesInRes]
		frames = append(frames, frame{size: w, png: pngData})
	}
	return frames, nil
}

func main() {
	exeDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	repoDir := exeDir
	for {
		if _, err := os.Stat(filepath.Join(repoDir, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(repoDir)
		if parent == repoDir {
			panic("could not find repository root containing go.mod")
		}
		repoDir = parent
	}

	icoPath := filepath.Join(repoDir, "desktop", "fluenta.ico")
	frames, err := readICO(icoPath)
	if err != nil {
		panic(err)
	}

	versionBlob := buildVersionResource(map[string]string{
		"Comments":         "FluentA AI Writing & Grammar Assistant",
		"CompanyName":      "FluentA",
		"FileDescription":  "FluentA Desktop Assistant",
		"FileVersion":      "1.0.0.0",
		"InternalName":     "FluentA.exe",
		"LegalCopyright":   "Copyright © 2026 FluentA. All rights reserved.",
		"OriginalFilename": "FluentA.exe",
		"ProductName":      "FluentA",
		"ProductVersion":   "1.0.0.0",
	})
	manifestBlob := buildManifestResource()
	groupIconBlob := buildGroupIconDir(frames)

	nIcons := len(frames)
	totalBlobs := nIcons + 1 + 1 + 1

	type3Off := uint32(48)
	type14Off := type3Off + uint32(16+nIcons*8)
	type16Off := type14Off + uint32(16+1*8)
	type24Off := type16Off + uint32(16+1*8)
	langDirsOff := type24Off + uint32(16+1*8)
	dataEntriesOff := langDirsOff + uint32(totalBlobs*24)
	dataStartOff := dataEntriesOff + uint32(totalBlobs*16)

	var rsrc bytes.Buffer
	writeDirHeader(&rsrc, 0, 4)
	writeDirEntry(&rsrc, 3, type3Off, true)
	writeDirEntry(&rsrc, 14, type14Off, true)
	writeDirEntry(&rsrc, 16, type16Off, true)
	writeDirEntry(&rsrc, 24, type24Off, true)

	writeDirHeader(&rsrc, 0, uint16(nIcons))
	for i := 0; i < nIcons; i++ {
		writeDirEntry(&rsrc, uint32(i+1), langDirsOff+uint32(i*24), true)
	}

	writeDirHeader(&rsrc, 0, 1)
	writeDirEntry(&rsrc, 1, langDirsOff+uint32(nIcons*24), true)

	writeDirHeader(&rsrc, 0, 1)
	writeDirEntry(&rsrc, 1, langDirsOff+uint32((nIcons+1)*24), true)

	writeDirHeader(&rsrc, 0, 1)
	writeDirEntry(&rsrc, 1, langDirsOff+uint32((nIcons+2)*24), true)

	for i := 0; i < totalBlobs; i++ {
		writeDirHeader(&rsrc, 0, 1)
		writeDirEntry(&rsrc, 0x0409, dataEntriesOff+uint32(i*16), false)
	}

	blobs := make([][]byte, 0, totalBlobs)
	for _, f := range frames {
		blobs = append(blobs, f.png)
	}
	blobs = append(blobs, groupIconBlob)
	blobs = append(blobs, versionBlob)
	blobs = append(blobs, manifestBlob)

	var relocOffsets []uint32
	curDataOff := dataStartOff
	for _, blob := range blobs {
		relocOffsets = append(relocOffsets, uint32(rsrc.Len()))
		_ = binary.Write(&rsrc, binary.LittleEndian, curDataOff)
		_ = binary.Write(&rsrc, binary.LittleEndian, uint32(len(blob)))
		_ = binary.Write(&rsrc, binary.LittleEndian, uint32(0))
		_ = binary.Write(&rsrc, binary.LittleEndian, uint32(0))
		curDataOff += uint32(len(blob))
		for curDataOff%4 != 0 {
			curDataOff++
		}
	}

	for _, blob := range blobs {
		rsrc.Write(blob)
		pad4(&rsrc)
	}

	rawData := rsrc.Bytes()
	rawOff := uint32(20 + 40)
	relocOff := rawOff + uint32(len(rawData))
	symOff := relocOff + uint32(len(relocOffsets)*10)

	var out bytes.Buffer
	_ = binary.Write(&out, binary.LittleEndian, uint16(0x8664))
	_ = binary.Write(&out, binary.LittleEndian, uint16(1))
	_ = binary.Write(&out, binary.LittleEndian, uint32(0))
	_ = binary.Write(&out, binary.LittleEndian, symOff)
	_ = binary.Write(&out, binary.LittleEndian, uint32(1))
	_ = binary.Write(&out, binary.LittleEndian, uint16(0))
	_ = binary.Write(&out, binary.LittleEndian, uint16(0x0104))

	secName := [8]byte{'.', 'r', 's', 'r', 'c'}
	out.Write(secName[:])
	_ = binary.Write(&out, binary.LittleEndian, uint32(0))
	_ = binary.Write(&out, binary.LittleEndian, uint32(0))
	_ = binary.Write(&out, binary.LittleEndian, uint32(len(rawData)))
	_ = binary.Write(&out, binary.LittleEndian, rawOff)
	_ = binary.Write(&out, binary.LittleEndian, relocOff)
	_ = binary.Write(&out, binary.LittleEndian, uint32(0))
	_ = binary.Write(&out, binary.LittleEndian, uint16(len(relocOffsets)))
	_ = binary.Write(&out, binary.LittleEndian, uint16(0))
	_ = binary.Write(&out, binary.LittleEndian, uint32(0x40000040))

	out.Write(rawData)

	for _, rvaOff := range relocOffsets {
		_ = binary.Write(&out, binary.LittleEndian, rvaOff)
		_ = binary.Write(&out, binary.LittleEndian, uint32(0))
		_ = binary.Write(&out, binary.LittleEndian, uint16(3))
	}

	out.Write(secName[:])
	_ = binary.Write(&out, binary.LittleEndian, uint32(0))
	_ = binary.Write(&out, binary.LittleEndian, uint16(1))
	_ = binary.Write(&out, binary.LittleEndian, uint16(0))
	out.WriteByte(3)
	out.WriteByte(0)
	_ = binary.Write(&out, binary.LittleEndian, uint32(4))

	sysoPath := filepath.Join(repoDir, "cmd", "desktop", "rsrc_windows_amd64.syso")
	if err := os.WriteFile(sysoPath, out.Bytes(), 0644); err != nil {
		panic(err)
	}
	fmt.Printf("Generated %s (%d bytes)\n", sysoPath, out.Len())
}

