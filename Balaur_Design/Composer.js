/* Composer — the owner's single seat of action. One artisanal wood ledge that
   unifies every input the owner gives:
     • draft   — the textarea + tool row + typing-sound toggle (the old ChatBar
                 and MessageDraft, merged), portrait glows teal while typing.
     • choices — DialogueChoices fold in here: spoken-line buttons, keys 1–9.
     • task    — a TaskCard surfaces here with its Done / Snooze / Drop actions.
     • memory  — a proposed KnowledgeCard surfaces here with Approve / Dismiss.
   Whatever Balaur asks for, the owner answers in this one fixed place — never by
   scrolling back into the conversation. Plain-JS React (no JSX, no transpile). */
(function () {
  var R = window.React;
  if (!R) return;
  var h = R.createElement;
  var useState = R.useState, useRef = R.useRef, useEffect = R.useEffect;

  var DEFAULT_TOOLS = [
    { icon: 'scroll', title: 'Attach a scroll' },
    { icon: 'tome', title: 'Add from memory' },
    { icon: 'lens', title: 'Recall a thread' },
  ];

  function Composer(props) {
    var who = props.who || 'You';
    var placeholder = props.placeholder || 'Speak; I am listening.';
    var sendLabel = props.sendLabel || 'Send';
    var assetBase = props.assetBase || 'assets';
    var tools = props.tools || DEFAULT_TOOLS;
    var onSend = props.onSend;
    var prompt = props.prompt || null;
    var icon = function (n) { return assetBase + '/icons/' + n + '.png'; };
    var DS = window.BalaurBasmDesignSystem_0c1b20 || {};

    var ref = useRef(null);
    var typingTimer = useRef(null);
    var audioRef = useRef(null);
    var panelRef = useRef(null);
    var typingState = useState(false), typing = typingState[0], setTyping = typingState[1];
    var focusState = useState(false), focused = focusState[0], setFocused = focusState[1];
    var soundState = useState(function () {
      try { return localStorage.getItem('balaur-composer-sounds') === 'on'; } catch (e) { return false; }
    });
    var sounds = soundState[0], setSounds = soundState[1];
    var answeredState = useState(false), answered = answeredState[0], setAnswered = answeredState[1];
    var flashState = useState(null), flash = flashState[0], setFlash = flashState[1];
    var flashTimer = useRef(null);

    // A stable signature of the current prompt, so the resolved/draft state
    // only re-arms when Balaur actually asks something new.
    var promptKey = prompt ? [prompt.kind, prompt.label || '',
      prompt.choices ? prompt.choices.map(function (c) { return c.label; }).join(',') : '',
      prompt.task ? prompt.task.title : '', prompt.card ? prompt.card.title : ''].join('|') : '';
    useEffect(function () { setAnswered(false); setFlash(null); }, [promptKey]);
    useEffect(function () { return function () { if (flashTimer.current) clearTimeout(flashTimer.current); }; }, []);

    function onAnswer(text) {
      setAnswered(true);
      setFlash(text || null);
      if (flashTimer.current) clearTimeout(flashTimer.current);
      flashTimer.current = setTimeout(function () { setFlash(null); }, 2400);
    }

    useEffect(function () { return function () { if (typingTimer.current) clearTimeout(typingTimer.current); }; }, []);

    // Keys 1–9 pick a dialogue choice while a choices prompt is open.
    var choices = prompt && prompt.kind === 'choices' && !answered ? (prompt.choices || []) : null;
    useEffect(function () {
      if (!choices) return;
      function onKey(e) {
        var t = e.target;
        if (t && (t.tagName === 'TEXTAREA' || t.tagName === 'INPUT' || t.isContentEditable)) return;
        if (e.metaKey || e.ctrlKey || e.altKey) return;
        var d = parseInt(e.key, 10);
        if (d >= 1 && d <= choices.length) { e.preventDefault(); resolveChoice(choices[d - 1], d - 1); }
      }
      window.addEventListener('keydown', onKey);
      return function () { window.removeEventListener('keydown', onKey); };
    });
    function resolveChoice(c, i) {
      if (c && c.onPick) c.onPick(c, i); else if (prompt && prompt.onPick) prompt.onPick(c, i);
      onAnswer('✓ ' + (c ? c.label : ''));
    }
    function answerByText(v) {
      var s = (v || '').trim(); if (!s) return;
      var M = (window.BalaurBasmDesignSystem_0c1b20 || {}).matchChoice;
      var hit = (M && choices) ? M(s, choices) : null;
      if (hit) { resolveChoice(hit.choice, hit.index); }
      else { if (onSend) onSend(s); onAnswer('“' + s + '”'); }
    }

    function ensureCtx() {
      if (!audioRef.current) {
        var AC = window.AudioContext || window.webkitAudioContext;
        if (AC) audioRef.current = new AC();
      }
      if (audioRef.current && audioRef.current.state === 'suspended') audioRef.current.resume();
      return audioRef.current;
    }
    function blip() {
      var ctx = audioRef.current; if (!ctx) return;
      var t = ctx.currentTime;
      var o = ctx.createOscillator(), g = ctx.createGain();
      o.type = 'triangle';
      o.frequency.value = 880 + Math.random() * 520;
      g.gain.setValueAtTime(0.0001, t);
      g.gain.exponentialRampToValueAtTime(0.045, t + 0.004);
      g.gain.exponentialRampToValueAtTime(0.0001, t + 0.05);
      o.connect(g); g.connect(ctx.destination);
      o.start(t); o.stop(t + 0.06);
    }
    function toggleSounds() {
      var next = !sounds;
      setSounds(next);
      try { localStorage.setItem('balaur-composer-sounds', next ? 'on' : 'off'); } catch (e) {}
      if (next) { ensureCtx(); blip(); }
    }
    function markTyping() {
      setTyping(true);
      if (typingTimer.current) clearTimeout(typingTimer.current);
      typingTimer.current = setTimeout(function () { setTyping(false); }, 850);
    }
    function autosize() {
      var elx = ref.current; if (!elx) return;
      elx.style.height = 'auto';
      elx.style.height = Math.min(elx.scrollHeight, 180) + 'px';
    }
    function submit() {
      var v = ref.current ? ref.current.value.trim() : '';
      if (!v) return;
      if (onSend) onSend(v);
      if (ref.current) { ref.current.value = ''; ref.current.style.height = 'auto'; ref.current.focus(); }
      setTyping(false);
    }
    function onKeyDown(e) {
      if (e.key === 'Enter' && !e.shiftKey && !e.altKey && !e.ctrlKey && !e.metaKey) { e.preventDefault(); submit(); return; }
      if (sounds && !e.metaKey && !e.ctrlKey && !e.altKey && (e.key.length === 1 || e.key === 'Backspace')) blip();
    }

    var live = typing || focused;
    var deciding = !!prompt && !answered;

    function corner(key, st) {
      var base = { position: 'absolute', width: 9, height: 9, pointerEvents: 'none', borderColor: 'var(--gold-deep)', borderStyle: 'solid', borderWidth: 0, opacity: 0.85 };
      for (var k in st) base[k] = st[k];
      return h('span', { key: key, style: base });
    }

    // ── Top row ────────────────────────────────────────────────────
    var topPortrait = h('span', { className: 'balaur-avatar balaur-avatar-soul', 'data-kind': 'soul', 'data-state': (typing && !deciding) ? 'thinking' : 'idle', title: who, style: {
      '--avatar-size': '42px', border: '2px solid var(--outline-2)', background: '#101314', boxShadow: 'inset 0 0 0 2px var(--indigo), var(--drop-hard)', padding: 2, flexShrink: 0,
    } }, h('img', { src: props.avatarSrc, alt: '', decoding: 'async', style: { display: 'block', width: '100%', height: '100%', imageRendering: 'pixelated', transform: 'scaleX(-1)' } }));

    var toolBtns = tools.map(function (t, i) {
      return h('button', { key: i, type: 'button', title: deciding ? undefined : t.title, disabled: deciding, onClick: deciding ? undefined : (t.onClick || function () {}), style: {
        display: 'inline-flex', alignItems: 'center', justifyContent: 'center', width: 42, height: 42, padding: 0, cursor: deciding ? 'default' : 'pointer', pointerEvents: deciding ? 'none' : 'auto', opacity: deciding ? 0.38 : 1,
        background: 'var(--chrome-2)', backgroundImage: 'var(--grain-warm)', backgroundSize: '4px 4px', border: '2px solid var(--outline-2)', borderRadius: 0, boxShadow: 'var(--bevel-in)',
      } }, h('img', { src: icon(t.icon), alt: '', decoding: 'async', style: { width: 20, height: 20, imageRendering: 'pixelated' } }));
    });
    var kicker = deciding
      ? h('div', { style: { minWidth: 0, overflow: 'hidden' } },
          h('div', { style: { fontFamily: 'var(--font-mono)', fontSize: 10, fontWeight: 700, letterSpacing: '.08em', textTransform: 'uppercase', color: 'var(--gold)', whiteSpace: 'nowrap', textAlign: 'center', overflow: 'hidden', textOverflow: 'ellipsis' } }, prompt.label || 'Your word')
        )
      : h('div');
    var soundBtn = h('button', { type: 'button', title: sounds ? 'Typing sounds on' : 'Typing sounds off', disabled: deciding, onClick: deciding ? undefined : toggleSounds, style: {
      display: 'inline-flex', alignItems: 'center', justifyContent: 'center', width: 42, height: 42, padding: 0, cursor: deciding ? 'default' : 'pointer', pointerEvents: deciding ? 'none' : 'auto', opacity: deciding ? 0.38 : 1,
      background: sounds ? 'color-mix(in srgb, var(--teal) 20%, var(--chrome-2))' : 'var(--chrome-2)', backgroundImage: sounds ? 'none' : 'var(--grain-warm)', backgroundSize: '4px 4px',
      border: '2px solid ' + (sounds ? 'var(--teal-deep)' : 'var(--outline-2)'), borderRadius: 0, boxShadow: 'var(--bevel-in)',
    } }, h('img', { src: icon('bell'), alt: '', decoding: 'async', style: { width: 20, height: 20, imageRendering: 'pixelated', opacity: sounds ? 1 : 0.4 } }));
    var topRow = h('div', { style: { display: 'grid', gridTemplateColumns: 'auto minmax(0,1fr) auto', alignItems: 'center', columnGap: 8, marginBottom: deciding ? 11 : 10, minHeight: 42 } },
      h('div', { style: { display: 'flex', alignItems: 'center', gap: 8 } }, toolBtns, soundBtn),
      kicker,
      h('div', { style: { display: 'flex', alignItems: 'center', gap: 8 } }, topPortrait)
    );

    // ── Main slot — draft, or the decision that surfaced ───────────
    var mainSlot;
    if (!deciding) {
      mainSlot = h('div', { style: {
        position: 'relative', minWidth: 0, background: 'var(--surface)', backgroundImage: 'var(--grain-ink)', backgroundSize: '4px 4px', color: 'var(--ink)',
        border: '2px ' + (live ? 'solid' : 'dashed') + ' ' + (live ? 'var(--gold-deep)' : 'var(--parch-edge)'),
        boxShadow: live ? 'var(--parch-bevel)' : 'none', padding: '10px 12px 8px', display: 'flex', flexDirection: 'column', gap: 6, transition: 'border-color .12s',
      } },
        h('textarea', { ref: ref, name: 'message', rows: 2, placeholder: placeholder, autoComplete: 'off',
          onInput: function () { autosize(); markTyping(); }, onKeyDown: onKeyDown,
          onFocus: function () { setFocused(true); }, onBlur: function () { setFocused(false); },
          style: { width: '100%', background: 'transparent', border: 0, resize: 'none', font: '16px/1.6 var(--font-body)', color: 'var(--ink)', caretColor: 'var(--ember-deep)', padding: '2px 2px', minHeight: 48, maxHeight: 180, outline: 'none' } }),
        h('div', { style: { display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 10, borderTop: '2px dotted var(--parch-edge)', paddingTop: 8 } },
          h('span', { style: { fontFamily: 'var(--font-mono)', fontSize: 10.5, textTransform: 'uppercase', letterSpacing: '.05em', color: flash ? 'var(--good-ink)' : 'var(--ink-muted)' } }, flash ? flash : (typing ? 'writing…' : 'unsent · enter speaks')),
          h('button', { type: 'submit', className: 'btn btn-primary btn-sm' }, sendLabel)
        )
      );
    } else if (prompt.kind === 'choices') {
      var cs = prompt.choices || [];
      mainSlot = h('div', { className: 'choices-panel', ref: panelRef, style: { minHeight: 0 } },
        cs.map(function (c, i) {
          return h('button', { key: i, type: 'button', className: 'choice', onClick: function () { resolveChoice(c, i); } },
            h('span', { className: 'choice-key', 'aria-hidden': 'true' }, i + 1),
            h('span', { className: 'choice-label' }, c.label),
            c.hint ? h('span', { className: 'choice-hint' }, c.hint) : null
          );
        }),
        h('div', { key: 'type', className: 'choice choice-type', style: { alignItems: 'center', cursor: 'text' }, onClick: function (e) { var inp = e.currentTarget.querySelector('input'); if (inp) inp.focus(); } },
          h('span', { className: 'choice-key', 'aria-hidden': 'true' }, cs.length + 1),
          h('input', { type: 'text', placeholder: 'type your answer\u2026', autoComplete: 'off',
            onKeyDown: function (e) { if (e.key === 'Enter') { e.preventDefault(); answerByText(e.target.value); e.target.value = ''; } },
            style: { flex: '1 1 auto', minWidth: 0, font: '16px/1.45 var(--font-body)', background: 'transparent', border: 0, color: 'var(--ink)', caretColor: 'var(--ember-deep)', padding: 0, outline: 'none' } }),
          h('span', { className: 'choice-hint' }, 'enter')
        )
      );
    } else if (prompt.kind === 'task') {
      mainSlot = DS.TaskCard ? h(DS.TaskCard, Object.assign({ status: 'open',
        onDone: function () { if (prompt.onDone) prompt.onDone(); onAnswer('✓ done · ' + (prompt.task ? prompt.task.title : '')); },
        onSnooze: function () { if (prompt.onSnooze) prompt.onSnooze(); onAnswer('✓ snoozed · ' + (prompt.task ? prompt.task.title : '')); },
        onDrop: function () { if (prompt.onDrop) prompt.onDrop(); onAnswer('✓ dropped · ' + (prompt.task ? prompt.task.title : '')); }
      }, prompt.task || {})) : null;
    } else if (prompt.kind === 'memory') {
      mainSlot = DS.KnowledgeCard ? h(DS.KnowledgeCard, Object.assign({ status: 'proposed',
        onApprove: function () { if (prompt.onApprove) prompt.onApprove(); onAnswer('✓ kept in memory'); },
        onDismiss: function () { if (prompt.onDismiss) prompt.onDismiss(); onAnswer('✓ not kept'); }
      }, prompt.card || {})) : null;
    }

    var body = deciding
      ? h('div', { style: { margin: 0 } }, mainSlot)
      : h('form', { onSubmit: function (e) { e.preventDefault(); submit(); }, style: { margin: 0 } }, mainSlot);

    return h('div', { className: 'balaur-composer', style: {
      position: 'relative', width: '100%', '--portrait-size': '64px',
      background: 'var(--chrome)', backgroundImage: 'var(--wood-planks), var(--grain-warm)', backgroundSize: 'auto, 4px 4px',
      border: '2px solid ' + (deciding ? 'var(--gold-deep)' : 'var(--outline-2)'), boxShadow: 'var(--bevel-up)', padding: '12px 14px 14px',
    } },
      corner('tl', { top: 4, left: 4, borderTopWidth: 3, borderLeftWidth: 3 }),
      corner('tr', { top: 4, right: 4, borderTopWidth: 3, borderRightWidth: 3 }),
      corner('bl', { bottom: 4, left: 4, borderBottomWidth: 3, borderLeftWidth: 3 }),
      corner('br', { bottom: 4, right: 4, borderBottomWidth: 3, borderRightWidth: 3 }),
      topRow,
      body
    );
  }

  window.Composer = Composer;
})();
