import 'package:craftsky_app/l10n/generated/app_localizations.dart';

enum AppErrorKind {
  networkUnavailable,
  serviceUnavailable,
  sessionExpired,
  permissionDenied,
  contentUnavailable,
  storageUnavailable,
  initializationFailed,
  navigationFailed,
  actionFailed,
  backgroundLoadFailed,
  unexpected,
}

enum AppErrorSeverity { info, warning, error }

final class AppError {
  const AppError(
    this.kind, {
    this.safeDiagnostics = const {},
    this.reportableOverride,
    this.sentryClassificationOverride,
  });

  final AppErrorKind kind;
  final Map<String, Object?> safeDiagnostics;
  final bool? reportableOverride;
  final String? sentryClassificationOverride;

  AppErrorMetadata get metadata => kind.metadata;
  bool get reportable => reportableOverride ?? metadata.reportableByDefault;
  String get sentryClassification =>
      sentryClassificationOverride ?? metadata.sentryClassification;
}

final class AppErrorMetadata {
  const AppErrorMetadata({
    required this.severity,
    required this.reportableByDefault,
    required this.sentryClassification,
  });

  final AppErrorSeverity severity;
  final bool reportableByDefault;
  final String sentryClassification;
}

extension AppErrorKindMetadata on AppErrorKind {
  AppErrorMetadata get metadata {
    return switch (this) {
      AppErrorKind.networkUnavailable => const AppErrorMetadata(
        severity: AppErrorSeverity.warning,
        reportableByDefault: false,
        sentryClassification: 'network.unavailable',
      ),
      AppErrorKind.serviceUnavailable => const AppErrorMetadata(
        severity: AppErrorSeverity.error,
        reportableByDefault: false,
        sentryClassification: 'service.unavailable',
      ),
      AppErrorKind.sessionExpired => const AppErrorMetadata(
        severity: AppErrorSeverity.warning,
        reportableByDefault: false,
        sentryClassification: 'auth.session_expired',
      ),
      AppErrorKind.permissionDenied => const AppErrorMetadata(
        severity: AppErrorSeverity.warning,
        reportableByDefault: false,
        sentryClassification: 'permission.denied',
      ),
      AppErrorKind.contentUnavailable => const AppErrorMetadata(
        severity: AppErrorSeverity.warning,
        reportableByDefault: false,
        sentryClassification: 'content.unavailable',
      ),
      AppErrorKind.storageUnavailable => const AppErrorMetadata(
        severity: AppErrorSeverity.error,
        reportableByDefault: true,
        sentryClassification: 'storage.unavailable',
      ),
      AppErrorKind.initializationFailed => const AppErrorMetadata(
        severity: AppErrorSeverity.error,
        reportableByDefault: true,
        sentryClassification: 'initialization.failed',
      ),
      AppErrorKind.navigationFailed => const AppErrorMetadata(
        severity: AppErrorSeverity.error,
        reportableByDefault: true,
        sentryClassification: 'navigation.failed',
      ),
      AppErrorKind.actionFailed => const AppErrorMetadata(
        severity: AppErrorSeverity.error,
        reportableByDefault: true,
        sentryClassification: 'action.failed',
      ),
      AppErrorKind.backgroundLoadFailed => const AppErrorMetadata(
        severity: AppErrorSeverity.warning,
        reportableByDefault: true,
        sentryClassification: 'background_load.failed',
      ),
      AppErrorKind.unexpected => const AppErrorMetadata(
        severity: AppErrorSeverity.error,
        reportableByDefault: true,
        sentryClassification: 'unexpected',
      ),
    };
  }
}

extension AppErrorPresentation on AppError {
  String message(AppLocalizations l10n) {
    return switch (kind) {
      AppErrorKind.networkUnavailable => l10n.errorNetworkUnavailable,
      AppErrorKind.serviceUnavailable => l10n.errorServiceUnavailable,
      AppErrorKind.sessionExpired => l10n.errorSessionExpired,
      AppErrorKind.permissionDenied => l10n.errorPermissionDenied,
      AppErrorKind.contentUnavailable => l10n.errorContentUnavailable,
      AppErrorKind.storageUnavailable => l10n.errorStorageUnavailable,
      AppErrorKind.initializationFailed => l10n.errorInitializationFailed,
      AppErrorKind.navigationFailed => l10n.errorNavigationFailed,
      AppErrorKind.actionFailed => l10n.errorActionFailed,
      AppErrorKind.backgroundLoadFailed => l10n.errorBackgroundLoadFailed,
      AppErrorKind.unexpected => l10n.errorUnexpected,
    };
  }

  String? actionLabel(AppLocalizations l10n) {
    return switch (kind) {
      AppErrorKind.permissionDenied || AppErrorKind.contentUnavailable => null,
      AppErrorKind.sessionExpired => l10n.errorActionSignIn,
      AppErrorKind.navigationFailed => l10n.goHomeButton,
      _ => l10n.retryButton,
    };
  }
}
