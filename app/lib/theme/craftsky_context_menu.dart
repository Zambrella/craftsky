import 'dart:async';
import 'dart:math' as math;

import 'package:craftsky_app/theme/craftsky_divider.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

const _popupScreenPadding = 8.0;
const _popupAnchorGap = 4.0;
const _popupVerticalPadding = 16.0;
const _popupItemHeight = 56.0;
const _popupDescribedItemHeight = 72.0;
const _popupDividerHeight = 1.0;

/// Visual treatment for a [CraftskyContextMenuItem].
enum CraftskyContextMenuItemStyle { normal, destructive }

/// Configuration for a single row in a CraftSky context menu.
class CraftskyContextMenuItem {
  const CraftskyContextMenuItem({
    required this.text,
    required this.icon,
    required this.onPressed,
    this.description,
    this.semanticHint,
    this.isSelected = false,
    this.style = CraftskyContextMenuItemStyle.normal,
  });

  final String text;
  final IconData icon;
  final FutureOr<void> Function()? onPressed;
  final String? description;
  final String? semanticHint;
  final bool isSelected;
  final CraftskyContextMenuItemStyle style;
}

/// Logical grouping for context menu rows.
class CraftskyContextMenuGroup {
  const CraftskyContextMenuGroup({required this.items});

  final List<CraftskyContextMenuItem> items;
}

/// Returns an overlay-relative position anchored to [context]'s render box.
RelativeRect craftskyContextMenuAnchorPosition(BuildContext context) {
  final renderObject = context.findRenderObject();
  final overlayObject = Overlay.of(context).context.findRenderObject();
  if (renderObject is! RenderBox || overlayObject is! RenderBox) {
    return RelativeRect.fill;
  }
  final topLeft = renderObject.localToGlobal(
    Offset.zero,
    ancestor: overlayObject,
  );
  final bottomRight = renderObject.localToGlobal(
    renderObject.size.bottomRight(Offset.zero),
    ancestor: overlayObject,
  );
  return RelativeRect.fromRect(
    Rect.fromPoints(topLeft, bottomRight),
    Offset.zero & overlayObject.size,
  );
}

/// Icon button that opens a responsive CraftSky context menu.
class CraftskyContextMenuButton extends StatelessWidget {
  const CraftskyContextMenuButton({
    required this.groups,
    this.icon = Icons.more_horiz,
    this.tooltip,
    this.enabled = true,
    super.key,
  });

  final List<CraftskyContextMenuGroup> groups;
  final IconData icon;
  final String? tooltip;
  final bool enabled;

  @override
  Widget build(BuildContext context) {
    return IconButton(
      icon: Icon(icon, size: 22),
      tooltip: tooltip,
      padding: EdgeInsets.zero,
      onPressed: enabled
          ? () {
              unawaited(
                showCraftskyContextMenu(
                  context,
                  position: craftskyContextMenuAnchorPosition(context),
                  groups: groups,
                ),
              );
            }
          : null,
    );
  }
}

/// Shows a CraftSky context menu as a bottom sheet on compact screens and an
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

    var selectedAction = Future<void>.value();
    await showModalBottomSheet<void>(
      context: context,
      useSafeArea: true,
      useRootNavigator: true,
      backgroundColor: swatches.paper3,
      barrierColor: Colors.black54,
      clipBehavior: Clip.antiAlias,
      shape: RoundedRectangleBorder(
        borderRadius: radius,
        side: theme.brightness == Brightness.dark
            ? BorderSide.none
            : BorderSide(color: theme.colorScheme.onSurface, width: 1.5),
      ),
      builder: (_) => _CraftskyContextMenuSheet(
        groups: groups,
        onSelected: (item) {
          Navigator.of(context, rootNavigator: true).pop();
          selectedAction = Future<void>.microtask(() async {
            await item.onPressed?.call();
          });
        },
      ),
    );
    await selectedAction;
    return;
  }

  final theme = Theme.of(context);
  final radii = theme.extension<RadiusTheme>()!;
  final swatches = theme.extension<BrandSwatchTheme>()!;
  final popupPosition = _positionOutsideAnchor(
    context,
    anchor: position,
    estimatedPopupHeight: _estimateContextMenuHeight(groups),
  );
  final selected = await showMenu<CraftskyContextMenuItem>(
    context: context,
    position: popupPosition,
    // [position] is measured against the nearest overlay. Keeping the popup
    // on that navigator preserves the same coordinate space when the app's
    // large-screen rail sits outside the content navigator.
    color: swatches.paper3,
    surfaceTintColor: Colors.transparent,
    elevation: 0,
    shape: _contextMenuShape(theme, radii),
    items: _popupEntries(groups),
  );
  await selected?.onPressed?.call();
}

/// Shows arbitrary interactive content using the same large-screen surface as
/// CraftSky context menus.
///
/// Unlike a disabled [PopupMenuItem], this entry leaves descendant controls
/// interactive and does not apply Flutter's disabled text/icon opacity.
Future<void> showCraftskyContextPopover(
  BuildContext context, {
  required RelativeRect position,
  required double estimatedHeight,
  required Widget child,
}) async {
  final theme = Theme.of(context);
  final radii = theme.extension<RadiusTheme>()!;
  final swatches = theme.extension<BrandSwatchTheme>()!;
  await showMenu<Object?>(
    context: context,
    position: _positionOutsideAnchor(
      context,
      anchor: position,
      estimatedPopupHeight: estimatedHeight,
    ),
    color: swatches.paper3,
    surfaceTintColor: Colors.transparent,
    elevation: 0,
    menuPadding: EdgeInsets.zero,
    shape: _contextMenuShape(theme, radii),
    items: [_CraftskyContextPopoverEntry(child: child)],
  );
}

RelativeRect _positionOutsideAnchor(
  BuildContext context, {
  required RelativeRect anchor,
  required double estimatedPopupHeight,
}) {
  final overlayObject = Overlay.of(context).context.findRenderObject();
  if (overlayObject is! RenderBox) return anchor;

  final overlayBounds = Offset.zero & overlayObject.size;
  final anchorRect = anchor.toRect(overlayBounds);
  final mediaPadding = MediaQuery.paddingOf(context);
  final topLimit = math.max(
    _popupScreenPadding,
    mediaPadding.top + _popupScreenPadding,
  );
  final bottomLimit =
      overlayObject.size.height -
      math.max(_popupScreenPadding, mediaPadding.bottom + _popupScreenPadding);
  final belowTop = anchorRect.bottom + _popupAnchorGap;
  final aboveTop = anchorRect.top - _popupAnchorGap - estimatedPopupHeight;
  final spaceBelow = bottomLimit - belowTop;
  final spaceAbove = anchorRect.top - _popupAnchorGap - topLimit;
  final opensBelow =
      estimatedPopupHeight <= spaceBelow ||
      (estimatedPopupHeight > spaceAbove && spaceBelow >= spaceAbove);
  final popupTop = opensBelow ? belowTop : math.max(topLimit, aboveTop);

  return RelativeRect.fromLTRB(
    anchor.left,
    popupTop,
    anchor.right,
    overlayObject.size.height - popupTop,
  );
}

double _estimateContextMenuHeight(
  List<CraftskyContextMenuGroup> groups,
) {
  var height = _popupVerticalPadding;
  var nonEmptyGroupCount = 0;

  for (final group in groups) {
    if (group.items.isEmpty) continue;
    if (nonEmptyGroupCount > 0) height += _popupDividerHeight;
    nonEmptyGroupCount++;
    for (final item in group.items) {
      height += item.description == null
          ? _popupItemHeight
          : _popupDescribedItemHeight;
    }
  }

  return height;
}

RoundedRectangleBorder _contextMenuShape(
  ThemeData theme,
  RadiusTheme radii,
) => RoundedRectangleBorder(
  borderRadius: BorderRadius.circular(radii.r3),
  side: BorderSide(color: theme.colorScheme.onSurface, width: 1.5),
);

class _CraftskyContextPopoverEntry extends PopupMenuEntry<Object?> {
  const _CraftskyContextPopoverEntry({required this.child});

  final Widget child;

  @override
  double get height => 0;

  @override
  bool represents(Object? value) => false;

  @override
  State<_CraftskyContextPopoverEntry> createState() =>
      _CraftskyContextPopoverEntryState();
}

class _CraftskyContextPopoverEntryState
    extends State<_CraftskyContextPopoverEntry> {
  @override
  Widget build(BuildContext context) => widget.child;
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
  const _CraftskyContextMenuSheet({
    required this.groups,
    required this.onSelected,
  });

  final List<CraftskyContextMenuGroup> groups;
  final void Function(CraftskyContextMenuItem item) onSelected;

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
            onTap: item.onPressed == null ? null : () => onSelected(item),
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

    return Semantics(
      label: item.text,
      hint: item.semanticHint,
      button: true,
      enabled: !isDisabled,
      excludeSemantics: true,
      child: Material(
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
      ),
    );
  }
}
