import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/shared/widgets/sort_menu_button.dart';
import 'package:flutter/material.dart';

class SavedPostSortButton extends StatelessWidget {
  const SavedPostSortButton({
    required this.value,
    required this.onChanged,
    super.key,
  });

  final SavedPostSort value;
  final ValueChanged<SavedPostSort> onChanged;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return SortMenuButton<SavedPostSort>(
      selectedValue: value,
      onChanged: onChanged,
      options: [
        SortMenuOption(
          value: SavedPostSort.newest,
          label: l10n.searchSortNewest,
          description: l10n.savedPostsSortNewestDescription,
        ),
        SortMenuOption(
          value: SavedPostSort.oldest,
          label: l10n.savedPostsSortOldest,
          description: l10n.savedPostsSortOldestDescription,
        ),
      ],
    );
  }
}
