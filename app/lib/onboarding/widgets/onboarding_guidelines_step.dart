import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/material.dart';

class OnboardingGuidelinesStep extends StatelessWidget {
  const OnboardingGuidelinesStep({
    required this.onViewFullGuidelines,
    super.key,
  });

  final VoidCallback onViewFullGuidelines;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(
          l10n.onboardingGuidelinesTitle,
          style: theme.textTheme.headlineMedium,
        ),
        const SizedBox(height: 24),
        _GuidelineSection(
          heading: l10n.onboardingGuidelinesWelcomeHeading,
          body: l10n.onboardingGuidelinesWelcomeBody,
        ),
        const SizedBox(height: 20),
        _GuidelineSection(
          heading: l10n.onboardingGuidelinesRespectHeading,
          body: l10n.onboardingGuidelinesRespectBody,
        ),
        const SizedBox(height: 20),
        _GuidelineSection(
          heading: l10n.onboardingGuidelinesCraftHeading,
          body: l10n.onboardingGuidelinesCraftBody,
        ),
        const SizedBox(height: 20),
        _GuidelineSection(
          heading: l10n.onboardingGuidelinesCheckHeading,
          body: l10n.onboardingGuidelinesCheckBody,
        ),
        const SizedBox(height: 24),
        Align(
          alignment: AlignmentDirectional.centerStart,
          child: OutlinedButton.icon(
            onPressed: onViewFullGuidelines,
            icon: const Icon(CraftskyIconsBold.externalLink),
            label: Text(l10n.onboardingGuidelinesViewFull),
          ),
        ),
      ],
    );
  }
}

class _GuidelineSection extends StatelessWidget {
  const _GuidelineSection({required this.heading, required this.body});

  final String heading;
  final String body;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(heading, style: theme.textTheme.titleMedium),
        const SizedBox(height: 6),
        Text(body, style: theme.textTheme.bodyLarge),
      ],
    );
  }
}
