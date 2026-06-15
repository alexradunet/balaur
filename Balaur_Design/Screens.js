/* Screens — composed reference layouts that assemble the kit into real pages.
   Life dashboard: Topbar + metric cards + week strip + day timeline + habits
   list + a standing alert + the Composer ledge. Plain-JS React (no JSX). */
(function () {
  var R = window.React;
  if (!R) return;
  var h = R.createElement;
  var NS = function () { return window.BalaurBasmDesignSystem_0c1b20 || {}; };
  var D = function () { return window.BalaurData || {}; };
  var U = function () { return window.BalaurUtil || {}; };
  var P = function () { return window.BalaurPrim || {}; };

  var HEAD = 'assets/avatars/balaur-01.png';
  var SOUL = 'assets/avatars/soul-05.png';
  var NAV = ['chat', 'tasks', 'memory', 'life', 'profile'];

  function sectionLabel(text, accent) {
    return h('div', { style: { display: 'flex', alignItems: 'center', gap: 10, marginBottom: 13 } },
      h('span', { style: { fontFamily: 'var(--font-mono)', fontSize: 11, letterSpacing: '.1em', textTransform: 'uppercase', color: accent || 'var(--gold)', whiteSpace: 'nowrap' } }, text),
      h('span', { style: { flex: 1, height: 2, background: 'linear-gradient(to right, var(--hair) 50%, transparent 50%) 0 0/8px 2px repeat-x' } })
    );
  }

  function LifeDashboard(props) {
    var ds = NS(), data = D(), util = U();
    var week = [
      { day: 9, pips: 1 }, { day: 10, pips: 0 }, { day: 11, pips: 2 }, { day: 12, pips: 1 },
      { day: 13, pips: 3 }, { day: 14, pips: 2, today: true }, { day: 15, pips: 0, selected: true },
    ];

    var topbar = ds.Topbar ? h(ds.Topbar, { brand: 'Balaur', crestSrc: 'assets/crest.png', links: NAV, active: 'life', onNavigate: function () {}, onToggleTheme: props.onToggleTheme || function () {}, themeGlyph: '\u25D1' }) : null;

    var stats = h('div', { style: { display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(min(220px,100%),1fr))', gap: 16 } },
      data.StatCard ? h(data.StatCard, { icon: 'gem', label: 'Weight', value: '81.2', unit: 'kg', delta: '0.6 this week', deltaTone: 'down', data: [83, 82.6, 82.1, 82.4, 81.9, 81.6, 81.2] }) : null,
      data.StatCard ? h(data.StatCard, { icon: 'flame', label: 'Steps', value: '8,210', delta: '12% vs avg', deltaTone: 'up', data: [6100, 7200, 5400, 8900, 7600, 9100, 8210] }) : null,
      data.StatCard ? h(data.StatCard, { icon: 'hourglass', label: 'Sleep', value: '7.1', unit: 'h', delta: 'steady', deltaTone: 'flat', data: [6.8, 7.2, 6.5, 7.4, 7.0, 7.3, 7.1] }) : null
    );

    var weekStrip = h('div', { style: { display: 'grid', gridTemplateColumns: 'repeat(7, 1fr)', gap: 8 } },
      week.map(function (c, i) { return data.CalendarCell ? h(data.CalendarCell, Object.assign({ key: i }, c)) : null; }));

    var timeline = h('div', { style: { background: 'var(--surface)', backgroundImage: 'var(--grain-ink)', backgroundSize: '4px 4px', border: '2px solid var(--parch-edge)', boxShadow: 'var(--parch-bevel)', padding: '18px 18px 6px' } },
      data.DayEntry ? h(data.DayEntry, { time: '06:30', title: 'Fed the hens', detail: 'daily \u00b7 streak 12', tone: 'gold' }) : null,
      data.DayEntry ? h(data.DayEntry, { time: '07:15', title: 'Logged sleep \u2014 7.1 h', detail: 'life log', tone: 'teal' }) : null,
      data.DayEntry ? h(data.DayEntry, { time: '13:00', title: 'Logged weight \u2014 81.2 kg', detail: 'down 0.6 this week', tone: 'teal' }) : null,
      data.DayEntry ? h(data.DayEntry, { time: '18:00', title: 'Water the tomatoes', detail: 'every 2 days \u00b7 due', tone: 'ember', last: true }) : null
    );

    var habits = data.List ? h(data.List, { title: 'Habits', items: [
      { icon: 'flame', title: 'Morning walk', subtitle: 'daily', meta: 'streak 8' },
      { icon: 'gem', title: 'Weigh-in', subtitle: 'every morning', meta: 'done' },
      { icon: 'tome', title: 'Evening pages', subtitle: 'most nights', meta: 'skipped', metaTone: 'warn' },
    ] }) : null;

    var alert = util.Alert ? h(util.Alert, { tone: 'info', title: 'Running on llama-3.1', children: 'A local head. Your measures stay on this box \u2014 nothing here is sent anywhere.' }) : null;

    var composer = window.Composer ? h(window.Composer, { who: 'You', avatarSrc: SOUL, placeholder: 'Log a measure, or ask about your week\u2026', sendLabel: 'Log', onSend: function () {} }) : null;

    return h('div', { style: { flex: 1, minHeight: 0, height: '100%', display: 'flex', flexDirection: 'column', background: 'var(--bg)', backgroundImage: 'var(--grain-warm)', backgroundSize: '4px 4px' } },
      h('div', { style: { position: 'relative', zIndex: 5 } }, topbar),
      h('div', { style: { flex: 1, overflowY: 'auto', padding: '26px clamp(16px, 4vw, 40px) 30px' } },
        h('div', { style: { maxWidth: 980, margin: '0 auto', display: 'flex', flexDirection: 'column', gap: 26 } },
          h('div', null,
            h('div', { style: { fontFamily: 'var(--font-mono)', fontSize: 11, letterSpacing: '.12em', textTransform: 'uppercase', color: 'var(--gold)', marginBottom: 7 } }, 'Tuesday \u00b7 14 May'),
            h('h1', { style: { margin: 0, fontFamily: 'var(--font-display)', fontSize: 'clamp(28px, 5vw, 40px)', color: 'var(--fg-strong)', lineHeight: 1 } }, 'Your week, kept.')
          ),
          alert,
          h('section', null, sectionLabel('Measures'), stats),
          h('div', { style: { display: 'grid', gridTemplateColumns: 'minmax(0, 1.4fr) minmax(0, 1fr)', gap: 22, alignItems: 'start' }, className: 'life-cols' },
            h('section', null, sectionLabel('Today'), timeline),
            h('section', null, sectionLabel('This week', 'var(--violet)'),
              h('div', { style: { display: 'flex', flexDirection: 'column', gap: 18 } }, weekStrip, habits))
          )
        )
      ),
      h('div', { style: { borderTop: '2px solid var(--outline-2)', padding: 'clamp(12px,3vw,18px) clamp(16px,4vw,40px)', background: 'var(--bg)' } },
        h('div', { style: { maxWidth: 980, margin: '0 auto' } }, composer))
    );
  }

  function appFrameInner(top, scrollBody, footer) {
    return h('div', { style: { flex: 1, minHeight: 0, height: '100%', display: 'flex', flexDirection: 'column', background: 'var(--bg)', backgroundImage: 'var(--grain-warm)', backgroundSize: '4px 4px' } },
      h('div', { style: { position: 'relative', zIndex: 5 } }, top),
      h('div', { style: { flex: 1, overflowY: 'auto', padding: '24px clamp(16px, 4vw, 40px) 28px' } },
        h('div', { style: { maxWidth: 860, margin: '0 auto', display: 'flex', flexDirection: 'column', gap: 22 } }, scrollBody)),
      footer ? h('div', { style: { borderTop: '2px solid var(--outline-2)', padding: 'clamp(12px,3vw,18px) clamp(16px,4vw,40px)', background: 'var(--bg)' } },
        h('div', { style: { maxWidth: 860, margin: '0 auto' } }, footer)) : null
    );
  }

  function screenTitle(eyebrow, title) {
    return h('div', null,
      h('div', { style: { fontFamily: 'var(--font-mono)', fontSize: 11, letterSpacing: '.12em', textTransform: 'uppercase', color: 'var(--gold)', marginBottom: 7 } }, eyebrow),
      h('h1', { style: { margin: 0, fontFamily: 'var(--font-display)', fontSize: 'clamp(28px, 5vw, 40px)', color: 'var(--fg-strong)', lineHeight: 1 } }, title)
    );
  }

  /* Tasks page — Topbar, a tab filter, the evening nudge, open task cards and a
     closed group, with the Composer ledge to add a task. */
  function TasksPage(props) {
    var ds = NS(), domain = window.BalaurDomain || {};
    var tabState = R.useState('open'), tab = tabState[0], setTab = tabState[1];
    var TABS = ['all', 'open', 'done', 'snoozed'];

    var open = [
      { title: 'Water the tomatoes', status: 'open', dueLine: 'due today 18:00', recurLine: 'every 2 days' },
      { title: 'Mend the deer fence', status: 'open', dueLine: 'overdue \u00b7 yesterday', overdue: true },
      { title: 'Call the seed supplier', status: 'open', dueLine: 'due Fri', notes: 'Ask about the heritage tomato stock.' },
    ];
    var closed = [
      { title: 'Feed the hens', status: 'done', dueLine: '', recurLine: 'daily 06:30' },
      { title: 'Move the firewood', status: 'snoozed', dueLine: 'snoozed \u00b7 tomorrow' },
    ];
    var showOpen = tab === 'all' || tab === 'open';
    var showClosed = tab === 'all' || tab === 'done' || tab === 'snoozed';
    var closedFiltered = closed.filter(function (c) { return tab === 'all' ? true : c.status === tab; });

    function cardList(items) {
      return h('div', { style: { display: 'flex', flexDirection: 'column', gap: 12 } },
        items.map(function (t, i) { return ds.TaskCard ? h(ds.TaskCard, Object.assign({ key: i, onDone: function () {}, onSnooze: function () {}, onDrop: function () {} }, t)) : null; }));
    }

    var topbar = ds.Topbar ? h(ds.Topbar, { brand: 'Balaur', crestSrc: 'assets/crest.png', links: NAV, active: 'tasks', onNavigate: function () {}, onToggleTheme: props.onToggleTheme || function () {}, themeGlyph: '\u25D1' }) : null;
    var tabs = ds.Tabs ? h(ds.Tabs, { items: TABS, active: tab, onSelect: setTab }) : null;
    var nudge = domain.NudgeBanner ? h(domain.NudgeBanner, { when: '18:00', message: 'The evening comes, and the tomatoes thirst. Will you tend them now?', onDone: function () {}, onSnooze: function () {}, onTomorrow: function () {} }) : null;
    var composer = window.Composer ? h(window.Composer, { who: 'You', avatarSrc: SOUL, placeholder: 'Add a task, or ask what\u2019s due\u2026', sendLabel: 'Add', onSend: function () {} }) : null;

    var body = [
      h('div', { key: 'h', style: { display: 'flex', flexDirection: 'column', gap: 15 } }, screenTitle('Tuesday \u00b7 14 May', 'On the book.'), h('div', null, tabs)),
      nudge ? h('div', { key: 'n' }, nudge) : null,
      showOpen ? h('section', { key: 'o' }, sectionLabel('Active'), cardList(open)) : null,
      showClosed && closedFiltered.length ? h('section', { key: 'c' },
        ds.Stitch ? h(ds.Stitch, { style: { margin: '2px 0 16px' } }) : null,
        sectionLabel('Settled', 'var(--smoke)'), cardList(closedFiltered)) : null
    ];
    return appFrameInner(topbar, body, composer);
  }

  /* Chat view — Topbar, a recap of earlier context, the framed conversation
     with a tool event woven in, and the Composer in the owner's seat. */
  function ChatView(props) {
    var ds = NS(), domain = window.BalaurDomain || {}, chat = window.ChatExplore || {};
    var msg = function (role, text, key) { return chat.Message ? h(chat.Message, { key: key, role: role, text: text, headSrc: HEAD, soulSrc: SOUL }) : null; };

    var topbar = ds.Topbar ? h(ds.Topbar, { brand: 'Balaur', crestSrc: 'assets/crest.png', links: NAV, active: 'chat', onNavigate: function () {}, onToggleTheme: props.onToggleTheme || function () {}, themeGlyph: '\u25D1' }) : null;
    var recap = domain.RecapCard ? h(domain.RecapCard, { when: 'earlier today', summary: 'We planned the orchard work and set the tomato watering. You asked me to keep two things.', points: ['Garden \u2014 tomatoes & peppers, watered at dusk', 'Notes exported as Markdown'] }) : null;
    var toolRow = ds.ToolRow ? h('div', { style: { '--portrait-size': '64px', '--chat-gutter': '74px' } }, h(ds.ToolRow, { tool: 'task_add', icon: 'assets/icons/quill.png' }, 'added task: water the tomatoes \u00b7 every 2 days 18:00')) : null;
    var composer = window.Composer ? h(window.Composer, { who: 'You', avatarSrc: SOUL, placeholder: 'Speak; I am listening.', sendLabel: 'Send', onSend: function () {} }) : null;

    var conversation = h('div', { style: { display: 'flex', flexDirection: 'column', gap: 20, '--portrait-size': '64px', '--chat-gutter': '74px' } },
      msg('balaur', 'I am here. The hearth is lit and your words stay on this box. What shall we weigh today?', 'm1'),
      msg('user', 'Remind me to water the tomatoes every two days, in the evenings.', 'm2'),
      toolRow,
      msg('balaur', 'Every second evening at 18:00, then. It is written \u2014 I shall nudge you when the hour comes.', 'm3'),
      msg('user', 'And remember I prefer my notes as Markdown.', 'm4'),
      msg('balaur', 'Kept. Markdown whenever I write for you. The garden and your notes are part of your story now.', 'm5')
    );

    return appFrameInner(topbar, [recap ? h('div', { key: 'r' }, recap) : null, conversation], composer);
  }

  window.BalaurScreens = { LifeDashboard: LifeDashboard, TasksPage: TasksPage, ChatView: ChatView };
})();
