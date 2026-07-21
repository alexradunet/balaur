# Generative canvas and live cards

This is the proposed bridge between JSON Canvas, live HTML/Three.js cards, and the ideas in Phil Holden's MIT-licensed [`partialupdate`](https://github.com/philholden/partialupdate) experiment.

## Keep the host document standard

JSON Canvas 1.0 has no `html`, `widget`, or `threejs` node type. Adding one would make the document an application-specific dialect. Instead, Orbit treats a standard file node whose `file` ends in `.html` as a live card:

```json
{
  "id": "focus-orbit",
  "type": "file",
  "file": "widgets/focus-orbit.html",
  "x": 100,
  "y": 100,
  "width": 480,
  "height": 290,
  "color": "5"
}
```

Other JSON Canvas clients still see a valid file attachment. Orbit renders it in an iframe with `sandbox="allow-scripts"`. HTML, CSS, SVG, Canvas 2D, and Three.js can all run inside that boundary.

The sandbox deliberately omits `allow-same-origin`, top navigation, popups, forms, downloads, camera, and microphone permissions. A future desktop build should serve workspace attachments through an app protocol rather than grant widgets direct filesystem access.

## Partial updates

The useful idea from `partialupdate` is addressable UI regions, not unrestricted access to the application DOM. A live card can expose regions such as:

```text
/nodes/focus-orbit/style
/nodes/focus-orbit/content
/nodes/focus-orbit/data/progress
```

The AI can return a `<template for="…">` update for one region without regenerating the card or canvas. The card runtime receives updates over `postMessage`; only the sandbox applies generated HTML/CSS/JS.

For canvas data, use structured operations rather than HTML templates:

```json
[
  {
    "type": "node.update",
    "id": "goal-1",
    "patch": { "text": "# Run 10K\n\n- [x] Choose a plan" }
  },
  {
    "type": "theme.set",
    "theme": "warm"
  }
]
```

The current demo exposes this boundary as:

```js
window.orbitCanvas.getDocument()
window.orbitCanvas.getSummary()
window.orbitCanvas.applyOperations(operations)
```

All operations are allowlisted and the resulting document is validated before commit.

## Reactive AI operator cards

An AI operator remains a standard JSON Canvas text node. Orbit recognizes a portable Markdown marker rather than introducing a custom node type:

```markdown
<!-- orbit:ai-card -->
# Weekly synthesis
Summarize the connected notes and recommend the next action.
```

Incoming edges define its context. The operator sends the prompt and connected node content to the configured provider, creates a standard text node for the result, and connects it with an edge labeled `AI output`. Subsequent executions update that same note instead of generating duplicates.

Orbit computes signatures from the prompt, incoming edge set, and input content. Content or connection changes queue a debounced regeneration. Coordinates and card dimensions do not trigger requests. Directed cycles pause automatic execution, and an update arriving during a request queues one follow-up run.

This representation remains readable in other JSON Canvas clients: they see ordinary text nodes and edges even if they do not understand Orbit's marker.

## AI request flow

1. The app derives a compact context: visible nodes, selected nodes, nearby relationships, open tasks, and available widget regions.
2. The model returns a typed plan, not executable host-page code.
3. The app validates the plan and shows a human-readable preview.
4. The user approves changes.
5. A single transaction applies the plan and records an inverse transaction for undo.
6. Widget-region changes are sent to the relevant sandbox through `postMessage`.

The model should receive only the relevant canvas slice by default, not every attachment in a workspace.

## Canvas styling

Theme state is not part of JSON Canvas 1.0. Store it in optional workspace metadata:

```json
{
  "document": "life.canvas",
  "theme": {
    "preset": "warm",
    "tokens": {
      "canvas.background": "#15110e",
      "card.radius": 10,
      "edge.width": 1.5
    }
  }
}
```

Models should update validated design tokens. Do not let a model inject arbitrary CSS into the host app. Arbitrary CSS is acceptable only inside a sandboxed live card.

## Provider modes

The GitHub Pages demo supports two dependency-free modes:

1. local canvas commands with no network access;
2. direct browser calls to an OpenAI-compatible `/chat/completions` endpoint.

Mistral can be configured with `https://api.mistral.ai/v1` and a model such as `mistral-small-latest`. Provider metadata is saved locally. The secret remains in `sessionStorage` by default, or in `localStorage` only when the user explicitly enables **Remember API key**.

Direct client access is useful for a personal proof of concept, but any script running under the application's origin can potentially read a browser-stored key. A distributed multi-user product should use a trusted backend or local model service. The Cloudflare Worker and Durable Object architecture in `partialupdate` remains a useful reference for streaming, authorization, rate limits, and multi-user sessions.

## Required hardening

- Content Security Policy for host and widget documents
- strict `postMessage` schema and source checks
- operation count and payload-size limits
- model/tool rate limits and spending limits
- no implicit network permission for generated widgets
- user confirmation for deletions, bulk updates, and external requests
- transaction log, undo, and recovery after interrupted writes
- sanitize generated HTML even inside the sandbox to reduce phishing and resource abuse

`partialupdate` itself warns that generated scripts can submit prompts or requests in loops. Isolation and budgets are product requirements, not later polish.
