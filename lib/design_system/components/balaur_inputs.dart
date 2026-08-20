import 'package:balaur/design_system/components/balaur_surface.dart';
import 'package:balaur/design_system/foundations/balaur_colors.dart';
import 'package:flutter/material.dart';

/// Displays a labelled parchment text field.
class BalaurTextField extends StatelessWidget {
  const BalaurTextField({
    super.key,
    this.controller,
    this.label,
    this.placeholder,
    this.hint,
    this.error,
    this.enabled = true,
    this.maxLines = 1,
    this.onChanged,
    this.onSubmitted,
  });

  final TextEditingController? controller;
  final String? label;
  final String? placeholder;
  final String? hint;
  final String? error;
  final bool enabled;
  final int? maxLines;
  final ValueChanged<String>? onChanged;
  final ValueChanged<String>? onSubmitted;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        if (label case final label?) ...[
          Text(
            label.toUpperCase(),
            style: Theme.of(context).textTheme.labelSmall?.copyWith(
              color: colors.muted,
              fontWeight: FontWeight.w700,
              letterSpacing: 0.8,
            ),
          ),
          const SizedBox(height: 6),
        ],
        TextField(
          controller: controller,
          enabled: enabled,
          maxLines: maxLines,
          onChanged: onChanged,
          onSubmitted: onSubmitted,
          style: Theme.of(context).textTheme.bodyMedium
              ?.copyWith(color: colors.ink),
          decoration: InputDecoration(
            hintText: placeholder,
            errorText: error,
            helperText: error == null ? hint : null,
            contentPadding: const EdgeInsets.symmetric(
              horizontal: 12,
              vertical: 10,
            ),
          ),
        ),
      ],
    );
  }
}

/// Displays a square Hearthwood toggle.
class BalaurToggle extends StatelessWidget {
  const BalaurToggle({
    super.key,
    required this.checked,
    required this.onChanged,
    this.label,
    this.enabled = true,
  });

  final bool checked;
  final ValueChanged<bool>? onChanged;
  final String? label;
  final bool enabled;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    final canChange = enabled && onChanged != null;
    final track = SizedBox.square(
      dimension: 48,
      child: Center(
        child: Opacity(
          opacity: enabled ? 1 : 0.5,
          child: BalaurSurface(
            material: BalaurSurfaceMaterial.inset,
            hardShadow: false,
            child: SizedBox(
              width: 46,
              height: 26,
              child: AnimatedAlign(
                duration: const Duration(milliseconds: 80),
                alignment: checked
                    ? Alignment.centerRight
                    : Alignment.centerLeft,
                child: Container(
                  width: 18,
                  height: 18,
                  margin: const EdgeInsets.symmetric(horizontal: 2),
                  decoration: BoxDecoration(
                    color: checked ? colors.gold : colors.surface2,
                    border: Border.all(color: colors.outline, width: 2),
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
    return Semantics(
      container: true,
      label: label ?? 'Toggle',
      toggled: checked,
      enabled: canChange,
      button: true,
      child: ExcludeSemantics(
        child: GestureDetector(
          behavior: HitTestBehavior.opaque,
          onTap: canChange ? () => onChanged!(!checked) : null,
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              track,
              if (label case final label?) ...[
                const SizedBox(width: 11),
                Text(
                  label.toUpperCase(),
                  style: Theme.of(context).textTheme.labelLarge
                      ?.copyWith(color: colors.foreground, letterSpacing: 0.5),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

/// Displays a labelled parchment selector.
class BalaurSelect extends StatelessWidget {
  const BalaurSelect({
    super.key,
    required this.options,
    required this.value,
    required this.onChanged,
    this.label,
    this.enabled = true,
  });

  final List<String> options;
  final String value;
  final ValueChanged<String?>? onChanged;
  final String? label;
  final bool enabled;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        if (label case final label?) ...[
          Text(
            label.toUpperCase(),
            style: Theme.of(context).textTheme.labelSmall?.copyWith(
              color: colors.muted,
              fontWeight: FontWeight.w700,
              letterSpacing: 0.8,
            ),
          ),
          const SizedBox(height: 6),
        ],
        DropdownButtonFormField<String>(
          initialValue: value,
          items: [
            for (final option in options)
              DropdownMenuItem(
                value: option,
                child: Text(option.toUpperCase()),
              ),
          ],
          onChanged: enabled ? onChanged : null,
          icon: Text('▾', style: TextStyle(color: colors.goldInk)),
          style: Theme.of(context).textTheme.labelLarge
              ?.copyWith(color: colors.ink),
          decoration: const InputDecoration(
            contentPadding: EdgeInsets.symmetric(horizontal: 12, vertical: 11),
          ),
        ),
      ],
    );
  }
}
