import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/onboarding/models/onboarding_flow_state.dart';
import 'package:flutter/material.dart';

class OnboardingProgress extends StatelessWidget {
  const OnboardingProgress({required this.step, super.key});

  final OnboardingStep step;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    const total = 3;
    final label = l10n.onboardingStepProgress(step.number, total);
    return Semantics(
      label: l10n.onboardingProgressSemantics(step.number, total),
      value: label,
      excludeSemantics: true,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(label, style: Theme.of(context).textTheme.labelLarge),
          const SizedBox(height: 8),
          LinearProgressIndicator(value: step.progress),
        ],
      ),
    );
  }
}
