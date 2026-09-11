import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/data/crafts_catalog.dart';
import 'package:craftsky_app/shared/widgets/craft_icon.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// Multi-select chip grid backed by the [Craft] catalog. Tapping a chip
/// toggles its presence in [selected] via [onToggle]. Renders every
/// currently selectable catalog entry in canonical order.
class EditProfileCraftsPicker extends StatelessWidget {
  const EditProfileCraftsPicker({
    required this.selected,
    required this.onToggle,
    required this.onRequestMore,
    super.key,
  });

  final Set<Craft> selected;
  final ValueChanged<Craft> onToggle;
  final VoidCallback? onRequestMore;

  @override
  Widget build(BuildContext context) {
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    final l10n = AppLocalizations.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Wrap(
          spacing: spacing.sp2,
          runSpacing: spacing.sp2,
          children: [
            for (final craft in canonicalSelectableCrafts)
              _CraftChoiceChip(
                craft: craft,
                isSelected: selected.contains(craft),
                onTap: () => onToggle(craft),
              ),
          ],
        ),
        SizedBox(height: spacing.sp2),
        TextButton(
          onPressed: onRequestMore,
          child: Text(l10n.craftsRequestMoreAction),
        ),
      ],
    );
  }
}

class _CraftChoiceChip extends StatelessWidget {
  const _CraftChoiceChip({
    required this.craft,
    required this.isSelected,
    required this.onTap,
  });

  final Craft craft;
  final bool isSelected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    final l10n = AppLocalizations.of(context);

    return FilterChip(
      selected: isSelected,
      onSelected: (_) => onTap(),
      label: CraftIconLabel(
        craft: craft.id,
        label: craftLabel(craft, l10n),
        gap: spacing.sp1,
        flexibleLabel: true,
      ),
    );
  }
}
