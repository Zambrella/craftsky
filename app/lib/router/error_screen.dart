import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/errors/app_error.dart';
import 'package:craftsky_app/shared/errors/app_error_mapper.dart';
import 'package:flutter/material.dart';

class ErrorScreen extends StatelessWidget {
  const ErrorScreen({required this.error, super.key});

  /// Type is `Object` rather than `Exception` because `GoRouterState.error`
  /// is `Object?` in go_router 17 — routing errors are not always exceptions.
  final Object error;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final appError = AppErrorMapper.map(
      error,
      fallbackKind: AppErrorKind.navigationFailed,
      source: 'routing',
    );
    return Scaffold(
      body: Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Icon(
                Icons.broken_image_outlined,
                size: 64,
                color: theme.colorScheme.error,
              ),
              const SizedBox(height: 16),
              Text(
                l10n.routingErrorTitle,
                style: theme.textTheme.headlineSmall,
              ),
              const SizedBox(height: 8),
              Text(
                appError.message(l10n),
                textAlign: TextAlign.center,
                style: theme.textTheme.bodyMedium,
              ),
              const SizedBox(height: 24),
              ElevatedButton.icon(
                onPressed: () => const FeedRoute().go(context),
                icon: const Icon(Icons.home),
                label: Text(l10n.goHomeButton),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
