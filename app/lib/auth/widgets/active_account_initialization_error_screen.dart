import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/errors/app_error.dart';
import 'package:craftsky_app/shared/errors/app_error_mapper.dart';
import 'package:craftsky_app/shared/widgets/action_progress.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:material_ui/material_ui.dart';

class ActiveAccountInitializationErrorScreen extends StatelessWidget {
  const new({
    required this.error,
    required this.onRetry,
    required this.onSignOut,
    required this.busy,
    this.onSwitchAccount,
    this.isSigningOut = false,
    super.key,
  });

  final VoidCallback onRetry;
  final Object error;
  final VoidCallback? onSwitchAccount;
  final VoidCallback onSignOut;
  final bool busy;
  final bool isSigningOut;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final appError = AppErrorMapper.map(
      error,
      fallbackKind: AppErrorKind.initializationFailed,
      source: 'initialization',
    );
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
                  CraftskyIcons.offline,
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
                  appError.kind == AppErrorKind.initializationFailed
                      ? l10n.activeAccountInitializationFailedBody
                      : appError.message(l10n),
                  textAlign: TextAlign.center,
                  style: theme.textTheme.bodyMedium,
                ),
                const SizedBox(height: 24),
                FilledButton.icon(
                  onPressed: busy ? null : onRetry,
                  icon: const Icon(CraftskyIconsBold.refresh),
                  label: Text(l10n.retryButton),
                ),
                if (onSwitchAccount != null) ...[
                  const SizedBox(height: 8),
                  OutlinedButton.icon(
                    onPressed: busy ? null : onSwitchAccount,
                    icon: const Icon(CraftskyIconsBold.switchAccount),
                    label: Text(l10n.activeAccountSwitchAction),
                  ),
                ],
                const SizedBox(height: 8),
                TextButton.icon(
                  onPressed: busy ? null : onSignOut,
                  icon: ActionProgress(
                    isLoading: isSigningOut,
                    child: const Icon(CraftskyIconsBold.logout),
                  ),
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
