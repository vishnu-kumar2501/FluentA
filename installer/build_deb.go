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

type tarItem struct {
	name  string
	mode  int64
	isDir bool
	data  []byte
}

func writeTarGzEntries(items []tarItem) ([]byte, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	now := time.Now()

	for _, it := range items {
		if it.isDir {
			hdr := &tar.Header{
				Name:     it.name,
				Mode:     0755,
				Typeflag: tar.TypeDir,
				ModTime:  now,
				Uid:      0,
				Gid:      0,
				Uname:    "root",
				Gname:    "root",
			}
			if err := tw.WriteHeader(hdr); err != nil {
				return nil, err
			}
		} else {
			hdr := &tar.Header{
				Name:     it.name,
				Mode:     it.mode,
				Size:     int64(len(it.data)),
				Typeflag: tar.TypeReg,
				ModTime:  now,
				Uid:      0,
				Gid:      0,
				Uname:    "root",
				Gname:    "root",
			}
			if err := tw.WriteHeader(hdr); err != nil {
				return nil, err
			}
			if _, err := tw.Write(it.data); err != nil {
				return nil, err
			}
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
	if _, err := os.Stat(filepath.Join(rootDir, "GrammarDesktop")); err == nil {
		rootDir = filepath.Join(rootDir, "GrammarDesktop")
	} else if filepath.Base(rootDir) == "installer" {
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

	// 1. control.tar.gz with directory entries
	controlItems := []tarItem{
		{name: "./", isDir: true},
		{name: "./control", mode: 0644, data: []byte(controlContent)},
	}
	controlTarGz, err := writeTarGzEntries(controlItems)
	if err != nil {
		panic(err)
	}

	// 2. data.tar.gz with complete directory tree
	dataItems := []tarItem{
		{name: "./", isDir: true},
		{name: "./usr/", isDir: true},
		{name: "./usr/bin/", isDir: true},
		{name: "./usr/bin/fluenta", mode: 0755, data: cliBytes},
		{name: "./usr/bin/fluenta-desktop", mode: 0755, data: deskBytes},
		{name: "./usr/share/", isDir: true},
		{name: "./usr/share/applications/", isDir: true},
		{name: "./usr/share/applications/fluenta.desktop", mode: 0644, data: []byte(desktopFile)},
		{name: "./usr/share/pixmaps/", isDir: true},
		{name: "./usr/share/pixmaps/fluenta.png", mode: 0644, data: iconBytes},
		{name: "./usr/share/icons/", isDir: true},
		{name: "./usr/share/icons/hicolor/", isDir: true},
		{name: "./usr/share/icons/hicolor/256x256/", isDir: true},
		{name: "./usr/share/icons/hicolor/256x256/apps/", isDir: true},
		{name: "./usr/share/icons/hicolor/256x256/apps/fluenta.png", mode: 0644, data: iconBytes},
	}
	dataTarGz, err := writeTarGzEntries(dataItems)
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
	fmt.Printf("Created standard DEB package: %s (%.2f MB)\n", outDeb, float64(fi.Size())/1024/1024)
}
