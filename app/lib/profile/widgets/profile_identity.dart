import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:flutter/material.dart';

/// Display name (DM Serif Display) + optional pronouns + `@handle` block,
/// matching the order from the design mockup. Pronouns are currently a
/// placeholder — there's no wire field for them yet.
class ProfileIdentity extends StatelessWidget {
  const ProfileIdentity({
    required this.handle,
    this.displayName,
    this.pronouns,
    super.key,
  });

  final String handle;
  final String? displayName;
  final String? pronouns;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final name = (displayName?.isNotEmpty ?? false) ? displayName! : '@$handle';

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        Row(
          crossAxisAlignment: CrossAxisAlignment.baseline,
          textBaseline: TextBaseline.alphabetic,
          children: [
            Flexible(
              child: Text(
                name,
                style: theme.textTheme.headlineMedium,
                overflow: TextOverflow.ellipsis,
              ),
            ),
            if (pronouns != null) ...[
              const SizedBox(width: 8),
              Text(
                pronouns!,
                style: theme.textTheme.bodySmall?.copyWith(
                  color: BrandColors.ink3,
                ),
              ),
            ],
          ],
        ),
        if (displayName?.isNotEmpty ?? false) ...[
          const SizedBox(height: 2),
          Text(
            '@$handle',
            style: theme.textTheme.bodyMedium?.copyWith(
              color: BrandColors.ink3,
            ),
          ),
        ],
      ],
    );
  }
}
