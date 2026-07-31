import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:flutter/material.dart';

class ActiveAccountInitializationErrorScreen extends StatelessWidget {
  const ActiveAccountInitializationErrorScreen({
    required this.onRetry,
    required this.onSignOut,
    required this.busy,
    this.onSwitchAccount,
    super.key,
  });

  final VoidCallback onRetry;
  final VoidCallback? onSwitchAccount;
  final VoidCallback onSignOut;
  final bool busy;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    return Scaffold(
      body: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 440),
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Icon(
                  Icons.cloud_off_outlined,
                  color: theme.colorScheme.error,
                  size: 64,
                ),
                const SizedBox(height: 16),
                Text(
                  l10n.activeAccountInitializationFailedTitle,
                  textAlign: TextAlign.center,
                  style: theme.textTheme.headlineSmall,
                ),
                const SizedBox(height: 8),
                Text(
                  l10n.activeAccountInitializationFailedBody,
                  textAlign: TextAlign.center,
                  style: theme.textTheme.bodyMedium,
                ),
                const SizedBox(height: 24),
                FilledButton.icon(
                  onPressed: busy ? null : onRetry,
                  icon: const Icon(Icons.refresh),
                  label: Text(l10n.retryButton),
                ),
                if (onSwitchAccount != null) ...[
                  const SizedBox(height: 8),
                  OutlinedButton.icon(
                    onPressed: busy ? null : onSwitchAccount,
                    icon: const Icon(Icons.switch_account_outlined),
                    label: Text(l10n.activeAccountSwitchAction),
                  ),
                ],
                const SizedBox(height: 8),
                TextButton.icon(
                  onPressed: busy ? null : onSignOut,
                  icon: const Icon(Icons.logout),
                  label: Text(l10n.activeAccountSignOutAction),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
