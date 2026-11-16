(() => {
  "use strict";

  function boot() {
    const state = {
      sourceText: "",
      aiOutputText: "",
      outputText: "",
      lastCheckedSource: "",
      mode: "fix",
      view: "diff", // "diff" | "list" | "split" | "clean"
      streaming: false,
      replaceAllMode: false,
      canReplaceTarget: false,
      hotkeyLabel: "Ctrl+Alt+G",
      hotkeyConflict: false,
      hotkeyNotice: "",
      hotkeyRecording: false,
      hotkeyCheckState: "idle",
      hotkeyCheckMessage: "",
      modelName: "qwen3.5:4b",
      watchEnabled: true,
      paused: false,
      language: "Auto-detect",
      defaultMode: "fix",
      themePref: "system",
      excludedApps: [],
      history: [],
      historyFilter: "",
      activePanel: "editor", // "editor" | "history" | "settings" | "onboarding"
      aiConnected: true,
      aiError: "",
      aiProviderType: "cloud",
      aiBaseURL: "https://fluenta-proxy.kumarvishnu610.workers.dev",
      aiModel: "openai/gpt-oss-120b",
      aiApiKey: "",
      aiTesting: false,
      error: "",
      startedAt: 0,
      segments: [], // { kind: "eq", text } | { kind: "hunk", id, before, after, accepted, category, reason, explaining }
      selectedHunkId: "",
      explainCache: new Map(), // key: before + "\0" + after -> { category, reason }
      explainingAll: false,
      insights: null,
      analyzingInsights: false,
      undoStack: [],
      redoStack: [],
    };

    const els = {
      windowHeader: document.getElementById("window-header"),
      modelTag: document.getElementById("model-tag"),
      hotkeyTag: document.getElementById("hotkey-tag"),
      watchToggleBtn: document.getElementById("btn-watch-toggle"),
      watchToggleLabel: document.getElementById("watch-toggle-label"),
      pauseToggleBtn: document.getElementById("btn-pause-toggle"),
      pauseToggleLabel: document.getElementById("pause-toggle-label"),
      statusPill: document.getElementById("status-pill"),
      statusText: document.getElementById("status-text"),
      historyToggleBtn: document.getElementById("btn-history-toggle"),
      settingsToggleBtn: document.getElementById("btn-settings-toggle"),
      themeBtn: document.getElementById("btn-theme"),
      toolbarRow: document.getElementById("toolbar-row"),
      languageSelect: document.getElementById("language-select"),
      paneSource: document.getElementById("pane-source"),
      paneResult: document.getElementById("pane-result"),
      modeButtons: Array.from(document.querySelectorAll(".mode-btn")),
      sourceInput: document.getElementById("source-input"),
      sourceStats: document.getElementById("source-stats"),
      clearInputBtn: document.getElementById("btn-clear-input"),
      errorBanner: document.getElementById("error-banner"),
      errorMessage: document.getElementById("error-message"),
      retryBtn: document.getElementById("btn-retry"),
      hotkeyNoticeBanner: document.getElementById("hotkey-notice-banner"),
      hotkeyNoticeMessage: document.getElementById("hotkey-notice-message"),
      hotkeyNoticeSettingsBtn: document.getElementById("btn-hotkey-notice-settings"),
      hotkeyNoticeDismissBtn: document.getElementById("btn-hotkey-notice-dismiss"),
      tabDiff: document.getElementById("tab-diff"),
      tabList: document.getElementById("tab-list"),
      tabSplit: document.getElementById("tab-split"),
      tabClean: document.getElementById("tab-clean"),
      tabInsights: document.getElementById("tab-insights"),
      hunkCountBadge: document.getElementById("hunk-count-badge"),
      undoBtn: document.getElementById("btn-undo"),
      redoBtn: document.getElementById("btn-redo"),
      diffSummary: document.getElementById("diff-summary"),
      inspectorBar: document.getElementById("hunk-inspector-bar"),
      inspectorPreview: document.getElementById("inspector-preview"),
      inspectorReason: document.getElementById("inspector-reason"),
      inspectorAcceptBtn: document.getElementById("btn-inspector-accept"),
      inspectorRejectBtn: document.getElementById("btn-inspector-reject"),
      inspectorWhyBtn: document.getElementById("btn-inspector-why"),
      diffView: document.getElementById("diff-view"),
      correctionsView: document.getElementById("corrections-view"),
      correctionsMeta: document.getElementById("corrections-meta"),
      explainAllBtn: document.getElementById("btn-explain-all"),
      acceptAllBtn: document.getElementById("btn-accept-all"),
      rejectAllBtn: document.getElementById("btn-reject-all"),
      correctionsList: document.getElementById("corrections-list"),
      splitView: document.getElementById("split-view"),
      splitOriginal: document.getElementById("split-original"),
      splitUpdated: document.getElementById("split-updated"),
      outputInput: document.getElementById("output-input"),
      insightsView: document.getElementById("insights-view"),
      analyzeWritingBtn: document.getElementById("btn-analyze-writing"),
      metricWords: document.getElementById("metric-words"),
      metricSentences: document.getElementById("metric-sentences"),
      metricReadability: document.getElementById("metric-readability"),
      metricReadtime: document.getElementById("metric-readtime"),
      aiInsightsEmpty: document.getElementById("ai-insights-empty"),
      aiInsightsContent: document.getElementById("ai-insights-content"),
      aiScoreValue: document.getElementById("ai-score-value"),
      aiToneBadge: document.getElementById("ai-tone-badge"),
      aiToneSummary: document.getElementById("ai-tone-summary"),
      aiClaritySummary: document.getElementById("ai-clarity-summary"),
      vocabList: document.getElementById("vocab-list"),
      historyPanel: document.getElementById("history-panel"),
      historyBackBtn: document.getElementById("btn-history-back"),
      historySearch: document.getElementById("history-search"),
      historyClearBtn: document.getElementById("btn-history-clear"),
      historyList: document.getElementById("history-list"),
      settingsPanel: document.getElementById("settings-panel"),
      settingsBackBtn: document.getElementById("btn-settings-back"),
      settingDefaultMode: document.getElementById("setting-default-mode"),
      settingLanguage: document.getElementById("setting-language"),
      settingTheme: document.getElementById("setting-theme"),
      settingWatchEnabled: document.getElementById("setting-watch-enabled"),
      settingHotkey: document.getElementById("setting-hotkey"),
      hotkeyRecordBtn: document.getElementById("btn-hotkey-record"),
      hotkeyCheckBtn: document.getElementById("btn-hotkey-check"),
      hotkeyDefaultBtn: document.getElementById("btn-hotkey-default"),
      hotkeyCheckBadge: document.getElementById("hotkey-check-badge"),
      hotkeyCheckDetail: document.getElementById("hotkey-check-detail"),
      settingExcludedApps: document.getElementById("setting-excluded-apps"),
      settingsCancelBtn: document.getElementById("btn-settings-cancel"),
      settingsSaveBtn: document.getElementById("btn-settings-save"),
      privacyPanel: document.getElementById("privacy-panel"),
      btnPrivacyBack: document.getElementById("btn-privacy-back"),
      btnViewPrivacy: document.getElementById("btn-view-privacy"),
      aiStatusBadge: document.getElementById("ai-status-badge"),
      aiStatusBadgeText: document.getElementById("ai-status-badge-text"),
      onboardingPanel: document.getElementById("onboarding-panel"),
      btnOnboardCloud: document.getElementById("btn-onboard-cloud"),
      btnOnboardOllama: document.getElementById("btn-onboard-ollama"),
      btnOnboardSkip: document.getElementById("btn-onboard-skip"),
      onboardCloudKey: document.getElementById("onboard-cloud-key"),
      btnToggleOnboardKey: document.getElementById("btn-toggle-onboard-key"),
      onboardOllamaUrl: document.getElementById("onboard-ollama-url"),
      onboardStatusBanner: document.getElementById("onboard-status-banner"),
      onboardStatusText: document.getElementById("onboard-status-text"),
      settingAIProvider: document.getElementById("setting-ai-provider"),
      settingAIBaseURL: document.getElementById("setting-ai-base-url"),
      settingAIModel: document.getElementById("setting-ai-model"),
      settingAIApiKey: document.getElementById("setting-ai-api-key"),
      rowSettingAIKey: document.getElementById("row-setting-ai-key"),
      btnToggleSettingKey: document.getElementById("btn-toggle-setting-key"),
      btnTestAI: document.getElementById("btn-test-ai"),
      aiTestBadge: document.getElementById("ai-test-badge"),
      aiTestDetail: document.getElementById("ai-test-detail"),
      btnPresetGroq: document.getElementById("btn-preset-groq"),
      btnPresetOllama: document.getElementById("btn-preset-ollama"),
      hotkeyHint: document.getElementById("hotkey-hint"),
      cancelBtn: document.getElementById("btn-cancel"),
      recheckBtn: document.getElementById("btn-recheck"),
      recheckLabel: document.getElementById("recheck-label"),
      copyBtn: document.getElementById("btn-copy"),
      copyLabel: document.getElementById("copy-label"),
      replaceBtn: document.getElementById("btn-replace"),
      replaceLabel: document.getElementById("replace-label"),
    };

    let copyResetTimer = null;
    let editUndoTimer = null;
    let hotkeyCheckTimer = null;
    let lastSnapshotHash = "";
    let explainReqCounter = 0;

    function postToHost(payload) {
      if (window.chrome && window.chrome.webview) {
        window.chrome.webview.postMessage(JSON.stringify(payload));
      } else {
        fetch("/api/message", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload)
        }).catch((err) => console.error("postToHost error:", err));
      }
    }

    function tokenize(text) {
      if (!text) return [];
      return text.match(/[A-Za-z0-9'’-]+|\s+|[^\sA-Za-z0-9'’-]+/g) || [text];
    }

    function computeRawDiffOps(originalText, revisedText) {
      const a = tokenize(originalText);
      const b = tokenize(revisedText);

      let prefix = 0;
      while (prefix < a.length && prefix < b.length && a[prefix] === b[prefix]) {
        prefix++;
      }

      let suffix = 0;
      while (
        suffix < a.length - prefix &&
        suffix < b.length - prefix &&
        a[a.length - 1 - suffix] === b[b.length - 1 - suffix]
      ) {
        suffix++;
      }

      const midA = a.slice(prefix, a.length - suffix);
      const midB = b.slice(prefix, b.length - suffix);

      const ops = [];
      for (let i = 0; i < prefix; i++) {
        ops.push({ type: "eq", text: a[i] });
      }

      if (midA.length === 0 && midB.length === 0) {
        // Exact match
      } else if (midA.length * midB.length > 900000) {
        if (midA.length > 0) ops.push({ type: "del", text: midA.join("") });
        if (midB.length > 0) ops.push({ type: "ins", text: midB.join("") });
      } else {
        const m = midA.length;
        const n = midB.length;
        const stride = n + 1;
        const dp = new Int32Array((m + 1) * stride);

        for (let i = m - 1; i >= 0; i--) {
          const row = i * stride;
          const nextRow = (i + 1) * stride;
          for (let j = n - 1; j >= 0; j--) {
            if (midA[i] === midB[j]) {
              dp[row + j] = dp[nextRow + j + 1] + 1;
            } else {
              const down = dp[nextRow + j];
              const right = dp[row + j + 1];
              dp[row + j] = down >= right ? down : right;
            }
          }
        }

        let i = 0;
        let j = 0;
        while (i < m && j < n) {
          if (midA[i] === midB[j]) {
            ops.push({ type: "eq", text: midA[i] });
            i++;
            j++;
          } else if (dp[(i + 1) * stride + j] >= dp[i * stride + j + 1]) {
            ops.push({ type: "del", text: midA[i++] });
          } else {
            ops.push({ type: "ins", text: midB[j++] });
          }
        }
        while (i < m) ops.push({ type: "del", text: midA[i++] });
        while (j < n) ops.push({ type: "ins", text: midB[j++] });
      }

      for (let k = a.length - suffix; k < a.length; k++) {
        ops.push({ type: "eq", text: a[k] });
      }

      const merged = [];
      for (const op of ops) {
        const last = merged[merged.length - 1];
        if (last && last.type === op.type) {
          last.text += op.text;
        } else {
          merged.push({ type: op.type, text: op.text });
        }
      }
      return merged;
    }

    function buildChangeSegments(originalText, revisedText) {
      const ops = computeRawDiffOps(originalText, revisedText);
      const segments = [];
      let hunkIdx = 0;

      for (let i = 0; i < ops.length; i++) {
        const cur = ops[i];
        if (cur.type === "eq") {
          segments.push({ kind: "eq", text: cur.text });
          continue;
        }

        let before = "";
        let after = "";
        if (cur.type === "del") {
          before = cur.text;
          if (i + 1 < ops.length && ops[i + 1].type === "ins") {
            after = ops[i + 1].text;
            i++;
          }
        } else if (cur.type === "ins") {
          after = cur.text;
          if (i + 1 < ops.length && ops[i + 1].type === "del") {
            before = ops[i + 1].text;
            i++;
          }
        }

        if (before.trim() === "" && after.trim() === "") {
          segments.push({ kind: "eq", text: after || before });
          continue;
        }

        hunkIdx++;
        const cacheKey = before + "\0" + after;
        const cached = state.explainCache.get(cacheKey);
        segments.push({
          kind: "hunk",
          id: "c" + hunkIdx,
          before,
          after,
          accepted: true,
          category: cached ? cached.category : "",
          reason: cached ? cached.reason : "",
          explaining: false,
        });
      }
      return segments;
    }

    function getHunks() {
      return state.segments.filter((s) => s.kind === "hunk");
    }

    function findHunk(id) {
      return state.segments.find((s) => s.kind === "hunk" && s.id === id) || null;
    }

    function materializeFromSegments() {
      let out = "";
      for (const seg of state.segments) {
        if (seg.kind === "eq") {
          out += seg.text;
        } else {
          out += seg.accepted ? seg.after : seg.before;
        }
      }
      return out;
    }

    function rebuildModel(source, revised) {
      if (state.mode === "explain" || !revised) {
        state.segments = [];
        state.selectedHunkId = "";
        return;
      }
      state.segments = buildChangeSegments(source, revised);
      const hunks = getHunks();
      if (!findHunk(state.selectedHunkId)) {
        state.selectedHunkId = hunks.length > 0 ? hunks[0].id : "";
      }
    }

    function snapshotState() {
      return {
        outputText: state.outputText,
        aiOutputText: state.aiOutputText,
        acceptedById: getHunks().map((h) => [h.id, h.accepted]),
      };
    }

    function snapshotKey(snap) {
      return JSON.stringify(snap);
    }

    function resetUndoHistory() {
      state.undoStack = [];
      state.redoStack = [];
      lastSnapshotHash = snapshotKey(snapshotState());
    }

    function pushUndoPoint() {
      const snap = snapshotState();
      const hash = snapshotKey(snap);
      if (hash === lastSnapshotHash) return;
      state.undoStack.push(JSON.parse(lastSnapshotHash));
      if (state.undoStack.length > 80) state.undoStack.shift();
      state.redoStack = [];
      lastSnapshotHash = hash;
    }

    function applySnapshot(snap) {
      state.aiOutputText = snap.aiOutputText;
      state.outputText = snap.outputText;
      if (state.aiOutputText && state.mode !== "explain") {
        rebuildModel(state.sourceText, state.aiOutputText);
        const map = new Map(snap.acceptedById || []);
        for (const h of getHunks()) {
          if (map.has(h.id)) h.accepted = map.get(h.id);
        }
        if (snap.acceptedById && snap.acceptedById.length > 0) {
          state.outputText = materializeFromSegments();
        }
      } else {
        state.segments = [];
      }
      lastSnapshotHash = snapshotKey(snapshotState());
      syncControls();
    }

    function undo() {
      if (state.undoStack.length === 0) return;
      const cur = snapshotState();
      const prev = state.undoStack.pop();
      state.redoStack.push(cur);
      applySnapshot(prev);
    }

    function redo() {
      if (state.redoStack.length === 0) return;
      const cur = snapshotState();
      const next = state.redoStack.pop();
      state.undoStack.push(cur);
      applySnapshot(next);
    }

    function setHunkAccepted(id, accepted) {
      const hunk = findHunk(id);
      if (!hunk || hunk.accepted === accepted) return;
      hunk.accepted = accepted;
      state.selectedHunkId = id;
      state.outputText = materializeFromSegments();
      pushUndoPoint();
      syncControls();
    }

    function setAllHunksAccepted(accepted) {
      const hunks = getHunks();
      if (hunks.length === 0) return;
      let changed = false;
      for (const h of hunks) {
        if (h.accepted !== accepted) {
          h.accepted = accepted;
          changed = true;
        }
      }
      if (!changed) return;
      state.outputText = materializeFromSegments();
      pushUndoPoint();
      syncControls();
    }

    function requestExplanations(hunksToExplain) {
      const pending = hunksToExplain.filter((h) => !h.reason && !h.explaining);
      if (pending.length === 0) return;
      explainReqCounter++;
      const requestId = "exp-" + explainReqCounter;
      for (const h of pending) h.explaining = true;
      if (pending.length > 1) state.explainingAll = true;
      renderInspectorBar();
      renderCorrectionsList();

      postToHost({
        type: "explainChanges",
        requestId,
        original: state.sourceText,
        revised: state.aiOutputText || state.outputText,
        changes: pending.map((h) => ({
          id: h.id,
          before: h.before,
          after: h.after,
        })),
      });
    }

    function formatChangeLabel(h) {
      const before = h.before.trim() || "(none)";
      const after = h.after.trim() || "(removed)";
      return before + " -> " + after;
    }

    function renderInspectorBar() {
      const hunks = getHunks();
      if (
        state.streaming ||
        state.mode === "explain" ||
        state.view !== "diff" ||
        hunks.length === 0
      ) {
        els.inspectorBar.hidden = true;
        return;
      }

      const sel = findHunk(state.selectedHunkId) || hunks[0];
      state.selectedHunkId = sel.id;
      els.inspectorBar.hidden = false;
      els.inspectorPreview.textContent = formatChangeLabel(sel);
      if (sel.explaining) {
        els.inspectorReason.textContent = "Asking model why...";
      } else if (sel.reason) {
        els.inspectorReason.textContent = (sel.category ? "[" + sel.category + "] " : "") + sel.reason;
      } else {
        els.inspectorReason.textContent = sel.accepted ? "Accepted — click Why? for reason" : "Rejected (kept original)";
      }

      els.inspectorAcceptBtn.dataset.active = sel.accepted ? "true" : "false";
      els.inspectorRejectBtn.dataset.active = !sel.accepted ? "true" : "false";
      els.inspectorWhyBtn.disabled = Boolean(sel.explaining);
    }

    function renderDiffView() {
      const container = els.diffView;
      container.textContent = "";

      if (state.streaming) {
        const textNode = document.createTextNode(state.outputText);
        const caret = document.createElement("span");
        caret.className = "streaming-caret";
        caret.setAttribute("aria-hidden", "true");
        container.appendChild(textNode);
        container.appendChild(caret);
        container.scrollTop = container.scrollHeight;
        els.diffSummary.textContent = "Streaming...";
        return;
      }

      if (!state.outputText && state.segments.length === 0) {
        const wrap = document.createElement("div");
        wrap.className = "empty-state";
        const title = document.createElement("p");
        title.className = "empty-title";
        title.textContent = "No suggestion yet";
        const hint = document.createElement("p");
        hint.className = "empty-hint";
        hint.textContent =
          "Highlight text anywhere and press " +
          state.hotkeyLabel +
          ", or edit text above and click Re-check.";
        wrap.appendChild(title);
        wrap.appendChild(hint);
        container.appendChild(wrap);
        els.diffSummary.textContent = "Waiting";
        return;
      }

      if (state.mode === "explain") {
        container.textContent = state.outputText;
        els.diffSummary.textContent = "Explanation";
        return;
      }

      if (state.segments.length === 0 && state.outputText) {
        container.textContent = state.outputText;
        return;
      }

      const frag = document.createDocumentFragment();
      for (const seg of state.segments) {
        if (seg.kind === "eq") {
          frag.appendChild(document.createTextNode(seg.text));
          continue;
        }
        const wrap = document.createElement("span");
        wrap.className = "diff-hunk";
        wrap.dataset.id = seg.id;
        wrap.dataset.status = seg.accepted ? "accepted" : "rejected";
        wrap.dataset.selected = seg.id === state.selectedHunkId ? "true" : "false";
        wrap.title = "Click to inspect or accept/reject";

        if (seg.before) {
          const del = document.createElement("del");
          del.className = "diff-del";
          del.textContent = seg.before;
          wrap.appendChild(del);
        }
        if (seg.after) {
          const ins = document.createElement("ins");
          ins.className = "diff-ins";
          ins.textContent = seg.after;
          wrap.appendChild(ins);
        }
        wrap.addEventListener("click", () => {
          state.selectedHunkId = seg.id;
          syncControls();
        });
        frag.appendChild(wrap);
      }
      container.appendChild(frag);
    }

    function renderCorrectionsList() {
      const hunks = getHunks();
      if (hunks.length > 0 && state.mode !== "explain") {
        els.hunkCountBadge.hidden = false;
        els.hunkCountBadge.textContent = String(hunks.length);
      } else {
        els.hunkCountBadge.hidden = true;
      }

      const acceptedCount = hunks.filter((h) => h.accepted).length;
      els.correctionsMeta.textContent =
        hunks.length === 0
          ? "No changes found"
          : acceptedCount + " of " + hunks.length + " accepted";

      els.explainAllBtn.disabled =
        hunks.length === 0 || state.explainingAll || hunks.every((h) => Boolean(h.reason));
      els.explainAllBtn.textContent = state.explainingAll ? "Explaining..." : "Explain all";
      els.acceptAllBtn.disabled = hunks.length === 0;
      els.rejectAllBtn.disabled = hunks.length === 0;

      els.correctionsList.textContent = "";
      if (hunks.length === 0) {
        const empty = document.createElement("div");
        empty.className = "empty-state";
        const t = document.createElement("p");
        t.className = "empty-title";
        t.textContent = state.outputText ? "Text already looks good" : "No corrections yet";
        empty.appendChild(t);
        els.correctionsList.appendChild(empty);
        return;
      }

      const frag = document.createDocumentFragment();
      for (const h of hunks) {
        const card = document.createElement("div");
        card.className = "correction-item";
        card.dataset.status = h.accepted ? "accepted" : "rejected";

        const top = document.createElement("div");
        top.className = "correction-row-top";

        const change = document.createElement("div");
        change.className = "correction-change";
        if (h.before) {
          const del = document.createElement("del");
          del.className = "diff-del";
          del.textContent = h.before;
          change.appendChild(del);
        }
        if (h.after) {
          const ins = document.createElement("ins");
          ins.className = "diff-ins";
          ins.textContent = h.after;
          change.appendChild(ins);
        }
        if (h.category) {
          const cat = document.createElement("span");
          cat.className = "correction-cat";
          cat.textContent = h.category;
          change.appendChild(cat);
        }

        const actions = document.createElement("div");
        actions.className = "hunk-inspector-actions";

        const accBtn = document.createElement("button");
        accBtn.type = "button";
        accBtn.className = "mini-action-btn mini-accept";
        accBtn.dataset.active = h.accepted ? "true" : "false";
        accBtn.textContent = "Accept";
        accBtn.addEventListener("click", () => setHunkAccepted(h.id, true));

        const rejBtn = document.createElement("button");
        rejBtn.type = "button";
        rejBtn.className = "mini-action-btn mini-reject";
        rejBtn.dataset.active = !h.accepted ? "true" : "false";
        rejBtn.textContent = "Reject";
        rejBtn.addEventListener("click", () => setHunkAccepted(h.id, false));

        const whyBtn = document.createElement("button");
        whyBtn.type = "button";
        whyBtn.className = "mini-action-btn";
        whyBtn.textContent = h.explaining ? "..." : "Why?";
        whyBtn.disabled = Boolean(h.explaining || h.reason);
        whyBtn.addEventListener("click", () => requestExplanations([h]));

        actions.appendChild(accBtn);
        actions.appendChild(rejBtn);
        actions.appendChild(whyBtn);
        top.appendChild(change);
        top.appendChild(actions);
        card.appendChild(top);

        if (h.reason || h.explaining) {
          const reasonP = document.createElement("p");
          reasonP.className = "correction-reason";
          reasonP.textContent = h.explaining ? "Fetching explanation from model..." : h.reason;
          card.appendChild(reasonP);
        }

        frag.appendChild(card);
      }
      els.correctionsList.appendChild(frag);
    }

    function renderSplitView() {
      els.splitOriginal.textContent = "";
      els.splitUpdated.textContent = "";

      if (state.segments.length === 0) {
        els.splitOriginal.textContent = state.sourceText;
        els.splitUpdated.textContent = state.outputText;
        return;
      }

      const origFrag = document.createDocumentFragment();
      const updFrag = document.createDocumentFragment();
      for (const seg of state.segments) {
        if (seg.kind === "eq") {
          origFrag.appendChild(document.createTextNode(seg.text));
          updFrag.appendChild(document.createTextNode(seg.text));
          continue;
        }
        if (seg.before) {
          if (seg.accepted) {
            const del = document.createElement("del");
            del.className = "diff-del";
            del.textContent = seg.before;
            origFrag.appendChild(del);
          } else {
            origFrag.appendChild(document.createTextNode(seg.before));
          }
        }
        if (seg.accepted) {
          if (seg.after) {
            const ins = document.createElement("ins");
            ins.className = "diff-ins";
            ins.textContent = seg.after;
            updFrag.appendChild(ins);
          }
        } else if (seg.before) {
          updFrag.appendChild(document.createTextNode(seg.before));
        }
      }
      els.splitOriginal.appendChild(origFrag);
      els.splitUpdated.appendChild(updFrag);
    }

    function renderDiffSummary() {
      if (state.streaming) {
        els.diffSummary.textContent = "Streaming...";
        return;
      }
      if (!state.outputText && state.segments.length === 0) {
        els.diffSummary.textContent = "Waiting";
        return;
      }
      if (state.mode === "explain") {
        els.diffSummary.textContent = "Explanation";
        return;
      }

      const hunks = getHunks();
      els.diffSummary.textContent = "";
      if (hunks.length === 0) {
        const badge = document.createElement("span");
        badge.className = "diff-count-clean";
        badge.textContent = "No changes needed";
        els.diffSummary.appendChild(badge);
        return;
      }
      let add = 0;
      let del = 0;
      for (const h of hunks) {
        if (!h.accepted) continue;
        if (h.after.trim()) add++;
        if (h.before.trim()) del++;
      }
      if (add === 0 && del === 0) {
        const badge = document.createElement("span");
        badge.className = "diff-count-clean";
        badge.textContent = "All rejected";
        els.diffSummary.appendChild(badge);
        return;
      }
      if (add > 0) {
        const addSpan = document.createElement("span");
        addSpan.className = "diff-count-add";
        addSpan.textContent = "+" + add;
        els.diffSummary.appendChild(addSpan);
      }
      if (del > 0) {
        const delSpan = document.createElement("span");
        delSpan.className = "diff-count-del";
        delSpan.textContent = "-" + del;
        els.diffSummary.appendChild(delSpan);
      }
    }

    function updateStats() {
      const src = state.sourceText;
      const trimmed = src.trim();
      const words = trimmed ? trimmed.split(/\s+/).length : 0;
      els.sourceStats.textContent = words + (words === 1 ? " word" : " words") + " · " + src.length + " chars";
      els.clearInputBtn.hidden = src.length === 0;
    }

    function setStatus(pillState, text) {
      els.statusPill.dataset.state = pillState;
      els.statusText.textContent = text;
    }

    function countSyllables(word) {
      const clean = word.toLowerCase().replace(/[^a-z]/g, "");
      if (!clean) return 0;
      if (clean.length <= 3) return 1;
      const stripped = clean.replace(/(?:[^laeiouy]es|ed|[^laeiouy]e)$/, "").replace(/^y/, "");
      const groups = stripped.match(/[aeiouy]{1,2}/g);
      return groups ? Math.max(1, groups.length) : 1;
    }

    function computeReadability(text) {
      const trimmed = (text || "").trim();
      if (!trimmed) {
        return { words: 0, sentences: 0, scoreLabel: "—", readTime: "0s" };
      }
      const wordList = trimmed.match(/[A-Za-z0-9'’-]+/g) || [];
      const words = wordList.length;
      const sentences = Math.max(1, (trimmed.match(/[.!?]+(?:\s|$)/g) || []).length);
      let syllables = 0;
      for (const w of wordList) {
        syllables += countSyllables(w);
      }
      const flesch = words > 0
        ? Math.round(206.835 - 1.015 * (words / sentences) - 84.6 * (syllables / words))
        : 0;
      const clamped = Math.max(0, Math.min(100, flesch));
      let band = "Standard";
      if (clamped >= 80) band = "Easy";
      else if (clamped >= 60) band = "Clear";
      else if (clamped < 40) band = "Dense";
      const secs = Math.max(1, Math.round((words / 220) * 60));
      const readTime = secs >= 60 ? Math.floor(secs / 60) + "m " + (secs % 60) + "s" : secs + "s";
      return {
        words,
        sentences,
        scoreLabel: words > 0 ? clamped + " · " + band : "—",
        readTime,
      };
    }

    function escapeRegExp(str) {
      return str.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    }

    function applyVocabReplacement(originalWord, replacement) {
      const target = state.outputText || state.sourceText;
      if (!target || !originalWord || !replacement) return;
      const re = new RegExp("\\b" + escapeRegExp(originalWord) + "\\b", "i");
      const match = target.match(re);
      if (!match) return;
      const matched = match[0];
      let nextRep = replacement;
      if (matched[0] === matched[0].toUpperCase() && matched[0] !== matched[0].toLowerCase()) {
        nextRep = replacement.charAt(0).toUpperCase() + replacement.slice(1);
      }
      const updated = target.replace(re, nextRep);
      if (updated === state.outputText) return;
      state.outputText = updated;
      state.aiOutputText = updated;
      rebuildModel(state.sourceText, updated);
      pushUndoPoint();
      syncControls();
    }

    function renderInsights() {
      const target = (state.outputText || "").trim() ? state.outputText : state.sourceText;
      const stats = computeReadability(target);
      els.metricWords.textContent = String(stats.words);
      els.metricSentences.textContent = String(stats.sentences);
      els.metricReadability.textContent = stats.scoreLabel;
      els.metricReadtime.textContent = stats.readTime;

      const hasText = target.trim().length > 0;
      els.analyzeWritingBtn.disabled = !hasText || state.streaming || state.analyzingInsights;
      els.analyzeWritingBtn.textContent = state.analyzingInsights ? "Analyzing..." : "Analyze Writing";

      if (!state.insights) {
        els.aiInsightsEmpty.hidden = false;
        els.aiInsightsContent.hidden = true;
        els.aiInsightsEmpty.textContent = state.analyzingInsights
          ? "Analyzing writing score, tone, clarity, and vocabulary..."
          : "Click Analyze Writing for AI writing score, tone detection, and vocabulary upgrades.";
        return;
      }

      els.aiInsightsEmpty.hidden = true;
      els.aiInsightsContent.hidden = false;
      els.aiScoreValue.textContent = String(state.insights.writingScore ?? "—");
      els.aiToneBadge.textContent = state.insights.tone || "Neutral";
      els.aiToneSummary.textContent = state.insights.toneSummary || "";
      els.aiClaritySummary.textContent = state.insights.clarity || "";

      els.vocabList.innerHTML = "";
      const vocab = Array.isArray(state.insights.vocabulary) ? state.insights.vocabulary : [];
      if (vocab.length === 0) {
        const none = document.createElement("div");
        none.className = "vocab-reason";
        none.textContent = "Vocabulary is concise and natural — no weak words flagged.";
        els.vocabList.appendChild(none);
        return;
      }

      for (const item of vocab) {
        const card = document.createElement("div");
        card.className = "vocab-card";

        const row = document.createElement("div");
        row.className = "vocab-word-row";
        const wordEl = document.createElement("span");
        wordEl.className = "vocab-word";
        wordEl.textContent = item.word;
        row.appendChild(wordEl);

        if (item.reason) {
          const reasonEl = document.createElement("span");
          reasonEl.className = "vocab-reason";
          reasonEl.textContent = item.reason;
          row.appendChild(reasonEl);
        }
        card.appendChild(row);

        const chips = document.createElement("div");
        chips.className = "vocab-chips";
        for (const alt of item.alternatives || []) {
          const btn = document.createElement("button");
          btn.type = "button";
          btn.className = "vocab-chip";
          btn.textContent = "→ " + alt;
          btn.title = "Replace \"" + item.word + "\" with \"" + alt + "\"";
          btn.addEventListener("click", () => applyVocabReplacement(item.word, alt));
          chips.appendChild(btn);
        }
        card.appendChild(chips);
        els.vocabList.appendChild(card);
      }
    }

    function requestInsights() {
      const target = (state.outputText || "").trim() ? state.outputText : state.sourceText;
      if (!target.trim() || state.streaming) return;
      state.error = "";
      state.analyzingInsights = true;
      syncControls();
      explainReqCounter++;
      postToHost({
        type: "analyzeWriting",
        requestId: "an-" + explainReqCounter,
        text: target,
        language: state.language || "",
      });
    }

    function applyTheme(pref) {
      state.themePref = pref || "system";
      let resolvedDark = false;
      if (state.themePref === "light" || state.themePref === "dark") {
        document.documentElement.setAttribute("data-theme", state.themePref);
        resolvedDark = state.themePref === "dark";
      } else {
        document.documentElement.removeAttribute("data-theme");
        resolvedDark = Boolean(window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches);
      }
      postToHost({ type: "theme", theme: resolvedDark ? "dark" : "light" });
    }

    function formatHistoryTime(iso) {
      if (!iso) return "";
      const d = new Date(iso);
      if (Number.isNaN(d.getTime())) return iso;
      return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
    }

    function renderHistory() {
      if (!els.historyList) return;
      els.historyList.innerHTML = "";
      const q = (state.historyFilter || "").trim().toLowerCase();
      const items = (state.history || []).filter((it) => {
        if (!q) return true;
        return (
          (it.sourceText || "").toLowerCase().includes(q) ||
          (it.outputText || "").toLowerCase().includes(q) ||
          (it.mode || "").toLowerCase().includes(q)
        );
      });
      if (items.length === 0) {
        const empty = document.createElement("div");
        empty.className = "history-empty";
        empty.textContent = q ? "No matching checks found." : "No saved checks yet. Run a check and it will appear here.";
        els.historyList.appendChild(empty);
        return;
      }

      for (const item of items) {
        const card = document.createElement("div");
        card.className = "history-card";

        const top = document.createElement("div");
        top.className = "history-card-top";

        const badges = document.createElement("div");
        badges.className = "history-badges";
        const modeBadge = document.createElement("span");
        modeBadge.className = "history-mode-badge";
        modeBadge.textContent = item.mode || "fix";
        badges.appendChild(modeBadge);

        if (item.language && item.language !== "Auto-detect") {
          const langBadge = document.createElement("span");
          langBadge.className = "history-time";
          langBadge.textContent = item.language;
          badges.appendChild(langBadge);
        }
        const timeEl = document.createElement("span");
        timeEl.className = "history-time";
        timeEl.textContent = formatHistoryTime(item.timestamp);
        badges.appendChild(timeEl);
        top.appendChild(badges);

        const actions = document.createElement("div");
        actions.className = "history-card-actions";
        const restoreBtn = document.createElement("button");
        restoreBtn.type = "button";
        restoreBtn.className = "mini-action-btn mini-accept";
        restoreBtn.textContent = "Restore";
        restoreBtn.addEventListener("click", () => {
          state.sourceText = item.sourceText || "";
          state.lastCheckedSource = state.sourceText;
          state.aiOutputText = item.outputText || "";
          state.outputText = item.outputText || "";
          if (item.mode) state.mode = item.mode;
          if (item.language) state.language = item.language;
          state.activePanel = "editor";
          state.view = state.mode === "explain" ? "clean" : "diff";
          rebuildModel(state.sourceText, state.outputText);
          resetUndoHistory();
          setStatus("done", "Restored from history");
          syncControls();
        });
        actions.appendChild(restoreBtn);

        const copyBtn = document.createElement("button");
        copyBtn.type = "button";
        copyBtn.className = "mini-action-btn";
        copyBtn.textContent = "Copy";
        copyBtn.addEventListener("click", () => {
          postToHost({ type: "copy", text: item.outputText || "" });
        });
        actions.appendChild(copyBtn);

        top.appendChild(actions);
        card.appendChild(top);

        const orig = document.createElement("div");
        orig.className = "history-snippet history-snippet-orig";
        orig.textContent = item.sourceText || "";
        card.appendChild(orig);

        const out = document.createElement("div");
        out.className = "history-snippet";
        out.textContent = item.outputText || "";
        card.appendChild(out);

        els.historyList.appendChild(card);
      }
    }

    function renderHotkeyCheckStatus() {
      if (!els.hotkeyCheckBadge) return;
      const st = state.hotkeyCheckState || "idle";
      els.hotkeyCheckBadge.dataset.state = st;
      if (st === "checking") {
        els.hotkeyCheckBadge.textContent = "Checking...";
      } else if (st === "ok") {
        els.hotkeyCheckBadge.textContent = state.hotkeyCheckMessage || "Available";
      } else if (st === "conflict") {
        els.hotkeyCheckBadge.textContent = "Already in use";
      } else if (st === "error") {
        els.hotkeyCheckBadge.textContent = "Invalid shortcut";
      } else {
        els.hotkeyCheckBadge.textContent = state.hotkeyConflict ? "Conflict" : "Active";
        if (state.hotkeyConflict) {
          els.hotkeyCheckBadge.dataset.state = "conflict";
        }
      }
      if (els.hotkeyCheckDetail) {
        if (state.hotkeyRecording) {
          els.hotkeyCheckDetail.textContent = "Press your modifier + key combination now (e.g. Ctrl+Shift+G)...";
        } else if (st === "conflict" || st === "error") {
          els.hotkeyCheckDetail.textContent = state.hotkeyCheckMessage || "This shortcut cannot be used. Pick another modifier + key combination.";
        } else if (st === "ok") {
          els.hotkeyCheckDetail.textContent = "Shortcut is free in Windows and ready to save.";
        } else {
          els.hotkeyCheckDetail.textContent = "Click Record and press a modifier + key (e.g. Ctrl+Shift+G), or type it and click Check.";
        }
      }
    }

    function setHotkeyRecording(active) {
      state.hotkeyRecording = Boolean(active);
      if (els.settingHotkey) {
        els.settingHotkey.classList.toggle("recording", state.hotkeyRecording);
      }
      if (els.hotkeyRecordBtn) {
        els.hotkeyRecordBtn.textContent = state.hotkeyRecording ? "Listening..." : "Record";
        els.hotkeyRecordBtn.setAttribute("aria-pressed", state.hotkeyRecording ? "true" : "false");
      }
      renderHotkeyCheckStatus();
    }

    function requestHotkeyCheck(spec) {
      const cleaned = (spec || "").trim();
      if (!cleaned) {
        state.hotkeyCheckState = "error";
        state.hotkeyCheckMessage = "Enter a shortcut such as Ctrl+Alt+G";
        renderHotkeyCheckStatus();
        return;
      }
      state.hotkeyCheckState = "checking";
      state.hotkeyCheckMessage = "";
      renderHotkeyCheckStatus();
      postToHost({ type: "checkHotkey", hotkey: cleaned });
    }

    function formatCapturedHotkey(e) {
      const ignored = ["Control", "Shift", "Alt", "Meta", "OS"];
      if (ignored.includes(e.key)) return "";
      const mods = [];
      if (e.ctrlKey) mods.push("Ctrl");
      if (e.shiftKey) mods.push("Shift");
      if (e.altKey) mods.push("Alt");
      if (e.metaKey) mods.push("Win");
      if (mods.length === 0) return null;

      let key = "";
      if (e.code && /^Key([A-Z])$/.test(e.code)) {
        key = e.code.slice(3);
      } else if (e.code && /^Digit([0-9])$/.test(e.code)) {
        key = e.code.slice(5);
      } else if (e.code === "Space" || e.key === " ") {
        key = "Space";
      } else if (/^F([1-9]|1[0-2])$/i.test(e.key)) {
        key = e.key.toUpperCase();
      } else if (e.key && /^[a-z0-9]$/i.test(e.key)) {
        key = e.key.toUpperCase();
      } else {
        return null;
      }
      return mods.concat(key).join("+");
    }

    function syncAIProviderFields() {
      const isCloud = els.settingAIProvider ? els.settingAIProvider.value === "cloud" : true;
      if (els.rowSettingAIKey) {
        els.rowSettingAIKey.hidden = !isCloud;
      }
    }

    function renderAITestStatus() {
      if (!els.aiTestBadge) return;
      if (state.aiTesting) {
        els.aiTestBadge.dataset.state = "checking";
        els.aiTestBadge.textContent = "Testing...";
        if (els.aiTestDetail) els.aiTestDetail.textContent = "Connecting to AI endpoint...";
      } else if (state.aiConnected) {
        els.aiTestBadge.dataset.state = "ok";
        els.aiTestBadge.textContent = "Connected";
        if (els.aiTestDetail) els.aiTestDetail.textContent = "AI connection verified and ready.";
      } else {
        els.aiTestBadge.dataset.state = "error";
        els.aiTestBadge.textContent = "Disconnected";
        if (els.aiTestDetail) els.aiTestDetail.textContent = state.aiError ? `Error: ${state.aiError}` : "Could not reach AI provider.";
      }
    }

    function renderOnboardStatus(override) {
      if (!els.onboardStatusBanner || !els.onboardStatusText) return;
      if (override === "testing" || state.aiTesting) {
        els.onboardStatusBanner.hidden = false;
        els.onboardStatusBanner.dataset.state = "checking";
        els.onboardStatusText.textContent = "Connecting & testing AI endpoint...";
      } else if (state.aiConnected) {
        els.onboardStatusBanner.hidden = false;
        els.onboardStatusBanner.dataset.state = "ok";
        els.onboardStatusText.textContent = "Connected successfully! Launching editor...";
        setTimeout(() => {
          state.activePanel = "editor";
          syncControls();
        }, 700);
      } else if (state.aiError) {
        els.onboardStatusBanner.hidden = false;
        els.onboardStatusBanner.dataset.state = "error";
        els.onboardStatusText.textContent = `Connection failed: ${state.aiError}`;
      } else {
        els.onboardStatusBanner.hidden = true;
      }
    }

    function populateSettingsForm() {
      setHotkeyRecording(false);
      if (els.settingDefaultMode) els.settingDefaultMode.value = state.defaultMode || "fix";
      if (els.settingLanguage) els.settingLanguage.value = state.language || "Auto-detect";
      if (els.settingTheme) els.settingTheme.value = state.themePref || "system";
      if (els.settingWatchEnabled) els.settingWatchEnabled.checked = Boolean(state.watchEnabled);
      if (els.settingHotkey) els.settingHotkey.value = state.hotkeyLabel || "Ctrl+Alt+G";
      state.hotkeyCheckState = state.hotkeyConflict ? "conflict" : "idle";
      state.hotkeyCheckMessage = state.hotkeyConflict ? (state.hotkeyNotice || "Shortcut already used by another app.") : "";
      renderHotkeyCheckStatus();
      if (els.settingExcludedApps) {
        els.settingExcludedApps.value = (state.excludedApps || []).join("\n");
      }

      if (els.settingAIProvider) {
        els.settingAIProvider.value = state.aiProviderType || "cloud";
      }
      if (els.settingAIBaseURL) {
        els.settingAIBaseURL.value = state.aiBaseURL || (state.aiProviderType === "ollama" ? "http://127.0.0.1:11434" : "https://fluenta-proxy.kumarvishnu610.workers.dev");
      }
      if (els.settingAIModel) {
        let currentModel = state.aiModel;
        if (currentModel === "llama-3.3-70b-versatile") {
          currentModel = "openai/gpt-oss-120b";
        }
        els.settingAIModel.value = currentModel || (state.aiProviderType === "ollama" ? "qwen3.5:4b" : "openai/gpt-oss-120b");
      }
      if (els.settingAIApiKey) {
        els.settingAIApiKey.value = state.aiApiKey || "";
      }
      syncAIProviderFields();
      renderAITestStatus();
    }

    function openPanel(panel) {
      state.activePanel = state.activePanel === panel ? "editor" : panel;
      if (state.activePanel === "settings") {
        populateSettingsForm();
      } else if (state.activePanel === "history") {
        renderHistory();
      }
      syncControls();
    }

    function syncControls() {
      if (els.sourceInput.value !== state.sourceText) {
        els.sourceInput.value = state.sourceText;
      }
      if (els.outputInput.value !== state.outputText) {
        els.outputInput.value = state.outputText;
      }

      const displayModel = state.modelName || (state.aiProviderType === "cloud" ? "gpt-oss-120b" : "qwen3.5:4b");
      els.modelTag.textContent = displayModel.replace("openai/", "");

      if (els.aiStatusBadge && els.aiStatusBadgeText) {
        if (state.aiTesting) {
          els.aiStatusBadge.dataset.state = "testing";
          els.aiStatusBadgeText.textContent = "Testing...";
        } else if (state.aiConnected) {
          els.aiStatusBadge.dataset.state = "connected";
          els.aiStatusBadgeText.textContent = "Online";
        } else {
          els.aiStatusBadge.dataset.state = "disconnected";
          els.aiStatusBadgeText.textContent = "Offline";
        }
      }

      if (els.watchToggleBtn) {
        const watchActive = state.watchEnabled && !state.paused;
        els.watchToggleBtn.setAttribute("aria-pressed", watchActive ? "true" : "false");
        els.watchToggleLabel.textContent = state.watchEnabled ? "Watcher ON" : "Watcher OFF";
      }
      if (els.pauseToggleBtn) {
        els.pauseToggleBtn.setAttribute("aria-pressed", state.paused ? "true" : "false");
        els.pauseToggleLabel.textContent = state.paused ? "Paused" : "Active";
      }
      if (els.historyToggleBtn) {
        els.historyToggleBtn.setAttribute("aria-pressed", state.activePanel === "history" ? "true" : "false");
      }
      if (els.settingsToggleBtn) {
        els.settingsToggleBtn.setAttribute("aria-pressed", state.activePanel === "settings" ? "true" : "false");
      }
      if (els.languageSelect && els.languageSelect.value !== (state.language || "Auto-detect")) {
        els.languageSelect.value = state.language || "Auto-detect";
      }

      const inEditor = state.activePanel === "editor";
      if (els.toolbarRow) els.toolbarRow.hidden = !inEditor;
      if (els.paneSource) els.paneSource.hidden = !inEditor;
      if (els.paneResult) els.paneResult.hidden = !inEditor;
      if (els.hotkeyTag) {
        els.hotkeyTag.textContent = state.paused
          ? "Paused"
          : (state.hotkeyConflict ? "Hotkey conflict" : (state.hotkeyLabel || "Ctrl+Alt+G"));
      }
      if (els.hotkeyNoticeBanner && els.hotkeyNoticeMessage) {
        const showNotice = Boolean(state.hotkeyNotice) && inEditor;
        els.hotkeyNoticeBanner.hidden = !showNotice;
        if (showNotice) {
          els.hotkeyNoticeMessage.textContent = state.hotkeyNotice;
        }
      }
      if (els.historyPanel) els.historyPanel.hidden = state.activePanel !== "history";
      if (els.settingsPanel) els.settingsPanel.hidden = state.activePanel !== "settings";
      if (els.onboardingPanel) els.onboardingPanel.hidden = state.activePanel !== "onboarding";
      if (els.privacyPanel) els.privacyPanel.hidden = state.activePanel !== "privacy";

      for (const btn of els.modeButtons) {
        btn.setAttribute("aria-pressed", btn.dataset.mode === state.mode ? "true" : "false");
      }

      els.tabDiff.setAttribute("aria-selected", state.view === "diff" ? "true" : "false");
      els.tabList.setAttribute("aria-selected", state.view === "list" ? "true" : "false");
      els.tabSplit.setAttribute("aria-selected", state.view === "split" ? "true" : "false");
      els.tabClean.setAttribute("aria-selected", state.view === "clean" ? "true" : "false");
      els.tabInsights.setAttribute("aria-selected", state.view === "insights" ? "true" : "false");

      els.diffView.hidden = state.view !== "diff";
      els.correctionsView.hidden = state.view !== "list";
      els.splitView.hidden = state.view !== "split";
      els.outputInput.hidden = state.view !== "clean";
      els.insightsView.hidden = state.view !== "insights";

      els.undoBtn.disabled = state.undoStack.length === 0 || state.streaming;
      els.redoBtn.disabled = state.redoStack.length === 0 || state.streaming;

      if (state.error) {
        els.errorBanner.hidden = false;
        els.errorMessage.textContent = state.error;
      } else {
        els.errorBanner.hidden = true;
        els.errorMessage.textContent = "";
      }

      const hasOutput = state.outputText.trim().length > 0;
      els.copyBtn.disabled = !hasOutput || state.streaming;
      els.replaceBtn.disabled = !hasOutput || state.streaming;

      if (state.replaceAllMode) {
        els.replaceLabel.textContent = "Replace All";
      } else if (state.canReplaceTarget) {
        els.replaceLabel.textContent = "Replace Selection";
      } else {
        els.replaceLabel.textContent = "Apply & Copy";
      }

      els.recheckLabel.textContent = state.streaming ? "Stop" : "Re-check";
      els.recheckBtn.disabled = !state.streaming && state.sourceText.trim().length === 0;

      updateStats();
      renderDiffSummary();
      renderDiffView();
      renderInspectorBar();
      renderCorrectionsList();
      renderSplitView();
      renderInsights();
    }

    function triggerRecheck() {
      const trimmed = state.sourceText.trim();
      if (!trimmed) {
        state.error = "Enter or paste text above to check.";
        setStatus("error", "Empty input");
        syncControls();
        els.sourceInput.focus();
        return;
      }
      state.error = "";
      state.lastCheckedSource = state.sourceText;
      postToHost({
        type: "recheck",
        text: state.sourceText,
        mode: state.mode,
      });
    }

    function triggerReplace() {
      if (state.streaming || !state.outputText.trim()) return;
      postToHost({
        type: "replace",
        text: state.outputText,
      });
    }

    function triggerCopy() {
      if (state.streaming || !state.outputText.trim()) return;
      postToHost({
        type: "copy",
        text: state.outputText,
      });
      els.copyLabel.textContent = "Copied";
      if (copyResetTimer) clearTimeout(copyResetTimer);
      copyResetTimer = setTimeout(() => {
        els.copyLabel.textContent = "Copy";
      }, 1400);
    }

    function triggerCancel() {
      postToHost({ type: "cancel" });
    }

    window.__grammarApp = {
      onHostEvent(evt) {
        if (!evt || typeof evt !== "object") return;
        switch (evt.type) {
          case "config":
            if (evt.hotkeyLabel) state.hotkeyLabel = evt.hotkeyLabel;
            if (typeof evt.hotkeyConflict === "boolean") state.hotkeyConflict = evt.hotkeyConflict;
            if (typeof evt.hotkeyNotice === "string") state.hotkeyNotice = evt.hotkeyNotice;
            if (evt.modelName) state.modelName = evt.modelName;
            if (evt.mode) state.mode = evt.mode;
            if (evt.defaultMode) state.defaultMode = evt.defaultMode;
            if (typeof evt.watchEnabled === "boolean") state.watchEnabled = evt.watchEnabled;
            if (typeof evt.paused === "boolean") state.paused = evt.paused;
            if (evt.language) state.language = evt.language;
            if (evt.theme) applyTheme(evt.theme);
            if (Array.isArray(evt.excludedApps)) state.excludedApps = evt.excludedApps;
            if (Array.isArray(evt.history)) state.history = evt.history;
            if (typeof evt.aiConnected === "boolean") state.aiConnected = evt.aiConnected;
            if (typeof evt.aiError === "string") state.aiError = evt.aiError;
            if (evt.aiProviderType) state.aiProviderType = evt.aiProviderType;
            if (evt.aiBaseURL) state.aiBaseURL = evt.aiBaseURL;
            if (evt.aiModel) state.aiModel = evt.aiModel;
            if (typeof evt.aiApiKey === "string") state.aiApiKey = evt.aiApiKey;
            if (!state.aiConnected && state.activePanel === "editor") {
              state.activePanel = "onboarding";
            }
            if (state.activePanel === "history") renderHistory();
            if (state.activePanel === "settings") populateSettingsForm();
            if (state.activePanel === "onboarding") renderOnboardStatus();
            syncControls();
            break;

          case "aiTestResult":
            state.aiTesting = false;
            state.aiConnected = Boolean(evt.aiConnected);
            state.aiError = evt.aiError || "";
            renderAITestStatus();
            renderOnboardStatus();
            syncControls();
            break;

          case "hotkeyCheck":
            if (evt.hotkeyAvailable) {
              state.hotkeyCheckState = "ok";
              state.hotkeyCheckMessage = evt.hotkeyLabel ? `Available (${evt.hotkeyLabel})` : "Available";
              if (evt.hotkeyLabel && els.settingHotkey) {
                els.settingHotkey.value = evt.hotkeyLabel;
              }
            } else if (evt.hotkeyConflict) {
              state.hotkeyCheckState = "conflict";
              state.hotkeyCheckMessage = evt.error || "Already in use by another app or Windows";
              if (state.activePanel !== "settings") {
                state.hotkeyNotice = state.hotkeyCheckMessage;
                syncControls();
              }
            } else {
              state.hotkeyCheckState = "error";
              state.hotkeyCheckMessage = evt.error || "Invalid shortcut";
            }
            renderHotkeyCheckStatus();
            break;

          case "history":
            state.history = Array.isArray(evt.history) ? evt.history : [];
            if (state.activePanel === "history") renderHistory();
            break;

          case "chunkProgress":
            setStatus("streaming", evt.chunkInfo || "Checking...");
            break;

          case "statusSync":
            if (evt.hotkeyLabel) state.hotkeyLabel = evt.hotkeyLabel;
            if (typeof evt.hotkeyConflict === "boolean") state.hotkeyConflict = evt.hotkeyConflict;
            if (typeof evt.hotkeyNotice === "string") state.hotkeyNotice = evt.hotkeyNotice;
            if (typeof evt.watchEnabled === "boolean") state.watchEnabled = evt.watchEnabled;
            if (typeof evt.paused === "boolean") state.paused = evt.paused;
            syncControls();
            break;

          case "open":
            state.activePanel = "editor";
            state.sourceText = evt.sourceText || "";
            state.lastCheckedSource = state.sourceText;
            state.aiOutputText = evt.outputText || "";
            state.outputText = evt.outputText || "";
            if (evt.mode) state.mode = evt.mode;
            state.replaceAllMode = Boolean(evt.replaceAllMode);
            state.canReplaceTarget = Boolean(evt.canReplaceTarget);
            state.hotkeyLabel = evt.hotkeyLabel || state.hotkeyLabel;
            state.modelName = evt.modelName || state.modelName;
            state.streaming = Boolean(evt.streaming);
            state.error = evt.error || "";
            state.explainingAll = false;
            state.view = state.mode === "explain" ? "clean" : "diff";
            rebuildModel(state.sourceText, state.streaming ? "" : state.aiOutputText);
            resetUndoHistory();

            if (state.streaming) {
              state.startedAt = performance.now();
              setStatus("streaming", "Checking...");
            } else if (state.error) {
              setStatus("error", "Check failed");
            } else if (state.outputText.trim()) {
              setStatus("done", "Suggestion ready");
            } else {
              setStatus("idle", "Ready");
            }
            syncControls();
            if (!state.sourceText.trim()) {
              els.sourceInput.focus();
            }
            break;

          case "start":
            state.aiOutputText = "";
            state.outputText = "";
            state.segments = [];
            state.selectedHunkId = "";
            state.error = "";
            state.explainingAll = false;
            state.insights = null;
            state.analyzingInsights = false;
            state.streaming = true;
            state.startedAt = performance.now();
            if (evt.mode) state.mode = evt.mode;
            resetUndoHistory();
            setStatus("streaming", "Checking...");
            syncControls();
            break;

          case "chunk":
            if (!state.streaming) {
              state.streaming = true;
              state.startedAt = state.startedAt || performance.now();
            }
            state.outputText += evt.text || "";
            syncControls();
            break;

          case "done": {
            state.streaming = false;
            state.error = "";
            if (typeof evt.text === "string") {
              state.aiOutputText = evt.text;
              state.outputText = evt.text;
            } else {
              state.aiOutputText = state.outputText;
            }
            rebuildModel(state.sourceText, state.aiOutputText);
            resetUndoHistory();
            const elapsedMs = state.startedAt ? Math.round(performance.now() - state.startedAt) : 0;
            setStatus("done", elapsedMs > 0 ? "Done · " + elapsedMs + " ms" : "Done");
            syncControls();
            break;
          }

          case "explanations": {
            state.explainingAll = false;
            for (const h of getHunks()) {
              h.explaining = false;
            }
            if (evt.error) {
              state.error = "Explanation failed: " + evt.error;
              syncControls();
              break;
            }
            if (Array.isArray(evt.explanations)) {
              for (const item of evt.explanations) {
                const h = findHunk(item.id);
                if (h) {
                  h.category = item.category || "grammar";
                  h.reason = item.reason || "";
                  state.explainCache.set(h.before + "\0" + h.after, {
                    category: h.category,
                    reason: h.reason,
                  });
                }
              }
            }
            syncControls();
            break;
          }

          case "insights":
            state.analyzingInsights = false;
            if (evt.error) {
              state.error = "Analysis failed: " + evt.error;
              syncControls();
              break;
            }
            if (evt.insights) {
              state.insights = evt.insights;
            }
            syncControls();
            break;

          case "error":
            state.streaming = false;
            state.explainingAll = false;
            state.analyzingInsights = false;
            state.error = evt.error || "Unknown error";
            setStatus("error", "Error");
            syncControls();
            break;
        }
      },
    };

    els.modeButtons.forEach((btn) => {
      btn.addEventListener("click", () => {
        const nextMode = btn.dataset.mode;
        if (!nextMode || nextMode === state.mode) return;
        state.mode = nextMode;
        state.view = state.mode === "explain" ? "clean" : state.view;
        postToHost({ type: "mode", mode: state.mode });
        if (state.sourceText.trim()) {
          triggerRecheck();
        } else {
          syncControls();
        }
      });
    });

    [els.tabDiff, els.tabList, els.tabSplit, els.tabClean, els.tabInsights].forEach((tab) => {
      tab.addEventListener("click", () => {
        state.view = tab.dataset.view;
        syncControls();
        if (state.view === "clean") els.outputInput.focus();
      });
    });

    if (els.watchToggleBtn) {
      els.watchToggleBtn.addEventListener("click", () => {
        state.watchEnabled = !state.watchEnabled;
        postToHost({ type: "setWatch", enabled: state.watchEnabled });
        syncControls();
      });
    }

    if (els.pauseToggleBtn) {
      els.pauseToggleBtn.addEventListener("click", () => {
        state.paused = !state.paused;
        postToHost({ type: "setPause", enabled: state.paused });
        syncControls();
      });
    }

    els.analyzeWritingBtn.addEventListener("click", requestInsights);

    if (els.historyToggleBtn) {
      els.historyToggleBtn.addEventListener("click", () => openPanel("history"));
    }
    if (els.historyBackBtn) {
      els.historyBackBtn.addEventListener("click", () => {
        state.activePanel = "editor";
        syncControls();
      });
    }
    if (els.historySearch) {
      els.historySearch.addEventListener("input", () => {
        state.historyFilter = els.historySearch.value || "";
        renderHistory();
      });
    }
    if (els.historyClearBtn) {
      els.historyClearBtn.addEventListener("click", () => {
        state.history = [];
        renderHistory();
        postToHost({ type: "clearHistory" });
      });
    }

    if (els.settingsToggleBtn) {
      els.settingsToggleBtn.addEventListener("click", () => openPanel("settings"));
    }
    if (els.settingsBackBtn) {
      els.settingsBackBtn.addEventListener("click", () => {
        state.activePanel = "editor";
        syncControls();
      });
    }
    if (els.settingsCancelBtn) {
      els.settingsCancelBtn.addEventListener("click", () => {
        state.activePanel = "editor";
        syncControls();
      });
    }
    if (els.btnViewPrivacy) {
      els.btnViewPrivacy.addEventListener("click", () => {
        state.activePanel = "privacy";
        syncControls();
      });
    }
    if (els.btnPrivacyBack) {
      els.btnPrivacyBack.addEventListener("click", () => {
        state.activePanel = "settings";
        syncControls();
      });
    }
    if (els.hotkeyNoticeSettingsBtn) {
      els.hotkeyNoticeSettingsBtn.addEventListener("click", () => {
        state.activePanel = "settings";
        populateSettingsForm();
        syncControls();
        if (els.settingHotkey) els.settingHotkey.focus();
      });
    }
    if (els.hotkeyNoticeDismissBtn) {
      els.hotkeyNoticeDismissBtn.addEventListener("click", () => {
        state.hotkeyNotice = "";
        syncControls();
      });
    }
    if (els.hotkeyRecordBtn) {
      els.hotkeyRecordBtn.addEventListener("click", () => {
        const next = !state.hotkeyRecording;
        setHotkeyRecording(next);
        if (next && els.settingHotkey) {
          els.settingHotkey.focus();
        }
      });
    }
    if (els.hotkeyCheckBtn) {
      els.hotkeyCheckBtn.addEventListener("click", () => {
        setHotkeyRecording(false);
        requestHotkeyCheck(els.settingHotkey ? els.settingHotkey.value : "");
      });
    }
    if (els.hotkeyDefaultBtn) {
      els.hotkeyDefaultBtn.addEventListener("click", () => {
        setHotkeyRecording(false);
        if (els.settingHotkey) els.settingHotkey.value = "Ctrl+Alt+G";
        requestHotkeyCheck("Ctrl+Alt+G");
      });
    }
    if (els.settingHotkey) {
      els.settingHotkey.addEventListener("keydown", (e) => {
        if (!state.hotkeyRecording) return;
        e.preventDefault();
        e.stopPropagation();
        if (e.key === "Escape") {
          setHotkeyRecording(false);
          return;
        }
        const captured = formatCapturedHotkey(e);
        if (captured === "") return; // lone modifier pressed
        if (captured === null) {
          state.hotkeyCheckState = "error";
          state.hotkeyCheckMessage = "Use Ctrl, Shift, Alt, or Win + A-Z, 0-9, Space, or F1-F12";
          renderHotkeyCheckStatus();
          return;
        }
        els.settingHotkey.value = captured;
        setHotkeyRecording(false);
        requestHotkeyCheck(captured);
      });
      els.settingHotkey.addEventListener("input", () => {
        if (hotkeyCheckTimer) clearTimeout(hotkeyCheckTimer);
        hotkeyCheckTimer = setTimeout(() => {
          requestHotkeyCheck(els.settingHotkey.value);
        }, 260);
      });
    }

    if (els.settingsSaveBtn) {
      els.settingsSaveBtn.addEventListener("click", () => {
        if (state.hotkeyCheckState === "conflict" || state.hotkeyCheckState === "error") {
          renderHotkeyCheckStatus();
          if (els.settingHotkey) els.settingHotkey.focus();
          return;
        }
        const nextMode = els.settingDefaultMode ? els.settingDefaultMode.value : state.defaultMode;
        const nextLang = els.settingLanguage ? els.settingLanguage.value : state.language;
        const nextTheme = els.settingTheme ? els.settingTheme.value : state.themePref;
        const nextWatch = els.settingWatchEnabled ? Boolean(els.settingWatchEnabled.checked) : state.watchEnabled;
        const nextHotkey = els.settingHotkey ? els.settingHotkey.value.trim() : state.hotkeyLabel;
        const rawExcl = els.settingExcludedApps ? els.settingExcludedApps.value : "";
        const nextExcl = rawExcl
          .split(/[\r\n,]+/)
          .map((s) => s.trim())
          .filter(Boolean);

        const nextAIProvider = els.settingAIProvider ? els.settingAIProvider.value : state.aiProviderType;
        const nextAIBaseURL = els.settingAIBaseURL ? els.settingAIBaseURL.value.trim() : state.aiBaseURL;
        const nextAIModel = els.settingAIModel ? els.settingAIModel.value.trim() : state.aiModel;
        const nextAIApiKey = els.settingAIApiKey ? els.settingAIApiKey.value.trim() : state.aiApiKey;

        state.defaultMode = nextMode;
        state.mode = nextMode;
        state.language = nextLang;
        state.watchEnabled = nextWatch;
        if (nextHotkey) state.hotkeyLabel = nextHotkey;
        state.excludedApps = nextExcl;
        state.aiProviderType = nextAIProvider;
        state.aiBaseURL = nextAIBaseURL;
        state.aiModel = nextAIModel;
        state.aiApiKey = nextAIApiKey;

        applyTheme(nextTheme);
        state.activePanel = "editor";
        postToHost({
          type: "saveSettings",
          settings: {
            defaultMode: nextMode,
            language: nextLang,
            theme: nextTheme,
            watchEnabled: nextWatch,
            paused: state.paused,
            hotkey: nextHotkey,
            excludedApps: nextExcl,
            aiProviderType: nextAIProvider,
            aiBaseURL: nextAIBaseURL,
            aiModel: nextAIModel,
            aiApiKey: nextAIApiKey,
          },
        });
        syncControls();
      });
    }

    if (els.aiStatusBadge) {
      els.aiStatusBadge.addEventListener("click", () => {
        openPanel("settings");
      });
    }

    if (els.btnToggleSettingKey && els.settingAIApiKey) {
      els.btnToggleSettingKey.addEventListener("click", () => {
        const isPass = els.settingAIApiKey.type === "password";
        els.settingAIApiKey.type = isPass ? "text" : "password";
        els.btnToggleSettingKey.textContent = isPass ? "Hide" : "Show";
      });
    }

    if (els.btnToggleOnboardKey && els.onboardCloudKey) {
      els.btnToggleOnboardKey.addEventListener("click", () => {
        const isPass = els.onboardCloudKey.type === "password";
        els.onboardCloudKey.type = isPass ? "text" : "password";
        els.btnToggleOnboardKey.textContent = isPass ? "Hide" : "Show";
      });
    }

    if (els.settingAIProvider) {
      els.settingAIProvider.addEventListener("change", () => {
        syncAIProviderFields();
        if (els.settingAIProvider.value === "cloud") {
          if (!els.settingAIBaseURL.value || els.settingAIBaseURL.value.includes("127.0.0.1")) {
            els.settingAIBaseURL.value = "https://fluenta-proxy.kumarvishnu610.workers.dev";
          }
          if (!els.settingAIModel.value || els.settingAIModel.value.includes("qwen") || els.settingAIModel.value.includes("llama")) {
            els.settingAIModel.value = "openai/gpt-oss-120b";
          }
        } else {
          if (!els.settingAIBaseURL.value || els.settingAIBaseURL.value.includes("workers.dev") || els.settingAIBaseURL.value.includes("groq.com")) {
            els.settingAIBaseURL.value = "http://127.0.0.1:11434";
          }
          if (!els.settingAIModel.value || els.settingAIModel.value.includes("gpt") || els.settingAIModel.value.includes("llama")) {
            els.settingAIModel.value = "qwen3.5:4b";
          }
        }
      });
    }

    if (els.btnPresetGroq) {
      els.btnPresetGroq.addEventListener("click", () => {
        if (els.settingAIProvider) els.settingAIProvider.value = "cloud";
        if (els.settingAIBaseURL) els.settingAIBaseURL.value = "https://fluenta-proxy.kumarvishnu610.workers.dev";
        if (els.settingAIModel) els.settingAIModel.value = "openai/gpt-oss-120b";
        syncAIProviderFields();
      });
    }

    if (els.btnPresetOllama) {
      els.btnPresetOllama.addEventListener("click", () => {
        if (els.settingAIProvider) els.settingAIProvider.value = "ollama";
        if (els.settingAIBaseURL) els.settingAIBaseURL.value = "http://127.0.0.1:11434";
        if (els.settingAIModel) els.settingAIModel.value = "qwen3.5:4b";
        syncAIProviderFields();
      });
    }

    if (els.btnTestAI) {
      els.btnTestAI.addEventListener("click", () => {
        state.aiTesting = true;
        renderAITestStatus();
        postToHost({
          type: "testAIConnection",
          aiProviderType: els.settingAIProvider ? els.settingAIProvider.value : "cloud",
          aiBaseURL: els.settingAIBaseURL ? els.settingAIBaseURL.value.trim() : "",
          aiModel: els.settingAIModel ? els.settingAIModel.value.trim() : "",
          aiApiKey: els.settingAIApiKey ? els.settingAIApiKey.value.trim() : "",
        });
      });
    }

    if (els.btnOnboardCloud) {
      els.btnOnboardCloud.addEventListener("click", () => {
        const key = (els.onboardCloudKey ? els.onboardCloudKey.value.trim() : "") || state.aiApiKey;
        state.aiTesting = true;
        state.aiProviderType = "cloud";
        state.aiBaseURL = "https://fluenta-proxy.kumarvishnu610.workers.dev";
        state.aiModel = "openai/gpt-oss-120b";
        state.aiApiKey = key;
        renderOnboardStatus("testing");
        postToHost({
          type: "saveSettings",
          settings: {
            defaultMode: state.defaultMode,
            language: state.language,
            theme: state.themePref,
            watchEnabled: state.watchEnabled,
            paused: state.paused,
            hotkey: state.hotkeyLabel,
            excludedApps: state.excludedApps,
            aiProviderType: "cloud",
            aiBaseURL: "https://fluenta-proxy.kumarvishnu610.workers.dev",
            aiModel: "openai/gpt-oss-120b",
            aiApiKey: key,
          },
        });
      });
    }

    if (els.btnOnboardOllama) {
      els.btnOnboardOllama.addEventListener("click", () => {
        const url = (els.onboardOllamaUrl ? els.onboardOllamaUrl.value.trim() : "") || "http://127.0.0.1:11434";
        state.aiTesting = true;
        state.aiProviderType = "ollama";
        state.aiBaseURL = url;
        state.aiModel = "qwen3.5:4b";
        renderOnboardStatus("testing");
        postToHost({
          type: "saveSettings",
          settings: {
            defaultMode: state.defaultMode,
            language: state.language,
            theme: state.themePref,
            watchEnabled: state.watchEnabled,
            paused: state.paused,
            hotkey: state.hotkeyLabel,
            excludedApps: state.excludedApps,
            aiProviderType: "ollama",
            aiBaseURL: url,
            aiModel: "qwen3.5:4b",
          },
        });
      });
    }

    if (els.btnOnboardSkip) {
      els.btnOnboardSkip.addEventListener("click", () => {
        state.activePanel = "editor";
        syncControls();
      });
    }

    if (els.languageSelect) {
      els.languageSelect.addEventListener("change", () => {
        const nextLang = els.languageSelect.value || "Auto-detect";
        state.language = nextLang;
        postToHost({ type: "setLanguage", language: nextLang });
        if (state.sourceText.trim()) {
          triggerRecheck();
        }
      });
    }

    els.inspectorAcceptBtn.addEventListener("click", () => {
      if (state.selectedHunkId) setHunkAccepted(state.selectedHunkId, true);
    });
    els.inspectorRejectBtn.addEventListener("click", () => {
      if (state.selectedHunkId) setHunkAccepted(state.selectedHunkId, false);
    });
    els.inspectorWhyBtn.addEventListener("click", () => {
      const sel = findHunk(state.selectedHunkId);
      if (sel) requestExplanations([sel]);
    });

    els.explainAllBtn.addEventListener("click", () => requestExplanations(getHunks()));
    els.acceptAllBtn.addEventListener("click", () => setAllHunksAccepted(true));
    els.rejectAllBtn.addEventListener("click", () => setAllHunksAccepted(false));

    els.undoBtn.addEventListener("click", undo);
    els.redoBtn.addEventListener("click", redo);

    els.sourceInput.addEventListener("input", () => {
      state.sourceText = els.sourceInput.value;
      updateStats();
      els.recheckBtn.disabled = !state.streaming && state.sourceText.trim().length === 0;
    });

    els.outputInput.addEventListener("input", () => {
      state.outputText = els.outputInput.value;
      state.aiOutputText = state.outputText;
      rebuildModel(state.sourceText, state.outputText);
      const hasOutput = state.outputText.trim().length > 0;
      els.copyBtn.disabled = !hasOutput || state.streaming;
      els.replaceBtn.disabled = !hasOutput || state.streaming;
      renderDiffSummary();
      renderCorrectionsList();

      if (editUndoTimer) clearTimeout(editUndoTimer);
      editUndoTimer = setTimeout(() => {
        pushUndoPoint();
        els.undoBtn.disabled = state.undoStack.length === 0 || state.streaming;
        els.redoBtn.disabled = state.redoStack.length === 0 || state.streaming;
      }, 350);
    });

    els.clearInputBtn.addEventListener("click", () => {
      if (state.streaming) {
        postToHost({ type: "stop" });
        state.streaming = false;
      }
      state.sourceText = "";
      state.aiOutputText = "";
      state.outputText = "";
      state.segments = [];
      state.error = "";
      resetUndoHistory();
      setStatus("idle", "Ready");
      syncControls();
      els.sourceInput.focus();
    });

    els.recheckBtn.addEventListener("click", () => {
      if (state.streaming) {
        postToHost({ type: "stop" });
        state.streaming = false;
        setStatus("idle", "Stopped");
        syncControls();
        return;
      }
      triggerRecheck();
    });

    els.retryBtn.addEventListener("click", triggerRecheck);
    els.copyBtn.addEventListener("click", triggerCopy);
    els.replaceBtn.addEventListener("click", triggerReplace);
    els.cancelBtn.addEventListener("click", triggerCancel);

    els.themeBtn.addEventListener("click", () => {
      const current = document.documentElement.getAttribute("data-theme");
      const prefersDark = window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches;
      const effective = current || (prefersDark ? "dark" : "light");
      const next = effective === "dark" ? "light" : "dark";
      applyTheme(next);
    });

    if (els.windowHeader) {
      els.windowHeader.addEventListener("mousedown", (e) => {
        if (e.button !== 0) return;
        if (e.target && typeof e.target.closest === "function" && e.target.closest("button, a, input, select")) {
          return;
        }
        postToHost({ type: "drag" });
      });
    }

    document.querySelectorAll(".resize-edge").forEach((el) => {
      el.addEventListener("mousedown", (e) => {
        if (e.button !== 0) return;
        const edge = el.dataset && el.dataset.edge;
        if (edge) {
          postToHost({ type: "resize", edge });
        }
      });
    });

    window.addEventListener("keydown", (e) => {
      if (state.hotkeyRecording) {
        if (e.key === "Escape") {
          e.preventDefault();
          setHotkeyRecording(false);
        }
        return;
      }
      if (e.key === "Escape") {
        e.preventDefault();
        triggerCancel();
        return;
      }
      if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
        e.preventDefault();
        if (
          document.activeElement === els.sourceInput &&
          (!state.outputText.trim() || state.sourceText !== state.lastCheckedSource)
        ) {
          triggerRecheck();
        } else {
          triggerReplace();
        }
        return;
      }
      const inTextarea =
        document.activeElement === els.sourceInput || document.activeElement === els.outputInput;
      if (!inTextarea && (e.ctrlKey || e.metaKey)) {
        const k = e.key.toLowerCase();
        if (k === "z" && !e.shiftKey) {
          e.preventDefault();
          undo();
        } else if (k === "y" || (k === "z" && e.shiftKey)) {
          e.preventDefault();
          redo();
        }
      }
    });

    if (!window.chrome || !window.chrome.webview) {
      try {
        const sse = new EventSource("/api/events");
        sse.onmessage = (e) => {
          try {
            const evt = JSON.parse(e.data);
            if (window.__grammarApp && window.__grammarApp.onHostEvent) {
              window.__grammarApp.onHostEvent(evt);
            }
          } catch (err) {
            console.error("SSE message parse error:", err);
          }
        };
      } catch (err) {
        console.warn("SSE connection error:", err);
      }
    }

    resetUndoHistory();
    syncControls();
    postToHost({ type: "ready" });
    const initDark = window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches;
    postToHost({ type: "theme", theme: initDark ? "dark" : "light" });
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", boot, { once: true });
  } else {
    boot();
  }
})();
