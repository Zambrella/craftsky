import 'dart:convert';

import 'package:craftsky_app/service_status/models/service_status_document.dart';
import 'package:craftsky_app/service_status/providers/service_status_controller.dart';
import 'package:craftsky_app/service_status/service_status_estimate_formatter.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:timezone/data/latest.dart' as tzdata;
import 'package:timezone/timezone.dart' as tz;

import 'announcement_dismissal_test.dart' show FakeDismissalStore;
import 'service_status_controller_test.dart' show FakeStatusRepository;

void main() {
  // UT-006 / FR-002, NFR-003, RULE-001 / AC-003, AC-017.
  test(
    'formats the same instant in local time across daylight saving changes',
    () {
      tzdata.initializeTimeZones();
      final london = tz.getLocation('Europe/London');
      DateTime localize(DateTime instant) =>
          tz.TZDateTime.from(instant, london);
      expect(
        formatServiceStatusEstimate(
          DateTime.utc(2026, 3, 29, 0, 30),
          'en',
          localize: localize,
        ),
        'Mar 29, 2026 12:30\u202fAM',
      );
      expect(
        formatServiceStatusEstimate(
          DateTime.utc(2026, 3, 29, 1, 30),
          'en',
          localize: localize,
        ),
        'Mar 29, 2026 2:30\u202fAM',
      );
      final document = ServiceStatusDocument.decode(
        utf8.encode(
          '{"schemaVersion":1,"mode":"maintenance","revision":"A",'
          '"title":"Maintenance","message":"Public",'
          '"estimatedRecoveryAt":"2026-03-29T03:30:00+02:00"}',
        ),
      );
      expect(
        formatServiceStatusEstimate(
          document.estimatedRecoveryAt!,
          'en',
          localize: localize,
        ),
        'Mar 29, 2026 2:30\u202fAM',
      );
      expect(document.mode, ServiceStatusMode.maintenance);
    },
  );
  test(
    'a newly fetched maintenance with a past estimate '
    'still authorizes the cover',
    () async {
      final repository = FakeStatusRepository();
      final container = ProviderContainer(
        overrides: [
          serviceStatusRepositoryProvider.overrideWithValue(repository),
          announcementDismissalStoreProvider.overrideWithValue(
            FakeDismissalStore(),
          ),
        ],
      );
      final controller = container.read(
        serviceStatusControllerProvider.notifier,
      );
      final fetch = controller.refresh();
      repository.requests.last.complete(
        ServiceStatusDocument(
          mode: ServiceStatusMode.maintenance,
          revision: 'A',
          title: 'Maintenance',
          message: 'Public',
          estimatedRecoveryAt: DateTime.utc(2000),
        ),
      );
      await fetch;
      expect(controller.maintenanceActive, isTrue);
      container.dispose();
    },
  );
}
