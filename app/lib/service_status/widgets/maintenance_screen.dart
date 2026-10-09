import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/service_status/models/service_status_document.dart';
import 'package:craftsky_app/service_status/service_status_estimate_formatter.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:flutter/material.dart';

class MaintenanceScreen extends StatelessWidget {
  const MaintenanceScreen({
    required this.document,
    required this.onRetry,
    this.fetching = false,
    super.key,
  });
  final ServiceStatusDocument document;
  final VoidCallback onRetry;
  final bool fetching;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context);
    return Material(
      color: theme.colorScheme.surface,
      child: SafeArea(
        child: LayoutBuilder(
          builder: (context, constraints) => SingleChildScrollView(
            child: ConstrainedBox(
              constraints: BoxConstraints(minHeight: constraints.maxHeight),
              child: Center(
                child: ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 560),
                  child: Padding(
                    padding: const EdgeInsets.all(24),
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Icon(
                          Icons.handyman_outlined,
                          size: 64,
                          color: theme.colorScheme.primary,
                        ),
                        const SizedBox(height: 24),
                        Semantics(
                          header: true,
                          child: Text(
                            document.title!,
                            textAlign: TextAlign.center,
                            style: theme.textTheme.headlineMedium,
                          ),
                        ),
                        const SizedBox(height: 16),
                        Text(
                          document.message!,
                          textAlign: TextAlign.center,
                          style: theme.textTheme.bodyLarge,
                        ),
                        if (document.estimatedRecoveryAt != null) ...[
                          const SizedBox(height: 16),
                          Text(
                            l10n.serviceStatusEstimatedRecovery(
                              formatServiceStatusEstimate(
                                document.estimatedRecoveryAt!,
                                l10n.localeName,
                              ),
                            ),
                            textAlign: TextAlign.center,
                          ),
                        ],
                        const SizedBox(height: 24),
                        ChunkyButton(
                          autofocus: true,
                          onPressed: fetching ? null : onRetry,
                          child: Text(l10n.serviceStatusTryAgain),
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
