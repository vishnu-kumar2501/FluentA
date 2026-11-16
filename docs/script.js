/**
 * FluentA Landing Page - Interactive Demo Controller
 */

document.addEventListener('DOMContentLoaded', () => {
  // Demo preset states
  const presets = {
    fix: {
      input: "Their going to announce the results tomorrow and its gonna be huge.",
      outputHtml: `<span class="diff-del">Their</span><span class="diff-add">They're</span> going to announce the results tomorrow<span class="diff-add">,</span> and <span class="diff-del">its</span><span class="diff-add">it's</span> <span class="diff-del">gonna</span><span class="diff-add">going to</span> be huge.`,
      score: "98/100",
      tone: "Clear & Correct"
    },
    enhance: {
      input: "This project is really good and makes things faster for us.",
      outputHtml: `This project <span class="diff-del">is really good</span><span class="diff-add">delivers exceptional value</span> and <span class="diff-del">makes things faster for us</span><span class="diff-add">substantially accelerates our core workflows</span>.`,
      score: "96/100",
      tone: "Engaging & Impactful"
    },
    formal: {
      input: "Hey boss, cant make it to the meeting today, got some urgent stuff to sort out.",
      outputHtml: `<span class="diff-del">Hey boss, cant make it to the meeting today, got some urgent stuff to sort out.</span><span class="diff-add">Good morning, I will be unable to attend today's meeting due to pressing personal matters requiring my immediate attention.</span>`,
      score: "99/100",
      tone: "Executive & Professional"
    },
    concise: {
      input: "In order to ensure that we are able to meet our stated quarterly objectives on time, we need to quickly prioritize tasks.",
      outputHtml: `<span class="diff-del">In order to ensure that we are able to</span><span class="diff-add">To</span> meet our <span class="diff-del">stated</span> quarterly <span class="diff-del">objectives</span><span class="diff-add">goals</span> on time, we <span class="diff-del">need to quickly prioritize tasks</span><span class="diff-add">must prioritize our deliverables</span>.`,
      score: "94/100",
      tone: "Lean & Direct"
    }
  };

  const tabs = document.querySelectorAll('.mode-tab');
  const inputArea = document.getElementById('demo-input');
  const outputArea = document.getElementById('demo-output');
  const toneBadge = document.getElementById('demo-tone');
  const copyBtn = document.getElementById('demo-copy-btn');

  let activeMode = 'fix';

  function applyPreset(modeKey) {
    activeMode = modeKey;
    const data = presets[modeKey];
    if (!data) return;

    // Update active tab UI
    tabs.forEach(tab => {
      if (tab.dataset.mode === modeKey) {
        tab.classList.add('active');
      } else {
        tab.classList.remove('active');
      }
    });

    // Populate fields
    if (inputArea) inputArea.value = data.input;
    if (outputArea) {
      outputArea.style.opacity = '0.3';
      setTimeout(() => {
        outputArea.innerHTML = data.outputHtml;
        outputArea.style.opacity = '1';
      }, 150);
    }
    if (toneBadge) toneBadge.textContent = data.tone;
  }

  // Handle Tab Click
  tabs.forEach(tab => {
    tab.addEventListener('click', () => {
      applyPreset(tab.dataset.mode);
    });
  });

  // Handle Input Typing
  if (inputArea) {
    inputArea.addEventListener('input', () => {
      const text = inputArea.value.trim();
      if (!text) {
        outputArea.innerHTML = '<span style="color: var(--text-muted); font-style: italic;">Enter text on the left to see instant corrections...</span>';
        return;
      }

      // Simple real-time client-side heuristics for interactive feel
      let enhanced = text
        .replace(/\bteh\b/gi, '<span class="diff-del">teh</span><span class="diff-add">the</span>')
        .replace(/\brecieve\b/gi, '<span class="diff-del">recieve</span><span class="diff-add">receive</span>')
        .replace(/\bseperate\b/gi, '<span class="diff-del">seperate</span><span class="diff-add">separate</span>')
        .replace(/\btheir\b/gi, '<span class="diff-del">their</span><span class="diff-add">they\'re</span>')
        .replace(/\bgonna\b/gi, '<span class="diff-del">gonna</span><span class="diff-add">going to</span>');

      if (enhanced === text) {
        outputArea.innerHTML = `${text} <span style="display:inline-block; margin-left:8px; font-size: 0.8rem; color: var(--accent-emerald);">✓ Perfect</span>`;
      } else {
        outputArea.innerHTML = enhanced;
      }
    });
  }

  // Copy button
  if (copyBtn) {
    copyBtn.addEventListener('click', () => {
      const plainText = outputArea.innerText || outputArea.textContent;
      navigator.clipboard.writeText(plainText).then(() => {
        const originalText = copyBtn.innerHTML;
        copyBtn.innerHTML = '<span>Copied!</span>';
        setTimeout(() => {
          copyBtn.innerHTML = originalText;
        }, 1800);
      });
    });
  }

  // Initialize
  applyPreset('fix');
});
