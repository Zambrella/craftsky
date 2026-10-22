import 'dart:async';

import 'package:craftsky_app/service_status/data/announcement_dismissal_store.dart';
import 'package:craftsky_app/service_status/data/service_status_repository.dart';
import 'package:craftsky_app/service_status/models/service_status_document.dart';
import 'package:craftsky_app/service_status/models/status_app_version.dart';
import 'package:craftsky_app/service_status/providers/service_status_controller.dart';
import 'package:craftsky_app/service_status/providers/status_runtime_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

class Repository extends ServiceStatusRepository {
  final requests = <Completer<ServiceStatusDocument>>[];
  bool failStart = false;
  @override
  StatusRequest start() {
    if (failStart) throw StateError('start failure');
    final request = Completer<ServiceStatusDocument>();
    requests.add(request);
    return StatusRequest(request.future, () {});
  }
}

class Store implements AnnouncementDismissalStore {
  String? revision;
  Completer<void>? pending;
  bool failWrite = false;
  final writes = <String>[];
  @override
  Future<String?> readRevision() async => revision;
  @override
  Future<void> writeRevision(String value) async {
    writes.add(value);
    await pending?.future;
    if (failWrite) throw StateError('storage failure');
    revision = value;
  }
}

const message = StatusMessage(
  revision: 'A',
  minVersion: '1.8.0',
  title: 'News',
  description: 'Hello',
);
const maintenance = StatusMaintenance(title: 'Break', description: 'Back soon');
const minima = StatusMinimumVersions(ios: '1.9.0');

void main() {
  // IR-001 / UT-006 / FR-008 / AC-012, AC-022
  testWidgets(
    'late CTA success preserves a newer successful dismissal',
    (tester) async {
      final repository = Repository();
      final store = Store();
      final container = ProviderContainer(
        overrides: [
          statusMessageAudienceReadyProvider.overrideWithValue(true),
          serviceStatusRepositoryProvider.overrideWithValue(repository),
          announcementDismissalStoreProvider.overrideWithValue(store),
          statusRuntimeVersionProvider.overrideWith(
            (ref) async => StatusAppVersion.parse('1.8.0'),
          ),
          statusPlatformProvider.overrideWithValue(StatusPlatform.ios),
        ],
      );
      addTearDown(container.dispose);
      final controller = container.read(
        serviceStatusControllerProvider.notifier,
      );
      await tester.pump();
      Future<void> accept(String revision) async {
        final pending = controller.refresh();
        repository.requests.last.complete(
          ServiceStatusDocument(
            revision: 'D$revision',
            message: StatusMessage(
              revision: revision,
              minVersion: '1.8.0',
              title: 'News',
              description: 'Body',
            ),
          ),
        );
        await tester.pump();
        await pending;
      }

      await accept('A');
      final activation = Completer<bool>();
      final a = controller.activateMessage('A', () => activation.future);
      await accept('B');
      expect(await controller.dismissAnnouncement(), isTrue);
      expect(controller.announcement, isNull);
      expect(store.revision, 'B');
      activation.complete(true);
      expect(await a, isTrue);
      expect(controller.announcement, isNull);
      expect(store.revision, 'B');
      await accept('B');
      expect(controller.announcement, isNull);
    },
  );

  // IR-001 / AT-008 / FR-008 / AC-022
  for (final failure in ['activation', 'persistence']) {
    testWidgets(
      'newer $failure failure does not discard an older successful dismissal',
      (tester) async {
        final repository = Repository();
        final store = Store();
        final container = ProviderContainer(
          overrides: [
            statusMessageAudienceReadyProvider.overrideWithValue(true),
            serviceStatusRepositoryProvider.overrideWithValue(repository),
            announcementDismissalStoreProvider.overrideWithValue(store),
            statusRuntimeVersionProvider.overrideWith(
              (ref) async => StatusAppVersion.parse('1.8.0'),
            ),
            statusPlatformProvider.overrideWithValue(StatusPlatform.ios),
          ],
        );
        addTearDown(container.dispose);
        final controller = container.read(
          serviceStatusControllerProvider.notifier,
        );
        await tester.pump();
        Future<void> accept(String revision) async {
          final pending = controller.refresh();
          repository.requests.last.complete(
            ServiceStatusDocument(
              revision: 'D$revision',
              message: StatusMessage(
                revision: revision,
                minVersion: '1.8.0',
                title: 'News',
                description: 'Body',
              ),
            ),
          );
          await tester.pump();
          await pending;
        }

        await accept('A');
        final activation = Completer<bool>();
        final a = controller.activateMessage('A', () => activation.future);
        await accept('B');
        final didActivate = failure == 'persistence';
        store.failWrite = didActivate;
        expect(
          await controller.activateMessage('B', () async => didActivate),
          didActivate,
        );
        expect(store.revision, isNull);
        store.failWrite = false;
        activation.complete(true);
        expect(await a, isTrue);
        expect(store.revision, 'A');
        await accept('B');
        expect(controller.announcement?.revision, didActivate ? null : 'B');
        expect(
          container.read(serviceStatusControllerProvider).actionFailed,
          !didActivate,
        );
      },
    );
  }

  // UT-008 / RULE-003, FR-001 / AC-005, AC-018, AC-021
  testWidgets(
    'late runtime resolution preserves requests; '
    'start and construction failures release accepted updates',
    (tester) async {
      final repository = Repository();
      final store = Store();
      final version = Completer<StatusAppVersion?>();
      var failConstruction = false;
      final container = ProviderContainer(
        retry: (_, _) => null,
        overrides: [
          statusMessageAudienceReadyProvider.overrideWithValue(true),
          serviceStatusRepositoryProvider.overrideWith((ref) {
            if (failConstruction) throw StateError('construction failure');
            return repository;
          }),
          announcementDismissalStoreProvider.overrideWithValue(store),
          statusRuntimeVersionProvider.overrideWith((ref) => version.future),
          statusPlatformProvider.overrideWithValue(StatusPlatform.ios),
        ],
      );
      final controller = container.read(
        serviceStatusControllerProvider.notifier,
      );
      final first = controller.refresh();
      await tester.pump();
      version.complete(StatusAppVersion.parse('1.8.0'));
      await tester.pump();
      expect(repository.requests, hasLength(1));
      repository.requests.last.complete(
        const ServiceStatusDocument(revision: 'D', minimumVersions: minima),
      );
      await tester.pump();
      await first;
      expect(controller.presentation, StatusPresentation.updateRequired);
      repository.failStart = true;
      await controller.refresh();
      expect(controller.presentation, StatusPresentation.none);
      repository.failStart = false;
      final valid = controller.refresh();
      repository.requests.last.complete(
        const ServiceStatusDocument(revision: 'D2', minimumVersions: minima),
      );
      await tester.pump();
      await valid;
      expect(controller.presentation, StatusPresentation.updateRequired);
      failConstruction = true;
      container.invalidate(serviceStatusRepositoryProvider);
      await controller.refresh();
      expect(controller.presentation, StatusPresentation.none);
      container.dispose();
    },
  );

  // AT-008 / FR-008, FR-010 / AC-022
  testWidgets(
    'CTA success hides captured revision even on storage failure; '
    'late completion respects replacement and cover',
    (tester) async {
      final repository = Repository();
      final store = Store();
      final container = ProviderContainer(
        overrides: [
          statusMessageAudienceReadyProvider.overrideWithValue(true),
          serviceStatusRepositoryProvider.overrideWithValue(repository),
          announcementDismissalStoreProvider.overrideWithValue(store),
          statusRuntimeVersionProvider.overrideWith(
            (ref) async => StatusAppVersion.parse('1.8.0'),
          ),
          statusPlatformProvider.overrideWithValue(StatusPlatform.ios),
        ],
      );
      final controller = container.read(
        serviceStatusControllerProvider.notifier,
      );
      await tester.pump();
      Future<void> accept(String revision, {bool covered = false}) async {
        final refresh = controller.refresh();
        repository.requests.last.complete(
          ServiceStatusDocument(
            revision: 'D$revision',
            maintenance: covered ? maintenance : null,
            message: StatusMessage(
              revision: revision,
              minVersion: '1.8.0',
              title: 'News',
              description: 'Body',
            ),
          ),
        );
        await tester.pump();
        await refresh;
      }

      await accept('A');
      expect(await controller.activateMessage('A', () async => false), isFalse);
      expect(controller.announcement?.revision, 'A');
      expect(store.writes, isEmpty);
      final activation = Completer<bool>();
      var launches = 0;
      final pending = controller.activateMessage('A', () {
        launches++;
        return activation.future;
      });
      expect(launches, 1);
      expect(
        await controller.activateMessage('A', () async {
          launches++;
          return true;
        }),
        isFalse,
      );
      await accept('B');
      activation.complete(true);
      await pending;
      expect(controller.announcement?.revision, 'B');
      expect(store.revision, 'A');
      store.failWrite = true;
      expect(await controller.activateMessage('B', () async => true), isTrue);
      expect(controller.announcement, isNull);
      await accept('B');
      expect(controller.announcement, isNull);
      final late = Completer<bool>();
      await accept('C');
      final c = controller.activateMessage('C', () => late.future);
      await accept('C', covered: true);
      late.complete(true);
      await c;
      expect(controller.presentation, StatusPresentation.maintenance);
      expect(launches, 1);
      container.dispose();
    },
  );

  // UT-006 / FR-008, FR-009 / AC-012, AC-013
  testWidgets(
    'revision dismissal serializes writes and cannot clear replacement state',
    (tester) async {
      final repository = Repository();
      final store = Store();
      final container = ProviderContainer(
        overrides: [
          statusMessageAudienceReadyProvider.overrideWithValue(true),
          serviceStatusRepositoryProvider.overrideWithValue(repository),
          announcementDismissalStoreProvider.overrideWithValue(store),
          statusRuntimeVersionProvider.overrideWith(
            (ref) async => StatusAppVersion.parse('1.8.0'),
          ),
          statusPlatformProvider.overrideWithValue(StatusPlatform.ios),
        ],
      );
      final controller = container.read(
        serviceStatusControllerProvider.notifier,
      );
      await tester.pump();
      Future<void> accept(String root, String revision) async {
        final pending = controller.refresh();
        repository.requests.last.complete(
          ServiceStatusDocument(
            revision: root,
            message: StatusMessage(
              revision: revision,
              minVersion: '1.8.0',
              title: 'News',
              description: 'Body',
            ),
          ),
        );
        await tester.pump();
        await pending;
      }

      await accept('D1', 'A');
      store.pending = Completer<void>();
      final a = controller.dismissAnnouncement();
      await accept('D2', 'B');
      expect(
        container.read(serviceStatusControllerProvider).dismissing,
        isFalse,
      );
      final b = controller.dismissAnnouncement();
      await tester.pump();
      expect(store.writes, ['A']);
      store.pending!.complete();
      await a;
      await b;
      expect(store.writes, ['A', 'B']);
      expect(store.revision, 'B');
      expect(controller.announcement, isNull);
      await accept('D3', 'B');
      expect(controller.announcement, isNull);
      await accept('D4', 'A');
      expect(controller.announcement?.revision, 'A');
      container.dispose();
    },
  );

  // UT-008 / RULE-001, RULE-003, FR-006 / AC-018, AC-021
  testWidgets(
    'failed refresh releases updates without renewing maintenance '
    'or reviving stale policy',
    (tester) async {
      final repository = Repository();
      final store = Store();
      final container = ProviderContainer(
        overrides: [
          statusMessageAudienceReadyProvider.overrideWithValue(true),
          serviceStatusRepositoryProvider.overrideWithValue(repository),
          announcementDismissalStoreProvider.overrideWithValue(store),
          statusRuntimeVersionProvider.overrideWith(
            (ref) async => StatusAppVersion.parse('1.8.0'),
          ),
          statusPlatformProvider.overrideWithValue(StatusPlatform.ios),
        ],
      );
      final controller = container.read(
        serviceStatusControllerProvider.notifier,
      );
      await tester.pump();
      Future<void> accept(ServiceStatusDocument document) async {
        final pending = controller.refresh();
        repository.requests.last.complete(document);
        await tester.pump();
        await pending;
      }

      expect(controller.presentation, StatusPresentation.none);
      await accept(
        const ServiceStatusDocument(
          revision: 'D1',
          maintenance: maintenance,
          minimumVersions: minima,
          message: message,
        ),
      );
      await tester.pump(const Duration(minutes: 4));
      final failed = controller.refresh();
      repository.requests.last.completeError(
        const FormatException('invalid snapshot'),
      );
      await tester.pump();
      await failed;
      expect(controller.presentation, StatusPresentation.maintenance);
      await tester.pump(const Duration(minutes: 1));
      expect(controller.presentation, StatusPresentation.message);
      await accept(
        const ServiceStatusDocument(
          revision: 'D2',
          minimumVersions: minima,
          message: message,
        ),
      );
      expect(controller.presentation, StatusPresentation.updateRequired);
      final timedOut = controller.refresh();
      await tester.pump(const Duration(seconds: 3));
      await timedOut;
      expect(controller.presentation, StatusPresentation.message);
      repository.requests.last.complete(
        const ServiceStatusDocument(revision: 'late', minimumVersions: minima),
      );
      await tester.pump();
      expect(controller.presentation, StatusPresentation.message);
      await accept(
        const ServiceStatusDocument(revision: 'D3', minimumVersions: minima),
      );
      expect(controller.presentation, StatusPresentation.updateRequired);
      container.dispose();
    },
  );

  // UT-003 / BR-001, RULE-001 / AC-001, AC-002
  testWidgets(
    'independent controls select maintenance then update then message',
    (tester) async {
      final repository = Repository();
      final store = Store();
      final container = ProviderContainer(
        overrides: [
          statusMessageAudienceReadyProvider.overrideWithValue(true),
          serviceStatusRepositoryProvider.overrideWithValue(repository),
          announcementDismissalStoreProvider.overrideWithValue(store),
          statusRuntimeVersionProvider.overrideWith(
            (ref) async => StatusAppVersion.parse('1.8.0+13'),
          ),
          statusPlatformProvider.overrideWithValue(StatusPlatform.ios),
        ],
      );
      final controller = container.read(
        serviceStatusControllerProvider.notifier,
      );
      await tester.pump();
      for (final (document, expected) in [
        (
          const ServiceStatusDocument(
            revision: 'D1',
            maintenance: maintenance,
            minimumVersions: minima,
            message: message,
          ),
          StatusPresentation.maintenance,
        ),
        (
          const ServiceStatusDocument(
            revision: 'D2',
            minimumVersions: minima,
            message: message,
          ),
          StatusPresentation.updateRequired,
        ),
        (
          const ServiceStatusDocument(revision: 'D3', message: message),
          StatusPresentation.message,
        ),
        (const ServiceStatusDocument(revision: 'D4'), StatusPresentation.none),
      ]) {
        final pending = controller.refresh();
        repository.requests.last.complete(document);
        await tester.pump();
        await pending;
        expect(controller.presentation, expected);
        expect(store.writes, isEmpty);
      }
      container.dispose();
    },
  );
}
