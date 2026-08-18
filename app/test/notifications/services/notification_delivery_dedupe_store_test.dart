import 'dart:io';
import 'dart:isolate';

import 'package:craftsky_app/notifications/services/notification_delivery_dedupe_store.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:path/path.dart' as path;
import 'package:sqflite_common_ffi/sqflite_ffi.dart';

void main() {
  sqfliteFfiInit();

  late Directory tempDirectory;
  late String databasePath;
  late DateTime now;

  setUp(() {
    tempDirectory = Directory.systemTemp.createTempSync(
      'craftsky-notification-dedupe-',
    );
    databasePath = path.join(tempDirectory.path, 'dedupe.sqlite');
    now = DateTime.utc(2026, 8, 14, 12);
  });

  tearDown(() async {
    await databaseFactoryFfi.deleteDatabase(databasePath);
    tempDirectory.deleteSync(recursive: true);
  });

  SqliteNotificationDeliveryDedupeStore store() =>
      SqliteNotificationDeliveryDedupeStore(
        databaseFactory: databaseFactoryFfi,
        databasePath: databasePath,
        capacity: 32,
        ttl: const Duration(hours: 1),
        now: () => now,
      );

  test(
    'UT-PUSH-003 stages are independent persisted compare-and-set keys',
    () async {
      final dedupe = store();
      const id = '00000000-0000-4000-8000-000000000001';
      const partition = 'routing-account-one';

      for (final stage in NotificationDeliveryStage.values) {
        expect(
          await dedupe.claim(
            notificationId: id,
            accountPartition: partition,
            stage: stage,
          ),
          isTrue,
          reason: '$stage first claim',
        );
        expect(
          await dedupe.claim(
            notificationId: id,
            accountPartition: partition,
            stage: stage,
          ),
          isFalse,
          reason: '$stage duplicate claim',
        );
      }
    },
  );

  test(
    'IT-PUSH-004 reconstruction and concurrent callers have one winner',
    () async {
      final first = store();
      final reconstructed = store();
      const persistedId = '00000000-0000-4000-8000-000000000002';
      const concurrentId = '00000000-0000-4000-8000-000000000003';
      const partition = 'routing-account-one';

      expect(
        await first.claim(
          notificationId: persistedId,
          accountPartition: partition,
          stage: NotificationDeliveryStage.presented,
        ),
        isTrue,
      );
      expect(
        await reconstructed.claim(
          notificationId: persistedId,
          accountPartition: partition,
          stage: NotificationDeliveryStage.presented,
        ),
        isFalse,
      );

      final outcomes = await Future.wait([
        for (var index = 0; index < 24; index++)
          (index.isEven ? first : reconstructed).claim(
            notificationId: concurrentId,
            accountPartition: partition,
            stage: NotificationDeliveryStage.opened,
          ),
      ]);
      expect(outcomes.where((outcome) => outcome), hasLength(1));
    },
  );

  test('IT-PUSH-005 separate isolates share the persisted CAS', () async {
    const id = '00000000-0000-4000-8000-000000000006';
    final outcomes = await Future.wait([
      for (var index = 0; index < 8; index++)
        Isolate.run(() => _claimFromIsolate(databasePath, id)),
    ]);

    expect(outcomes.where((outcome) => outcome), hasLength(1));
  });

  test('UT-PUSH-006 TTL and deterministic LRU stay bounded', () async {
    final dedupe = store();
    const partition = 'routing-account-one';
    for (var index = 1; index <= 33; index++) {
      final id = '00000000-0000-4000-8000-${index.toString().padLeft(12, '0')}';
      expect(
        await dedupe.claim(
          notificationId: id,
          accountPartition: partition,
          stage: NotificationDeliveryStage.presented,
        ),
        isTrue,
      );
      now = now.add(const Duration(milliseconds: 1));
    }

    expect(
      await dedupe.claim(
        notificationId: '00000000-0000-4000-8000-000000000001',
        accountPartition: partition,
        stage: NotificationDeliveryStage.presented,
      ),
      isTrue,
      reason: 'oldest entry should be evicted at capacity',
    );
    expect(
      await dedupe.claim(
        notificationId: '00000000-0000-4000-8000-000000000033',
        accountPartition: partition,
        stage: NotificationDeliveryStage.presented,
      ),
      isFalse,
      reason: 'newest entry should remain',
    );

    now = now.add(const Duration(hours: 2));
    expect(
      await dedupe.claim(
        notificationId: '00000000-0000-4000-8000-000000000033',
        accountPartition: partition,
        stage: NotificationDeliveryStage.presented,
      ),
      isTrue,
      reason: 'expired entry should be admitted again',
    );
  });

  test('UT-PUSH-007 account clearing removes only that partition', () async {
    final dedupe = store();
    const firstId = '00000000-0000-4000-8000-000000000004';
    const secondId = '00000000-0000-4000-8000-000000000005';
    for (final entry in [
      (id: firstId, partition: 'routing-account-one'),
      (id: secondId, partition: 'routing-account-two'),
    ]) {
      expect(
        await dedupe.claim(
          notificationId: entry.id,
          accountPartition: entry.partition,
          stage: NotificationDeliveryStage.presented,
        ),
        isTrue,
      );
    }

    await dedupe.clearAccountPartition('routing-account-one');
    expect(
      await dedupe.claim(
        notificationId: firstId,
        accountPartition: 'routing-account-one',
        stage: NotificationDeliveryStage.presented,
      ),
      isTrue,
    );
    expect(
      await dedupe.claim(
        notificationId: secondId,
        accountPartition: 'routing-account-two',
        stage: NotificationDeliveryStage.presented,
      ),
      isFalse,
    );
  });

  test('UT-PUSH-008 resets a corrupt disposable cache once', () async {
    File(databasePath).writeAsStringSync('not a SQLite database');
    final dedupe = store();
    const id = '00000000-0000-4000-8000-000000000007';

    expect(
      await dedupe.claim(
        notificationId: id,
        accountPartition: 'routing-account-one',
        stage: NotificationDeliveryStage.presented,
      ),
      isTrue,
    );
    expect(
      await dedupe.claim(
        notificationId: id,
        accountPartition: 'routing-account-one',
        stage: NotificationDeliveryStage.presented,
      ),
      isFalse,
    );
  });

  test('UT-PUSH-009 rejects unsafe cache bounds', () {
    expect(
      () => SqliteNotificationDeliveryDedupeStore(
        databaseFactory: databaseFactoryFfi,
        databasePath: databasePath,
        capacity: 4,
      ),
      throwsRangeError,
    );
    expect(
      () => SqliteNotificationDeliveryDedupeStore(
        databaseFactory: databaseFactoryFfi,
        databasePath: databasePath,
        ttl: const Duration(days: 31),
      ),
      throwsRangeError,
    );
  });
}

Future<bool> _claimFromIsolate(
  String databasePath,
  String notificationId,
) async {
  sqfliteFfiInit();
  final store = SqliteNotificationDeliveryDedupeStore(
    databaseFactory: databaseFactoryFfi,
    databasePath: databasePath,
  );
  return store.claim(
    notificationId: notificationId,
    accountPartition: 'routing-account-one',
    stage: NotificationDeliveryStage.presented,
  );
}
