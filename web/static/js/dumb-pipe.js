// YooMates Dumb Pipe JS - Zero Framework, Zero State (<230 lines)
// Event relay, offline cache, WASM search, voice VAD, and Quick Switcher.
(function () {
  'use strict';
  const $ = (id) => document.getElementById(id);
  const q = (sel) => document.querySelector(sel);
  const scrollStream = () => { const s = $('message-stream'); if (s) s.scrollTop = s.scrollHeight; };

  // 1. Service Worker & Network Status
  if ('serviceWorker' in navigator) window.addEventListener('load', () => navigator.serviceWorker.register('/sw.js').catch(() => {}));
  function updateNetworkStatus() {
    const b = $('network-status-badge');
    if (!b) return;
    b.textContent = navigator.onLine ? 'Online' : 'Offline';
    b.style.color = navigator.onLine ? 'var(--text-muted)' : 'var(--accent-red)';
    b.style.borderColor = navigator.onLine ? 'var(--border-muted)' : 'var(--accent-red)';
  }
  window.addEventListener('online', updateNetworkStatus);
  window.addEventListener('offline', updateNetworkStatus);

  // 2. 0ms Instant Cache Hydration & DOM Sync
  function initOfflineCache() {
    const cid = (q('input[name="channel_id"]') || {}).value || 'chan_general';
    if (!window.YooCache) return;
    window.YooCache.syncDOMToCache(cid);
    const box = $('messages-container');
    if (box && !box.querySelector('.message-item')) {
      window.YooCache.getChannelMessages(cid).then((msgs) => {
        if (!msgs) return;
        msgs.forEach((m) => {
          const el = document.createElement('article');
          el.className = 'message-item'; el.id = 'msg-' + m.id;
          el.innerHTML = '<div class="msg-avatar">' + (m.initials || 'U') + '</div><div class="msg-content-box"><header class="msg-meta"><span class="msg-author">' + (m.author_name || 'Member') + '</span><time class="msg-time">' + (m.time || '') + '</time></header><div class="msg-body">' + (m.content || '') + '</div></div>';
          box.appendChild(el);
        });
        scrollStream();
      });
    }
  }

  // 3. Scroll & Sync on HTMX Swaps & Clear Chat Input
  document.addEventListener('htmx:afterSwap', (evt) => {
    if (evt.target && (evt.target.id === 'message-stream' || evt.target.closest('#message-stream') || evt.target.id === 'messages-container')) {
      scrollStream();
      const cid = (q('input[name="channel_id"]') || {}).value || 'chan_general';
      if (window.YooCache) window.YooCache.syncDOMToCache(cid);
    }
  });
  document.addEventListener('htmx:wsAfterSend', () => {
    const inp = $('chat-input-field');
    if (inp) { inp.value = ''; inp.focus(); }
  });

  // 4. File Attachment Upload Pipeline
  function initAttachmentPipeline() {
    const btn = $('chat-attach-trigger'), fi = $('chat-file-input'), pb = $('attachment-preview-bar');
    const fn = $('preview-filename'), fs = $('preview-filesize'), rm = $('remove-attachment-btn'), st = $('upload-status');
    const u = $('chat-file-url'), n = $('chat-file-name'), t = $('chat-file-type'), s = $('chat-file-size');
    if (!btn || !fi) return;
    btn.onclick = () => fi.click();
    fi.onchange = () => {
      if (!fi.files || !fi.files[0]) return;
      const f = fi.files[0];
      if (pb) pb.style.display = 'flex'; if (st) st.textContent = 'Uploading...';
      if (fn) fn.textContent = f.name; if (fs) fs.textContent = '(' + (f.size / 1024).toFixed(1) + ' KB)';
      const fd = new FormData(); fd.append('file', f);
      fetch('/api/hubs/upload', { method: 'POST', body: fd })
        .then((r) => { if (!r.ok) throw new Error('Upload failed'); return r.json(); })
        .then((d) => {
          if (u) u.value = d.file_url; if (n) n.value = d.file_name;
          if (t) t.value = d.file_type; if (s) s.value = d.file_size;
          if (st) st.textContent = '';
        }).catch((err) => { if (st) st.textContent = 'Error: ' + err.message; });
    };
    const resetAttach = () => { [fi, u, n, t, s].forEach((el) => { if (el) el.value = ''; }); if (pb) pb.style.display = 'none'; if (st) st.textContent = ''; };
    if (rm) rm.onclick = resetAttach;
    document.addEventListener('htmx:wsAfterSend', resetAttach);
  }

  // 5. WebAssembly In-Memory Search Engine
  function initWasmSearch() {
    if (typeof Go !== 'undefined' && 'WebAssembly' in window) {
      const go = new Go();
      WebAssembly.instantiateStreaming(fetch('/static/wasm/search.wasm'), go.importObject)
        .then((res) => { go.run(res.instance); }).catch(() => {});
    }
    const si = $('channel-search-input'), sc = $('search-count');
    if (!si) return;
    si.addEventListener('input', () => {
      const qText = si.value.trim(), box = $('messages-container');
      if (!box) return;
      const items = box.querySelectorAll('.message-item');
      if (!qText) {
        items.forEach((el) => el.classList.remove('msg-search-hidden', 'msg-search-match'));
        if (sc) sc.style.display = 'none';
        return;
      }
      const msgs = Array.from(items).map((el) => ({
        id: el.id.replace('msg-', ''),
        author_name: (el.querySelector('.msg-author') || {}).textContent || '',
        content: (el.querySelector('.msg-body') || {}).textContent || '',
        file_name: (el.querySelector('.attachment-name') || {}).textContent || ''
      }));
      const matchIds = new Set();
      if (window.YooSearchWASM) {
        try {
          const res = JSON.parse(window.YooSearchWASM(qText, JSON.stringify(msgs)));
          if (res && res.results) res.results.forEach((r) => matchIds.add(r.id));
        } catch (_) {}
      } else {
        const ql = qText.toLowerCase();
        msgs.forEach((m) => { if (m.content.toLowerCase().includes(ql) || m.author_name.toLowerCase().includes(ql) || m.file_name.toLowerCase().includes(ql)) matchIds.add(m.id); });
      }
      items.forEach((el) => {
        const has = matchIds.has(el.id.replace('msg-', ''));
        el.classList.toggle('msg-search-match', has);
        el.classList.toggle('msg-search-hidden', !has);
      });
      if (sc) { sc.textContent = matchIds.size + ' matches'; sc.style.display = 'inline'; }
    });
    si.addEventListener('keydown', (e) => { if (e.key === 'Escape') { si.value = ''; si.dispatchEvent(new Event('input')); si.blur(); } });
  }

  // 6. Low-Latency Voice Activity Detection (VAD) & Gateway
  let vStream = null, vCtx = null, vInterval = null, isSpeaking = false, silenceFrames = 0;
  function initVoiceEngine() {
    function startVoice(cid) {
      if (vStream || !navigator.mediaDevices?.getUserMedia) return;
      navigator.mediaDevices.getUserMedia({ audio: true, video: false }).then((stream) => {
        vStream = stream; vCtx = new (window.AudioContext || window.webkitAudioContext)();
        const src = vCtx.createMediaStreamSource(stream), an = vCtx.createAnalyser();
        an.fftSize = 256; src.connect(an);
        const data = new Uint8Array(an.frequencyBinCount);
        vInterval = setInterval(() => {
          an.getByteFrequencyData(data);
          let sum = 0; for (let i = 0; i < data.length; i++) sum += data[i];
          if ((sum / data.length) > 18) {
            silenceFrames = 0; if (!isSpeaking) { isSpeaking = true; notifySpeaking(cid, true); }
          } else {
            silenceFrames++; if (isSpeaking && silenceFrames > 3) { isSpeaking = false; notifySpeaking(cid, false); }
          }
        }, 100);
      }).catch(() => {});
    }
    function stopVoice() { if (vInterval) clearInterval(vInterval); if (vStream) { vStream.getTracks().forEach((t) => t.stop()); vStream = null; } if (vCtx) { vCtx.close(); vCtx = null; } isSpeaking = false; }
    function notifySpeaking(cid, active) { const f = $('chat-input-form'); if (f) htmx.trigger(f, 'htmx:wsSend', { type: 'voice_speaking', channel_id: cid, is_speaking: active }); }
    const obs = new MutationObserver(() => {
      const p = $('voice-status-panel');
      if (p && p.style.display !== 'none' && p.innerHTML.trim() !== '') {
        const cn = p.querySelector('.voice-channel-name');
        startVoice(cn ? (cn.getAttribute('data-channel-id') || 'chan_voice_lounge') : 'chan_voice_lounge');
        const dis = p.querySelector('.voice-disconnect-btn'); if (dis) dis.onclick = stopVoice;
      } else { stopVoice(); }
    });
    const sb = q('.channel-sidebar'); if (sb) obs.observe(sb, { childList: true, subtree: true });
  }

  // 7. Universal Quick Switcher (Ctrl+K Command Palette)
  function initQuickSwitcher() {
    const m = $('quick-switcher-modal'), inp = $('qs-search-input'), list = $('qs-results-list');
    if (!m || !inp || !list) return;
    const vis = () => Array.from(list.querySelectorAll('.qs-item:not(.qs-hidden)'));
    let sel = 0;
    const updateSel = () => vis().forEach((el, i) => el.classList.toggle('qs-selected', i === sel));
    const filter = (val) => {
      const qText = val.toLowerCase().trim();
      let n = 0;
      list.querySelectorAll('.qs-item').forEach((el) => {
        const match = !qText || ((el.dataset.keywords || '') + ' ' + (el.textContent || '')).toLowerCase().includes(qText);
        el.classList.toggle('qs-hidden', !match);
        if (match) n++;
      });
      list.querySelectorAll('.qs-group-header').forEach((h) => {
        let s = h.nextElementSibling, any = false;
        while (s && !s.classList.contains('qs-group-header') && s.id !== 'qs-empty-msg') {
          if (!s.classList.contains('qs-hidden')) any = true;
          s = s.nextElementSibling;
        }
        h.style.display = any ? 'block' : 'none';
      });
      const em = $('qs-empty-msg'); if (em) em.style.display = n === 0 ? 'block' : 'none';
      sel = 0; updateSel();
    };
    const open = () => { m.style.display = 'flex'; inp.value = ''; filter(''); sel = 0; updateSel(); inp.focus(); };
    const close = () => { m.style.display = 'none'; };
    inp.oninput = () => filter(inp.value);
    inp.onkeydown = (e) => {
      const v = vis();
      if (e.key === 'ArrowDown') { e.preventDefault(); sel = (sel + 1) % (v.length || 1); updateSel(); }
      else if (e.key === 'ArrowUp') { e.preventDefault(); sel = (sel - 1 + v.length) % (v.length || 1); updateSel(); }
      else if (e.key === 'Enter') { e.preventDefault(); if (v[sel]) { close(); v[sel].click(); } }
      else if (e.key === 'Escape') { close(); }
    };
    m.onclick = (e) => { if (e.target === m) close(); };
    window.openQuickSwitcher = open; window.closeQuickSwitcher = close;
  }

  // 8. Global Shortcuts & Lifecycle Init
  window.addEventListener('DOMContentLoaded', () => {
    scrollStream();
    initOfflineCache();
    updateNetworkStatus();
    initAttachmentPipeline();
    initWasmSearch();
    initVoiceEngine();
    initQuickSwitcher();
  });

  window.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') { if (window.closeQuickSwitcher) window.closeQuickSwitcher(); if (document.activeElement?.blur) document.activeElement.blur(); }
    if ((e.ctrlKey || e.metaKey) && (e.key === 'f' || e.key === 'F')) { const s = $('channel-search-input'); if (s) { e.preventDefault(); s.focus(); s.select(); } }
    if ((e.ctrlKey || e.metaKey) && (e.key === 'k' || e.key === 'K')) { e.preventDefault(); if (window.openQuickSwitcher) window.openQuickSwitcher(); }
  });
})();
