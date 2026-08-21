import 'package:balaur/design_system/components/balaur_avatar.dart';
import 'package:balaur/design_system/components/balaur_core.dart';
import 'package:balaur/design_system/components/balaur_surface.dart';
import 'package:balaur/design_system/foundations/balaur_assets.dart';
import 'package:balaur/design_system/foundations/balaur_colors.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

/// Displays one visible tool or operating-system access record.
class BalaurToolRow extends StatelessWidget {
  const BalaurToolRow({
    super.key,
    required this.tool,
    required this.message,
    this.iconName,
    this.glyph = '⌥',
  });

  final String tool;
  final String message;
  final String? iconName;
  final String glyph;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    return BalaurSurface(
      material: BalaurSurfaceMaterial.inset,
      hardShadow: false,
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 9),
      child: Text.rich(
        TextSpan(
          children: [
            if (iconName == null)
              TextSpan(text: '$glyph  ')
            else
              WidgetSpan(
                alignment: PlaceholderAlignment.middle,
                child: Padding(
                  padding: const EdgeInsets.only(right: 8),
                  child: Image(
                    image: BalaurAssets.icon(iconName!),
                    width: 18,
                    height: 18,
                  ),
                ),
              ),
            TextSpan(
              text: 'tool · $tool',
              style: TextStyle(color: colors.teal),
            ),
            TextSpan(
              text: ' — $message',
              style: TextStyle(color: colors.chromeForeground),
            ),
          ],
        ),
        style: Theme.of(context).textTheme.labelLarge?.copyWith(fontSize: 13),
      ),
    );
  }
}

/// Contains one spoken owner choice.
@immutable
final class BalaurDialogueChoice {
  const BalaurDialogueChoice({required this.label, this.hint});

  final String label;
  final String? hint;
}

class _ComposerChoices extends StatelessWidget {
  const _ComposerChoices({required this.choices, required this.onPick});

  final List<BalaurDialogueChoice> choices;
  final ValueChanged<int> onPick;

  @override
  Widget build(BuildContext context) {
    final bindings = <ShortcutActivator, VoidCallback>{};
    final digits = [
      LogicalKeyboardKey.digit1,
      LogicalKeyboardKey.digit2,
      LogicalKeyboardKey.digit3,
      LogicalKeyboardKey.digit4,
      LogicalKeyboardKey.digit5,
      LogicalKeyboardKey.digit6,
      LogicalKeyboardKey.digit7,
      LogicalKeyboardKey.digit8,
      LogicalKeyboardKey.digit9,
    ];
    for (
      var index = 0;
      index < choices.length && index < digits.length;
      index++
    ) {
      final selected = index;
      bindings[SingleActivator(digits[index])] = () => onPick(selected);
    }

    return CallbackShortcuts(
      bindings: bindings,
      child: Focus(
        autofocus: true,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            for (var index = 0; index < choices.length; index++)
              _ChoiceButton(
                index: index,
                choice: choices[index],
                onPressed: () => onPick(index),
              ),
          ],
        ),
      ),
    );
  }
}

class _ChoiceButton extends StatelessWidget {
  const _ChoiceButton({
    required this.index,
    required this.choice,
    required this.onPressed,
  });

  final int index;
  final BalaurDialogueChoice choice;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    return Semantics(
      button: true,
      label:
          '${index + 1}. ${choice.label}${choice.hint == null ? '' : ', ${choice.hint}'}',
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          onTap: onPressed,
          hoverColor: colors.ink.withValues(alpha: 0.09),
          focusColor: colors.ink.withValues(alpha: 0.09),
          child: ConstrainedBox(
            constraints: const BoxConstraints(minHeight: 48),
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 9),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.baseline,
                textBaseline: TextBaseline.alphabetic,
                children: [
                  Text(
                    '${index + 1}',
                    style: Theme.of(context).textTheme.labelMedium
                        ?.copyWith(color: colors.inkMuted),
                  ),
                  const SizedBox(width: 10),
                  Text('▸', style: TextStyle(color: colors.emberDeep)),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      choice.label,
                      style: Theme.of(context).textTheme.bodyMedium
                          ?.copyWith(color: colors.ink, height: 1.45),
                    ),
                  ),
                  if (choice.hint case final hint?) ...[
                    const SizedBox(width: 10),
                    Flexible(
                      child: Text(
                        hint.toUpperCase(),
                        textAlign: TextAlign.right,
                        style: Theme.of(context).textTheme.labelSmall
                            ?.copyWith(color: colors.inkMuted),
                      ),
                    ),
                  ],
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

/// Contains one composer tool.
@immutable
final class BalaurComposerTool {
  const BalaurComposerTool({
    required this.iconName,
    required this.tooltip,
    required this.onPressed,
  });

  final String iconName;
  final String tooltip;
  final VoidCallback onPressed;
}

/// Displays the owner's single seat of action.
class BalaurComposer extends StatefulWidget {
  const BalaurComposer({
    super.key,
    required ValueChanged<String> onSend,
    this.controller,
    this.avatar,
    this.placeholder = 'Speak; I am listening.',
    this.sendLabel = 'Send',
    this.tools = const [],
    this.promptLabel,
    this.enabled = true,
    this.responding = false,
    this.onStop,
  }) : onSend = onSend,
       choices = null,
       onPick = null,
       assert(!responding || onStop != null);

  const BalaurComposer.choices({
    super.key,
    required List<BalaurDialogueChoice> choices,
    required ValueChanged<int> onPick,
    this.avatar,
    this.promptLabel = 'Your word',
  }) : choices = choices,
       onPick = onPick,
       onSend = null,
       controller = null,
       placeholder = '',
       sendLabel = '',
       tools = const [],
       enabled = true,
       responding = false,
       onStop = null;

  final ValueChanged<String>? onSend;
  final TextEditingController? controller;
  final ImageProvider? avatar;
  final String placeholder;
  final String sendLabel;
  final List<BalaurComposerTool> tools;
  final String? promptLabel;
  final bool enabled;
  final bool responding;
  final VoidCallback? onStop;
  final List<BalaurDialogueChoice>? choices;
  final ValueChanged<int>? onPick;

  @override
  State<BalaurComposer> createState() => _BalaurComposerState();
}

class _BalaurComposerState extends State<BalaurComposer> {
  TextEditingController? _localController;
  bool _focused = false;

  TextEditingController get _controller =>
      widget.controller ?? (_localController ??= TextEditingController());

  bool get _acceptsInput => widget.enabled && !widget.responding;

  @override
  void dispose() {
    _localController?.dispose();
    super.dispose();
  }

  void _send() {
    final onSend = widget.onSend;
    if (!_acceptsInput || onSend == null) {
      return;
    }
    final text = _controller.text.trim();
    if (text.isEmpty) {
      return;
    }
    _controller.clear();
    onSend(text);
  }

  void _insertNewline() {
    if (!_acceptsInput) {
      return;
    }
    final value = _controller.value;
    final selection = value.selection;
    final start = selection.isValid ? selection.start : value.text.length;
    final end = selection.isValid ? selection.end : value.text.length;
    final text = value.text.replaceRange(start, end, '\n');
    _controller.value = value.copyWith(
      text: text,
      selection: TextSelection.collapsed(offset: start + 1),
      composing: TextRange.empty,
    );
  }

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    final choices = widget.choices;
    return BalaurSurface(
      material: BalaurSurfaceMaterial.wood,
      ornate: true,
      borderColor: choices == null ? colors.outline : colors.goldDeep,
      padding: const EdgeInsets.fromLTRB(14, 12, 14, 14),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Row(
            children: [
              for (final tool in widget.tools)
                Padding(
                  padding: const EdgeInsets.only(right: 8),
                  child: Tooltip(
                    message: tool.tooltip,
                    child: _ComposerToolButton(
                      tool: tool,
                      enabled: _acceptsInput,
                    ),
                  ),
                ),
              const Spacer(),
              if (widget.promptLabel case final label?)
                Flexible(
                  child: Text(
                    label.toUpperCase(),
                    overflow: TextOverflow.ellipsis,
                    style: Theme.of(context).textTheme.labelSmall?.copyWith(
                      color: colors.gold,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                ),
              const Spacer(),
              BalaurAvatar(
                image: widget.avatar ?? BalaurAssets.soulAvatar(5),
                kind: BalaurAvatarKind.soul,
                size: 42,
                mirrored: true,
              ),
            ],
          ),
          const SizedBox(height: 10),
          if (choices != null)
            BalaurSurface(
              borderColor: colors.goldDeep,
              hardShadow: false,
              padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 4),
              child: _ComposerChoices(choices: choices, onPick: widget.onPick!),
            )
          else
            _buildDraft(context, colors),
        ],
      ),
    );
  }

  Widget _buildDraft(BuildContext context, BalaurColors colors) {
    return Focus(
      onFocusChange: (focused) => setState(() => _focused = focused),
      child: BalaurSurface(
        dashed: !_focused,
        hardShadow: _focused,
        borderColor: _focused ? colors.goldDeep : colors.parchmentEdge,
        padding: const EdgeInsets.fromLTRB(14, 12, 14, 10),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            CallbackShortcuts(
              bindings: {
                const SingleActivator(LogicalKeyboardKey.enter): _send,
                const SingleActivator(LogicalKeyboardKey.enter, shift: true):
                    _insertNewline,
                const SingleActivator(LogicalKeyboardKey.numpadEnter): _send,
                const SingleActivator(
                  LogicalKeyboardKey.numpadEnter,
                  shift: true,
                ): _insertNewline,
              },
              child: TextField(
                key: const Key('chat-composer'),
                controller: _controller,
                enabled: _acceptsInput,
                minLines: 2,
                maxLines: 6,
                textCapitalization: TextCapitalization.sentences,
                textInputAction: TextInputAction.newline,
                style: Theme.of(context).textTheme.bodyMedium
                    ?.copyWith(color: colors.ink),
                decoration: InputDecoration.collapsed(
                  hintText: widget.placeholder,
                ),
              ),
            ),
            const SizedBox(height: 8),
            Divider(color: colors.parchmentEdge, height: 2),
            const SizedBox(height: 8),
            Row(
              children: [
                Expanded(
                  child: Text(
                    'ENTER SENDS · SHIFT+ENTER ADDS A LINE',
                    style: Theme.of(context).textTheme.labelSmall
                        ?.copyWith(color: colors.inkMuted),
                  ),
                ),
                if (widget.responding)
                  BalaurButton(
                    key: const Key('stop-button'),
                    label: 'Stop',
                    onPressed: widget.onStop,
                    size: BalaurButtonSize.small,
                  )
                else
                  BalaurButton(
                    key: const Key('send-button'),
                    label: widget.sendLabel,
                    onPressed: widget.enabled ? _send : null,
                    size: BalaurButtonSize.small,
                  ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _ComposerToolButton extends StatelessWidget {
  const _ComposerToolButton({required this.tool, required this.enabled});

  final BalaurComposerTool tool;
  final bool enabled;

  @override
  Widget build(BuildContext context) {
    return Opacity(
      opacity: enabled ? 1 : 0.38,
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          onTap: enabled ? tool.onPressed : null,
          child: SizedBox.square(
            dimension: 48,
            child: BalaurSurface(
              material: BalaurSurfaceMaterial.inset,
              hardShadow: false,
              padding: const EdgeInsets.all(9),
              child: Image(
                image: BalaurAssets.icon(tool.iconName),
                width: 20,
                height: 20,
              ),
            ),
          ),
        ),
      ),
    );
  }
}
