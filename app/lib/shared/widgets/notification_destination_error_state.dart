import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/errors/notification_destination_error.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

class NotificationDestinationErrorState extends StatelessWidget {
  const NotificationDestinationErrorState({
    required this.error,
    required this.onRetry,
    required this.onBack,
    required this.onViewNotifications,
    super.key,
  });

  final Object error;
  final VoidCallback onRetry;
  final VoidCallback onBack;
  final VoidCallback onViewNotifications;

  @override
  Widget build(BuildContext context) {
    final kind = classifyNotificationDestinationError(error);
    if (kind == NotificationDestinationErrorKind.authenticationLost) {
      return const SizedBox.shrink();
    }

    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final isPermanent =
        kind == NotificationDestinationErrorKind.permanentUnavailable;

    return Center(
      child: Padding(
        padding: EdgeInsets.all(spacing.sp6),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(
              isPermanent
                  ? l10n.notificationDestinationUnavailableTitle
                  : l10n.notificationDestinationRetryTitle,
              style: theme.textTheme.headlineSmall,
              textAlign: TextAlign.center,
            ),
            SizedBox(height: spacing.sp2),
            Text(
              isPermanent
                  ? l10n.notificationDestinationUnavailableBody
                  : l10n.notificationDestinationRetryBody,
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.outline,
              ),
              textAlign: TextAlign.center,
            ),
            SizedBox(height: spacing.sp5),
            if (isPermanent)
              Wrap(
                alignment: WrapAlignment.center,
                spacing: spacing.sp2,
                children: [
                  TextButton(onPressed: onBack, child: Text(l10n.backButton)),
                  TextButton(
                    onPressed: onViewNotifications,
                    child: Text(l10n.notificationDestinationViewNotifications),
                  ),
                ],
              )
            else
              TextButton.icon(
                onPressed: onRetry,
                icon: const Icon(Icons.refresh),
                label: Text(l10n.retryButton),
              ),
          ],
        ),
      ),
    );
  }
}
