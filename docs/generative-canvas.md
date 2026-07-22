# Generative canvas and live cards

Balaur keeps generated presentation in two standard JSON Canvas file-node forms: declarative component-card Markdown and explicitly activated sandboxed HTML widgets. Models produce reviewed typed operations; they never execute generated code in the host page. The addressable-region ideas in Phil Holden's MIT-licensed [`partialupdate`](https://github.com/philholden/partialupdate) experiment remain reference material, not a shipped host-DOM or template-mutation API.

## Canonical life data and AI context

AI changes to life entities use the file repositories. A task, habit, journal entry, or calendar event is a canonical Markdown file; the in-memory index is only a disposable query projection. AI operations must not write projection rows or invent task-marker nodes. A task placement is a standard JSON Canvas `file` node, and removing a placement does not remove the canonical entity.

When an incoming AI context edge targets a canonical entity `file` node, Balaur preloads the referenced Markdown file and supplies its parsed title/body rather than treating the path as the content. Missing or unreadable files produce a diagnostic and a bounded fallback; they are never silently treated as the file body.

## Keep the host document standard

JSON Canvas 1.0 has no `component`, `html`, `widget`, or `webgl` node type. Balaur does not add one. A declarative card is a standard file node pointing to `cards/*.md`; a live widget is a standard file node pointing to `widgets/*.html`:

```json
{
  "id": "weekly-focus-placement",
  "type": "file",
  "file": "cards/weekly-focus--y-focus.md",
  "x": 100,
  "y": 100,
  "width": 360,
  "height": 220,
  "color": "5"
}
```

Other JSON Canvas clients still see valid file attachments. The immutable Orbit ID inside a component-card file is its identity; the safe widget path identifies widget source; the Canvas node ID is only a placement. One canonical file may be placed on multiple canvases. Component-card and widget catalogs preload parsed content, source, placements, and repair diagnostics so render does not read the vault once per card.

## Declarative component cards

Canonical `cards/*.md` files contain constrained frontmatter plus ordinary Markdown. The five recipes are metric, progress, callout, list, and timeline. `<balaur-component-card>` receives a parsed view model and renders only native declarative DOM—no source field, HTML injection, event handler, repository call, or executable host code is part of the schema. Invalid files render a readable raw fallback and diagnostic rather than disappearing or being overwritten.

Generated component-card operations are:

- `component-card.create`: allocate the immutable ID, safe canonical path, target Canvas, and standard file-node placement before review;
- `component-card.update`: patch allowlisted fields, optionally rename/rewrite placement paths, and optionally add a placement; and
- no generated delete operation—deletion remains a separate confirmed user action.

Validation rejects unknown/prototype-bearing data, unsupported operations, unsafe paths, unknown IDs/canvases, duplicate node IDs, invalid recipes and fields, bad colors/geometry, excessive operation counts/payloads, and an invalid resulting canvas. A recipe transition explicitly clears fields that no longer belong. Application calls go through `FileComponentCardRepository` with expected-hash checks. If a canonical create or update succeeds but its Canvas placement fails, the proposal reports the durable file and retries only the unfinished placement and untouched later operations.

## Typed AI operations

Canvas edits remain structured operations such as `node.add`, `node.update`, and `theme.set`. Shipped generated file operations add `component-card.create`, `component-card.update`, `widget.create`, and `widget.place`. The provider receives their closed schema. The app validates the complete evolving plan, provides a deterministic human-readable description including IDs, paths, fields, geometry, colors, and target Canvas, and requires approval.

`widget.create` discloses the complete self-contained HTML source and capability limits. Approval writes and places the canonical file but deliberately leaves it inactive. `widget.place` adds another standard file-node placement for an existing canonical source file. Widget deletion is not a generated operation.

The stable browser integration surface includes:

```js
window.orbitCanvas.getDocument()
window.orbitCanvas.getSummary()
window.orbitCanvas.validateOperations(operations)
window.orbitCanvas.applyOperations(operations)
```

Validation and confirmation are controls around canonical repository writes; provider output never receives direct repository, vault, DOM, or host-code execution access.

## Prompt-first AI notes

An AI note is a one-shot generation flow. Balaur first opens a native `<dialog>` for the question, calls the configured provider only after submission, and adds the resulting Markdown as an ordinary JSON Canvas text node. No placeholder node is created when the dialog is cancelled or the request fails.

## Reactive AI operator cards

An AI operator remains a standard JSON Canvas text node. Balaur recognizes the existing portable Markdown compatibility marker rather than introducing a custom node type:

```markdown
<!-- orbit:ai-card -->
# Weekly synthesis
Summarize the connected notes and recommend the next action.
```

Incoming edges define its context. The operator sends the prompt and connected node content to the configured provider, creates a standard text node for the result, and connects it with an edge labeled `AI output`. Subsequent executions update that same note instead of generating duplicates.

Balaur computes signatures from the prompt, incoming edge set, and input content. Content or connection changes queue a debounced regeneration. Coordinates and card dimensions do not trigger requests. Directed cycles pause automatic execution, and an update arriving during a request queues one follow-up run.

This representation remains readable in other JSON Canvas clients: they see ordinary text nodes and edges even if they do not understand the compatibility marker.

## AI request flow

1. The app derives a bounded context from the current canvas, selected and related nodes, canonical file bodies, open tasks, known component cards, and safe widget metadata.
2. The model returns a typed plain-data plan, not executable host-page code.
3. The app validates the complete evolving plan and resulting canvas and shows a deterministic review.
4. The user approves or discards it.
5. Canvas-only changes use validated canvas operations. Component cards and widgets use their file-first repositories.
6. Successful canonical writes reconcile catalogs and canvas placements; partial placement failure remains recoverable without repeating the durable write.

Browser checks cover the local component-card and widget proposal/apply flows, canonical create/update/place, controlled reload, multiple card placements, whole-space export/import, and file-first recovery after forced placement failures. Remote provider correctness and provider-specific failure behavior remain dependent on the configured external service.

The model should receive only the relevant canvas slice by default, not every attachment in a workspace.

## Widget review, sandbox, and lifecycle

Generated widget source passes a conservative dependency-free scanner before it can be written or executed. The scanner rejects external resource/navigation/form/worker/frame channels, oversized or ambiguous markup/script/style, and animation without reduced-motion handling. This is review-time validation, not the security boundary.

After explicit **Run**, the host builds a trusted document in this order: restrictive CSP and referrer metadata, trusted bootstrap, reviewed generated source, then the trusted diagnostic boundary. The CSP is:

```text
default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline';
img-src data: blob:; media-src data: blob:; font-src 'none';
connect-src 'none'; frame-src 'none'; worker-src 'none'; object-src 'none';
base-uri 'none'; form-action 'none'
```

The document is loaded from a Blob URL into an opaque-origin iframe with exactly `sandbox="allow-scripts"`, `referrerpolicy="no-referrer"`, `loading="lazy"`, and no Permissions Policy capabilities. Omitting `allow-same-origin` prevents access to the host origin, cookies, storage, or DOM. A transferred private `MessageChannel` accepts only closed version-1 host/widget schemas. The host sends bounded theme tokens, reduced-motion/transparency/contrast preferences, visibility, and pause; the widget may send ready, heartbeat, bounded status/diagnostic/resize messages. Canonical content, secrets, mutation commands, repositories, callbacks, and host objects are never projected.

Limits are exact: 128 KiB source, 500 static elements, 64 KiB aggregate script, 64 KiB aggregate style, 64 KiB serialized message, six active widgets, 30 messages/second sustained with burst 60, and three missed five-second heartbeats. Pause removes the iframe, closes the channel, and revokes the Blob URL. Reload creates a fresh document/channel. Source, path, or title changes; disconnect; policy/schema/rate failures; missed heartbeats; and unexpected post-initialization loads all tear down execution. Widgets never auto-restart.

These are least-capability and lifecycle controls, not hard CPU isolation. The CSP and scanner cover the supported request/resource channels, while hostile browser probes confirm that tested parent DOM/storage/cookie/fetch/image/font/form/popup/navigation/worker/frame attempts fail. Unexpected self-navigation is detected and paused; Balaur does not claim that the attempted request was suppressed. A malicious script can still consume its frame's main-thread time until lifecycle controls or the browser intervene.

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

## Security boundary summary

- Generated host-page code is never an operation.
- Declarative component cards render allowlisted data through native DOM.
- Widget source is fully disclosed, saved inactive, and runs only after explicit review and Run.
- The opaque-origin sandbox omits same-origin, navigation, popup, form, download, device, and filesystem permissions.
- CSP, conservative source validation, closed private-channel schemas, message/source/element caps, rate limits, active-instance caps, heartbeats, visibility handling, and teardown are all enforced.
- Generated widgets receive no canonical/user data, provider key, repository, host object, callback, or mutation command.
- Provider calls remain outside the Service Worker and are unavailable offline.
- Provider keys remain in `sessionStorage` unless the user explicitly enables **Remember API key**.
- Self-navigation is detected and torn down; hard request suppression and hard CPU isolation are explicitly not claimed.
