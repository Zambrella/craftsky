import 'dart:async';

import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class AccountEligibilityPage extends ConsumerWidget {
  const AccountEligibilityPage({required this.appealGuidance, super.key});

  final String? appealGuidance;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(l10n.accountEligibilityTitle)),
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.all(24),
          children: [
            Text(
              l10n.accountEligibilityHeading,
              style: Theme.of(context).textTheme.headlineMedium,
            ),
            const SizedBox(height: 12),
            Text(appealGuidance ?? l10n.accountEligibilityAppealGuidance),
            const SizedBox(height: 24),
            FilledButton(
              onPressed: () => const AccountStandingRoute().go(context),
              child: Text(l10n.accountEligibilityViewStanding),
            ),
            OutlinedButton.icon(
              icon: const Icon(CraftskyIcons.muted),
              label: Text(l10n.settingsMutedAccounts),
              onPressed: () => const MutedAccountsRoute().go(context),
            ),
            OutlinedButton.icon(
              icon: const Icon(CraftskyIconsBold.block),
              label: Text(l10n.settingsBlockedAccounts),
              onPressed: () => const BlockedAccountsRoute().go(context),
            ),
            OutlinedButton.icon(
              icon: const Icon(CraftskyIcons.privacy),
              label: Text(l10n.settingsPrivacyPolicy),
              onPressed: () => const AboutRoute().go(context),
            ),
            TextButton(
              onPressed: () => const AccountRoute().go(context),
              child: Text(l10n.accountEligibilityManageAccount),
            ),
            TextButton(
              onPressed: () => unawaited(
                ref.read(authControllerProvider.notifier).signOut(),
              ),
              child: Text(l10n.settingsSignOut),
            ),
          ],
        ),
      ),
    );
  }
}
