import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// Banner-overlay status chip ("JACKET WEATHER" in the design mockup).
/// Currently a passive label; user-settable status is future work.
class ProfileBannerChip extends StatelessWidget {
  const ProfileBannerChip({required this.label, super.key});

  final String label;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(
        color: swatches.paper3,
        borderRadius: BorderRadius.circular(radii.r2),
        border: Border.all(color: theme.colorScheme.onSurface, width: 1.5),
      ),
      child: Text(
        label.toUpperCase(),
        style: theme.textTheme.labelSmall?.copyWith(
          color: theme.colorScheme.onSurface,
        ),
      ),
    );
  }
}
