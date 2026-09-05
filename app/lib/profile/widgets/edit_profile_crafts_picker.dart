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
      excludeSemantics: true,
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
          child: CraftIconLabel(
            craft: craft.id,
            label: craftLabel(craft, l10n),
            gap: spacing.sp1,
            flexibleLabel: true,
            style: theme.textTheme.labelMedium?.copyWith(color: foreground),
          ),
        ),
      ),
    );
  }
}
