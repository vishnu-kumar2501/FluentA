#!/usr/bin/env bash
set -e

echo "=== Installing FluentA for Linux ==="

# Check permissions
if [ "$EUID" -ne 0 ]; then
  echo "Please run as root or with sudo: sudo ./install.sh"
  exit 1
fi

BIN_DIR="/usr/local/bin"
APP_DIR="/usr/share/applications"
ICON_DIR="/usr/share/pixmaps"

# 1. Install binaries to both /usr/bin and /usr/local/bin
install -m 755 fluenta-cli /usr/bin/fluenta
install -m 755 fluenta-desktop /usr/bin/fluenta-desktop
install -m 755 fluenta-cli /usr/local/bin/fluenta
install -m 755 fluenta-desktop /usr/local/bin/fluenta-desktop
if [ -f "uninstall.sh" ]; then
  install -m 755 uninstall.sh /usr/bin/fluenta-uninstall
  install -m 755 uninstall.sh /usr/local/bin/fluenta-uninstall
fi
echo "✓ Installed binaries to /usr/bin and /usr/local/bin"

# 2. Install desktop shortcut
install -m 644 fluenta.desktop "$APP_DIR/fluenta.desktop"
echo "✓ Installed application launcher to $APP_DIR/fluenta.desktop"

# 3. Install icon to pixmaps and hicolor icon themes
if [ -f "fluenta-logo.png" ]; then
  install -m 644 fluenta-logo.png "$ICON_DIR/fluenta.png"
  mkdir -p /usr/share/icons/hicolor/256x256/apps/
  install -m 644 fluenta-logo.png /usr/share/icons/hicolor/256x256/apps/fluenta.png
  echo "✓ Installed application icon to $ICON_DIR/fluenta.png and hicolor theme"
fi

# 4. Update desktop and icon databases
if command -v update-desktop-database >/dev/null 2>&1; then
  update-desktop-database "$APP_DIR" || true
fi
if command -v gtk-update-icon-cache >/dev/null 2>&1; then
  gtk-update-icon-cache -f -t /usr/share/icons/hicolor >/dev/null 2>&1 || true
fi

echo ""
echo "=========================================================="
echo " FluentA installed successfully!"
echo " - Run GUI Desktop App:   fluenta-desktop"
echo " - Run CLI Proofreader:   fluenta \"your text here\""
echo " - Active Selection Fix:  fluenta --selection --replace"
echo " - To Uninstall:          sudo fluenta-uninstall (or sudo ./uninstall.sh)"
echo "=========================================================="
