import 'dart:async';

import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

class SortMenuOption<T> {
  const SortMenuOption({
    required this.value,
    required this.label,
    required this.description,
  });

  final T value;
  final String label;
  final String description;
}

class SortMenuButton<T> extends StatelessWidget {
  const SortMenuButton({
    required this.selectedValue,
    required this.options,
    required this.onChanged,
    super.key,
  });

  final T selectedValue;
  final List<SortMenuOption<T>> options;
  final ValueChanged<T> onChanged;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>() ?? const SpacingTheme();
    final selected = options.firstWhere(
      (option) => option.value == selectedValue,
      orElse: () => options.first,
    );

    return OutlinedButton.icon(
      onPressed: () => _showMenu(context),
      icon: const Icon(Icons.filter_list, size: 18),
      label: Text(selected.label),
      style: OutlinedButton.styleFrom(
        foregroundColor: theme.colorScheme.onSurface,
        side: BorderSide(
          color: theme.colorScheme.outlineVariant,
          width: 1.5,
        ),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(spacing.sp2),
        ),
        padding: EdgeInsets.symmetric(
          horizontal: spacing.sp3,
          vertical: spacing.sp2,
        ),
      ),
    );
  }

  void _showMenu(BuildContext context) {
    final rect = craftskyContextMenuAnchorPosition(context);

    unawaited(
      showCraftskyContextMenu(
        context,
        position: rect,
        groups: [
          CraftskyContextMenuGroup(
            items: [
              for (final option in options)
                CraftskyContextMenuItem(
                  text: option.label,
                  description: option.description,
                  icon: Icons.check_box_outline_blank,
                  isSelected: selectedValue == option.value,
                  onPressed: selectedValue == option.value
                      ? () {}
                      : () => onChanged(option.value),
                ),
            ],
          ),
        ],
      ),
    );
  }
}
