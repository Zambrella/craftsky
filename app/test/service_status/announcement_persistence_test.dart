import 'dart:async';

import 'package:craftsky_app/service_status/data/announcement_dismissal_store.dart';
import 'package:craftsky_app/service_status/models/service_status_document.dart';
import 'package:craftsky_app/service_status/providers/service_status_controller.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'status_flows_controller_test.dart' show Repository;
import 'support/status_test_fixtures.dart';

class PausedDismissalStore implements AnnouncementDismissalStore {
  final inner = PreferencesAnnouncementDismissalStore();
  Completer<void>? pending;
  @override
  Future<String?> readRevision() => inner.readRevision();
  @override
  Future<void> writeRevision(String revision) async {
    await pending?.future;
    await inner.writeRevision(revision);
  }
}

void main() {
  // IR-001 / AT-008, IT-002 / FR-008 / AC-012, AC-022.
  // Real adapter with mocked platform storage; not device durability proof.
  for (final cta in [false, true]) {
    for (final delayed in [false, true]) {
      for (final cover in [
        StatusPresentation.maintenance,
        StatusPresentation.updateRequired,
      ]) {
        testWidgets(
          'late A activation preserves B dismissal: '
          'cta=$cta delayed=$delayed cover=$cover across controller restart',
          (tester) async {
            SharedPreferences.setMockInitialValues({
              'service_status.dismissed_revision': 'legacy',
            });
            final store = PausedDismissalStore();
            var repository = Repository();
            ProviderContainer createContainer(
              AnnouncementDismissalStore store,
            ) => ProviderContainer(
              overrides: [
                ...statusRuntimeOverrides,
                serviceStatusRepositoryProvider.overrideWithValue(repository),
                announcementDismissalStoreProvider.overrideWithValue(store),
              ],
            );
            var container = createContainer(store);
            addTearDown(() => container.dispose());
            var controller = container.read(
              serviceStatusControllerProvider.notifier,
            );
            await tester.pump();
            Future<void> accept(String revision, {bool covered = false}) async {
              final pending = controller.refresh();
              repository.requests.last.complete(
                ServiceStatusDocument(
                  revision: 'D$revision',
                  maintenance:
                      covered && cover == StatusPresentation.maintenance
                      ? const StatusMaintenance(
                          title: 'Break',
                          description: 'Soon',
                        )
                      : null,
                  minimumVersions:
                      covered && cover == StatusPresentation.updateRequired
                      ? const StatusMinimumVersions(ios: '1.9.0')
                      : null,
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
            if (delayed) store.pending = Completer<void>();
            final b = cta
                ? controller.activateMessage('B', () async => true)
                : controller.dismissAnnouncement();
            await tester.pump();
            if (!delayed) expect(await b, isTrue);
            await accept('B', covered: true);
            activation.complete(true);
            await tester.pump();
            expect(controller.presentation, cover);
            store.pending?.complete();
            expect(await b, isTrue);
            expect(await a, isTrue);
            expect(controller.presentation, cover);
            expect(await store.inner.readRevision(), 'B');
            await accept('B');
            expect(controller.announcement, isNull);
            await accept('A');
            expect(
              controller.announcement,
              isNull,
              reason: 'The captured successful action stays session-dismissed',
            );
            await accept('C');
            expect(controller.announcement?.revision, 'C');

            container.dispose();
            repository = Repository();
            final freshStore = PreferencesAnnouncementDismissalStore();
            container = createContainer(freshStore);
            controller = container.read(
              serviceStatusControllerProvider.notifier,
            );
            await tester.pump();
            await accept('B');
            expect(controller.announcement, isNull);
            expect(await freshStore.readRevision(), 'B');
            await accept('A');
            expect(
              controller.announcement?.revision,
              'A',
              reason: 'Only B is durable; no historical ledger was added',
            );
            final preferences = await SharedPreferences.getInstance();
            expect(preferences.getKeys(), {
              'service_status.dismissed_message_revision',
              'service_status.dismissed_revision',
            });
            expect(
              preferences.getString('service_status.dismissed_revision'),
              'legacy',
            );
          },
        );
      }
    }
  }

  // IT-002 / FR-006, FR-008 / AC-009, AC-011. Mock corroboration only.
  test('awaited dismissal is read after reload by a fresh store', () async {
    SharedPreferences.setMockInitialValues({
      'service_status.dismissed_revision': 'legacy',
    });
    final first = PreferencesAnnouncementDismissalStore();
    expect(await first.readRevision(), isNull);
    await first.writeRevision('A');
    final preferences = await SharedPreferences.getInstance();
    await preferences.reload();
    expect(
      preferences.getString('service_status.dismissed_message_revision'),
      'A',
    );
    expect(await PreferencesAnnouncementDismissalStore().readRevision(), 'A');
    expect(preferences.getKeys(), {
      'service_status.dismissed_message_revision',
      'service_status.dismissed_revision',
    });
    expect(
      preferences.getString('service_status.dismissed_revision'),
      'legacy',
    );
  });
}
