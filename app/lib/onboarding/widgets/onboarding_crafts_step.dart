import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/onboarding/models/onboarding_flow_state.dart';
import 'package:craftsky_app/profile/data/crafts_catalog.dart';
import 'package:craftsky_app/profile/widgets/edit_profile_crafts_picker.dart';
import 'package:material_ui/material_ui.dart';

class OnboardingCraftsStep extends StatelessWidget {
  const new({
    required this.state,
    required this.onToggle,
    required this.onReorder,
    required this.onRequestMore,
    super.key,
  });

  final OnboardingFlowState state;
  final ValueChanged<Craft> onToggle;
  final ValueChanged<List<Craft>> onReorder;
  final VoidCallback onRequestMore;

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
            for (final id in state.selectedCraftIds) ?Craft.fromId(id),
          },
          onToggle: onToggle,
          onReorder: onReorder,
          enabled: !state.saving,
          onRequestMore: onRequestMore,
        ),
      ],
    );
  }
}
