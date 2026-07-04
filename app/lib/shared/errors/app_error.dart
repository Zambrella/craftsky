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

enum AppErrorSurface { fullScreen, inline, messenger, silent }

enum AppErrorActionPolicy { none, retry, signIn, goHome }

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
    required this.localizationKey,
    required this.severity,
    required this.surface,
    required this.actionPolicy,
    required this.reportableByDefault,
    required this.sentryClassification,
  });

  final String localizationKey;
  final AppErrorSeverity severity;
  final AppErrorSurface surface;
  final AppErrorActionPolicy actionPolicy;
  final bool reportableByDefault;
  final String sentryClassification;
}

extension AppErrorKindMetadata on AppErrorKind {
  AppErrorMetadata get metadata {
    return switch (this) {
      AppErrorKind.networkUnavailable => const AppErrorMetadata(
        localizationKey: 'errorNetworkUnavailable',
        severity: AppErrorSeverity.warning,
        surface: AppErrorSurface.messenger,
        actionPolicy: AppErrorActionPolicy.retry,
        reportableByDefault: false,
        sentryClassification: 'network.unavailable',
      ),
      AppErrorKind.serviceUnavailable => const AppErrorMetadata(
        localizationKey: 'errorServiceUnavailable',
        severity: AppErrorSeverity.error,
        surface: AppErrorSurface.inline,
        actionPolicy: AppErrorActionPolicy.retry,
        reportableByDefault: false,
        sentryClassification: 'service.unavailable',
      ),
      AppErrorKind.sessionExpired => const AppErrorMetadata(
        localizationKey: 'errorSessionExpired',
        severity: AppErrorSeverity.warning,
        surface: AppErrorSurface.messenger,
        actionPolicy: AppErrorActionPolicy.signIn,
        reportableByDefault: false,
        sentryClassification: 'auth.session_expired',
      ),
      AppErrorKind.permissionDenied => const AppErrorMetadata(
        localizationKey: 'errorPermissionDenied',
        severity: AppErrorSeverity.warning,
        surface: AppErrorSurface.inline,
        actionPolicy: AppErrorActionPolicy.none,
        reportableByDefault: false,
        sentryClassification: 'permission.denied',
      ),
      AppErrorKind.contentUnavailable => const AppErrorMetadata(
        localizationKey: 'errorContentUnavailable',
        severity: AppErrorSeverity.warning,
        surface: AppErrorSurface.inline,
        actionPolicy: AppErrorActionPolicy.none,
        reportableByDefault: false,
        sentryClassification: 'content.unavailable',
      ),
      AppErrorKind.storageUnavailable => const AppErrorMetadata(
        localizationKey: 'errorStorageUnavailable',
        severity: AppErrorSeverity.error,
        surface: AppErrorSurface.messenger,
        actionPolicy: AppErrorActionPolicy.retry,
        reportableByDefault: true,
        sentryClassification: 'storage.unavailable',
      ),
      AppErrorKind.initializationFailed => const AppErrorMetadata(
        localizationKey: 'errorInitializationFailed',
        severity: AppErrorSeverity.error,
        surface: AppErrorSurface.fullScreen,
        actionPolicy: AppErrorActionPolicy.retry,
        reportableByDefault: true,
        sentryClassification: 'initialization.failed',
      ),
      AppErrorKind.navigationFailed => const AppErrorMetadata(
        localizationKey: 'errorNavigationFailed',
        severity: AppErrorSeverity.error,
        surface: AppErrorSurface.fullScreen,
        actionPolicy: AppErrorActionPolicy.goHome,
        reportableByDefault: true,
        sentryClassification: 'navigation.failed',
      ),
      AppErrorKind.actionFailed => const AppErrorMetadata(
        localizationKey: 'errorActionFailed',
        severity: AppErrorSeverity.error,
        surface: AppErrorSurface.messenger,
        actionPolicy: AppErrorActionPolicy.retry,
        reportableByDefault: true,
        sentryClassification: 'action.failed',
      ),
      AppErrorKind.backgroundLoadFailed => const AppErrorMetadata(
        localizationKey: 'errorBackgroundLoadFailed',
        severity: AppErrorSeverity.warning,
        surface: AppErrorSurface.inline,
        actionPolicy: AppErrorActionPolicy.retry,
        reportableByDefault: true,
        sentryClassification: 'background_load.failed',
      ),
      AppErrorKind.unexpected => const AppErrorMetadata(
        localizationKey: 'errorUnexpected',
        severity: AppErrorSeverity.error,
        surface: AppErrorSurface.messenger,
        actionPolicy: AppErrorActionPolicy.retry,
        reportableByDefault: true,
        sentryClassification: 'unexpected',
      ),
    };
  }
}
