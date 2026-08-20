---
name: widgetbook-story-first
description: |
  Develop Balaur Flutter widgets, screens, themes, and design tokens with
  Widgetbook Stories first. Use before each reusable UI addition or change,
  including responsive, state, accessibility, and design-system work.
---

# Story-first UI development

Use a Story as the first integration point for reusable user interface work.
Build the widget in isolation before you connect it to an application screen.

Read `../widgetbook-v4/SKILL.md` before you change Widgetbook files.
Read its linked Story or testing reference when that branch applies.

## Workflow

1. Define the widget interface and list each meaningful visible state.
2. Create or update the mirrored Story file in `widgetbook/lib/`.
3. Declare `Meta`, then run generation before you use generated types.
4. Add one `$Story` for each state that developers must browse.
5. Use the production Balaur theme and design-system widgets.
6. Develop the implementation while Widgetbook runs in isolation.
7. Check compact, wide, light, dark, and large-text states when applicable.
8. Run the Widgetbook Scenario tests.
9. Integrate the widget into the application only after isolated checks pass.
10. Run the root analysis and tests after application integration.

Use watch mode during steps 3 through 7:

```bash
cd widgetbook
dart run build_runner watch
```

Run the isolated catalog in a second terminal:

```bash
cd widgetbook
flutter run -d linux
```

## State rules

Use one Story for each browsable state.
Use Scenarios for fixed captures, interactions, and environment Modes.
Use Args to make useful visual inputs adjustable.
Keep callbacks and infrastructure adapters fixed or fake.
Never use a live provider key or a production network dependency in a Story.

## Screen rules

A screen is also a Story target.
Give each changed screen a bounded phone viewport and a narrow viewport.
Also check the wide layout that Balaur supports.
Provide fake adapters at the existing composition seam.
Keep data and agent behavior outside the design-system module.

## Completion criterion

The widget has current Stories for all meaningful states.
The catalog builds and runs without runtime errors.
The applicable Scenario tests, root tests, and analysis pass.
