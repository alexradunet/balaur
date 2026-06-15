/* Utilities — the small connective pieces: Badge, Tooltip, Skeleton, Alert,
   Breadcrumb, Pagination. Hearthwood-styled, plain-JS React (no JSX). */
(function () {
  var R = window.React;
  if (!R) return;
  var h = R.createElement;
  var useState = R.useState;
  var ic = function (n) { return 'assets/icons/' + n + '.png'; };

  /* Badge — a small count / status chip. Tones: gold (default), ember (urgent),
     teal (info), wood (neutral). dot=true renders a bare marker. */
  function Badge(props) {
    var tone = props.tone || 'gold';
    var map = {
      gold: { bg: 'var(--gold)', fg: '#1c0d04', bd: 'var(--gold-deep)' },
      ember: { bg: 'var(--ember)', fg: '#1c0d04', bd: 'var(--ember-deep)' },
      teal: { bg: 'var(--teal-deep)', fg: '#06120f', bd: 'var(--outline-2)' },
      wood: { bg: 'var(--chrome-2)', fg: 'var(--chrome-fg)', bd: 'var(--outline-2)' },
    };
    var c = map[tone] || map.gold;
    if (props.dot) return h('span', { style: { display: 'inline-block', width: 9, height: 9, background: c.bg, border: '2px solid ' + c.bd } });
    return h('span', { style: {
      display: 'inline-flex', alignItems: 'center', justifyContent: 'center', minWidth: 20, height: 20, padding: '0 6px', boxSizing: 'border-box',
      fontFamily: 'var(--font-mono)', fontSize: 11, fontWeight: 700, lineHeight: 1, letterSpacing: '.02em', color: c.fg,
      background: c.bg, border: '2px solid ' + c.bd, boxShadow: 'inset 0 1px 0 rgba(255,255,255,.25)',
    } }, props.children);
  }

  /* Tooltip — a wood label on hover/focus. position: top | bottom. */
  function Tooltip(props) {
    var st = useState(false), open = st[0], set = st[1];
    var pos = props.position || 'top';
    var bubble = open ? h('span', { role: 'tooltip', style: {
      position: 'absolute', left: '50%', transform: 'translateX(-50%)', bottom: pos === 'top' ? '100%' : 'auto', top: pos === 'bottom' ? '100%' : 'auto',
      marginBottom: pos === 'top' ? 8 : 0, marginTop: pos === 'bottom' ? 8 : 0, zIndex: 30, whiteSpace: 'nowrap',
      fontFamily: 'var(--font-mono)', fontSize: 11, letterSpacing: '.02em', color: 'var(--chrome-fg)',
      background: 'var(--chrome)', backgroundImage: 'var(--wood-planks), var(--grain-warm)', backgroundSize: 'auto, 4px 4px',
      border: '2px solid var(--outline-2)', boxShadow: 'var(--bevel-up)', padding: '5px 9px',
    } }, props.label) : null;
    return h('span', { style: { position: 'relative', display: 'inline-flex' },
      onMouseEnter: function () { set(true); }, onMouseLeave: function () { set(false); },
      onFocus: function () { set(true); }, onBlur: function () { set(false); } }, props.children, bubble);
  }

  /* Skeleton — a carved loading placeholder with a sliding sheen. variant:
     line | block | avatar. The breathing comes from basm-glow-ish keyframes
     defined inline via animation on a gradient. */
  function Skeleton(props) {
    var v = props.variant || 'line';
    var base = {
      background: 'var(--surface-2)', backgroundImage: 'linear-gradient(100deg, transparent 30%, rgba(255,255,255,.10) 50%, transparent 70%)',
      backgroundSize: '220% 100%', border: '2px solid var(--parch-edge)', boxShadow: 'var(--bevel-in)',
      animation: 'sk-sheen 1.25s linear infinite',
    };
    if (v === 'avatar') { base.width = props.size || 48; base.height = props.size || 48; }
    else if (v === 'block') { base.width = props.width || '100%'; base.height = props.height || 64; }
    else { base.width = props.width || '100%'; base.height = props.height || 13; }
    return h('span', { 'aria-hidden': 'true', style: Object.assign({ display: 'block' }, base) });
  }

  /* Alert — a callout band. tone: info | warn | danger. Pixel icon + title +
     body. Heavier than a Toast; it stays on the surface. */
  function Alert(props) {
    var tone = props.tone || 'info';
    var map = {
      info: { edge: 'var(--gold-deep)', kick: 'var(--gold-ink)', icon: 'orb' },
      warn: { edge: 'var(--ember-deep)', kick: 'var(--ember-deep)', icon: 'shield' },
      danger: { edge: 'var(--ember-red)', kick: 'var(--ember-red)', icon: 'flame' },
    };
    var c = map[tone] || map.info;
    return h('div', { role: tone === 'info' ? 'note' : 'alert', style: {
      display: 'grid', gridTemplateColumns: '26px 1fr', columnGap: 12, alignItems: 'start',
      background: 'var(--surface)', backgroundImage: 'var(--grain-ink)', backgroundSize: '4px 4px', color: 'var(--ink)',
      borderLeft: '6px solid ' + c.edge, borderTop: '2px solid var(--parch-edge)', borderRight: '2px solid var(--parch-edge)', borderBottom: '2px solid var(--parch-edge)',
      boxShadow: 'var(--parch-bevel)', padding: '13px 16px 14px',
    } },
      h('img', { src: ic(props.icon || c.icon), alt: '', decoding: 'async', style: { width: 22, height: 22, imageRendering: 'pixelated', marginTop: 1 } }),
      h('div', null,
        props.title ? h('div', { style: { fontFamily: 'var(--font-mono)', fontSize: 11, fontWeight: 700, letterSpacing: '.06em', textTransform: 'uppercase', color: c.kick, marginBottom: 4 } }, props.title) : null,
        h('div', { style: { fontSize: 14, lineHeight: 1.5 } }, props.children)
      )
    );
  }

  /* Breadcrumb — wood-chrome path trail; last crumb is gold and current. */
  function Breadcrumb(props) {
    var items = props.items || [];
    var kids = [];
    items.forEach(function (it, i) {
      var last = i === items.length - 1;
      kids.push(h(last ? 'span' : 'button', {
        key: 'i' + i, type: last ? undefined : 'button', onClick: last ? undefined : it.onClick,
        style: {
          fontFamily: 'var(--font-mono)', fontSize: 11.5, letterSpacing: '.03em', textTransform: 'uppercase',
          color: last ? 'var(--gold)' : 'var(--chrome-fg)', background: 'none', border: 'none', padding: 0,
          cursor: last ? 'default' : 'pointer',
        },
      }, it.label));
      if (!last) kids.push(h('span', { key: 's' + i, 'aria-hidden': 'true', style: { color: 'var(--smoke)', fontFamily: 'var(--font-mono)', fontSize: 11 } }, '\u203A'));
    });
    return h('nav', { style: { display: 'inline-flex', alignItems: 'center', gap: 9, flexWrap: 'wrap',
      background: 'var(--chrome)', backgroundImage: 'var(--wood-planks), var(--grain-warm)', backgroundSize: 'auto, 4px 4px',
      border: '2px solid var(--outline-2)', boxShadow: 'var(--bevel-up)', padding: '7px 12px' } }, kids);
  }

  /* Pagination — prev / numbered slabs / next. The active page is a raised gold
     chip; the rest are inset wells. */
  function Pagination(props) {
    var total = props.total || 1, page = props.page || 1;
    function go(p) { if (p >= 1 && p <= total && p !== page && props.onChange) props.onChange(p); }
    function slab(content, p, opts) {
      opts = opts || {};
      var on = opts.active;
      var disabled = opts.disabled;
      return h('button', { key: opts.key, type: 'button', disabled: disabled, onClick: function () { go(p); }, style: {
        minWidth: 34, height: 34, padding: '0 9px', fontFamily: 'var(--font-mono)', fontSize: 12, cursor: disabled ? 'default' : 'pointer', borderRadius: 0,
        color: on ? 'var(--gold)' : 'var(--chrome-fg)', background: on ? 'var(--chrome)' : 'var(--chrome-2)', backgroundImage: 'var(--grain-warm)', backgroundSize: '4px 4px',
        border: '2px solid ' + (on ? 'var(--gold-deep)' : 'var(--outline-2)'),
        boxShadow: on ? 'inset 0 2px 0 var(--bevel-light), inset 0 -2px 0 var(--bevel-dark), var(--drop-hard)' : 'var(--bevel-in)',
        opacity: disabled ? 0.4 : 1,
      } }, content);
    }
    // window of pages around current
    var pages = [];
    var start = Math.max(1, Math.min(page - 1, total - 2));
    var end = Math.min(total, start + 2);
    for (var p = start; p <= end; p++) pages.push(p);
    var slabs = [slab('\u2039', page - 1, { key: 'prev', disabled: page <= 1 })];
    if (start > 1) slabs.push(h('span', { key: 'e1', style: { color: 'var(--smoke)', fontFamily: 'var(--font-mono)', alignSelf: 'end', padding: '0 2px' } }, '\u2026'));
    pages.forEach(function (p) { slabs.push(slab(String(p), p, { key: 'p' + p, active: p === page })); });
    if (end < total) slabs.push(h('span', { key: 'e2', style: { color: 'var(--smoke)', fontFamily: 'var(--font-mono)', alignSelf: 'end', padding: '0 2px' } }, '\u2026'));
    slabs.push(slab('\u203A', page + 1, { key: 'next', disabled: page >= total }));
    return h('div', { style: { display: 'inline-flex', alignItems: 'center', gap: 6 } }, slabs);
  }

  window.BalaurUtil = { Badge: Badge, Tooltip: Tooltip, Skeleton: Skeleton, Alert: Alert, Breadcrumb: Breadcrumb, Pagination: Pagination };
})();
