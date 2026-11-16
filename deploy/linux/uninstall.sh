#!/usr/bin/env bash
set -e

echo "=== Uninstalling FluentA ==="

# Check permissions
if [ "$EUID" -ne 0 ]; then
  echo "Please run as root or with sudo: sudo ./uninstall.sh (or sudo fluenta-uninstall)"
  exit 1
fi

# 1. Terminate running background or desktop instances
pkill -f fluenta-desktop 2>/dev/null || true
pkill -f fluenta-cli 2>/dev/null || true
echo "✓ Stopped running processes"

# 2. Remove binaries from /usr/bin and /usr/local/bin
rm -f /usr/bin/fluenta /usr/bin/fluenta-desktop /usr/bin/fluenta-uninstall
rm -f /usr/local/bin/fluenta /usr/local/bin/fluenta-desktop /usr/local/bin/fluenta-uninstall
echo "✓ Removed binaries from /usr/bin and /usr/local/bin"

# 3. Remove desktop launcher and application icons
rm -f /usr/share/applications/fluenta.desktop
rm -f /usr/share/pixmaps/fluenta.png
rm -f /usr/share/icons/hicolor/256x256/apps/fluenta.png
echo "✓ Removed desktop launcher and icons"

# 4. Refresh desktop and icon caches
if command -v update-desktop-database >/dev/null 2>&1; then
  update-desktop-database /usr/share/applications || true
fi
if command -v gtk-update-icon-cache >/dev/null 2>&1; then
  gtk-update-icon-cache -f -t /usr/share/icons/hicolor >/dev/null 2>&1 || true
fi

echo ""
echo "=========================================================="
echo " FluentA has been completely uninstalled from this system."
echo " Note: User configuration in ~/.config/FluentA was preserved."
echo " (To remove user data: rm -rf ~/.config/FluentA)"
echo "=========================================================="
