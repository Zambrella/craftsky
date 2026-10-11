import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/error_reporting_controller.dart';
import 'package:craftsky_app/shared/observability/observability_bootstrap.dart';
import 'package:craftsky_app/shared/observability/sentry_config.dart';
import 'package:craftsky_app/shared/observability/sentry_error_reporter.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:sentry_flutter/sentry_flutter.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../test_support/serialized_sentry_transport.dart';

void main() {
  final config = SentryConfig.fromValues(
    dsn: 'https://public@example.invalid/1',
    environment: 'production',
  );
  const context = ReportContext(
    feature: 'Test',
    operation: 'consent',
    classification: 'test.consent',
  );

  setUp(() => SharedPreferences.setMockInitialValues({}));
  tearDown(Sentry.close);

  test(
    'no consent skips SDK, opt-in exports, opt-out stops and persists',
    () async {
      final prefs = await SharedPreferences.getInstance();
      final adapter = _Adapter();
      final controller = ErrorReportingController(
        preferences: prefs,
        config: config,
        adapter: adapter,
        closeSdk: Sentry.close,
      );
      await controller.initialize();
      expect(controller.optedIn, isFalse);
      expect(Sentry.isEnabled, isFalse);
      expect(adapter.calls, 0);
      expect(
        await controller.captureException(
          StateError('PRIVATE_OFF'),
          context: context,
        ),
        isNull,
      );

      await controller.setEnabled(enabled: true);
      expect(controller.enabled, isTrue);
      expect(Sentry.isEnabled, isTrue);
      expect(
        await controller.captureException(
          StateError('PRIVATE_ON'),
          context: context,
        ),
        isNotNull,
      );
      expect(adapter.transport.payloads.join(), contains('StateError'));
      expect(adapter.transport.payloads.join(), isNot(contains('PRIVATE_ON')));

      await controller.setEnabled(enabled: false);
      expect(controller.enabled, isFalse);
      expect(Sentry.isEnabled, isFalse);
      final sent = adapter.transport.payloads.length;
      await controller.captureException(
        StateError('PRIVATE_OFF_AGAIN'),
        context: context,
      );
      await Sentry.captureException(StateError('PRIVATE_AUTOMATIC_OFF'));
      expect(adapter.transport.payloads.length, sent);

      final restarted = ErrorReportingController(
        preferences: prefs,
        config: config,
        adapter: adapter,
        closeSdk: Sentry.close,
      );
      await restarted.initialize();
      expect(restarted.optedIn, isFalse);
      expect(adapter.calls, 1);
      await restarted.setEnabled(enabled: true);
      expect(adapter.calls, 2);
      expect(restarted.enabled, isTrue);
    },
  );

  test('failed persistence cannot start SDK or change consent', () async {
    final adapter = _Adapter();
    final controller = ErrorReportingController(
      preferences: _RejectingPreferences(),
      config: config,
      adapter: adapter,
      closeSdk: Sentry.close,
    );
    await expectLater(controller.setEnabled(enabled: true), throwsStateError);
    expect(controller.optedIn, isFalse);
    expect(controller.enabled, isFalse);
    expect(adapter.calls, 0);
    expect(Sentry.isEnabled, isFalse);
  });

  test('saved opt-in initializes SDK on restart', () async {
    SharedPreferences.setMockInitialValues({
      ErrorReportingController.preferenceKey: true,
    });
    final adapter = _Adapter();
    final controller = ErrorReportingController(
      preferences: await SharedPreferences.getInstance(),
      config: config,
      adapter: adapter,
      closeSdk: Sentry.close,
    );
    await controller.initialize();
    expect(controller.optedIn, isTrue);
    expect(controller.enabled, isTrue);
    expect(adapter.calls, 1);
  });

  test('consent cannot override a build without a DSN', () async {
    final adapter = _Adapter();
    final controller = ErrorReportingController(
      preferences: await SharedPreferences.getInstance(),
      config: SentryConfig.fromValues(environment: 'production'),
      adapter: adapter,
      closeSdk: Sentry.close,
    );
    await controller.setEnabled(enabled: true);
    expect(controller.optedIn, isTrue);
    expect(controller.enabled, isFalse);
    expect(adapter.calls, 0);
  });
}

class _Adapter implements SentryBootstrapAdapter {
  int calls = 0;
  final transport = SerializedTransport();

  @override
  Future<ErrorReporter> initialize(SentryConfig config) async {
    calls++;
    await Sentry.init((options) {
      options
        ..dsn = config.dsn
        ..transport = transport;
      configureDiagnosticOptions(options);
    });
    return const SentryErrorReporter();
  }
}

class _RejectingPreferences extends Fake implements SharedPreferences {
  @override
  bool? getBool(String key) => false;

  @override
  Future<bool> setBool(String key, bool value) async => false;
}
