# FluentA Privacy Policy & User Transparency

**Last Updated:** November 2026  
**App Version:** 1.0.0+  
**Website / Repository:** https://github.com/FluentA

At **FluentA**, privacy is not an afterthought or an optional toggle—it is the foundational architecture of the application. FluentA is designed to be a private, transparent, and user-controlled desktop writing assistant.

This document clearly explains what data FluentA touches, what data it never collects, and how you retain total ownership over your writing.

---

## 1. Core Privacy Commitments

1. **Zero Keylogging:** FluentA does **not** record keystrokes. It only inspects text when you explicitly press the global hotkey (`Ctrl+Alt+G`) or when you pause typing in a supported text field.
2. **Zero Data Monetization:** We never sell, rent, monetize, or broker your personal information, text, or writing habits.
3. **Zero Model Training:** Your writing is **never** used to train, retrain, or fine-tune public AI models.
4. **Air-Gapped Offline Capability:** If you choose the **Local Ollama** engine, 100% of grammar and style inference occurs on your own computer. Zero bytes of text ever leave your device.
5. **Full User Control:** You can disable the background watcher at any time, exclude sensitive applications (e.g., password managers, banking apps, terminals), and purge your history with one click.

---

## 2. What Data FluentA Processes (And Why)

| Data Type | Purpose | Where It Is Stored |
| :--- | :--- | :--- |
| **Selected / Input Text** | Sent to the AI engine solely to detect spelling, grammar, punctuation, and style edits. | Kept temporarily in memory during inference. If history is enabled, stored strictly on your local disk at `%APPDATA%\FluentA\history.json`. |
| **Active Application Name** | Used strictly to compare against your **Excluded Apps** privacy blocklist (e.g., `1password.exe`, `keepass.exe`). | Checked in memory only. Never logged or transmitted anywhere. |
| **User Preferences** | Stores your chosen shortcut hotkey, default language, theme, and AI engine preferences. | Stored locally at `%APPDATA%\FluentA\settings.json`. |
| **Local Application Logs** | Diagnostic error logs (e.g., network failure, hotkey conflict) to help you troubleshoot issues. | Stored locally at `%APPDATA%\FluentA\fluenta.log`. |

---

## 3. What FluentA NEVER Collects

FluentA is strictly a desktop utility, not a tracking platform:

- ❌ **No Keyloggers:** We do not hook global keyboard scan codes to record passwords or general typing.
- ❌ **No Password Fields:** Windows UI Automation classifies password and PIN input fields securely. FluentA explicitly ignores all password controls.
- ❌ **No Analytics or Telemetry:** No Google Analytics, no telemetry beacons, no user fingerprinting, and no background crash reporting pings to third parties.
- ❌ **No Identity or Account Tracking:** You do not need to create a FluentA account, sign in, or provide an email address.
- ❌ **No File System Scanning:** FluentA never scans, reads, or indexes your personal files, documents, photos, or browser history.

---

## 4. How AI Inference Handles Your Text

FluentA supports two distinct AI modes:

### Option A: Local Ollama (100% Air-Gapped & Offline)
- When using **Local Ollama** (default model: `qwen3.5:4b` at `http://127.0.0.1:11434`), your text **never leaves your PC**.
- All natural language processing executes entirely on your CPU / GPU.
- Ideal for legal professionals, medical staff, software developers working on proprietary code, and anyone with strict data sovereignty requirements.

### Option B: Cloud AI (Fast Inference via Encrypted HTTPS)
- When using **Cloud AI** (such as Groq or an OpenAI-compatible API endpoint), your text is transmitted over an encrypted **TLS 1.3** connection.
- Text is processed ephemerally in server RAM to generate the grammar corrections and is immediately discarded.
- Under enterprise API agreements (such as Groq and OpenAI API), data sent via API endpoints is **not used for model training**.
- You can provide your own private API key or connect to your own self-hosted reverse proxy.

---

## 5. Security & Prompt Injection Defense

To prevent malicious prompts or instruction hijacking:
- All text sent to the AI engine is wrapped in strict passive boundary tags (`<text>...</text>`).
- System instructions enforce that text inside the boundaries is treated strictly as **passive content to proofread**, never instructions to execute.
- Even if your text contains commands, questions, or prompts (e.g., *"Write an essay"*, *"Ignore previous rules"*), FluentA will never follow them—it strictly corrects the grammar and spelling.

---

## 6. Your Privacy Controls

You have complete control over how FluentA operates:

1. **Excluded Applications (Blocklist):**
   - In FluentA **Preferences → Excluded Apps**, you can list executable names (e.g., `1password.exe`, `keepass.exe`, `cmd.exe`, `slack.exe`). FluentA will immediately deactivate itself whenever any of these apps are focused.
2. **Toggle Background Watcher:**
   - Prefer on-demand checks only? Turn off the **Background UIA watcher** in Preferences. FluentA will only check text when you explicitly highlight text and press `Ctrl+Alt+G`.
3. **Clear History:**
   - In the **History** tab, click **Clear all** to immediately wipe all saved diffs from your hard drive.
4. **Complete Data Removal:**
   - Uninstalling FluentA and deleting `%APPDATA%\FluentA` permanently removes every configuration file, log, and history record from your machine.

---

## 7. Contact & Open Source Transparency

FluentA believes in verifiable privacy:
- The entire application source code, build scripts, and prompt definitions are open for inspection.
- If you have questions or concerns about data security, please open an issue on our GitHub repository.
