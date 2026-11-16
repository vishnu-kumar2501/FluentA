/**
 * FluentA Options Page Logic
 */

document.addEventListener("DOMContentLoaded", () => {
  const providerSelect = document.getElementById("provider-select");
  const baseUrlInput = document.getElementById("base-url-input");
  const modelInput = document.getElementById("model-input");
  const apiKeyInput = document.getElementById("api-key-input");
  const defaultModeSelect = document.getElementById("default-mode-select");
  const badgeToggle = document.getElementById("badge-toggle");
  const excludedDomainsInput = document.getElementById("excluded-domains-input");
  const testConnectionBtn = document.getElementById("test-connection-btn");
  const testStatus = document.getElementById("test-status");
  const saveBtn = document.getElementById("save-settings-btn");
  const saveStatus = document.getElementById("save-status");
  const changeShortcutLink = document.getElementById("change-shortcut-link");

  const DEFAULT_SETTINGS = {
    provider: "cloud",
    baseURL: "https://fluenta-proxy.kumarvishnu610.workers.dev",
    model: "openai/gpt-oss-120b",
    apiKey: "",
    defaultMode: "fix",
    autoFloatingBadge: true,
    excludedDomains: [
      "paypal.com",
      "chase.com",
      "bankofamerica.com",
      "wellsfargo.com"
    ]
  };

  // 1. Load Current Settings
  chrome.storage.sync.get(DEFAULT_SETTINGS, (settings) => {
    providerSelect.value = settings.provider || "cloud";
    baseUrlInput.value = settings.baseURL || DEFAULT_SETTINGS.baseURL;
    modelInput.value = settings.model || DEFAULT_SETTINGS.model;
    apiKeyInput.value = settings.apiKey || "";
    defaultModeSelect.value = settings.defaultMode || "fix";
    badgeToggle.checked = settings.autoFloatingBadge !== false;
    excludedDomainsInput.value = (settings.excludedDomains || []).join("\n");
  });

  // 2. Provider Preset Switcher
  providerSelect.addEventListener("change", () => {
    const p = providerSelect.value;
    if (p === "cloud") {
      baseUrlInput.value = "https://fluenta-proxy.kumarvishnu610.workers.dev";
      modelInput.value = "openai/gpt-oss-120b";
      apiKeyInput.placeholder = "Leave empty to use built-in gateway";
    } else if (p === "ollama") {
      baseUrlInput.value = "http://127.0.0.1:11434";
      modelInput.value = "qwen2.5:latest";
      apiKeyInput.value = "";
      apiKeyInput.placeholder = "Not required for local Ollama";
    } else if (p === "openai") {
      baseUrlInput.value = "https://api.groq.com/openai/v1";
      modelInput.value = "llama-3.3-70b-versatile";
      apiKeyInput.placeholder = "gsk_... or sk-...";
    }
  });

  // 3. Test Connection
  testConnectionBtn.addEventListener("click", () => {
    testStatus.textContent = "Testing...";
    testStatus.style.color = "#a5b4fc";
    testConnectionBtn.disabled = true;

    // Temporarily save current form values to test
    const testConfig = {
      provider: providerSelect.value,
      baseURL: baseUrlInput.value.trim(),
      model: modelInput.value.trim(),
      apiKey: apiKeyInput.value.trim()
    };

    chrome.storage.sync.set(testConfig, () => {
      chrome.runtime.sendMessage({
        action: "CHECK_TEXT",
        text: "This is an test sentence.",
        mode: "fix"
      }, (res) => {
        testConnectionBtn.disabled = false;
        if (res && res.success) {
          testStatus.textContent = `✓ Connected! (${res.latencyMs}ms)`;
          testStatus.style.color = "#34d399";
        } else {
          testStatus.textContent = `✕ Failed: ${res ? res.error : "Unknown error"}`;
          testStatus.style.color = "#f87171";
        }
      });
    });
  });

  // 4. Save Settings
  saveBtn.addEventListener("click", () => {
    const rawDomains = excludedDomainsInput.value
      .split("\n")
      .map(d => d.trim().toLowerCase())
      .filter(d => d.length > 0);

    const newSettings = {
      provider: providerSelect.value,
      baseURL: baseUrlInput.value.trim(),
      model: modelInput.value.trim(),
      apiKey: apiKeyInput.value.trim(),
      defaultMode: defaultModeSelect.value,
      autoFloatingBadge: badgeToggle.checked,
      excludedDomains: rawDomains
    };

    saveBtn.disabled = true;
    chrome.storage.sync.set(newSettings, () => {
      saveBtn.disabled = false;
      saveStatus.textContent = "✓ Settings saved successfully!";
      saveStatus.style.color = "#34d399";
      setTimeout(() => {
        saveStatus.textContent = "";
      }, 2500);
    });
  });

  // 5. Shortcut Link
  changeShortcutLink.addEventListener("click", (e) => {
    e.preventDefault();
    chrome.tabs.create({ url: "chrome://extensions/shortcuts" });
  });
});
