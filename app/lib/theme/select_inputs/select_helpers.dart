part of '../craftsky_select_inputs.dart';

class _SelectedChips<T> extends StatelessWidget {
  const _SelectedChips({
    required this.name,
    required this.keyPrefix,
    required this.values,
    required this.labelByValue,
    required this.enabled,
    required this.onRemove,
  });

  final String name;
  final String keyPrefix;
  final List<T> values;
  final Map<T, String> labelByValue;
  final bool enabled;
  final ValueChanged<T> onRemove;

  @override
  Widget build(BuildContext context) {
    if (values.isEmpty) return const SizedBox(height: 32);
    return Wrap(
      spacing: 8,
      runSpacing: 4,
      children: [
        for (final value in values)
          InputChip(
            label: Text(labelByValue[value] ?? value.toString()),
            onDeleted: enabled ? () => onRemove(value) : null,
            deleteIcon: Icon(Icons.close, key: Key('$keyPrefix-remove-$value')),
          ),
      ],
    );
  }
}

class _CraftskyOptionsPanel extends StatelessWidget {
  const _CraftskyOptionsPanel({
    required this.child,
    super.key,
    this.scrollable = true,
  });

  final Widget child;
  final bool scrollable;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final radii = theme.extension<RadiusTheme>()!;
    return DecoratedBox(
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(radii.r3),
        border: Border.all(color: theme.colorScheme.outlineVariant),
        boxShadow: const [
          BoxShadow(
            blurRadius: 18,
            offset: Offset(0, 8),
            color: Color(0x26000000),
          ),
        ],
      ),
      child: ClipRRect(
        borderRadius: BorderRadius.circular(radii.r3),
        child: Material(
          type: MaterialType.transparency,
          child: scrollable
              ? SingleChildScrollView(
                  padding: const EdgeInsets.all(8),
                  child: child,
                )
              : child,
        ),
      ),
    );
  }
}

class _AnchoredSelectOverlay extends StatelessWidget {
  const _AnchoredSelectOverlay({
    required this.anchorKey,
    required this.layerLink,
    required this.onDismiss,
    required this.onEscape,
    required this.child,
  });

  final GlobalKey anchorKey;
  final LayerLink layerLink;
  final VoidCallback onDismiss;
  final VoidCallback onEscape;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final anchorBox = anchorKey.currentContext?.findRenderObject();
    if (anchorBox is! RenderBox) {
      return const SizedBox.shrink();
    }

    const gap = 4.0;
    const preferredMaxHeight = 280.0;

    return Stack(
      children: [
        Positioned.fill(
          child: GestureDetector(
            behavior: HitTestBehavior.translucent,
            onTap: onDismiss,
          ),
        ),
        Positioned.fill(
          child: CompositedTransformFollower(
            link: layerLink,
            targetAnchor: Alignment.bottomLeft,
            offset: const Offset(0, gap),
            child: Align(
              alignment: Alignment.topLeft,
              child: SizedBox(
                width: anchorBox.size.width,
                child: Focus(
                  onKeyEvent: (_, event) {
                    if (event is KeyDownEvent &&
                        event.logicalKey == LogicalKeyboardKey.escape) {
                      onEscape();
                      return KeyEventResult.handled;
                    }
                    return KeyEventResult.ignored;
                  },
                  child: ConstrainedBox(
                    constraints: const BoxConstraints(
                      maxHeight: preferredMaxHeight,
                    ),
                    child: child,
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

String _optionKey(String? keyPrefix, String label, Object? value) {
  return '${keyPrefix ?? label}-option-$value';
}

bool _listEquals<T>(List<T> a, List<T> b) {
  if (identical(a, b)) return true;
  if (a.length != b.length) return false;
  for (var index = 0; index < a.length; index++) {
    if (a[index] != b[index]) return false;
  }
  return true;
}
