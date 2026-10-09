import 'dart:async';

import 'package:craftsky_app/service_status/data/service_status_repository.dart';
import 'package:craftsky_app/service_status/models/service_status_document.dart';
import 'package:craftsky_app/service_status/providers/service_status_controller.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

class FakeStatusRepository extends ServiceStatusRepository {
  final requests = <Completer<ServiceStatusDocument>>[];
  int cancellations = 0;

  @override
  StatusRequest start() {
    final result = Completer<ServiceStatusDocument>();
    requests.add(result);
    return StatusRequest(result.future, () => cancellations++);
  }
}

void main() {
  // UT-003 / FR-004, NFR-001 / AC-006.
  testWidgets('coalesces triggers, times out, polls only in foreground', (
    tester,
  ) async {
    final repository = FakeStatusRepository();
    final container = ProviderContainer(
      overrides: [
        serviceStatusRepositoryProvider.overrideWithValue(repository),
      ],
    );
    final controller = container.read(serviceStatusControllerProvider.notifier);
    // Keep the resolved controller visible as the fixture under test.
    // ignore: cascade_invocations
    controller.start();
    final retry = controller.refresh();
    expect(identical(retry, controller.refresh()), isTrue);
    expect(repository.requests, hasLength(1));
    await tester.pump(const Duration(milliseconds: 2999));
    expect(container.read(serviceStatusControllerProvider).fetching, isTrue);
    await tester.pump(const Duration(milliseconds: 1));
    await retry;
    expect(repository.cancellations, 1);
    expect(container.read(serviceStatusControllerProvider).fetching, isFalse);
    await tester.pump(const Duration(seconds: 57));
    expect(repository.requests, hasLength(2));
    repository.requests.last.complete(
      const ServiceStatusDocument(
        mode: ServiceStatusMode.normal,
        revision: 'A',
      ),
    );
    await tester.pump();
    controller.setForeground(foreground: false);
    await tester.pump(const Duration(minutes: 2));
    expect(repository.requests, hasLength(2));
    controller.setForeground(foreground: true);
    expect(repository.requests, hasLength(3));
    container.dispose();
    expect(repository.cancellations, 2);
    await tester.pump(const Duration(minutes: 2));
    expect(repository.requests, hasLength(3));
  });
  for (final mode in ServiceStatusMode.values) {
    testWidgets('polls every 60 seconds in ${mode.name}', (tester) async {
      final repository = FakeStatusRepository();
      final container = ProviderContainer(
        overrides: [
          serviceStatusRepositoryProvider.overrideWithValue(repository),
        ],
      );
      final controller = container.read(
        serviceStatusControllerProvider.notifier,
      );
      // Explicit duplicate trigger verifies idempotence.
      // ignore: cascade_invocations
      controller
        ..start()
        ..start();
      repository.requests.single.complete(
        ServiceStatusDocument(
          mode: mode,
          revision: 'A',
          title: 'Title',
          message: 'Message',
        ),
      );
      await tester.pump();
      expect(
        container.read(serviceStatusControllerProvider).document?.mode,
        mode,
      );
      await tester.pump(const Duration(seconds: 59));
      expect(repository.requests, hasLength(1));
      await tester.pump(const Duration(seconds: 1));
      expect(repository.requests, hasLength(2));
      container.dispose();
    });
  }
  // UT-004 / FR-003, FR-004, FR-008, NFR-001, RULE-001 / AC-005, AC-007, AC-011.
  testWidgets(
    'expires maintenance exactly at five minutes without renewal on failure',
    (tester) async {
      final repository = FakeStatusRepository();
      final clock = FakeStatusClock();
      final container = ProviderContainer(
        overrides: [
          serviceStatusRepositoryProvider.overrideWithValue(repository),
          serviceStatusClockProvider.overrideWithValue(clock),
        ],
      );
      final controller = container.read(
        serviceStatusControllerProvider.notifier,
      );
      // Keep the resolved controller visible as the fixture under test.
      // ignore: cascade_invocations
      controller.start();
      repository.requests.single.complete(maintenance);
      await tester.pump();
      expect(controller.maintenanceActive, isTrue);
      controller.setForeground(foreground: false);
      clock.advance(const Duration(minutes: 4, seconds: 59));
      await tester.pump(const Duration(minutes: 4, seconds: 59));
      expect(controller.maintenanceActive, isTrue);
      final retry = controller.refresh();
      repository.requests.last.completeError(const FormatException('Invalid'));
      await tester.pump();
      await retry;
      clock.advance(const Duration(seconds: 1));
      await tester.pump(const Duration(seconds: 1));
      expect(controller.maintenanceActive, isFalse);
      expect(
        container.read(serviceStatusControllerProvider).document?.mode,
        ServiceStatusMode.maintenance,
      );
      container.dispose();
    },
  );

  testWidgets(
    'unchanged success renews; normal and announcement clear immediately',
    (tester) async {
      final repository = FakeStatusRepository();
      final clock = FakeStatusClock();
      final container = ProviderContainer(
        overrides: [
          serviceStatusRepositoryProvider.overrideWithValue(repository),
          serviceStatusClockProvider.overrideWithValue(clock),
        ],
      );
      final controller = container.read(
        serviceStatusControllerProvider.notifier,
      )..start();
      repository.requests.last.complete(maintenance);
      await tester.pump();
      controller.setForeground(foreground: false);
      clock.advance(const Duration(minutes: 4));
      await tester.pump(const Duration(minutes: 4));
      final renewal = controller.refresh();
      repository.requests.last.complete(maintenance);
      await tester.pump();
      await renewal;
      clock.advance(const Duration(minutes: 4));
      await tester.pump(const Duration(minutes: 4));
      expect(controller.maintenanceActive, isTrue);
      for (final mode in [
        ServiceStatusMode.normal,
        ServiceStatusMode.announcement,
      ]) {
        final cleared = controller.refresh();
        repository.requests.last.complete(
          ServiceStatusDocument(mode: mode, revision: 'B'),
        );
        await tester.pump();
        await cleared;
        expect(controller.maintenanceActive, isFalse);
      }
      container.dispose();
    },
  );

  testWidgets(
    'late timed-out maintenance cannot replace a newer normal result',
    (tester) async {
      final repository = FakeStatusRepository();
      final container = ProviderContainer(
        overrides: [
          serviceStatusRepositoryProvider.overrideWithValue(repository),
        ],
      );
      final controller = container.read(
        serviceStatusControllerProvider.notifier,
      )..start();
      final old = repository.requests.single;
      await tester.pump(const Duration(seconds: 3));
      final next = controller.refresh();
      repository.requests.last.complete(
        const ServiceStatusDocument(
          mode: ServiceStatusMode.normal,
          revision: 'B',
        ),
      );
      await tester.pump();
      await next;
      old.complete(maintenance);
      await tester.pump();
      expect(
        container.read(serviceStatusControllerProvider).document?.revision,
        'B',
      );
      expect(controller.maintenanceActive, isFalse);
      container.dispose();
    },
  );

  testWidgets(
    'wall suspension expires trust; backwards wall time invalidates it',
    (tester) async {
      final repository = FakeStatusRepository();
      final clock = FakeStatusClock();
      final container = ProviderContainer(
        overrides: [
          serviceStatusRepositoryProvider.overrideWithValue(repository),
          serviceStatusClockProvider.overrideWithValue(clock),
        ],
      );
      final controller = container.read(
        serviceStatusControllerProvider.notifier,
      )..start();
      repository.requests.last.complete(maintenance);
      await tester.pump();
      controller.setForeground(foreground: false);
      clock.wall = clock.wall.subtract(const Duration(seconds: 1));
      expect(controller.maintenanceActive, isFalse);
      clock.wall = clock.wall.add(const Duration(seconds: 2));
      expect(controller.maintenanceActive, isFalse);
      final renewed = controller.refresh();
      repository.requests.last.complete(maintenance);
      await tester.pump();
      await renewed;
      expect(controller.maintenanceActive, isTrue);
      clock.wall = clock.wall.add(const Duration(minutes: 10));
      controller.setForeground(foreground: true);
      expect(controller.maintenanceActive, isFalse);
      repository.requests.last.completeError(Exception('Offline'));
      await tester.pump();
      expect(controller.maintenanceActive, isFalse);
      // Once expired, a later wall correction cannot revive the old grant.
      clock.wall = clock.wall.subtract(const Duration(minutes: 9));
      expect(controller.maintenanceActive, isFalse);
      container.dispose();
    },
  );
}

const maintenance = ServiceStatusDocument(
  mode: ServiceStatusMode.maintenance,
  revision: 'A',
  title: 'Maintenance',
  message: 'Public message',
);

class FakeStatusClock implements ServiceStatusClock {
  Duration elapsed = Duration.zero;
  DateTime wall = DateTime.utc(2026, 10, 9);
  void advance(Duration duration) {
    elapsed += duration;
    wall = wall.add(duration);
  }

  @override
  StatusAgeSample sample() => StatusAgeSample(elapsed, wall);
}
