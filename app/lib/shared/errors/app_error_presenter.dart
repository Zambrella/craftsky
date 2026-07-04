import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/errors/app_error.dart';

final class AppErrorPresenter {
  const AppErrorPresenter._();

  static String message(AppLocalizations l10n, AppError error) {
    return switch (error.kind) {
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

  static String? actionLabel(AppLocalizations l10n, AppError error) {
    return switch (error.metadata.actionPolicy) {
      AppErrorActionPolicy.none => null,
      AppErrorActionPolicy.retry => l10n.retryButton,
      AppErrorActionPolicy.signIn => l10n.errorActionSignIn,
      AppErrorActionPolicy.goHome => l10n.goHomeButton,
    };
  }
}
