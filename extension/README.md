# FluentA Browser Extension (Manifest V3)

> **High-speed, privacy-first AI writing & grammar assistant for Google Chrome, Microsoft Edge, Brave, and all Chromium browsers.**

---

## Features

- **Floating Selection Trigger**: Highlight any text on any website (Gmail, Notion, Slack, Twitter/X, Reddit, Google Docs, ChatGPT) and an unobtrusive FluentA pill appears.
- **In-Place Universal Replacement**: 1-click text replacement directly inside `input`, `textarea`, and `contenteditable` editors with full `Ctrl+Z` undo history.
- **Word-Level Diff View**: Visual comparison highlighting insertions (green) and deletions (red strikethrough).
- **Multiple Writing Modes**:
  - **Fix Grammar**: Correct spelling, punctuation, and capitalization.
  - **Enhance Flow**: Improve clarity, vocabulary, and phrasing.
  - **Executive Formal**: Polish into executive workplace tone.
  - **Make Concise**: Trim wordiness and keep it punchy.
  - **Explain Changes**: Detailed grammatical breakdown.
- **Zero Configuration Required**: Uses the built-in serverless Cloudflare Edge Gateway by default.
- **100% Offline Ollama Support**: Switch provider to local Ollama (`http://127.0.0.1:11434`) for total private, offline proofreading.
- **Zero Keylogging**: Never logs keystrokes; only activates upon explicit user selection or hotkey trigger.

---

## Installation (Developer / Unpacked Mode)

1. Open your browser:
   - **Google Chrome / Brave**: Navigate to `chrome://extensions`
   - **Microsoft Edge**: Navigate to `edge://extensions`
2. Toggle on **"Developer mode"** in the top right corner.
3. Click **"Load unpacked"**.
4. Select the `extension` folder located at:
   ```
   D:\free lancing works\Grammar\GrammarDesktop\extension
   ```
5. Pin **FluentA** to your browser toolbar for quick access!

---

## Keyboard Shortcuts

- **Proofread Selection**: <kbd>Alt</kbd> + <kbd>G</kbd>
- **Replace Text**: Press <kbd>Enter</kbd> inside the modal
- **Close Modal**: Press <kbd>Esc</kbd>
- *To customize shortcuts, navigate to `chrome://extensions/shortcuts`.*

---

## Packaging for Chrome Web Store / Edge Add-ons

To create a clean release `.zip` for publishing to the Chrome Web Store Developer Dashboard:
```powershell
Compress-Archive -Path .\extension\* -DestinationPath .\dist\fluenta-browser-extension.zip -Force
```
Upload `dist\fluenta-browser-extension.zip` directly to the [Chrome Web Store Developer Dashboard](https://chrome.google.com/webstore/devconsole).
