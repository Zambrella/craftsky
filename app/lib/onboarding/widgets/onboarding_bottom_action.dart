import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/onboarding/models/onboarding_action_state.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/material.dart';

class OnboardingBottomAction extends StatelessWidget {
  const OnboardingBottomAction({
    required this.state,
    required this.onPressed,
    super.key,
  });

  final OnboardingActionState state;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final label = switch (state.kind) {
      OnboardingActionKind.next => l10n.onboardingNext,
      OnboardingActionKind.saveAndNext => l10n.onboardingSaveNext,
      OnboardingActionKind.finish => l10n.onboardingFinish,
    };
    return SafeArea(
      top: false,
      minimum: const EdgeInsets.fromLTRB(24, 12, 24, 16),
      child: Semantics(
        key: const Key('onboarding-primary-action-semantics'),
        excludeSemantics: true,
        button: true,
        enabled: state.canSubmit,
        label: label,
        value: state.busy ? l10n.loading : null,
        liveRegion: state.busy,
        onTap: state.canSubmit ? onPressed : null,
        child: SizedBox(
          width: double.infinity,
          child: ChunkyButton(
            onPressed: state.canSubmit ? onPressed : null,
            child: state.busy
                ? const StitchProgressIndicator(size: 20)
                : Text(label),
          ),
        ),
      ),
    );
  }
}
