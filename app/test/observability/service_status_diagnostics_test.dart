import 'dart:async';
import 'dart:io';

import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/service_status/data/service_status_repository.dart';
import 'package:craftsky_app/service_status/models/service_status_document.dart';
import 'package:craftsky_app/service_status/providers/service_status_controller.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/sentry_error_reporter.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:logging/logging.dart';
import 'package:sentry_flutter/sentry_flutter.dart';

import '../service_status/announcement_dismissal_test.dart'
    show FakeDismissalStore;
import '../service_status/service_status_controller_test.dart'
    show FakeStatusRepository;
import '../test_support/diagnostic_evidence.dart';
import '../test_support/serialized_sentry_transport.dart';

void main() {
  test(
    'provider-owned resolution failure is not captured again by the controller',
    () async {
      final reporter = _Reporter(const NoopErrorReporter());
      final container = ProviderContainer(
        retry: (_, _) => null,
        observers: [ProviderLogger(reporter: reporter)],
        overrides: [
          serviceStatusRepositoryProvider.overrideWith(
            (ref) => throw StateError('provider-secret-canary'),
          ),
          announcementDismissalStoreProvider.overrideWithValue(
            FakeDismissalStore(),
          ),
          serviceStatusReporterProvider.overrideWithValue(reporter),
        ],
      );
      await container.read(serviceStatusControllerProvider.notifier).refresh();
      expect(reporter.errors, hasLength(1));
      expect(container.read(serviceStatusControllerProvider).fetching, isFalse);
      container.dispose();
    },
  );
  testWidgets('throwing cancellation cannot strand retry or disposal', (
    tester,
  ) async {
    final reporter = _Reporter(const NoopErrorReporter())..fail = true;
    final container = ProviderContainer(
      overrides: [
        serviceStatusRepositoryProvider.overrideWithValue(_ThrowingCancel()),
        announcementDismissalStoreProvider.overrideWithValue(
          FakeDismissalStore(),
        ),
        serviceStatusReporterProvider.overrideWithValue(reporter),
      ],
    );
    final controller = container.read(serviceStatusControllerProvider.notifier);
    final pending = controller.refresh();
    await tester.pump(const Duration(seconds: 3));
    await pending;
    expect(container.read(serviceStatusControllerProvider).fetching, isFalse);
    final next = controller.refresh();
    container.dispose();
    await next;
  });

  // IT-006 / NFR-004, RULE-002 / AC-018, AC-019.
  test(
    'consumed failure has one owner with bounded serialized diagnostics',
    () async {
      final transport = SerializedTransport();
      await Sentry.init((options) {
        options
          ..dsn = 'https://public@example.invalid/1'
          ..transport = transport;
        configureDiagnosticOptions(options);
      });
      addTearDown(Sentry.close);
      final lines = <String>[];
      final emitter = DiagnosticEmitter(
        platformSink: lines.add,
        debugLogs: true,
      );
      Logger.root.level = Level.ALL;
      final subscription = Logger.root.onRecord.listen(emitter.emitLocal);
      addTearDown(subscription.cancel);
      final repository = FakeStatusRepository();
      final recording = _Reporter(const SentryErrorReporter());
      final container = ProviderContainer(
        observers: [const ProviderLogger()],
        overrides: [
          serviceStatusRepositoryProvider.overrideWithValue(repository),
          announcementDismissalStoreProvider.overrideWithValue(
            FakeDismissalStore(),
          ),
          serviceStatusReporterProvider.overrideWithValue(recording),
        ],
      );
      addTearDown(container.dispose);
      container.listen(serviceStatusControllerProvider, (_, _) {});
      final controller = container.read(
        serviceStatusControllerProvider.notifier,
      );
      Future<void> fail(Object error, [StackTrace? stack]) async {
        final pending = controller.refresh();
        repository.requests.last.completeError(error, stack);
        await pending;
      }

      for (var i = 0; i < 3; i++) {
        await fail(const SocketException('private-remote-body-canary'));
        await fail(const FormatException('private-title-canary'));
      }
      expect(recording.errors, isEmpty);
      final ready = controller.refresh();
      repository.requests.last.complete(
        const ServiceStatusDocument(
          mode: ServiceStatusMode.maintenance,
          revision: 'private-revision-canary',
          title: 'private-title-canary',
          message: 'private-remote-body-canary',
        ),
      );
      await ready;
      final error = StateError('private-token-canary');
      final stack = StackTrace.fromString(
        '#0 statusRead (package:craftsky_app/service_status/reader.dart:12:3)',
      );
      await fail(error, stack);
      await transport.issueReceived.future.timeout(const Duration(seconds: 3));
      expect(recording.errors, [same(error)]);
      expect(recording.stacks.single, same(stack));
      expect(controller.maintenanceActive, isTrue);
      expect(
        transport.payloads.where((line) => line.contains('"exception"')),
        hasLength(1),
      );
      final serialized = [...lines, ...transport.payloads].join('\n');
      expect(serialized, contains('ServiceStatus'));
      expect(serialized, contains('fetch'));
      expect(serialized, contains('StateError'));
      expect(serialized, contains('reader.dart'));
      for (final canary in [
        'private-title-canary',
        'private-remote-body-canary',
        'private-token-canary',
        'private-revision-canary',
      ]) {
        expect(serialized, isNot(contains(canary)));
      }
      recording.fail = true;
      await runZoned(
        () async {
          await fail(StateError('reporter-failure-canary'));
          await Future<void>.delayed(Duration.zero);
        },
        zoneSpecification: ZoneSpecification(
          print: (_, _, _, line) => lines.add(line),
        ),
      );
      expect(lines.join('\n'), contains('Telemetry reporting failed'));
      expect(lines.join('\n'), isNot(contains('reporter-failure-canary')));
      expect(container.read(serviceStatusControllerProvider).fetching, isFalse);
      expect(controller.maintenanceActive, isTrue);
      writeDiagnosticEvidence(
        'service-status',
        local: lines,
        exported: transport.payloads,
      );
    },
  );
}

class _Reporter implements ErrorReporter {
  _Reporter(this.delegate);
  final ErrorReporter delegate;
  final errors = <Object>[];
  final stacks = <StackTrace?>[];
  bool fail = false;
  @override
  bool get enabled => true;
  @override
  Future<String?> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) async {
    errors.add(error);
    stacks.add(stackTrace);
    if (fail) throw StateError('reporter-failure-canary');
    return delegate.captureException(
      error,
      context: context,
      stackTrace: stackTrace,
    );
  }
}

class _ThrowingCancel extends ServiceStatusRepository {
  @override
  StatusRequest start() => StatusRequest(
    Completer<ServiceStatusDocument>().future,
    () => throw StateError('cancel-secret-canary'),
  );
}
