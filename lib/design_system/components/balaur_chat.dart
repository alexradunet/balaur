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

/// Displays a numbered dialogue decision in the owner's voice.
class BalaurDialogueChoices extends StatelessWidget {
  const BalaurDialogueChoices({
    super.key,
    required this.choices,
    required this.onPick,
    this.kicker = 'Your word',
    this.who = 'You',
    this.avatar,
  });

  final List<BalaurDialogueChoice> choices;
  final ValueChanged<int> onPick;
  final String kicker;
  final String who;
  final ImageProvider? avatar;

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
        child: LayoutBuilder(
          builder: (context, constraints) {
            final showPortrait = constraints.maxWidth >= 420;
            final panel = Expanded(
              child: _DialoguePanel(
                choices: choices,
                onPick: onPick,
                kicker: kicker,
              ),
            );
            return Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                panel,
                if (showPortrait) ...[
                  const SizedBox(width: 14),
                  _OwnerPortrait(
                    image: avatar ?? BalaurAssets.soulAvatar(5),
                    who: who,
                  ),
                ],
              ],
            );
          },
        ),
      ),
    );
  }
}

class _DialoguePanel extends StatelessWidget {
  const _DialoguePanel({
    required this.choices,
    required this.onPick,
    required this.kicker,
  });

  final List<BalaurDialogueChoice> choices;
  final ValueChanged<int> onPick;
  final String kicker;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    return BalaurSurface(
      ornate: true,
      borderColor: colors.goldDeep,
      padding: const EdgeInsets.fromLTRB(14, 12, 14, 13),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Row(
            children: [
              const Expanded(child: BalaurStitch()),
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 10),
                child: Text(
                  kicker.toUpperCase(),
                  style: Theme.of(context).textTheme.labelSmall?.copyWith(
                    color: colors.goldInk,
                    fontWeight: FontWeight.w700,
                    letterSpacing: 0.8,
                  ),
                ),
              ),
              const Expanded(child: BalaurStitch()),
            ],
          ),
          const SizedBox(height: 4),
          for (var index = 0; index < choices.length; index++)
            _ChoiceButton(
              index: index,
              choice: choices[index],
              onPressed: () => onPick(index),
            ),
        ],
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

class _OwnerPortrait extends StatelessWidget {
  const _OwnerPortrait({required this.image, required this.who});

  final ImageProvider image;
  final String who;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        BalaurSurface(
          material: BalaurSurfaceMaterial.wood,
          padding: const EdgeInsets.all(5),
          child: ColoredBox(
            color: const Color(0xff101314),
            child: BalaurAvatar(
              image: image,
              kind: BalaurAvatarKind.soul,
              size: 54,
              mirrored: true,
            ),
          ),
        ),
        Transform.translate(
          offset: const Offset(0, -2),
          child: BalaurSurface(
            material: BalaurSurfaceMaterial.wood,
            hardShadow: false,
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
            child: Text(
              who.toUpperCase(),
              style: Theme.of(context).textTheme.labelSmall
                  ?.copyWith(color: colors.indigo, fontWeight: FontWeight.w700),
            ),
          ),
        ),
      ],
    );
  }
}

/// Displays the owner's unsent message in the dialogue column.
class BalaurMessageDraft extends StatefulWidget {
  const BalaurMessageDraft({
    super.key,
    required this.onSend,
    this.controller,
    this.who = 'You',
    this.avatar,
    this.placeholder = 'Speak; I am listening.',
    this.sendLabel = 'Send',
  });

  final ValueChanged<String> onSend;
  final TextEditingController? controller;
  final String who;
  final ImageProvider? avatar;
  final String placeholder;
  final String sendLabel;

  @override
  State<BalaurMessageDraft> createState() => _BalaurMessageDraftState();
}

class _BalaurMessageDraftState extends State<BalaurMessageDraft> {
  late final TextEditingController _localController = TextEditingController();
  bool _focused = false;

  TextEditingController get _controller =>
      widget.controller ?? _localController;

  @override
  void dispose() {
    _localController.dispose();
    super.dispose();
  }

  void _send() {
    final text = _controller.text.trim();
    if (text.isEmpty) {
      return;
    }
    widget.onSend(text);
    _controller.clear();
  }

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    return LayoutBuilder(
      builder: (context, constraints) {
        final showPortrait = constraints.maxWidth >= 420;
        return Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Expanded(
              child: Focus(
                onFocusChange: (focused) => setState(() => _focused = focused),
                child: BalaurSurface(
                  dashed: !_focused,
                  hardShadow: _focused,
                  borderColor: _focused
                      ? colors.goldDeep
                      : colors.parchmentEdge,
                  padding: const EdgeInsets.fromLTRB(14, 12, 14, 10),
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      TextField(
                        controller: _controller,
                        minLines: 2,
                        maxLines: 6,
                        textInputAction: TextInputAction.newline,
                        style: Theme.of(context).textTheme.bodyMedium
                            ?.copyWith(color: colors.ink),
                        decoration: InputDecoration.collapsed(
                          hintText: widget.placeholder,
                        ),
                      ),
                      const SizedBox(height: 8),
                      Divider(color: colors.parchmentEdge, height: 2),
                      const SizedBox(height: 8),
                      Row(
                        children: [
                          Expanded(
                            child: Text(
                              'UNSENT · ENTER SPEAKS',
                              style: Theme.of(context).textTheme.labelSmall
                                  ?.copyWith(color: colors.inkMuted),
                            ),
                          ),
                          BalaurButton(
                            label: widget.sendLabel,
                            onPressed: _send,
                            size: BalaurButtonSize.small,
                          ),
                        ],
                      ),
                    ],
                  ),
                ),
              ),
            ),
            if (showPortrait) ...[
              const SizedBox(width: 14),
              _OwnerPortrait(
                image: widget.avatar ?? BalaurAssets.soulAvatar(5),
                who: widget.who,
              ),
            ],
          ],
        );
      },
    );
  }
}

/// Displays a page-level wood input ledge.
class BalaurChatBar extends StatelessWidget {
  const BalaurChatBar({
    super.key,
    required this.onSend,
    this.controller,
    this.placeholder = 'Speak; I am listening.',
    this.sendLabel = 'Send',
    this.above,
  });

  final ValueChanged<String> onSend;
  final TextEditingController? controller;
  final String placeholder;
  final String sendLabel;
  final Widget? above;

  @override
  Widget build(BuildContext context) {
    return BalaurSurface(
      material: BalaurSurfaceMaterial.wood,
      padding: const EdgeInsets.fromLTRB(14, 14, 14, 16),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (above case final above?) ...[above, const SizedBox(height: 10)],
          Row(
            crossAxisAlignment: CrossAxisAlignment.end,
            children: [
              Expanded(
                child: TextField(
                  controller: controller,
                  minLines: 2,
                  maxLines: 6,
                  decoration: InputDecoration(hintText: placeholder),
                  onSubmitted: (text) {
                    if (text.trim().isNotEmpty) {
                      onSend(text.trim());
                    }
                  },
                ),
              ),
              const SizedBox(width: 10),
              BalaurButton(
                label: sendLabel,
                onPressed: () {
                  final text = controller?.text.trim() ?? '';
                  if (text.isNotEmpty) {
                    onSend(text);
                    controller?.clear();
                  }
                },
              ),
            ],
          ),
        ],
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
class BalaurComposer extends StatelessWidget {
  const BalaurComposer({
    super.key,
    required this.onSend,
    this.controller,
    this.who = 'You',
    this.avatar,
    this.placeholder = 'Speak; I am listening.',
    this.sendLabel = 'Send',
    this.tools = const [],
    this.promptLabel,
    this.decision,
  });

  final ValueChanged<String> onSend;
  final TextEditingController? controller;
  final String who;
  final ImageProvider? avatar;
  final String placeholder;
  final String sendLabel;
  final List<BalaurComposerTool> tools;
  final String? promptLabel;
  final Widget? decision;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    final deciding = decision != null;
    return BalaurSurface(
      material: BalaurSurfaceMaterial.wood,
      ornate: true,
      borderColor: deciding ? colors.goldDeep : colors.outline,
      padding: const EdgeInsets.fromLTRB(14, 12, 14, 14),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Row(
            children: [
              for (final tool in tools)
                Padding(
                  padding: const EdgeInsets.only(right: 8),
                  child: Tooltip(
                    message: tool.tooltip,
                    child: _ComposerToolButton(tool: tool, enabled: !deciding),
                  ),
                ),
              const Spacer(),
              if (promptLabel case final label?)
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
                image: avatar ?? BalaurAssets.soulAvatar(5),
                kind: BalaurAvatarKind.soul,
                size: 42,
                mirrored: true,
              ),
            ],
          ),
          const SizedBox(height: 10),
          decision ??
              BalaurMessageDraft(
                controller: controller,
                who: who,
                avatar: avatar,
                placeholder: placeholder,
                sendLabel: sendLabel,
                onSend: onSend,
              ),
        ],
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
