import 'dart:async';

import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/data/crafts_catalog.dart';
import 'package:craftsky_app/shared/widgets/craft_icon.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/gestures.dart';
import 'package:material_ui/material_ui.dart';

/// Selected crafts retain insertion order; the first is the default.
/// Unselected crafts remain available below the reorderable selection.
class EditProfileCraftsPicker extends StatelessWidget {
  const new({
    required this.selected,
    required this.onToggle,
    required this.onReorder,
    required this.onRequestMore,
    this.enabled = true,
    super.key,
  });

  final Set<Craft> selected;
  final ValueChanged<Craft> onToggle;
  final ValueChanged<List<Craft>> onReorder;
  final VoidCallback? onRequestMore;
  final bool enabled;

  void _move(List<Craft> crafts, int from, int to) {
    final next = List.of(crafts);
    next.insert(to, next.removeAt(from));
    onReorder(next);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final crafts = selected.where(canonicalSelectableCrafts.contains).toList();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        ReorderableListView(
          shrinkWrap: true,
          physics: const NeverScrollableScrollPhysics(),
          buildDefaultDragHandles: false,
          proxyDecorator: (child, index, animation) =>
              CraftskyCard(child: child),
          onReorderItem: (from, to) {
            if (enabled) _move(crafts, from, to);
          },
          children: [
            for (var index = 0; index < crafts.length; index++)
              _row(context, crafts[index], index: index, crafts: crafts),
          ],
        ),
        for (final craft in canonicalSelectableCrafts)
          if (!selected.contains(craft)) _row(context, craft),
        TextButton(
          onPressed: onRequestMore,
          child: Text(l10n.craftsRequestMoreAction),
        ),
      ],
    );
  }

  Widget _row(
    BuildContext context,
    Craft craft, {
    int? index,
    List<Craft> crafts = const [],
  }) {
    final l10n = AppLocalizations.of(context);
    final label = craftLabel(craft, l10n);
    final isSelected = index != null;
    final canReorder = enabled && crafts.length > 1;
    return ListTile(
      key: ValueKey('craft-choice-${craft.id}'),
      contentPadding: EdgeInsets.zero,
      onTap: enabled ? () => onToggle(craft) : null,
      leading: Checkbox(
        value: isSelected,
        semanticLabel: label,
        onChanged: enabled ? (_) => onToggle(craft) : null,
      ),
      title: CraftIconLabel(craft: craft.id, label: label, flexibleLabel: true),
      subtitle: index == 0
          ? Row(
              children: [
                const Icon(Icons.star, size: 16),
                const SizedBox(width: 4),
                Flexible(child: Text(l10n.craftsDefaultLabel)),
              ],
            )
          : null,
      trailing: isSelected && crafts.length > 1
          ? GestureDetector(
              // Consume taps on controls, including disabled ones, so they do
              // not toggle the enclosing selection row.
              behavior: HitTestBehavior.opaque,
              excludeFromSemantics: true,
              onTap: () {},
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  CraftskyContextMenuButton(
                    enabled: canReorder,
                    tooltip: l10n.craftsReorderLabel(label),
                    groups: [
                      CraftskyContextMenuGroup(
                        items: [
                          if (index > 0)
                            CraftskyContextMenuItem(
                              text: l10n.craftsMoveUp,
                              icon: CraftskyIcons.moveUp,
                              onPressed: () => _move(crafts, index, index - 1),
                            ),
                          if (index < crafts.length - 1)
                            CraftskyContextMenuItem(
                              text: l10n.craftsMoveDown,
                              icon: CraftskyIcons.moveDown,
                              onPressed: () => _move(crafts, index, index + 1),
                            ),
                        ],
                      ),
                    ],
                  ),
                  _CraftDragStartListener(
                    index: index,
                    enabled: canReorder,
                    child: Semantics(
                      label: l10n.craftsDragLabel(label),
                      enabled: canReorder,
                      child: SizedBox(
                        width: 48,
                        height: 48,
                        child: Icon(
                          Icons.drag_handle,
                          color: canReorder
                              ? null
                              : Theme.of(context).disabledColor,
                        ),
                      ),
                    ),
                  ),
                ],
              ),
            )
          : null,
    );
  }
}

/// Accept movement immediately, or pick up after a stationary long press.
class _CraftDragStartListener extends ReorderableDragStartListener {
  const new({
    required super.child,
    required super.index,
    super.enabled,
  });

  @override
  MultiDragGestureRecognizer createRecognizer() =>
      _CraftDragGestureRecognizer(debugOwner: this);
}

class _CraftDragGestureRecognizer extends ImmediateMultiDragGestureRecognizer {
  new({super.debugOwner});

  @override
  MultiDragPointerState createNewPointerState(PointerDownEvent event) =>
      _CraftDragPointerState(event.position, event.kind, gestureSettings);
}

class _CraftDragPointerState extends MultiDragPointerState {
  new(
    super.initialPosition,
    super.kind,
    super.gestureSettings,
  ) {
    _timer = Timer(kLongPressTimeout, _pickUp);
  }

  Timer? _timer;
  GestureMultiDragStartCallback? _starter;
  bool _ready = false;

  void _pickUp() {
    _timer?.cancel();
    _timer = null;
    _ready = true;
    if (_starter case final starter?) {
      _starter = null;
      starter(initialPosition);
    } else {
      resolve(GestureDisposition.accepted);
    }
  }

  @override
  void checkForResolutionAfterMove() {
    if (pendingDelta!.distance > computeHitSlop(kind, gestureSettings)) {
      _pickUp();
    }
  }

  @override
  void accepted(GestureMultiDragStartCallback starter) {
    if (_ready) {
      starter(initialPosition);
    } else {
      _starter = starter;
    }
  }

  @override
  void dispose() {
    _timer?.cancel();
    _starter = null;
    super.dispose();
  }
}
