import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/observability_bootstrap.dart';
import 'package:craftsky_app/shared/observability/sentry_config.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// Device-wide consent, shared by onboarding and settings. The stable reporter
/// lets existing error owners observe SDK changes without rebuilding the app.
final class ErrorReportingController implements ErrorReporter {
  new({
    required this.preferences,
    required this.config,
    required this.adapter,
    required this.closeSdk,
  }) : _optedIn = preferences.getBool(preferenceKey) ?? false;

  static const preferenceKey = 'error_reporting_enabled';
  final SharedPreferences preferences;
  final SentryConfig config;
  final SentryBootstrapAdapter adapter;
  final Future<void> Function() closeSdk;
  ErrorReporter _reporter = const NoopErrorReporter();
  bool _changing = false;
  bool _optedIn;

  bool get optedIn => _optedIn;

  Future<void> initialize() async {
    if (optedIn) {
      _reporter = await ObservabilityBootstrap.initialize(
        config: config,
        adapter: adapter,
      );
    }
  }

  Future<void> setEnabled({required bool enabled}) async {
    if (_changing) return;
    _changing = true;
    try {
      // Persist consent before starting the SDK. A rejected storage write must
      // never accidentally opt someone in.
      if (!await preferences.setBool(preferenceKey, enabled)) {
        throw StateError('Could not save error reporting preference');
      }
      _optedIn = enabled;
      if (enabled) {
        await initialize();
      } else {
        _reporter = const NoopErrorReporter();
        await closeSdk();
      }
    } finally {
      _changing = false;
    }
  }

  @override
  bool get enabled => optedIn && _reporter.enabled;

  @override
  Future<String?> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) async {
    if (!enabled) return null;
    return await _reporter.captureException(
      error,
      context: context,
      stackTrace: stackTrace,
    );
  }
}
