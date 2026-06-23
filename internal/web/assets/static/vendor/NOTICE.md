# Vendored client-side assets

Third-party assets embedded verbatim (no build step). Pinned by version + sha256;
re-vendoring must update the hash here.

## force-graph.min.js

- **Library:** force-graph (2D Canvas force-directed graph) by Vasco Asturiano
- **Version:** 1.51.4
- **License:** MIT
- **Source:** https://github.com/vasturiano/force-graph
- **Fetched from:** https://unpkg.com/force-graph@1.51.4/dist/force-graph.min.js
- **sha256:** `1008539bb9e171a0dc343453366451a1b3a6ded06028ef4f978608b658ba2d0a`
- **Global:** `window.ForceGraph` (UMD)
- **Used by:** `/static/graph-canvas.js` to render the interactive graph card.
