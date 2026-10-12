import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/shared/widgets/sort_menu_button.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:material_ui/material_ui.dart';

class SavedPostSortButton extends StatelessWidget {
  const new({
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
          icon: CraftskyIcons.schedule,
          label: l10n.searchSortNewest,
          description: l10n.savedPostsSortNewestDescription,
        ),
        SortMenuOption(
          value: SavedPostSort.oldest,
          icon: CraftskyIcons.history,
          label: l10n.savedPostsSortOldest,
          description: l10n.savedPostsSortOldestDescription,
        ),
      ],
    );
  }
}
