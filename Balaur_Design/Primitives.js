/* Primitives — the form & feedback batch the kit was missing: TextField,
   Toggle, Select, Toast, Dialog, EmptyState. Hearthwood-styled, plain-JS React
   (no JSX). Buttons reuse the design-system Button. */
(function () {
  var R = window.React;
  if (!R) return;
  var h = R.createElement;
  var DS = function () { return window.BalaurBasmDesignSystem_0c1b20 || {}; };
  var labelStyle = { fontFamily: 'var(--font-mono)', fontSize: 10, fontWeight: 700, letterSpacing: '.07em', textTransform: 'uppercase', color: 'var(--muted)' };

  function TextField(props) {
    var error = props.error;
    return h('label', { style: { display: 'flex', flexDirection: 'column', gap: 6, width: '100%' } },
      props.label ? h('span', { style: labelStyle }, props.label) : null,
      h('input', { className: 'prim-field', type: props.type || 'text', placeholder: props.placeholder, defaultValue: props.value, disabled: props.disabled, style: {
        font: '15px var(--font-body)', color: 'var(--ink)', background: 'var(--surface)', backgroundImage: 'var(--grain-ink)', backgroundSize: '4px 4px',
        border: '2px solid ' + (error ? 'var(--ember-red)' : 'var(--parch-edge)'), boxShadow: 'inset 0 2px 0 rgba(0,0,0,.12)', borderRadius: 0,
        padding: '10px 12px', outline: 'none', width: '100%', boxSizing: 'border-box', caretColor: 'var(--ember-deep)', opacity: props.disabled ? 0.55 : 1,
      } }),
      (error || props.hint) ? h('span', { style: { fontFamily: 'var(--font-mono)', fontSize: 10.5, letterSpacing: '.02em', color: error ? 'var(--ember-red)' : 'var(--muted)' } }, error || props.hint) : null
    );
  }

  function Toggle(props) {
    var on = !!props.checked;
    var track = h('span', { onClick: props.disabled ? undefined : function () { if (props.onChange) props.onChange(!on); }, style: {
      position: 'relative', display: 'inline-block', width: 46, height: 26, flexShrink: 0, cursor: props.disabled ? 'default' : 'pointer',
      background: on ? 'var(--teal-deep)' : 'var(--chrome-2)', backgroundImage: on ? 'none' : 'var(--grain-warm)', backgroundSize: '4px 4px',
      border: '2px solid var(--outline-2)', boxShadow: 'var(--bevel-in)', opacity: props.disabled ? 0.5 : 1,
    } }, h('span', { style: { position: 'absolute', top: 2, left: on ? 22 : 2, width: 18, height: 18, background: on ? 'var(--gold)' : 'var(--surface-2)', border: '2px solid var(--outline-2)', boxShadow: 'var(--bevel-up)', transition: 'left 80ms' } }));
    if (!props.label) return track;
    return h('label', { style: { display: 'inline-flex', alignItems: 'center', gap: 11, cursor: props.disabled ? 'default' : 'pointer' } },
      track, h('span', { style: { fontFamily: 'var(--font-mono)', fontSize: 12, textTransform: 'uppercase', letterSpacing: '.04em', color: 'var(--fg)' } }, props.label));
  }

  function Select(props) {
    var options = props.options || [];
    return h('label', { style: { display: 'flex', flexDirection: 'column', gap: 6, width: '100%' } },
      props.label ? h('span', { style: labelStyle }, props.label) : null,
      h('div', { style: { position: 'relative', width: '100%' } },
        h('select', { className: 'prim-field', value: props.value, onChange: function (e) { if (props.onChange) props.onChange(e.target.value); }, disabled: props.disabled, style: {
          appearance: 'none', WebkitAppearance: 'none', MozAppearance: 'none', font: '13px var(--font-mono)', textTransform: 'uppercase', letterSpacing: '.03em', color: 'var(--ink)',
          background: 'var(--surface)', backgroundImage: 'var(--grain-ink)', backgroundSize: '4px 4px', border: '2px solid var(--parch-edge)', boxShadow: 'inset 0 2px 0 rgba(0,0,0,.12)',
          borderRadius: 0, padding: '11px 34px 11px 12px', outline: 'none', width: '100%', boxSizing: 'border-box', cursor: 'pointer', opacity: props.disabled ? 0.55 : 1,
        } }, options.map(function (o, i) { return h('option', { key: i, value: o }, o); })),
        h('span', { 'aria-hidden': 'true', style: { position: 'absolute', right: 12, top: '50%', transform: 'translateY(-50%)', pointerEvents: 'none', fontFamily: 'var(--font-mono)', color: 'var(--gold-ink)', fontSize: 12 } }, '\u25BE')
      )
    );
  }

  function Toast(props) {
    var tone = props.tone || 'info';
    var iconName = props.icon || (tone === 'success' ? 'check' : tone === 'warn' ? 'shield' : 'quill');
    var edge = tone === 'warn' ? 'var(--ember-deep)' : tone === 'success' ? 'var(--good-ink)' : 'var(--gold-deep)';
    return h('div', { role: 'status', style: {
      display: 'inline-flex', alignItems: 'center', gap: 11, maxWidth: '100%',
      background: 'var(--surface)', backgroundImage: 'var(--grain-ink)', backgroundSize: '4px 4px', color: 'var(--ink)',
      border: '2px solid ' + edge, boxShadow: 'var(--parch-bevel)', padding: '11px 15px',
    } },
      h('img', { src: 'assets/icons/' + iconName + '.png', alt: '', decoding: 'async', style: { width: 20, height: 20, imageRendering: 'pixelated', flexShrink: 0 } }),
      h('span', { style: { fontSize: 14, lineHeight: 1.45 } }, props.children)
    );
  }

  function corner(key, st) {
    var base = { position: 'absolute', width: 10, height: 10, pointerEvents: 'none', borderColor: 'var(--gold-deep)', borderStyle: 'solid', borderWidth: 0 };
    for (var k in st) base[k] = st[k];
    return h('span', { key: key, style: base });
  }

  function Dialog(props) {
    var panel = h('div', { style: {
      position: 'relative', width: props.inline ? '100%' : 'min(520px, calc(100% - 40px))', maxWidth: 520,
      background: 'var(--surface)', backgroundImage: 'var(--grain-ink)', backgroundSize: '4px 4px', color: 'var(--ink)',
      border: '2px solid var(--gold-deep)', boxShadow: 'var(--parch-bevel)', padding: '22px 22px 20px',
    } },
      corner('tl', { top: 5, left: 5, borderTopWidth: 3, borderLeftWidth: 3 }),
      corner('tr', { top: 5, right: 5, borderTopWidth: 3, borderRightWidth: 3 }),
      corner('bl', { bottom: 5, left: 5, borderBottomWidth: 3, borderLeftWidth: 3 }),
      corner('br', { bottom: 5, right: 5, borderBottomWidth: 3, borderRightWidth: 3 }),
      props.kicker ? h('div', { style: { fontFamily: 'var(--font-mono)', fontSize: 10, fontWeight: 700, letterSpacing: '.1em', textTransform: 'uppercase', color: 'var(--gold-ink)', marginBottom: 8 } }, props.kicker) : null,
      props.title ? h('h2', { style: { margin: '0 0 10px', fontFamily: 'var(--font-display)', fontSize: 24, color: 'var(--ink)', lineHeight: 1.1 } }, props.title) : null,
      h('div', { style: { fontSize: 15, lineHeight: 1.55 } }, props.children),
      (props.actions && props.actions.length) ? h('div', { style: { display: 'flex', flexWrap: 'wrap', gap: 10, marginTop: 18, justifyContent: 'flex-end' } },
        props.actions.map(function (a, i) { return h(DS().Button, { key: i, variant: a.variant, size: 'sm', onClick: a.onClick }, a.label); })) : null
    );
    if (props.inline) return panel;
    if (!props.open) return null;
    return h('div', { style: { position: 'fixed', inset: 0, zIndex: 100, display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 20, background: 'rgba(10,5,1,.72)' }, onClick: props.onClose },
      h('div', { onClick: function (e) { e.stopPropagation(); } }, panel));
  }

  function EmptyState(props) {
    return h('div', { style: { display: 'flex', flexDirection: 'column', alignItems: 'center', textAlign: 'center', gap: 14, padding: '30px 20px' } },
      props.crestSrc ? h('img', { src: props.crestSrc, alt: '', decoding: 'async', style: { width: 88, height: 88, imageRendering: 'pixelated', opacity: 0.92 } }) : null,
      h('h3', { style: { margin: 0, fontFamily: 'var(--font-display)', fontSize: 22, color: 'var(--fg-strong)' } }, props.title || 'Nothing on the book.'),
      props.line ? h('p', { style: { margin: 0, maxWidth: 360, color: 'var(--muted)', fontSize: 14, lineHeight: 1.55 } }, props.line) : null,
      props.actionLabel ? h('div', { style: { marginTop: 4 } }, h(DS().Button, { variant: 'wood', onClick: props.onAction }, props.actionLabel)) : null
    );
  }

  window.BalaurPrim = { TextField: TextField, Toggle: Toggle, Select: Select, Toast: Toast, Dialog: Dialog, EmptyState: EmptyState };
})();
