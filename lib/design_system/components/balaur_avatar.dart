import 'package:balaur/design_system/foundations/balaur_colors.dart';
import 'package:flutter/material.dart';

/// Identifies a portrait subject.
enum BalaurAvatarKind { balaur, soul }

/// Identifies the visible avatar activity.
enum BalaurAvatarState { idle, thinking, working }

/// Displays a borderless local pixel portrait.
class BalaurAvatar extends StatefulWidget {
  const BalaurAvatar({
    super.key,
    required this.image,
    this.kind = BalaurAvatarKind.balaur,
    this.state = BalaurAvatarState.idle,
    this.size = 54,
    this.semanticLabel,
    this.mirrored = false,
  });

  final ImageProvider image;
  final BalaurAvatarKind kind;
  final BalaurAvatarState state;
  final double size;
  final String? semanticLabel;
  final bool mirrored;

  @override
  State<BalaurAvatar> createState() => _BalaurAvatarState();
}

class _BalaurAvatarState extends State<BalaurAvatar>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller = AnimationController(
    vsync: this,
    duration: const Duration(milliseconds: 1600),
  );

  @override
  void didUpdateWidget(covariant BalaurAvatar oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.state != widget.state) {
      _syncAnimation();
    }
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _syncAnimation();
  }

  void _syncAnimation() {
    if (!mounted) {
      return;
    }
    final animate =
        widget.state != BalaurAvatarState.idle &&
        MediaQuery.maybeOf(context)?.disableAnimations != true;
    if (animate) {
      _controller.repeat(reverse: true);
    } else {
      _controller
        ..stop()
        ..value = 0;
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    final image = Image(
      image: widget.image,
      width: widget.size,
      height: widget.size,
      fit: BoxFit.cover,
      filterQuality: FilterQuality.none,
      semanticLabel: widget.semanticLabel,
      excludeFromSemantics: widget.semanticLabel == null,
    );
    return AnimatedBuilder(
      animation: _controller,
      builder: (context, child) {
        final glow = widget.state == BalaurAvatarState.idle
            ? 0.0
            : 10 + 16 * _controller.value;
        return DecoratedBox(
          decoration: BoxDecoration(
            boxShadow: glow == 0
                ? const []
                : [
                    BoxShadow(
                      color: colors.teal.withValues(
                        alpha: 0.3 + 0.3 * _controller.value,
                      ),
                      blurRadius: glow,
                    ),
                  ],
          ),
          child: child,
        );
      },
      child: widget.mirrored
          ? Transform.flip(flipX: true, child: image)
          : image,
    );
  }
}
