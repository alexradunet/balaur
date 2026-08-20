# Flutter Resources

## Knowledge

- [Flutter learning pathway](https://docs.flutter.dev/learn/pathway)
  Official guided introduction to Flutter's declarative UI model. Use for: widgets, layout, input, state, and the first small applications.
- [Flutter architectural overview](https://docs.flutter.dev/resources/architectural-overview)
  Primary explanation of the widget, element, and render trees and the framework-to-engine layers. Use for: building an accurate mental model and debugging rebuild/layout behavior.
- [Flutter application architecture guide](https://docs.flutter.dev/app-architecture/guide)
  Official opinionated guide to separating UI and data responsibilities. Use for: evolving the household MVP beyond toy-sized `StatefulWidget` state.
- [Flutter interactivity guide](https://docs.flutter.dev/ui/interactivity)
  Official guide to stateful widgets, callbacks, and state ownership. Use for: local interactions and parent-child update paths.
- [Ephemeral state and application state](https://docs.flutter.dev/data-and-backend/state-mgmt/ephemeral-vs-app)
  Official comparison of local UI state and shared application state. Use for: selecting the correct owner for each household value.
- [Dart language documentation](https://dart.dev/language)
  Canonical language reference. Use for: Dart syntax, types, patterns, asynchronous code, and language semantics rather than relying on JS analogies.
- [`StatefulWidget` API documentation](https://api.flutter.dev/flutter/widgets/StatefulWidget-class.html)
  Precise framework contract for mutable UI state and widget identity. Use for: lifecycle, state placement, keys, and rebuild-cost questions.
- [Flutter testing overview](https://docs.flutter.dev/testing/overview)
  Official guide to unit, widget, and integration tests. Use for: selecting the fastest feedback loop for each household behavior.
- [Flutter adaptive and responsive design](https://docs.flutter.dev/ui/adaptive-responsive)
  Official guidance for adapting one app to different screen sizes and capabilities. Use for: phone and always-on household display interfaces.

## Wisdom (Communities)

- [Flutter community channels](https://flutter.dev/community)
  Official directory of community spaces and events. Use for: finding practitioner feedback on platform behavior and production trade-offs.
- [Flutter issue tracker](https://github.com/flutter/flutter/issues)
  Maintainer and contributor discussions grounded in reproducible framework issues. Use for: confirming suspected framework defects after creating a minimal reproduction.

## Gaps

- The MVP's backend, synchronization, offline, privacy, and agent-safety requirements are not yet concrete enough to curate implementation resources.
- The first end-to-end household workflow has not yet been selected, so domain-specific sources remain intentionally uncurated.
