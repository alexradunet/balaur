/* Domain — Balaur's signature components: the guardian OS-access card, the
   model switcher, the recap card, and the evening nudge. Hearthwood-styled,
   plain-JS React (no JSX). Buttons reuse the design-system Button. */
(function () {
  var R = window.React;
  if (!R) return;
  var h = R.createElement;
  var DS = function () { return window.BalaurBasmDesignSystem_0c1b20 || {}; };
  var ic = function (n) { return 'assets/icons/' + n + '.png'; };
  var kickerStyle = function (color) { return { fontFamily: 'var(--font-mono)', fontSize: 10, fontWeight: 700, letterSpacing: '.1em', textTransform: 'uppercase', color: color }; };

  function corner(key, st) {
    var base = { position: 'absolute', width: 10, height: 10, pointerEvents: 'none', borderColor: 'var(--gold-deep)', borderStyle: 'solid', borderWidth: 0 };
    for (var k in st) base[k] = st[k];
    return h('span', { key: key, style: base });
  }

  /* GuardianCard — an OS-access request. The quiet guardian asks before it
     touches the owner's box. Gold-bracketed parchment; the data beat is plain. */
  function GuardianCard(props) {
    return h('article', { style: { position: 'relative', background: 'var(--surface)', backgroundImage: 'var(--grain-ink)', backgroundSize: '4px 4px', color: 'var(--ink)', border: '2px solid var(--gold-deep)', boxShadow: 'var(--parch-bevel)', padding: '16px 18px 16px' } },
      corner('tl', { top: 5, left: 5, borderTopWidth: 3, borderLeftWidth: 3 }),
      corner('tr', { top: 5, right: 5, borderTopWidth: 3, borderRightWidth: 3 }),
      corner('bl', { bottom: 5, left: 5, borderBottomWidth: 3, borderLeftWidth: 3 }),
      corner('br', { bottom: 5, right: 5, borderBottomWidth: 3, borderRightWidth: 3 }),
      h('header', { style: { display: 'flex', alignItems: 'center', gap: 10, marginBottom: 10 } },
        h('img', { src: ic('shield'), alt: '', decoding: 'async', style: { width: 24, height: 24, imageRendering: 'pixelated' } }),
        h('span', { style: kickerStyle('var(--gold-ink)') }, props.kicker || 'OS access')
      ),
      h('h3', { style: { margin: '0 0 7px', fontFamily: 'var(--font-display)', fontSize: 21, color: 'var(--ink)', lineHeight: 1.12 } }, props.title || 'Read your Documents folder?'),
      props.detail ? h('p', { style: { margin: '0 0 11px', fontSize: 14.5, lineHeight: 1.5 } }, props.detail) : null,
      props.scope ? h('div', { style: { fontFamily: 'var(--font-mono)', fontSize: 11.5, color: 'var(--ink-muted)', background: 'var(--surface-2)', border: '2px solid var(--parch-edge)', boxShadow: 'var(--bevel-in)', padding: '7px 10px', marginBottom: 13, overflowWrap: 'anywhere' } }, props.scope) : null,
      h('footer', { style: { display: 'flex', flexWrap: 'wrap', gap: 9 } },
        h(DS().Button, { size: 'sm', onClick: props.onAllowOnce }, 'Allow once'),
        h(DS().Button, { variant: 'ghost', size: 'sm', onClick: props.onAllowAlways }, 'Always'),
        h(DS().Button, { variant: 'ghost', size: 'sm', onClick: props.onDeny }, 'Deny')
      )
    );
  }

  /* RecapCard — "further back…". The orb-marked summary of earlier context, so
     the owner sees what Balaur is carrying forward. */
  function RecapCard(props) {
    var points = props.points || [];
    return h('article', { style: { position: 'relative', background: 'var(--surface)', backgroundImage: 'var(--grain-ink)', backgroundSize: '4px 4px', color: 'var(--ink)', border: '2px solid var(--parch-edge)', boxShadow: 'var(--parch-bevel)', padding: '15px 17px 16px' } },
      h('span', { style: { position: 'absolute', top: 6, right: 6, width: 7, height: 7, background: 'var(--gold-ink)' } }),
      h('header', { style: { display: 'flex', alignItems: 'center', gap: 9, marginBottom: 9 } },
        h('img', { src: ic('orb'), alt: '', decoding: 'async', style: { width: 20, height: 20, imageRendering: 'pixelated' } }),
        h('span', { style: kickerStyle('var(--teal-ink)') }, props.kicker || 'Recap'),
        h('span', { style: { marginLeft: 'auto', fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--ink-muted)', textTransform: 'uppercase', letterSpacing: '.05em' } }, props.when || 'earlier today')
      ),
      props.summary ? h('p', { style: { margin: '0 0 ' + (points.length ? '11px' : '0'), fontSize: 15, lineHeight: 1.55 } }, props.summary) : null,
      points.length ? h('ul', { style: { margin: 0, padding: 0, listStyle: 'none', display: 'flex', flexDirection: 'column', gap: 6 } },
        points.map(function (p, i) { return h('li', { key: i, style: { display: 'grid', gridTemplateColumns: '14px 1fr', gap: 8, fontSize: 13.5, lineHeight: 1.45, color: 'var(--ink)' } }, h('span', { style: { color: 'var(--teal-ink)' } }, '\u25AA'), h('span', null, p)); })
      ) : null
    );
  }

  /* NudgeBanner — the evening reminder. Bell, the spoken ask, and the owner's
     established replies (done / nightfall / tomorrow). */
  function NudgeBanner(props) {
    var choices = [
      { label: 'It is done.', hint: 'mark done', cb: props.onDone },
      { label: 'At nightfall.', hint: 'snooze \u00b7 21:00', cb: props.onSnooze },
      { label: 'Tomorrow, I swear it.', hint: 'snooze \u00b7 tomorrow', cb: props.onTomorrow },
    ];
    return h('div', { style: { background: 'var(--surface)', backgroundImage: 'var(--grain-ink)', backgroundSize: '4px 4px', color: 'var(--ink)', border: '2px solid var(--gold-deep)', boxShadow: 'var(--parch-bevel)', padding: '13px 16px 14px' } },
      h('div', { style: { display: 'flex', alignItems: 'center', gap: 10, marginBottom: 10 } },
        h('img', { src: ic('bell'), alt: '', decoding: 'async', style: { width: 20, height: 20, imageRendering: 'pixelated' } }),
        h('span', { style: kickerStyle('var(--gold-ink)') }, props.kicker || 'Nudge'),
        h('span', { style: { marginLeft: 'auto', fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--ink-muted)', textTransform: 'uppercase', letterSpacing: '.05em' } }, props.when || '18:00')
      ),
      h('p', { style: { margin: '0 0 12px', fontSize: 15.5, lineHeight: 1.5 } }, props.message || 'The evening comes, and the tomatoes thirst. Will you tend them now?'),
      h('div', { style: { display: 'flex', flexWrap: 'wrap', gap: 8 } },
        choices.map(function (c, i) {
          return h('button', { key: i, type: 'button', onClick: c.cb, style: {
            display: 'inline-flex', alignItems: 'baseline', gap: 9, cursor: 'pointer', textAlign: 'left', borderRadius: 0,
            background: 'var(--surface-2)', border: '2px solid var(--parch-edge)', boxShadow: 'inset 0 1px 0 rgba(255,255,255,.4)', padding: '7px 11px', font: '14px var(--font-body)', color: 'var(--ink)',
          } },
            h('span', null, c.label),
            h('span', { style: { fontFamily: 'var(--font-mono)', fontSize: 9.5, textTransform: 'uppercase', letterSpacing: '.04em', color: 'var(--ink-muted)' } }, c.hint)
          );
        })
      )
    );
  }

  window.BalaurDomain = { GuardianCard: GuardianCard, RecapCard: RecapCard, NudgeBanner: NudgeBanner };
})();
