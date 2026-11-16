/**
 * FluentA Content Script
 * Injected into web pages to detect selections, show floating UI, and perform in-place replacements.
 * Isolated completely inside a Shadow DOM to prevent CSS leaks.
 */

(function () {
  if (window.__fluentaInitialized) return;
  window.__fluentaInitialized = true;

  // 1. Create Isolated Shadow Host
  const host = document.createElement("div");
  host.id = "fluenta-extension-host";
  document.documentElement.appendChild(host);
  const shadow = host.attachShadow({ mode: "open" });

  // 2. Inject FluentA Styles into Shadow Root
  const style = document.createElement("style");
  style.textContent = `
    * {
      box-sizing: border-box;
      margin: 0;
      padding: 0;
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
    }

    /* Floating Badge Trigger */
    .fluenta-badge {
      position: absolute;
      display: flex;
      align-items: center;
      gap: 6px;
      padding: 5px 10px;
      background: linear-gradient(135deg, #1e1b4b 0%, #0f172a 100%);
      border: 1px solid rgba(99, 102, 241, 0.4);
      border-radius: 20px;
      color: #ffffff;
      font-size: 12px;
      font-weight: 600;
      cursor: pointer;
      box-shadow: 0 4px 14px rgba(0, 0, 0, 0.4), 0 0 10px rgba(99, 102, 241, 0.3);
      z-index: 2147483647;
      user-select: none;
      transition: all 0.15s cubic-bezier(0.16, 1, 0.3, 1);
      transform: translateY(0);
      opacity: 0;
      pointer-events: none;
    }

    .fluenta-badge.visible {
      opacity: 1;
      pointer-events: auto;
      transform: translateY(-2px);
    }

    .fluenta-badge:hover {
      background: linear-gradient(135deg, #312e81 0%, #1e293b 100%);
      border-color: rgba(99, 102, 241, 0.8);
      box-shadow: 0 6px 20px rgba(99, 102, 241, 0.5);
    }

    .fluenta-badge-icon {
      width: 14px;
      height: 14px;
      border-radius: 3px;
    }

    /* FluentA In-Page Modal */
    .fluenta-modal {
      position: fixed;
      width: 440px;
      max-width: 90vw;
      background: #0d121d;
      border: 1px solid rgba(255, 255, 255, 0.12);
      border-radius: 12px;
      box-shadow: 0 20px 45px rgba(0, 0, 0, 0.75), 0 0 0 1px rgba(99, 102, 241, 0.2);
      color: #e2e8f0;
      z-index: 2147483647;
      display: none;
      flex-direction: column;
      overflow: hidden;
      font-size: 13px;
      line-height: 1.5;
      animation: fluentaFadeIn 0.2s cubic-bezier(0.16, 1, 0.3, 1);
    }

    @keyframes fluentaFadeIn {
      from { opacity: 0; transform: scale(0.96) translateY(-4px); }
      to { opacity: 1; transform: scale(1) translateY(0); }
    }

    .fluenta-modal.visible {
      display: flex;
    }

    /* Modal Header */
    .fluenta-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 10px 14px;
      background: rgba(15, 23, 42, 0.8);
      border-bottom: 1px solid rgba(255, 255, 255, 0.08);
    }

    .fluenta-brand {
      display: flex;
      align-items: center;
      gap: 7px;
      font-weight: 700;
      font-size: 13px;
      color: #ffffff;
      letter-spacing: -0.01em;
    }

    .fluenta-brand-icon {
      width: 16px;
      height: 16px;
      border-radius: 3px;
    }

    .fluenta-close-btn {
      background: transparent;
      border: none;
      color: #94a3b8;
      cursor: pointer;
      font-size: 16px;
      line-height: 1;
      padding: 4px;
      border-radius: 4px;
      display: flex;
      align-items: center;
      justify-content: center;
    }

    .fluenta-close-btn:hover {
      color: #ffffff;
      background: rgba(255, 255, 255, 0.1);
    }

    /* Mode Tabs */
    .fluenta-modes {
      display: flex;
      padding: 8px 12px;
      gap: 4px;
      background: rgba(11, 15, 25, 0.5);
      border-bottom: 1px solid rgba(255, 255, 255, 0.06);
      overflow-x: auto;
    }

    .fluenta-mode-btn {
      background: transparent;
      border: 1px solid transparent;
      color: #94a3b8;
      font-size: 11px;
      font-weight: 600;
      padding: 4px 8px;
      border-radius: 6px;
      cursor: pointer;
      transition: all 0.12s ease;
      white-space: nowrap;
    }

    .fluenta-mode-btn:hover {
      color: #ffffff;
      background: rgba(255, 255, 255, 0.05);
    }

    .fluenta-mode-btn.active {
      background: rgba(99, 102, 241, 0.2);
      border-color: rgba(99, 102, 241, 0.4);
      color: #a5b4fc;
    }

    /* Output Body */
    .fluenta-body {
      padding: 14px;
      min-height: 100px;
      max-height: 240px;
      overflow-y: auto;
      font-size: 13.5px;
      line-height: 1.6;
      color: #f1f5f9;
      background: rgba(13, 18, 29, 0.6);
      word-break: break-word;
    }

    .fluenta-loading {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 8px;
      color: #94a3b8;
      height: 80px;
    }

    .fluenta-spinner {
      width: 16px;
      height: 16px;
      border: 2px solid rgba(99, 102, 241, 0.2);
      border-top-color: #6366f1;
      border-radius: 50%;
      animation: fluentaSpin 0.7s linear infinite;
    }

    @keyframes fluentaSpin {
      to { transform: rotate(360deg); }
    }

    /* Word Diff Highlighting */
    .fluenta-diff-del {
      background: rgba(239, 68, 68, 0.2);
      color: #fca5a5;
      text-decoration: line-through;
      padding: 1px 4px;
      border-radius: 3px;
      margin-right: 2px;
    }

    .fluenta-diff-add {
      background: rgba(16, 185, 129, 0.2);
      color: #6ee7b7;
      font-weight: 600;
      padding: 1px 4px;
      border-radius: 3px;
    }

    /* Meta Bar */
    .fluenta-meta {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 6px 14px;
      background: rgba(11, 15, 25, 0.7);
      border-top: 1px solid rgba(255, 255, 255, 0.05);
      font-size: 11px;
      color: #64748b;
    }

    .fluenta-latency {
      color: #34d399;
      font-family: monospace;
    }

    /* Action Footer */
    .fluenta-footer {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 10px 14px;
      background: #090d15;
      border-top: 1px solid rgba(255, 255, 255, 0.08);
    }

    .fluenta-actions {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-left: auto;
    }

    .fluenta-btn {
      padding: 6px 12px;
      border-radius: 6px;
      font-size: 12px;
      font-weight: 600;
      cursor: pointer;
      display: inline-flex;
      align-items: center;
      gap: 5px;
      transition: all 0.15s ease;
      border: 1px solid transparent;
    }

    .fluenta-btn-secondary {
      background: rgba(255, 255, 255, 0.06);
      border-color: rgba(255, 255, 255, 0.1);
      color: #cbd5e1;
    }

    .fluenta-btn-secondary:hover {
      background: rgba(255, 255, 255, 0.12);
      color: #ffffff;
    }

    .fluenta-btn-primary {
      background: #6366f1;
      color: #ffffff;
      box-shadow: 0 2px 8px rgba(99, 102, 241, 0.35);
    }

    .fluenta-btn-primary:hover {
      background: #4f46e5;
      box-shadow: 0 4px 12px rgba(99, 102, 241, 0.5);
    }
  `;
  shadow.appendChild(style);

  // 3. Create DOM Elements inside Shadow
  const badge = document.createElement("div");
  badge.className = "fluenta-badge";
  badge.innerHTML = `
    <svg class="fluenta-badge-icon" viewBox="0 0 24 24" fill="none" stroke="#818cf8" stroke-width="2.5">
      <path d="M12 2v20M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/>
    </svg>
    <span>FluentA</span>
  `;
  shadow.appendChild(badge);

  const modal = document.createElement("div");
  modal.className = "fluenta-modal";
  modal.innerHTML = `
    <div class="fluenta-header">
      <div class="fluenta-brand">
        <svg class="fluenta-brand-icon" viewBox="0 0 24 24" fill="none" stroke="#818cf8" stroke-width="2.5">
          <path d="M12 2v20M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/>
        </svg>
        <span>FluentA AI</span>
      </div>
      <button class="fluenta-close-btn" id="fluenta-close" title="Close (Esc)">✕</button>
    </div>

    <div class="fluenta-modes">
      <button class="fluenta-mode-btn active" data-mode="fix">Fix Grammar</button>
      <button class="fluenta-mode-btn" data-mode="enhance">Enhance Flow</button>
      <button class="fluenta-mode-btn" data-mode="formal">Formal</button>
      <button class="fluenta-mode-btn" data-mode="concise">Concise</button>
      <button class="fluenta-mode-btn" data-mode="draft">Draft Email</button>
      <button class="fluenta-mode-btn" data-mode="reply">Smart Reply</button>
      <button class="fluenta-mode-btn" data-mode="explain">Explain</button>
    </div>

    <div class="fluenta-body" id="fluenta-body">
      <div class="fluenta-loading">
        <div class="fluenta-spinner"></div>
        <span>Proofreading with AI...</span>
      </div>
    </div>

    <div class="fluenta-meta">
      <span id="fluenta-stat">Original: 0 chars</span>
      <span class="fluenta-latency" id="fluenta-latency">⚡ sub-second</span>
    </div>

    <div class="fluenta-footer">
      <span style="font-size:11px; color:#64748b;">Press <strong>Enter</strong> to replace</span>
      <div class="fluenta-actions">
        <button class="fluenta-btn fluenta-btn-secondary" id="fluenta-copy">Copy</button>
        <button class="fluenta-btn fluenta-btn-primary" id="fluenta-apply">Replace in Page</button>
      </div>
    </div>
  `;
  shadow.appendChild(modal);

  // 4. State Variables
  let currentSelectionText = "";
  let currentResultText = "";
  let activeElementTarget = null;
  let activeRange = null;
  let currentMode = "fix";
  let settings = null;

  // Load initial settings
  chrome.runtime.sendMessage({ action: "GET_SETTINGS" }, (s) => {
    settings = s;
  });

  // 5. Selection Handling
  document.addEventListener("mouseup", handleSelectionChange);
  document.addEventListener("keyup", (e) => {
    if (e.key === "Shift" || e.key.startsWith("Arrow")) {
      handleSelectionChange();
    }
  });

  function isDomainExcluded() {
    if (!settings || !settings.excludedDomains) return false;
    const host = window.location.hostname.toLowerCase();
    return settings.excludedDomains.some(d => host.includes(d.toLowerCase()));
  }

  function handleSelectionChange() {
    // If modal is currently open, don't auto-dismiss or update
    if (modal.classList.contains("visible")) return;

    if (isDomainExcluded()) {
      badge.classList.remove("visible");
      return;
    }

    const sel = window.getSelection();
    const text = sel ? sel.toString().trim() : "";

    // Ignore tiny or empty selections
    if (!text || text.length < 2) {
      badge.classList.remove("visible");
      return;
    }

    // Check if selection is inside a password field
    const activeEl = document.activeElement;
    if (activeEl && activeEl.type === "password") {
      badge.classList.remove("visible");
      return;
    }

    currentSelectionText = text;
    activeElementTarget = activeEl;

    try {
      activeRange = sel.rangeCount > 0 ? sel.getRangeAt(0).cloneRange() : null;
    } catch (_) {
      activeRange = null;
    }

    // Position floating badge
    if (settings && settings.autoFloatingBadge === false) {
      return; // Hotkey only mode
    }

    positionBadge(sel);
    badge.classList.add("visible");
  }

  function positionBadge(selection) {
    if (!selection || selection.rangeCount === 0) return;
    try {
      const rect = selection.getRangeAt(0).getBoundingClientRect();
      if (rect.width === 0 && rect.height === 0) return;

      const scrollX = window.scrollX || window.pageXOffset;
      const scrollY = window.scrollY || window.pageYOffset;

      let top = rect.bottom + scrollY + 8;
      let left = rect.left + scrollX;

      if (left + 120 > window.innerWidth + scrollX) {
        left = window.innerWidth + scrollX - 130;
      }
      if (left < scrollX + 10) left = scrollX + 10;

      badge.style.top = `${top}px`;
      badge.style.left = `${left}px`;
    } catch (_) {}
  }

  function positionModal(selection) {
    let top = 0;
    let left = 0;
    let positioned = false;

    if (selection && selection.rangeCount > 0) {
      try {
        const rect = selection.getRangeAt(0).getBoundingClientRect();
        if (rect.width > 0 || rect.height > 0) {
          top = rect.bottom + 10;
          left = rect.left;
          positioned = true;
        }
      } catch (_) {}
    }

    if (!positioned) {
      // Fallback: Center modal in visible viewport (ideal for context menu clicks)
      top = Math.max(30, (window.innerHeight - 320) / 2);
      left = Math.max(20, (window.innerWidth - 440) / 2);
    } else {
      // Viewport boundary clamping for fixed positioning
      if (top + 320 > window.innerHeight) {
        top = Math.max(20, window.innerHeight - 340);
      }
      if (left + 450 > window.innerWidth) {
        left = Math.max(20, window.innerWidth - 450);
      }
      if (left < 20) left = 20;
    }

    modal.style.top = `${top}px`;
    modal.style.left = `${left}px`;
  }

  // 6. Modal Open & Inference Execution
  badge.addEventListener("click", () => {
    badge.classList.remove("visible");
    openModal(currentSelectionText, currentMode);
  });

  function openModal(text, mode = "fix") {
    if (!text || !text.trim()) {
      const sel = window.getSelection();
      text = sel ? sel.toString().trim() : "";
    }
    if (!text || !text.trim()) return;

    currentSelectionText = text;
    currentMode = mode;

    const sel = window.getSelection();
    positionModal(sel);
    badge.classList.remove("visible");
    modal.classList.add("visible");

    // Highlight active mode tab
    shadow.querySelectorAll(".fluenta-mode-btn").forEach(btn => {
      btn.classList.toggle("active", btn.dataset.mode === mode);
    });

    const body = shadow.getElementById("fluenta-body");
    body.innerHTML = `
      <div class="fluenta-loading">
        <div class="fluenta-spinner"></div>
        <span>Proofreading with AI (${mode})...</span>
      </div>
    `;

    shadow.getElementById("fluenta-stat").textContent = `Original: ${text.length} chars`;
    shadow.getElementById("fluenta-latency").textContent = "⚡ running...";

    // Request check through background worker to avoid page CSP issues
    chrome.runtime.sendMessage({
      action: "CHECK_TEXT",
      text: text,
      mode: mode
    }, (response) => {
      if (!response) {
        body.innerHTML = `<span style="color:#ef4444;">Error: Extension background did not respond.</span>`;
        return;
      }
      if (!response.success) {
        body.innerHTML = `<span style="color:#ef4444;">Error: ${escapeHtml(response.error)}</span>`;
        return;
      }

      currentResultText = response.resultText;
      shadow.getElementById("fluenta-latency").textContent = `⚡ ${response.latencyMs}ms`;

      // Render Word Diff
      renderDiff(response.originalText, response.resultText, body);
    });
  }

  // 7. Mode Tab Switch Listener
  shadow.querySelectorAll(".fluenta-mode-btn").forEach(btn => {
    btn.addEventListener("click", () => {
      const mode = btn.dataset.mode;
      openModal(currentSelectionText, mode);
    });
  });

  // 8. Word Diff Renderer
  function renderDiff(orig, updated, container) {
    if (orig === updated) {
      container.innerHTML = `<span style="color:#10b981;">✓ No grammatical errors found! Text is clean.</span><br><br>${escapeHtml(updated)}`;
      return;
    }

    const origWords = orig.split(/(\s+)/);
    const updatedWords = updated.split(/(\s+)/);

    // Simple diff highlight
    let html = "";
    let i = 0;
    let j = 0;

    while (i < origWords.length || j < updatedWords.length) {
      if (i < origWords.length && j < updatedWords.length && origWords[i] === updatedWords[j]) {
        html += escapeHtml(origWords[i]);
        i++;
        j++;
      } else {
        if (i < origWords.length && (!updatedWords.includes(origWords[i], j))) {
          html += `<span class="fluenta-diff-del">${escapeHtml(origWords[i])}</span>`;
          i++;
        }
        if (j < updatedWords.length) {
          html += `<span class="fluenta-diff-add">${escapeHtml(updatedWords[j])}</span>`;
          j++;
        }
      }
    }

    container.innerHTML = html;
  }

  // 9. In-Place Text Replacement
  function replaceText() {
    if (!currentResultText) return;

    modal.classList.remove("visible");

    // Case 1: Active Input or Textarea
    if (activeElementTarget && (activeElementTarget.tagName === "TEXTAREA" || activeElementTarget.tagName === "INPUT")) {
      const el = activeElementTarget;
      el.focus();

      const start = el.selectionStart;
      const end = el.selectionEnd;

      if (start !== undefined && end !== undefined && start !== end) {
        // Modern execCommand for undo history
        const success = document.execCommand("insertText", false, currentResultText);
        if (!success) {
          el.setRangeText(currentResultText, start, end, "end");
          el.dispatchEvent(new Event("input", { bubbles: true }));
          el.dispatchEvent(new Event("change", { bubbles: true }));
        }
      } else {
        el.value = currentResultText;
        el.dispatchEvent(new Event("input", { bubbles: true }));
      }
      return;
    }

    // Case 2: ContentEditable (Gmail, Notion, Slack, Docs, Twitter)
    if (activeRange) {
      const sel = window.getSelection();
      sel.removeAllRanges();
      sel.addRange(activeRange);

      const success = document.execCommand("insertText", false, currentResultText);
      if (!success) {
        activeRange.deleteContents();
        activeRange.insertNode(document.createTextNode(currentResultText));
      }
      return;
    }

    // Fallback: Copy to clipboard if active target lost
    navigator.clipboard.writeText(currentResultText);
  }

  // Button Listeners
  shadow.getElementById("fluenta-apply").addEventListener("click", replaceText);
  shadow.getElementById("fluenta-copy").addEventListener("click", () => {
    if (currentResultText) {
      navigator.clipboard.writeText(currentResultText);
      const copyBtn = shadow.getElementById("fluenta-copy");
      copyBtn.textContent = "Copied!";
      setTimeout(() => copyBtn.textContent = "Copy", 1500);
    }
  });

  shadow.getElementById("fluenta-close").addEventListener("click", () => {
    modal.classList.remove("visible");
  });

  // Global Hotkey and Escape Listeners
  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape" && modal.classList.contains("visible")) {
      modal.classList.remove("visible");
    }
    if ((e.key === "Enter" || e.keyCode === 13) && modal.classList.contains("visible")) {
      replaceText();
    }
  });

  // Listen for Messages from Background Worker (Context Menu & Hotkey)
  chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
    if (request.action === "TRIGGER_SELECTION") {
      openModal(request.text, request.mode || "fix");
      if (sendResponse) sendResponse({ received: true });
      return true;
    }
    if (request.action === "TRIGGER_SHORTCUT") {
      const sel = window.getSelection();
      const text = sel ? sel.toString().trim() : "";
      if (text) {
        openModal(text, currentMode);
      }
      if (sendResponse) sendResponse({ received: true });
      return true;
    }
  });

  function escapeHtml(str) {
    if (!str) return "";
    return str.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
  }
})();
