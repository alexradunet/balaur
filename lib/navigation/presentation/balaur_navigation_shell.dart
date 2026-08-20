import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/material.dart';

enum BalaurDestination {
  chat('/chat'),
  settings('/settings');

  const BalaurDestination(this.location);

  final String location;
}

class BalaurNavigationShell extends StatelessWidget {
  static const _links = ['chat', 'settings'];
  const BalaurNavigationShell({
    super.key,
    required this.active,
    required this.onSelect,
    required this.child,
    this.onToggleTheme,
    this.themeGlyph = '◑',
  });

  final BalaurDestination active;
  final ValueChanged<BalaurDestination> onSelect;
  final Widget child;
  final VoidCallback? onToggleTheme;
  final String themeGlyph;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: BalaurTopbar(
        links: _links,
        active: active.name,
        onNavigate: (name) => onSelect(
          name == 'home'
              ? BalaurDestination.chat
              : BalaurDestination.values.byName(name),
        ),
        onToggleTheme: onToggleTheme,
        themeGlyph: themeGlyph,
      ),
      body: child,
    );
  }
}
