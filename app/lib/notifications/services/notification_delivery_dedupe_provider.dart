import 'package:craftsky_app/notifications/services/notification_delivery_dedupe_store.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Foreground account cleanup and background presentation open independent
/// SQLite handles to the same bounded cache. SQLite provides the cross-isolate
/// compare-and-set and targeted account-partition deletion contract.
final notificationDeliveryDedupeStoreProvider =
    Provider<NotificationDeliveryDedupeStore>(
      (_) => SqliteNotificationDeliveryDedupeStore(),
    );
