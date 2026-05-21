import 'package:craftsky_app/theme/craftsky_divider.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// Visual treatment for a [CraftskyContextMenuItem].
enum CraftskyContextMenuItemStyle { normal, destructive }

/// Configuration for a single row in a Craftsky context menu.
class CraftskyContextMenuItem {
  const CraftskyContextMenuItem({
    required this.text,
    required this.icon,
    required this.onPressed,
    this.description,
    this.isSelected = false,
    this.style = CraftskyContextMenuItemStyle.normal,
  });

  final String text;
  final IconData icon;
  final VoidCallback? onPressed;
  final String? description;
  final bool isSelected;
  final CraftskyContextMenuItemStyle style;
}

/// Logical grouping for context menu rows.
class CraftskyContextMenuGroup {
  const CraftskyContextMenuGroup({required this.items});

  final List<CraftskyContextMenuItem> items;
}

/// Icon button that opens a responsive Craftsky context menu.
class CraftskyContextMenuButton extends StatelessWidget {
  const CraftskyContextMenuButton({
    required this.groups,
    this.icon = Icons.more_horiz,
    this.tooltip,
    super.key,
  });

  final List<CraftskyContextMenuGroup> groups;
  final IconData icon;
  final String? tooltip;

  @override
  Widget build(BuildContext context) {
    return IconButton(
      icon: Icon(icon, size: 22),
      tooltip: tooltip,
      padding: EdgeInsets.zero,
      onPressed: () {
        final button = context.findRenderObject()! as RenderBox;
        final overlay =
            Navigator.of(context).overlay!.context.findRenderObject()!
                as RenderBox;
        final offset = button.localToGlobal(Offset.zero, ancestor: overlay);
        final position = RelativeRect.fromRect(
          offset & button.size,
          Offset.zero & overlay.size,
        );

        showCraftskyContextMenu(
          context,
          position: position,
          groups: groups,
        );
      },
    );
  }
}

/// Shows a Craftsky context menu as a bottom sheet on compact screens and an
/// anchored popup menu on larger screens.
Future<void> showCraftskyContextMenu(
  BuildContext context, {
  required RelativeRect position,
  required List<CraftskyContextMenuGroup> groups,
}) async {
  final width = MediaQuery.sizeOf(context).width;
  final isCompact = width <= 900;

  if (isCompact) {
    final theme = Theme.of(context);
    final radii = theme.extension<RadiusTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final radius = BorderRadius.vertical(top: Radius.circular(radii.r4));

    await showModalBottomSheet<void>(
      context: context,
      useSafeArea: true,
      useRootNavigator: true,
      backgroundColor: swatches.paper3,
      barrierColor: Colors.black54,
      clipBehavior: Clip.antiAlias,
      shape: RoundedRectangleBorder(
        borderRadius: radius,
        side: BorderSide(color: theme.colorScheme.onSurface, width: 1.5),
      ),
      builder: (_) => _CraftskyContextMenuSheet(groups: groups),
    );
    return;
  }

  final theme = Theme.of(context);
  final radii = theme.extension<RadiusTheme>()!;
  final swatches = theme.extension<BrandSwatchTheme>()!;
  final selected = await showMenu<CraftskyContextMenuItem>(
    context: context,
    position: position,
    useRootNavigator: true,
    color: swatches.paper3,
    elevation: 0,
    shape: RoundedRectangleBorder(
      borderRadius: BorderRadius.circular(radii.r3),
      side: BorderSide(color: theme.colorScheme.onSurface, width: 1.5),
    ),
    items: _popupEntries(groups),
  );
  selected?.onPressed?.call();
}

List<PopupMenuEntry<CraftskyContextMenuItem>> _popupEntries(
  List<CraftskyContextMenuGroup> groups,
) {
  final entries = <PopupMenuEntry<CraftskyContextMenuItem>>[];

  for (final group in groups) {
    if (group.items.isEmpty) continue;
    if (entries.isNotEmpty) {
      entries.add(const PopupMenuDivider(height: 1));
    }
    for (final item in group.items) {
      entries.add(
        PopupMenuItem<CraftskyContextMenuItem>(
          value: item,
          enabled: item.onPressed != null,
          padding: EdgeInsets.zero,
          child: _CraftskyContextMenuRow(item: item),
        ),
      );
    }
  }

  return entries;
}

class _CraftskyContextMenuSheet extends StatelessWidget {
  const _CraftskyContextMenuSheet({required this.groups});

  final List<CraftskyContextMenuGroup> groups;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final radii = theme.extension<RadiusTheme>()!;
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        SizedBox(height: radii.r3),
        ..._sheetChildren(context),
        SizedBox(height: MediaQuery.of(context).padding.bottom),
      ],
    );
  }

  List<Widget> _sheetChildren(BuildContext context) {
    final children = <Widget>[];

    for (final group in groups) {
      if (group.items.isEmpty) continue;
      if (children.isNotEmpty) {
        children.add(
          CraftskyDivider(color: Theme.of(context).colorScheme.onSurface),
        );
      }
      for (final item in group.items) {
        children.add(
          _CraftskyContextMenuRow(
            item: item,
            onTap: item.onPressed == null
                ? null
                : () {
                    Navigator.of(context).pop();
                    item.onPressed?.call();
                  },
          ),
        );
      }
    }

    return children;
  }
}

class _CraftskyContextMenuRow extends StatelessWidget {
  const _CraftskyContextMenuRow({required this.item, this.onTap});

  final CraftskyContextMenuItem item;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final semanticColors = theme.extension<SemanticColorsTheme>()!;
    final isDisabled = item.onPressed == null;
    final foreground = switch (item.style) {
      CraftskyContextMenuItemStyle.normal => theme.colorScheme.onSurface,
      CraftskyContextMenuItemStyle.destructive => semanticColors.error,
    };
    final color = isDisabled ? theme.colorScheme.outline : foreground;
    final selectedBackground = theme.colorScheme.primaryContainer.withValues(
      alpha: 0.4,
    );

    return Material(
      color: item.isSelected ? selectedBackground : Colors.transparent,
      child: ListTile(
        enabled: !isDisabled,
        onTap: onTap,
        contentPadding: EdgeInsets.symmetric(horizontal: spacing.sp4),
        horizontalTitleGap: spacing.sp3,
        leading: Icon(
          item.isSelected ? Icons.check_box : item.icon,
          color: color,
        ),
        title: Text(
          item.text,
          style: theme.textTheme.labelLarge?.copyWith(color: color),
        ),
        subtitle: item.description == null
            ? null
            : Text(
                item.description!,
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.outline,
                ),
              ),
      ),
    );
  }
}
