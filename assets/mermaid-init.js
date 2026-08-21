const mermaidScript = document.createElement('script');
mermaidScript.src = 'https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.min.js';
mermaidScript.onload = async () => {
  mermaid.initialize({
    startOnLoad: false,
    theme: window.matchMedia('(prefers-color-scheme: dark)').matches
      ? 'dark'
      : 'neutral',
  });
  await mermaid.run({ querySelector: '.mermaid' });
};
document.head.appendChild(mermaidScript);
