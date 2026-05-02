import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/data/crafts_catalog.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// Multi-select chip grid backed by the [Craft] catalog. Tapping a chip
/// toggles its presence in [selected] via [onToggle]. Renders every
/// catalog entry — order is the catalog's enum order, not the user's
/// selection order, so the grid layout is stable as the user toggles.
class EditProfileCraftsPicker extends StatelessWidget {
  const EditProfileCraftsPicker({
    required this.selected,
    required this.onToggle,
    super.key,
  });

  final Set<Craft> selected;
  final ValueChanged<Craft> onToggle;

  @override
  Widget build(BuildContext context) {
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    return Wrap(
      spacing: spacing.sp2,
      runSpacing: spacing.sp2,
      children: [
        for (final craft in Craft.values)
          _CraftChoiceChip(
            craft: craft,
            isSelected: selected.contains(craft),
            onTap: () => onToggle(craft),
          ),
      ],
    );
  }
}

/// Selectable variant of the profile-page craft pill. Selected state
/// fills the chip with the brand primary; unselected stays paper-on-
/// paper. Both share the chunky 1.5px ink border so the row reads as a
/// cohesive group.
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
    final theme = Theme.of(context);
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final spacing = theme.extension<SpacingTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    final l10n = AppLocalizations.of(context);

    final background = isSelected ? theme.colorScheme.primary : swatches.paper3;
    final foreground = isSelected
        ? theme.colorScheme.onPrimary
        : theme.colorScheme.onSurface;

    return Semantics(
      button: true,
      selected: isSelected,
      label: craftLabel(craft, l10n),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(radii.rPill),
        child: Container(
          padding: EdgeInsets.symmetric(
            horizontal: spacing.sp3,
            vertical: 6,
          ),
          decoration: BoxDecoration(
            color: background,
            borderRadius: BorderRadius.circular(radii.rPill),
            border: Border.all(
              color: theme.colorScheme.onSurface,
              width: 1.5,
            ),
          ),
          child: Text(
            craftLabel(craft, l10n),
            style: theme.textTheme.labelMedium?.copyWith(color: foreground),
          ),
        ),
      ),
    );
  }
}
