# Orbit × Pixel Loom (BASM)

Orbit uses the locally vendored [BASM / Pixel Loom design system](https://github.com/alexradunet/basm-design-system) as its visual foundation.

## What is consumed directly

- `vendor/pixel-loom/tokens/indigo.css`
- `vendor/pixel-loom/fonts.css`
- self-hosted Newsreader, Work Sans, and JetBrains Mono font files

The files are vendored rather than loaded from a CDN so the browser and eventual desktop application remain local-first and work offline.

Orbit currently defaults to Pixel Loom's dark Indigo palette. The application maps its structural aliases to the design-system tokens:

```css
:root {
  --bg: var(--background);
  --panel: var(--surface-container-lowest);
  --panel-2: var(--surface-container-low);
  --line: var(--outline-variant);
  --text: var(--on-surface);
  --accent: var(--primary);
}
```

JSON Canvas preset colors remain separate because red, orange, yellow, green, cyan, and purple carry document meaning rather than application-theme meaning.

## Why base.css is not loaded directly

Pixel Loom's existing `base.css` targets content-oriented sites with global rules for `section`, headings, page width, and body sizing. Orbit is a full-screen spatial application, so those global rules would interfere with the canvas, sidebar, dialogs, and inspector.

Instead, Orbit adopts the system's semantic tokens and principles while applying component rules in its own application layer:

- Newsreader for card headings
- Work Sans for interface text
- JetBrains Mono for metadata
- 4px radii and spacing rhythm
- crisp outlines and tonal layers instead of elevation shadows
- ochre 4×4 corner notches
- self-hosted assets

This keeps the design system reusable without creating a large override sheet.

## CSS organization

```text
styles/
  layers.css             explicit cascade order
  foundation.css         Pixel Loom aliases, JSON Canvas colors, reset
  shell.css              header and sidebar
  canvas.css             nodes, edges, minimap, camera controls
  components.css         inspector, assistant, native dialogs
  themes.css             AI-selectable canvas themes
  responsive.css         narrow viewport adaptations
```

Every application file uses a named cascade layer. Files are loaded with standard `<link>` elements, so there is no CSS preprocessor, package dependency, or build process. Native landmarks, forms, labels, buttons, `dialog`, `nav`, `main`, `aside`, and `article` provide the semantic structure; classes identify application components rather than acting as styling utilities.

## Updating the vendored system

Copy fonts, `fonts.css`, and the selected token file from the BASM repository. Review token changes before updating because the application deliberately does not inherit `base.css` or `components.css` automatically.
