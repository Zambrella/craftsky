import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/service_status/models/service_status_document.dart';
import 'package:craftsky_app/service_status/service_status_estimate_formatter.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:flutter/material.dart';

class AnnouncementModal extends StatelessWidget {
  const AnnouncementModal({
    required this.document,
    required this.onDismiss,
    this.dismissing = false,
    super.key,
  });
  final ServiceStatusDocument document;
  final VoidCallback onDismiss;
  final bool dismissing;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context);
    return SafeArea(
      child: Semantics(
        scopesRoute: true,
        explicitChildNodes: true,
        namesRoute: true,
        label: document.title,
        child: CraftskyDialog(
          title: document.title!,
          body: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(document.message!),
              if (document.estimatedRecoveryAt != null) ...[
                const SizedBox(height: 16),
                Text(
                  l10n.serviceStatusEstimatedRecovery(
                    formatServiceStatusEstimate(
                      document.estimatedRecoveryAt!,
                      l10n.localeName,
                    ),
                  ),
                ),
              ],
            ],
          ),
          actions: [
            ChunkyButton(
              autofocus: true,
              backgroundColor: theme.colorScheme.primary,
              onPressed: dismissing ? null : onDismiss,
              child: Text(l10n.serviceStatusDismiss),
            ),
          ],
        ),
      ),
    );
  }
}
