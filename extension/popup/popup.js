/**
 * FluentA Toolbar Popup Logic
 */

document.addEventListener("DOMContentLoaded", async () => {
  const inputEl = document.getElementById("scratchpad-input");
  const checkBtn = document.getElementById("check-btn");
  const copyBtn = document.getElementById("copy-btn");
  const outputContainer = document.getElementById("scratchpad-output-container");
  const outputEl = document.getElementById("scratchpad-output");
  const latencyEl = document.getElementById("output-latency");
  const siteToggle = document.getElementById("site-toggle");
  const siteLabel = document.getElementById("current-site-label");
  const openOptionsBtn = document.getElementById("open-options-btn");

  let currentMode = "fix";
  let lastResultText = "";
  let currentHost = "";

  // 1. Get current active tab hostname
  chrome.tabs.query({ active: true, currentWindow: true }, (tabs) => {
    if (tabs[0] && tabs[0].url) {
      try {
        const url = new URL(tabs[0].url);
        currentHost = url.hostname;
        siteLabel.textContent = `Active on ${currentHost}`;

        // Check if current host is excluded
        chrome.storage.sync.get({ excludedDomains: [] }, (res) => {
          const isExcluded = res.excludedDomains.some(d => currentHost.includes(d));
          siteToggle.checked = !isExcluded;
        });
      } catch (_) {
        siteLabel.textContent = "Active";
      }
    }
  });

  // 2. Open Settings
  openOptionsBtn.addEventListener("click", () => {
    chrome.runtime.openOptionsPage();
  });

  // 3. Mode Switcher
  document.querySelectorAll(".mode-tab").forEach(tab => {
    tab.addEventListener("click", () => {
      document.querySelectorAll(".mode-tab").forEach(t => t.classList.remove("active"));
      tab.classList.add("active");
      currentMode = tab.dataset.mode;
      if (inputEl.value.trim()) {
        runCheck();
      }
    });
  });

  // 4. Run Check
  async function runCheck() {
    const text = inputEl.value.trim();
    if (!text) return;

    checkBtn.disabled = true;
    checkBtn.textContent = "Checking...";
    outputContainer.style.display = "block";
    outputEl.innerHTML = `<span style="color:#94a3b8;">Processing with AI...</span>`;

    chrome.runtime.sendMessage({
      action: "CHECK_TEXT",
      text: text,
      mode: currentMode
    }, (response) => {
      checkBtn.disabled = false;
      checkBtn.textContent = "Check with AI";

      if (!response || !response.success) {
        outputEl.innerHTML = `<span style="color:#ef4444;">Error: ${response ? response.error : 'Unknown error'}</span>`;
        return;
      }

      lastResultText = response.resultText;
      latencyEl.textContent = `⚡ ${response.latencyMs}ms`;
      copyBtn.style.display = "inline-flex";

      renderDiff(response.originalText, response.resultText, outputEl);
    });
  }

  checkBtn.addEventListener("click", runCheck);

  // Ctrl+Enter shortcut in textarea
  inputEl.addEventListener("keydown", (e) => {
    if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
      runCheck();
    }
  });

  // 5. Copy Output
  copyBtn.addEventListener("click", () => {
    if (lastResultText) {
      navigator.clipboard.writeText(lastResultText);
      copyBtn.textContent = "Copied!";
      setTimeout(() => copyBtn.textContent = "Copy", 1500);
    }
  });

  // 6. Site Exclude Toggle
  siteToggle.addEventListener("change", () => {
    if (!currentHost) return;
    chrome.storage.sync.get({ excludedDomains: [] }, (res) => {
      let list = res.excludedDomains;
      if (!siteToggle.checked) {
        if (!list.includes(currentHost)) list.push(currentHost);
      } else {
        list = list.filter(d => !currentHost.includes(d));
      }
      chrome.storage.sync.set({ excludedDomains: list });
    });
  });

  function renderDiff(orig, updated, container) {
    if (orig === updated) {
      container.innerHTML = `<span style="color:#10b981;">✓ No grammatical errors found!</span>`;
      return;
    }

    const origWords = orig.split(/(\s+)/);
    const updatedWords = updated.split(/(\s+)/);

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
          html += `<span class="diff-del">${escapeHtml(origWords[i])}</span>`;
          i++;
        }
        if (j < updatedWords.length) {
          html += `<span class="diff-add">${escapeHtml(updatedWords[j])}</span>`;
          j++;
        }
      }
    }

    container.innerHTML = html;
  }

  function escapeHtml(str) {
    if (!str) return "";
    return str.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
  }
});
