# Balaur Widgetbook

This Flutter package builds the Balaur user interface in isolation.
It depends on the root Balaur package through a local path.

Keep each Story under `lib/` in the mirrored production path.
Use one Story for each meaningful state that developers must browse.

## Develop

Run generation in watch mode:

```bash
dart run build_runner watch
```

Run the catalog in a second terminal:

```bash
flutter run -d linux
```

## Validate

```bash
dart run build_runner build
dart format lib
flutter analyze
flutter test
flutter build linux --debug
flutter build web
```

Widgetbook is pinned to `4.0.0-beta.11`.
Do not use Widgetbook 3 annotations or knobs.
