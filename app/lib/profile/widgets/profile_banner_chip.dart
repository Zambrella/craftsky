import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:flutter/material.dart';

/// Banner-overlay status chip ("JACKET WEATHER" in the design mockup).
/// Currently a passive label; user-settable status is future work.
class ProfileBannerChip extends StatelessWidget {
  const ProfileBannerChip({required this.label, super.key});

  final String label;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(
        color: BrandColors.paper3,
        borderRadius: BorderRadius.circular(6),
        border: Border.all(color: theme.colorScheme.onSurface, width: 1.5),
      ),
      child: Text(
        label.toUpperCase(),
        style: theme.textTheme.labelSmall?.copyWith(color: BrandColors.ink),
      ),
    );
  }
}
