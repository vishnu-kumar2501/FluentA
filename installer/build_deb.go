package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

func writeTarGz(files map[string][]byte, modes map[string]int64) ([]byte, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for path, data := range files {
		mode := int64(0644)
		if m, ok := modes[path]; ok {
			mode = m
		}
		hdr := &tar.Header{
			Name:     path,
			Mode:     mode,
			Size:     int64(len(data)),
			ModTime:  time.Now(),
			Typeflag: tar.TypeReg,
			Uid:      0,
			Gid:      0,
			Uname:    "root",
			Gname:    "root",
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, err
		}
		if _, err := tw.Write(data); err != nil {
			return nil, err
		}
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeArEntry(w io.Writer, name string, data []byte) error {
	hdr := fmt.Sprintf("%-16s%-12d%-6d%-6d%-8s%-10d`\n",
		name,
		time.Now().Unix(),
		0,
		0,
		"100644",
		len(data),
	)
	if _, err := io.WriteString(w, hdr); err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	if len(data)%2 != 0 {
		if _, err := w.Write([]byte("\n")); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	exeDir, _ := os.Getwd()
	rootDir := exeDir
	if filepath.Base(rootDir) == "installer" {
		rootDir = filepath.Dir(rootDir)
	}

	distLinux := filepath.Join(rootDir, "dist", "linux")
	outDeb := filepath.Join(rootDir, "dist", "fluenta_1.0.0_amd64.deb")

	cliBytes, err := os.ReadFile(filepath.Join(distLinux, "fluenta-cli"))
	if err != nil {
		panic(fmt.Errorf("read fluenta-cli: %w", err))
	}
	deskBytes, err := os.ReadFile(filepath.Join(distLinux, "fluenta-desktop"))
	if err != nil {
		panic(fmt.Errorf("read fluenta-desktop: %w", err))
	}
	iconBytes, err := os.ReadFile(filepath.Join(rootDir, "desktop", "ui", "fluenta-logo.png"))
	if err != nil {
		panic(fmt.Errorf("read icon: %w", err))
	}

	desktopFile := `[Desktop Entry]
Version=1.0
Type=Application
Name=FluentA
Comment=Fast, Private AI Writing & Grammar Assistant
Exec=/usr/bin/fluenta-desktop
Icon=/usr/share/pixmaps/fluenta.png
Terminal=false
Categories=Utility;Office;TextEditor;
StartupNotify=true
StartupWMClass=FluentA
Keywords=grammar;spellcheck;ai;proofread;writing;
`

	controlContent := `Package: fluenta
Version: 1.0.0
Section: utils
Priority: optional
Architecture: amd64
Maintainer: Vishnu Kumar <kumarvishnu610@gmail.com>
Installed-Size: 22000
Depends: libc6
Recommends: xclip, xdotool, wl-clipboard
Description: FluentA — Fast, Private AI Writing & Grammar Assistant
 Native, privacy-first desktop writing assistant powered by modern AI.
 Proofreads grammar, fixes spelling errors, rewrites tone, and polishes
 your writing across any application.
`

	controlFiles := map[string][]byte{
		"./control": []byte(controlContent),
	}
	controlTarGz, err := writeTarGz(controlFiles, map[string]int64{"./control": 0644})
	if err != nil {
		panic(err)
	}

	dataFiles := map[string][]byte{
		"./usr/bin/fluenta":                                  cliBytes,
		"./usr/bin/fluenta-desktop":                          deskBytes,
		"./usr/share/applications/fluenta.desktop":           []byte(desktopFile),
		"./usr/share/pixmaps/fluenta.png":                    iconBytes,
		"./usr/share/icons/hicolor/256x256/apps/fluenta.png": iconBytes,
	}
	dataModes := map[string]int64{
		"./usr/bin/fluenta":                                  0755,
		"./usr/bin/fluenta-desktop":                          0755,
		"./usr/share/applications/fluenta.desktop":           0644,
		"./usr/share/pixmaps/fluenta.png":                    0644,
		"./usr/share/icons/hicolor/256x256/apps/fluenta.png": 0644,
	}
	dataTarGz, err := writeTarGz(dataFiles, dataModes)
	if err != nil {
		panic(err)
	}

	debFile, err := os.Create(outDeb)
	if err != nil {
		panic(err)
	}
	defer debFile.Close()

	if _, err := io.WriteString(debFile, "!<arch>\n"); err != nil {
		panic(err)
	}

	if err := writeArEntry(debFile, "debian-binary", []byte("2.0\n")); err != nil {
		panic(err)
	}
	if err := writeArEntry(debFile, "control.tar.gz", controlTarGz); err != nil {
		panic(err)
	}
	if err := writeArEntry(debFile, "data.tar.gz", dataTarGz); err != nil {
		panic(err)
	}

	fi, _ := debFile.Stat()
	fmt.Printf("Created DEB package: %s (%.2f MB)\n", outDeb, float64(fi.Size())/1024/1024)
}
