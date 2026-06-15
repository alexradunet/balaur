/* ChatExplore — the framed RPG conversation (the chosen direction).
   A wood-framed portrait sized to the message's minimum height, so the
   avatar's top and bottom borders line up with the speech panel's. The
   speaker's nameplate is embedded as a small tab straddling the panel's
   top border. Balaur speaks from the left, the owner answers from the right.
   Plain-JS React (no JSX). */
(function () {
  var R = window.React;
  if (!R) return;
  var h = R.createElement;

  var AV = 64; // avatar square = message minimum height

  var FULL = [
    { role: 'balaur', text: 'I am here. The hearth is lit and your words stay on this box. What shall we weigh today?' },
    { role: 'user', text: 'Remind me to water the tomatoes every two days, in the evenings.' },
    { role: 'balaur', text: 'Every second evening at 18:00, then. It is written \u2014 I shall nudge you when the hour comes.' },
    { role: 'user', text: 'And remember I prefer my notes as Markdown.' },
    { role: 'balaur', text: 'Kept. Markdown whenever I write for you. The garden and your notes are part of your story now.' },
  ];

  function frameMessage(m, who, src, key, pending) {
    var isUser = m.role === 'user';
    var accent = isUser ? 'var(--indigo-ink)' : 'var(--gold-ink)';
    var keyline = isUser ? 'var(--indigo)' : 'var(--gold-deep)';

    var portrait = h('div', { key: 'p', style: {
      width: AV, height: AV, flexShrink: 0, alignSelf: 'flex-start', boxSizing: 'border-box', overflow: 'hidden',
      border: '2px solid var(--outline-2)', background: '#101314', boxShadow: 'inset 0 0 0 2px ' + keyline + ', var(--drop-hard)', padding: 3,
      animation: pending ? 'basm-glow 1.6s ease-in-out infinite' : undefined,
    } }, h('img', { src: src, alt: '', decoding: 'async', style: { display: 'block', width: '100%', height: '100%', imageRendering: 'pixelated', transform: isUser ? 'scaleX(-1)' : 'none' } }));

    var panel = h('div', { key: 'm', style: {
      position: 'relative', minWidth: 0, minHeight: AV, boxSizing: 'border-box', display: 'flex', alignItems: 'center',
      background: 'var(--surface)', backgroundImage: 'var(--grain-ink)', backgroundSize: '4px 4px', color: 'var(--ink)',
      border: '2px solid ' + (pending ? 'var(--gold-deep)' : 'var(--parch-edge)'), boxShadow: 'var(--parch-bevel)', padding: '16px 16px 13px',
    } },
      // nameplate tab embedded on the top border
      h('div', { style: {
        position: 'absolute', top: 0, transform: 'translateY(-52%)', left: isUser ? 'auto' : 12, right: isUser ? 12 : 'auto',
        fontFamily: 'var(--font-mono)', fontSize: 10, fontWeight: 700, letterSpacing: '.07em', textTransform: 'uppercase', color: accent,
        background: 'var(--surface-2)', border: '2px solid var(--parch-edge)', boxShadow: 'inset 0 1px 0 rgba(255,255,255,.4)', padding: '1px 9px', lineHeight: 1.6,
      } }, who),
      pending
        ? h('span', { className: 'thinking thinking-dots' }, 'thinking')
        : h('div', { style: { fontSize: 16, lineHeight: 1.55, whiteSpace: 'pre-wrap', overflowWrap: 'anywhere', width: '100%' } }, m.text)
    );

    var cols = isUser ? [panel, portrait] : [portrait, panel];
    return h('div', { key: key, style: { display: 'flex', justifyContent: isUser ? 'flex-end' : 'flex-start' } },
      h('div', { style: { display: 'flex', gap: 10, alignItems: 'stretch', maxWidth: '88%' } }, cols));
  }

  /* A single message — the molecule. */
  function Message(props) {
    var role = props.role || 'balaur';
    var who = props.who || (role === 'user' ? 'You' : 'Balaur');
    var src = role === 'user' ? props.soulSrc : props.headSrc;
    return frameMessage({ role: role, text: props.text }, who, src, 'one', !!props.pending);
  }

  function ChatConversation(props) {
    var headSrc = props.headSrc, soulSrc = props.soulSrc, owner = props.owner || 'You';
    var msgs = props.compact ? [FULL[0], FULL[1]] : FULL;
    var children = msgs.map(function (m, i) {
      return frameMessage(m, m.role === 'user' ? owner : 'Balaur', m.role === 'user' ? soulSrc : headSrc, i);
    });
    if (props.withComposer && window.Composer) {
      children.push(h('div', { key: 'composer', style: { marginTop: 6 } },
        h(window.Composer, { who: owner, avatarSrc: soulSrc, placeholder: 'Speak; I am listening.', onSend: function () {} })));
    }
    return h('div', { style: { display: 'flex', flexDirection: 'column', gap: 20, width: '100%' } }, children);
  }

  window.ChatExplore = { ChatConversation: ChatConversation, Message: Message };
})();
