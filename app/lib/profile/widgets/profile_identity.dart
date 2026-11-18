import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/profile/models/profile_handle.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar_hero.dart';
import 'package:craftsky_app/profile/widgets/profile_business_hero.dart';
import 'package:craftsky_app/shared/widgets/business_identity_icon.dart';
import 'package:material_ui/material_ui.dart';

/// Display name (DM Serif Display) + optional pronouns + `@handle` block,
/// matching the order from the design mockup.
class ProfileIdentity extends StatelessWidget {
  const new({
    required this.handle,
    this.displayName,
    this.pronouns,
    this.businessLabel,
    this.centered = false,
    this.isBusiness = false,
    this.customisation = ProfileCustomisation.defaults,
    super.key,
  });

  final String handle;
  final String? displayName;
  final String? pronouns;
  final String? businessLabel;
  final bool centered;
  final bool isBusiness;
  final ProfileCustomisation customisation;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context);
    final profileHandle = ProfileHandle(handle);
    final visiblePronouns = pronouns?.trim();
    final name = profileHandle.displayLabel(
      displayName: displayName,
      unavailableLabel: l10n.handleUnavailable,
      includeAtSignWhenNoDisplayName: true,
    );

    // `outline` carries the brand's ink3 (tertiary text) per the
    // ColorScheme override in app_theme.dart.
    final mutedInk = theme.colorScheme.outline;

    return Column(
      crossAxisAlignment: centered
          ? CrossAxisAlignment.center
          : CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        SizedBox(
          width: double.infinity,
          child: FittedBox(
            fit: BoxFit.scaleDown,
            alignment: centered
                ? Alignment.center
                : AlignmentDirectional.centerStart,
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                if (isBusiness) ...[
                  Semantics(
                    label: l10n.businessProfileLabel,
                    image: true,
                    excludeSemantics: true,
                    child: ProfileBusinessHero.destination(
                      context,
                      child: BusinessIdentityIcon(
                        size: 24,
                        customisation: customisation,
                      ),
                    ),
                  ),
                  const SizedBox(width: 8),
                ],
                ProfileIdentityTextHero.destination(
                  context,
                  part: ProfileIdentityHeroPart.name,
                  child: Text(
                    name,
                    style: theme.textTheme.headlineMedium,
                    maxLines: 1,
                    softWrap: false,
                    textAlign: centered ? TextAlign.center : TextAlign.start,
                  ),
                ),
              ],
            ),
          ),
        ),
        if (visiblePronouns?.isNotEmpty ?? false) ...[
          const SizedBox(height: 2),
          Text(
            visiblePronouns!,
            textAlign: centered ? TextAlign.center : TextAlign.start,
            style: theme.textTheme.bodySmall?.copyWith(color: mutedInk),
          ),
        ],
        if (businessLabel != null) ...[
          const SizedBox(height: 2),
          Text(
            businessLabel!,
            textAlign: centered ? TextAlign.center : TextAlign.start,
            style: theme.textTheme.labelLarge?.copyWith(
              color: theme.colorScheme.primary,
              fontWeight: FontWeight.w700,
            ),
          ),
        ],
        if (displayName?.isNotEmpty ?? false) ...[
          const SizedBox(height: 2),
          ProfileIdentityTextHero.destination(
            context,
            part: ProfileIdentityHeroPart.handle,
            child: Text(
              profileHandle.currentLabel(
                unavailableLabel: l10n.handleUnavailable,
              ),
              // `onSurfaceVariant` (ink2) rather than `outline` (ink3) —
              // the @handle reads as a secondary identifier paired with
              // the display name, not tertiary metadata, so it wants
              // the darker secondary-text strength.
              style: theme.textTheme.bodyMedium?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
              textAlign: centered ? TextAlign.center : TextAlign.start,
            ),
          ),
        ],
      ],
    );
  }
}
