/* DataDisplay — the Tasks / Life / Day data set: List + ListItem, CalendarCell,
   DayEntry (timeline), and StatCard with a Sparkline. Hearthwood-styled,
   plain-JS React (no JSX). */
(function () {
  var R = window.React;
  if (!R) return;
  var h = R.createElement;
  var ic = function (n) { return 'assets/icons/' + n + '.png'; };

  /* List + ListItem — a parchment list. Each row: optional pixel icon, a title
     and subtitle, and a trailing meta/value. Rows divide with a hairline. */
  function ListItem(props) {
    return h('div', { onClick: props.onClick, style: {
      display: 'grid', gridTemplateColumns: (props.icon ? '28px ' : '') + '1fr auto', alignItems: 'center', columnGap: 12,
      padding: '11px 14px', cursor: props.onClick ? 'pointer' : 'default', borderTop: props.first ? 'none' : '1px solid var(--parch-edge)',
    } },
      props.icon ? h('img', { src: ic(props.icon), alt: '', decoding: 'async', style: { width: 22, height: 22, imageRendering: 'pixelated' } }) : null,
      h('div', { style: { minWidth: 0 } },
        h('div', { style: { fontSize: 14.5, color: 'var(--ink)', lineHeight: 1.3, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' } }, props.title),
        props.subtitle ? h('div', { style: { fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--ink-muted)', marginTop: 2, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' } }, props.subtitle) : null
      ),
      props.meta ? h('div', { style: { fontFamily: 'var(--font-mono)', fontSize: 11.5, color: props.metaTone === 'warn' ? 'var(--ember-deep)' : 'var(--ink-muted)', textTransform: 'uppercase', letterSpacing: '.03em', whiteSpace: 'nowrap', textAlign: 'right' } }, props.meta) : null
    );
  }

  function List(props) {
    var items = props.items || [];
    return h('div', { style: { background: 'var(--surface)', backgroundImage: 'var(--grain-ink)', backgroundSize: '4px 4px', border: '2px solid var(--parch-edge)', boxShadow: 'var(--parch-bevel)', overflow: 'hidden' } },
      props.title ? h('div', { style: { fontFamily: 'var(--font-mono)', fontSize: 10, fontWeight: 700, letterSpacing: '.09em', textTransform: 'uppercase', color: 'var(--gold-ink)', padding: '11px 14px 9px', borderBottom: '2px solid var(--parch-edge)' } }, props.title) : null,
      items.map(function (it, i) { return h(ListItem, Object.assign({ key: i, first: i === 0 && !props.title }, it)); })
    );
  }

  /* CalendarCell — one day in a month grid. Date number, today ring, selected
     fill, dim for other months, and up to three event pips. */
  function CalendarCell(props) {
    var selected = props.selected, today = props.today, dim = props.dim;
    var pips = props.pips || 0;
    return h('button', { type: 'button', onClick: props.onClick, style: {
      position: 'relative', width: '100%', aspectRatio: '1 / 1', minWidth: 38, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: 5,
      cursor: 'pointer', borderRadius: 0, padding: 4,
      background: selected ? 'var(--gold)' : 'var(--surface)', backgroundImage: selected ? 'none' : 'var(--grain-ink)', backgroundSize: '4px 4px',
      border: '2px solid ' + (today && !selected ? 'var(--gold-deep)' : 'var(--parch-edge)'),
      boxShadow: selected ? 'var(--parch-bevel)' : (today ? 'inset 0 0 0 1px var(--gold-deep)' : 'none'),
      opacity: dim ? 0.4 : 1,
    } },
      h('span', { style: { fontFamily: 'var(--font-mono)', fontSize: 13, fontWeight: today || selected ? 700 : 400, color: selected ? '#1c0d04' : 'var(--ink)', lineHeight: 1 } }, props.day),
      h('span', { style: { display: 'flex', gap: 3, height: 5 } },
        Array.apply(null, { length: pips }).map(function (_, i) { return h('span', { key: i, style: { width: 5, height: 5, background: selected ? '#1c0d04' : (i === 0 ? 'var(--ember)' : i === 1 ? 'var(--teal-ink)' : 'var(--gold-ink)') } }); }))
    );
  }

  /* DayEntry — a vertical timeline row: a time, a node on the rail, and content.
     last omits the trailing rail segment. */
  function DayEntry(props) {
    var tone = props.tone || 'gold';
    var node = tone === 'teal' ? 'var(--teal-ink)' : tone === 'ember' ? 'var(--ember-deep)' : 'var(--gold-ink)';
    return h('div', { style: { display: 'grid', gridTemplateColumns: '54px 22px 1fr', columnGap: 12, alignItems: 'stretch' } },
      h('div', { style: { fontFamily: 'var(--font-mono)', fontSize: 11.5, color: 'var(--ink-muted)', textTransform: 'uppercase', letterSpacing: '.03em', paddingTop: 1, textAlign: 'right' } }, props.time),
      h('div', { style: { position: 'relative', display: 'flex', justifyContent: 'center' } },
        h('span', { style: { position: 'absolute', top: 0, bottom: 0, width: 2, background: 'var(--parch-edge)' } }),
        h('span', { style: { position: 'relative', zIndex: 1, width: 12, height: 12, marginTop: 1, background: node, border: '2px solid var(--surface)', boxShadow: '0 0 0 2px ' + node } }),
        props.last ? h('span', { style: { position: 'absolute', top: 14, bottom: 0, width: 2, background: 'var(--surface)' } }) : null
      ),
      h('div', { style: { paddingBottom: props.last ? 0 : 18 } },
        h('div', { style: { fontSize: 14.5, color: 'var(--ink)', lineHeight: 1.4 } }, props.title),
        props.detail ? h('div', { style: { fontSize: 13, color: 'var(--ink-muted)', marginTop: 3, lineHeight: 1.45 } }, props.detail) : null
      )
    );
  }

  /* Sparkline — a tiny functional line chart. Pure SVG path; no axes. */
  function Sparkline(props) {
    var data = props.data || [];
    var w = props.width || 120, ht = props.height || 34, pad = 3;
    var min = Math.min.apply(null, data), max = Math.max.apply(null, data);
    var span = (max - min) || 1;
    var stepX = (w - pad * 2) / Math.max(1, data.length - 1);
    var pts = data.map(function (v, i) { return [pad + i * stepX, pad + (ht - pad * 2) * (1 - (v - min) / span)]; });
    var d = pts.map(function (p, i) { return (i ? 'L' : 'M') + p[0].toFixed(1) + ' ' + p[1].toFixed(1); }).join(' ');
    var area = d + ' L' + pts[pts.length - 1][0].toFixed(1) + ' ' + (ht - pad) + ' L' + pts[0][0].toFixed(1) + ' ' + (ht - pad) + ' Z';
    var color = props.color || 'var(--teal-ink)';
    var last = pts[pts.length - 1];
    return h('svg', { width: w, height: ht, viewBox: '0 0 ' + w + ' ' + ht, style: { display: 'block', shapeRendering: 'geometricPrecision' } },
      h('path', { d: area, fill: color, opacity: 0.12 }),
      h('path', { d: d, fill: 'none', stroke: color, strokeWidth: 2, strokeLinejoin: 'round', strokeLinecap: 'round' }),
      h('rect', { x: last[0] - 2.5, y: last[1] - 2.5, width: 5, height: 5, fill: color })
    );
  }

  /* StatCard — a Life metric: label, big value + unit, a delta, and a sparkline. */
  function StatCard(props) {
    var up = props.deltaTone === 'up', down = props.deltaTone === 'down';
    var deltaColor = up ? 'var(--good-ink)' : down ? 'var(--ember-deep)' : 'var(--ink-muted)';
    return h('article', { style: { background: 'var(--surface)', backgroundImage: 'var(--grain-ink)', backgroundSize: '4px 4px', color: 'var(--ink)', border: '2px solid var(--parch-edge)', boxShadow: 'var(--parch-bevel)', padding: '15px 16px 14px', display: 'flex', flexDirection: 'column', gap: 11 } },
      h('div', { style: { display: 'flex', alignItems: 'center', gap: 8 } },
        props.icon ? h('img', { src: ic(props.icon), alt: '', decoding: 'async', style: { width: 18, height: 18, imageRendering: 'pixelated' } }) : null,
        h('span', { style: { fontFamily: 'var(--font-mono)', fontSize: 10, fontWeight: 700, letterSpacing: '.08em', textTransform: 'uppercase', color: 'var(--ink-muted)' } }, props.label)
      ),
      h('div', { style: { display: 'flex', alignItems: 'baseline', gap: 7, flexWrap: 'wrap' } },
        h('span', { style: { fontFamily: 'var(--font-display)', fontSize: 32, color: 'var(--ink)', lineHeight: 0.9 } }, props.value),
        props.unit ? h('span', { style: { fontFamily: 'var(--font-mono)', fontSize: 12, color: 'var(--ink-muted)' } }, props.unit) : null,
        props.delta ? h('span', { style: { fontFamily: 'var(--font-mono)', fontSize: 11.5, color: deltaColor, marginLeft: 'auto' } }, (up ? '\u25B2 ' : down ? '\u25BC ' : '') + props.delta) : null
      ),
      props.data ? h(Sparkline, { data: props.data, color: down ? 'var(--ember-deep)' : up ? 'var(--good-ink)' : 'var(--teal-ink)', width: props.spark || 150 }) : null
    );
  }

  window.BalaurData = { List: List, ListItem: ListItem, CalendarCell: CalendarCell, DayEntry: DayEntry, Sparkline: Sparkline, StatCard: StatCard };
})();
