import 'package:balaur/design_system/components/balaur_surface.dart';
import 'package:balaur/design_system/foundations/balaur_assets.dart';
import 'package:balaur/design_system/foundations/balaur_colors.dart';
import 'package:flutter/material.dart';

/// Displays a row of parchment navigation tabs.
class BalaurTabs extends StatelessWidget {
  const BalaurTabs({
    super.key,
    required this.items,
    required this.active,
    required this.onSelect,
  });

  final List<String> items;
  final String active;
  final ValueChanged<String> onSelect;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    return Wrap(
      spacing: 6,
      runSpacing: 6,
      children: [
        for (final item in items)
          Semantics(
            selected: item == active,
            button: true,
            child: Material(
              color: item == active ? colors.gold : colors.surface,
              shape: BeveledRectangleBorder(
                side: BorderSide(
                  color: item == active
                      ? colors.goldDeep
                      : colors.parchmentEdge,
                  width: 2,
                ),
              ),
              child: InkWell(
                onTap: () => onSelect(item),
                child: ConstrainedBox(
                  constraints: const BoxConstraints(minHeight: 48),
                  child: Padding(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 12,
                      vertical: 8,
                    ),
                    child: Text(
                      item.toUpperCase(),
                      style: Theme.of(context).textTheme.labelMedium?.copyWith(
                        color: item == active
                            ? const Color(0xff1c0d04)
                            : colors.inkMuted,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                  ),
                ),
              ),
            ),
          ),
      ],
    );
  }
}

/// Displays the wood application nameplate and primary navigation.
class BalaurTopbar extends StatelessWidget implements PreferredSizeWidget {
  const BalaurTopbar({
    super.key,
    this.brand = 'Balaur',
    this.links = const [],
    this.active,
    this.onNavigate,
    this.onToggleTheme,
    this.themeGlyph = '◑',
  });

  final String brand;
  final List<String> links;
  final String? active;
  final ValueChanged<String>? onNavigate;
  final VoidCallback? onToggleTheme;
  final String themeGlyph;

  @override
  Size get preferredSize => const Size.fromHeight(62);

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    return BalaurSurface(
      material: BalaurSurfaceMaterial.wood,
      hardShadow: false,
      padding: const EdgeInsets.symmetric(horizontal: 18),
      child: SizedBox(
        height: preferredSize.height,
        child: LayoutBuilder(
          builder: (context, constraints) {
            final compact = constraints.maxWidth < 700;
            return Row(
              children: [
                InkWell(
                  onTap: onNavigate == null ? null : () => onNavigate!('home'),
                  child: SizedBox(
                    height: 48,
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        const Image(
                          image: BalaurAssets.crest,
                          width: 34,
                          height: 34,
                        ),
                        const SizedBox(width: 9),
                        Text(
                          brand.toUpperCase(),
                          style: const TextStyle(
                            fontFamily: 'Silkscreen',
                            package: 'balaur',
                            fontSize: 14,
                            fontWeight: FontWeight.w700,
                            letterSpacing: 1.1,
                          ).copyWith(color: colors.gold),
                        ),
                      ],
                    ),
                  ),
                ),
                const Spacer(),
                if (compact)
                  PopupMenuButton<String>(
                    tooltip: 'Navigation',
                    iconColor: colors.chromeForeground,
                    color: colors.chrome,
                    onSelected: onNavigate,
                    itemBuilder: (context) => [
                      for (final link in links)
                        PopupMenuItem(
                          value: link,
                          child: Text(
                            link.toUpperCase(),
                            style: Theme.of(context).textTheme.labelMedium
                                ?.copyWith(
                                  color: link == active
                                      ? colors.gold
                                      : colors.chromeForeground,
                                ),
                          ),
                        ),
                    ],
                  )
                else
                  for (final link in links)
                    TextButton(
                      onPressed: onNavigate == null
                          ? null
                          : () => onNavigate!(link),
                      child: Text(
                        link.toUpperCase(),
                        style: Theme.of(context).textTheme.labelMedium
                            ?.copyWith(
                              color: link == active
                                  ? colors.gold
                                  : colors.chromeForeground,
                            ),
                      ),
                    ),
                const SizedBox(width: 8),
                Semantics(
                  button: true,
                  label: 'Toggle light and dark mode',
                  child: IconButton.outlined(
                    onPressed: onToggleTheme,
                    icon: Text(
                      themeGlyph,
                      style: TextStyle(color: colors.chromeForeground),
                    ),
                    style: IconButton.styleFrom(
                      shape: BeveledRectangleBorder(
                        side: BorderSide(color: colors.chromeForeground),
                      ),
                    ),
                  ),
                ),
              ],
            );
          },
        ),
      ),
    );
  }
}

/// Contains one breadcrumb step.
@immutable
final class BalaurBreadcrumbItem {
  const BalaurBreadcrumbItem({required this.label, this.onPressed});

  final String label;
  final VoidCallback? onPressed;
}

/// Displays a wood path trail.
class BalaurBreadcrumb extends StatelessWidget {
  const BalaurBreadcrumb({super.key, required this.items});

  final List<BalaurBreadcrumbItem> items;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    return BalaurSurface(
      material: BalaurSurfaceMaterial.wood,
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 7),
      child: Wrap(
        spacing: 9,
        runSpacing: 5,
        crossAxisAlignment: WrapCrossAlignment.center,
        children: [
          for (var index = 0; index < items.length; index++) ...[
            if (index > 0) Text('›', style: TextStyle(color: colors.smoke)),
            InkWell(
              onTap: items[index].onPressed,
              child: ConstrainedBox(
                constraints: const BoxConstraints(minHeight: 48),
                child: Center(
                  child: Text(
                    items[index].label.toUpperCase(),
                    style: Theme.of(context).textTheme.labelMedium?.copyWith(
                      color: index == items.length - 1
                          ? colors.gold
                          : colors.chromeForeground,
                    ),
                  ),
                ),
              ),
            ),
          ],
        ],
      ),
    );
  }
}

/// Displays previous, current, and next page controls.
class BalaurPagination extends StatelessWidget {
  const BalaurPagination({
    super.key,
    required this.page,
    required this.total,
    required this.onChange,
  });

  final int page;
  final int total;
  final ValueChanged<int> onChange;

  @override
  Widget build(BuildContext context) {
    final start = total <= 3 ? 1 : (page - 1).clamp(1, total - 2);
    final end = (start + 2).clamp(1, total);
    return Wrap(
      spacing: 6,
      crossAxisAlignment: WrapCrossAlignment.center,
      children: [
        _PageButton(
          label: '‹',
          active: false,
          onPressed: page > 1 ? () => onChange(page - 1) : null,
        ),
        if (start > 1) const Text('…'),
        for (var value = start; value <= end; value++)
          _PageButton(
            label: '$value',
            active: value == page,
            onPressed: value == page ? null : () => onChange(value),
          ),
        if (end < total) const Text('…'),
        _PageButton(
          label: '›',
          active: false,
          onPressed: page < total ? () => onChange(page + 1) : null,
        ),
      ],
    );
  }
}

class _PageButton extends StatelessWidget {
  const _PageButton({
    required this.label,
    required this.active,
    required this.onPressed,
  });

  final String label;
  final bool active;
  final VoidCallback? onPressed;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    return Opacity(
      opacity: onPressed == null && !active ? 0.4 : 1,
      child: SizedBox.square(
        dimension: 48,
        child: Material(
          color: active ? colors.chrome : colors.chromeDeep,
          shape: BeveledRectangleBorder(
            side: BorderSide(
              color: active ? colors.goldDeep : colors.outline,
              width: 2,
            ),
          ),
          child: InkWell(
            onTap: onPressed,
            child: Center(
              child: Text(
                label,
                style: Theme.of(context).textTheme.labelLarge?.copyWith(
                  color: active ? colors.gold : colors.chromeForeground,
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
