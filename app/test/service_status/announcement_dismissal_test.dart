import 'dart:async';

import 'package:craftsky_app/service_status/data/announcement_dismissal_store.dart';
import 'package:craftsky_app/service_status/models/service_status_document.dart';
import 'package:craftsky_app/service_status/providers/service_status_controller.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'service_status_controller_test.dart'
    show FakeStatusRepository, maintenance;

class FakeDismissalStore implements AnnouncementDismissalStore {
  String? revision;
  Completer<void>? pendingWrite;
  @override
  Future<String?> readRevision() async => revision;
  @override
  Future<void> writeRevision(String value) async {
    await pendingWrite?.future;
    revision = value;
  }
}

ServiceStatusDocument announcement(String revision) => ServiceStatusDocument(
  mode: ServiceStatusMode.announcement,
  revision: revision,
  title: 'Notice',
  message: 'Public information',
);

void main() {
  // UT-005 / FR-006 / AC-004, AC-009.
  testWidgets(
    'dismissal applies only to its captured revision and never maintenance',
    (tester) async {
      final repository = FakeStatusRepository();
      final store = FakeDismissalStore();
      final container = ProviderContainer(
        overrides: [
          serviceStatusRepositoryProvider.overrideWithValue(repository),
          announcementDismissalStoreProvider.overrideWithValue(store),
        ],
      );
      final controller = container.read(
        serviceStatusControllerProvider.notifier,
      );
      Future<void> accept(ServiceStatusDocument document) async {
        final refresh = controller.refresh();
        repository.requests.last.complete(document);
        await tester.pump();
        await refresh;
      }

      await accept(announcement('A'));
      expect(controller.announcement?.revision, 'A');
      expect(await controller.dismissAnnouncement(), isTrue);
      expect(controller.announcement, isNull);
      expect(store.revision, 'A');
      await accept(announcement('A'));
      expect(controller.announcement, isNull);
      await accept(announcement('B'));
      expect(controller.announcement?.revision, 'B');
      store.pendingWrite = Completer<void>();
      final dismissB = controller.dismissAnnouncement();
      await accept(announcement('C'));
      store.pendingWrite!.complete();
      await dismissB;
      expect(controller.announcement?.revision, 'C');
      await accept(maintenance);
      expect(controller.maintenanceActive, isTrue);
      expect(await controller.dismissAnnouncement(), isFalse);
      expect(controller.maintenanceActive, isTrue);
      await tester.pump(const Duration(minutes: 5));
      expect(
        container.read(serviceStatusControllerProvider).dismissalReady,
        isTrue,
      );
      container.dispose();
    },
  );
}
