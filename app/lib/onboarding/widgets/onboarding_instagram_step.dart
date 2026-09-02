import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/instagram_migration/pages/instagram_migration_page.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:flutter/material.dart';

class OnboardingInstagramStep extends StatelessWidget {
  const OnboardingInstagramStep({required this.lease, super.key});

  final ActiveAccountLease lease;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(
          l10n.onboardingInstagramTitle,
          style: Theme.of(context).textTheme.headlineMedium,
        ),
        const SizedBox(height: 8),
        Text(l10n.onboardingInstagramDescription),
        const SizedBox(height: 24),
        InstagramOnboardingSections(lease: lease),
      ],
    );
  }
}
