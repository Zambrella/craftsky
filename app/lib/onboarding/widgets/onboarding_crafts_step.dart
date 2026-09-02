import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/onboarding/models/onboarding_flow_state.dart';
import 'package:craftsky_app/profile/data/crafts_catalog.dart';
import 'package:craftsky_app/profile/widgets/edit_profile_crafts_picker.dart';
import 'package:flutter/material.dart';

class OnboardingCraftsStep extends StatelessWidget {
  const OnboardingCraftsStep({
    required this.state,
    required this.onToggle,
    super.key,
  });

  final OnboardingFlowState state;
  final ValueChanged<Craft> onToggle;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(
          l10n.onboardingCraftsTitle,
          style: Theme.of(context).textTheme.headlineMedium,
        ),
        const SizedBox(height: 8),
        Text(l10n.onboardingCraftsDescription),
        const SizedBox(height: 24),
        EditProfileCraftsPicker(
          selected: {
            for (final craft in Craft.values)
              if (state.selectedCraftIds.contains(craft.id)) craft,
          },
          onToggle: onToggle,
        ),
      ],
    );
  }
}
