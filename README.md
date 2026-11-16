# FluentA — AI Writing & Grammar Assistant for Windows

[![Download FluentA](https://img.shields.io/badge/Download-FluentA%20v1.0.0%20(Windows)-6366F1?style=for-the-badge&logo=windows&logoColor=white)](https://github.com/vishnu-kumar2501/FluentA/releases/latest/download/FluentASetup.exe)
[![GitHub release](https://img.shields.io/github/v/release/vishnu-kumar2501/FluentA?style=for-the-badge&color=10B981)](https://github.com/vishnu-kumar2501/FluentA/releases)
[![Website](https://img.shields.io/badge/Website-Live%20Demo-06B6D4?style=for-the-badge)](https://vishnu-kumar2501.github.io/FluentA/)

**FluentA** is a fast, native, privacy-first desktop writing assistant for Windows. Powered by modern LLMs, it proofreads grammar, fixes spelling errors, rewrites tone, and polishes your writing across **any Windows app** (Word, Slack, Discord, Chrome, VS Code, Notepad) using a single global shortcut.

---

## ✨ Features

- ⚡ **Works Everywhere in Windows:** Highlight text in any application and press <kbd>Ctrl</kbd>+<kbd>Alt</kbd>+<kbd>G</kbd> to inspect and replace text in-place.
- 🚀 **Dual AI Architecture:**
  - **Cloud AI (Default):** Sub-second grammar checking powered by ultra-fast inference through a secure, rate-limited proxy. No setup or GPU required.
  - **Local Ollama (100% Offline):** Run entirely on your local CPU/GPU using `qwen3.5:4b`. Zero bytes of text ever leave your PC.
- 🛡️ **Privacy by Architecture:**
  - **Zero Keylogging:** Never intercepts or logs keystrokes. Only checks active fields upon user action.
  - **Zero Model Training:** Your writing is never used to train or fine-tune public AI models.
  - **Excluded Apps Blocklist:** Automatically deactivates over sensitive apps (e.g., password managers like 1Password or KeePass).
- 🎨 **Multiple Writing Modes:**
  - **Fix:** Instant spelling, grammar, and punctuation correction.
  - **Enhance:** Elevate clarity, flow, and natural phrasing.
  - **Formal:** Rewrite text for professional executive emails.
  - **Concise:** Tighten and shorten text without losing meaning.
  - **Explain:** Detailed breakdown of why each edit was suggested.
- 🔍 **Writing Insights & Tone Analysis:** Instant score (0–100), tone detection, and vocabulary improvement recommendations.
- 📌 **Draggable Floating Badge:** An unobtrusive status pill that snaps to your active text field and can be dragged anywhere on screen.

---

## 📥 Download & Installation

1. Download the latest **[FluentASetup.exe](https://github.com/vishnu-kumar2501/FluentA/releases/latest/download/FluentASetup.exe)** from the Releases page.
2. Run `FluentASetup.exe` to install.
   > **Note on Windows SmartScreen:** As this is a new open-source binary, Windows Defender may show a blue prompt on first launch. Click **"More info" → "Run anyway"**.
3. FluentA will launch minimized to your system tray. Press <kbd>Ctrl</kbd>+<kbd>Alt</kbd>+<kbd>G</kbd> to start writing!

---

## 🔒 Privacy & Data Transparency

For full details on our data protection commitments, read our **[Privacy Policy](PRIVACY.md)**.

---

## 🛠️ Building from Source

### Prerequisites
- [Go 1.22+](https://go.dev/)
- [Inno Setup 6 or 7](https://jrsoftware.org/isdl.php) (for building the installer)
- Microsoft Edge WebView2 Runtime (pre-installed on Windows 10 & 11)

### Build Steps
```powershell
# 1. Clone the repository
git clone https://github.com/vishnu-kumar2501/FluentA.git
cd FluentA

# 2. Build the release binary and installer
powershell -ExecutionPolicy Bypass -File .\installer\build_installer.ps1
```

The output installer will be created in `dist\FluentASetup.exe`.

---

## 📄 License
This project is licensed under the MIT License.
