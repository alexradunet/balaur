---
name: widgetbook-v4
description: |
  Configure and use Widgetbook 4.0.0-beta.11 in Balaur. Use for catalog
  setup, stories, generated Args, Scenarios, addons, isolated UI checks,
  accessibility captures, or Widgetbook migration questions.
---

# Widgetbook 4 beta.11

Use the Widgetbook 4.0.0-beta.11 interface only.
Widgetbook 3 examples use incompatible annotations and generated files.
Keep `widgetbook: 4.0.0-beta.11` pinned until this skill is updated.

The Balaur catalog is a separate Flutter package in `widgetbook/`.
Production widgets stay in the root `lib/` directory.
Story files stay in `widgetbook/lib/`.

Load only the reference that the task needs:

- Read [SETUP.md](SETUP.md) for installation, configuration, and addons.
- Read [STORIES.md](STORIES.md) before you add or change a Story.
- Read [TESTING.md](TESTING.md) for Scenarios, captures, or validation.

## Working sequence

1. Read the production widget and its related Story files.
2. Run generation after each `Meta` or constructor change.
3. Use one `$Story` for each meaningful state that developers must browse.
4. Use a Scenario for fixed captures, interactions, or environment Modes.
5. Run tests from the `widgetbook/` package.
6. Inspect compact, wide, light, dark, and large-text states when applicable.

Use these commands from `widgetbook/`:

```bash
dart run build_runner build
dart format lib
flutter test
flutter run -d linux
```

Use watch mode during isolated development:

```bash
dart run build_runner watch
```

Completion criterion: generated code is current, all Stories build, and the
isolated checks cover each meaningful state.
