import 'package:craftsky_app/account_eligibility/providers/account_eligibility_provider.dart';
import 'package:craftsky_app/auth/models/active_account_initialization.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/languages/providers/account_language_preferences_provider.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:craftsky_app/shared/errors/app_error.dart';
import 'package:craftsky_app/shared/errors/app_error_mapper.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:logging/logging.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'active_account_initialization_provider.g.dart';

final _log = Logger('ActiveAccountInitialization');

/// Supporting diagnostic for the initialization occurrence; ProviderLogger owns
/// its unexpected issue. Identity comes from the lease of this initialization.
void logActiveAccountInitializationFailure(
  Object error,
  StackTrace stackTrace, {
  String? accountDid,
}) {
  final appError = AppErrorMapper.map(
    error,
    fallbackKind: AppErrorKind.backgroundLoadFailed,
    source: 'initialization',
    fallbackClassification: 'initialization.failed',
  );
  _log.severe(
    DiagnosticMessage(
      'Active account failed to initialize',
      context: ReportContext(
        feature: 'ActiveAccountInitialization',
        operation: 'account.initialize',
        classification: appError.sentryClassification,
        safeDiagnostics: {
          ...appError.safeDiagnostics,
          'failureStage': 'initialization',
        },
        workflow: accountDid == null
            ? null
            : PublicRecordContext(actorDid: accountDid),
      ),
    ),
    error,
    stackTrace,
  );
}

/// Resolves the account-critical state for the exact active session lease.
///
/// A signed-out registry is a successful initialization with no account.
/// Loading and failures from either dependency remain visible to the gate.
@Riverpod(keepAlive: true)
FutureOr<ActiveAccountInitialization?> activeAccountInitialization(Ref ref) {
  final registry = ref.watch(sessionRegistryProvider).requireValue;
  final lease = registry.activeLease;
  if (lease == null) return null;

  final preferences = ref
      .watch(
        accountLanguagePreferencesProvider(lease),
      )
      .requireValue
      .preferences;
  final onboarding = ref
      .watch(onboardingStatusProvider(lease.session))
      .requireValue;
  final eligibility = ref.watch(accountEligibilityProvider(lease)).requireValue;
  return ActiveAccountInitialization(
    lease: lease,
    languagePreferences: preferences,
    onboardingComplete: onboarding.completed,
    accountEligibility: eligibility,
  );
}
