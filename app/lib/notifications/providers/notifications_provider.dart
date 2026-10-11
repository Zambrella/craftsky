import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:craftsky_app/notifications/models/notifications_state.dart';
import 'package:craftsky_app/notifications/providers/notification_filter_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/profile/providers/block_profile_overlay.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'notifications_provider.g.dart';

const notificationsPageLimit = 20;
int _nextRenderToken = 0;

@Riverpod(keepAlive: true)
class AccountNotifications extends _$AccountNotifications {
  int _generation = 0;

  @override
  Future<NotificationsState> build(AccountKey account) async {
    ++_generation;
    final category = ref.watch(notificationFilterProvider);
    final repo = await ref.watch(
      accountNotificationRepositoryProvider(account).future,
    );
    final registry = ref.read(sessionRegistryProvider).requireValue;
    final owner = registry.leaseFor(account);
    final page = await repo.list(
      limit: notificationsPageLimit,
      category: category,
    );
    return NotificationsState(
      items: _filterAndDedupe(ref.read, page.items),
      category: category,
      cursor: page.cursor,
      renderToken: ++_nextRenderToken,
      owner: owner,
    );
  }

  Future<void> loadMore() async {
    if (!state.hasValue || state.isLoading) return;
    final current = state.requireValue;
    final generation = _generation;
    final category = ref.read(notificationFilterProvider);
    if (!current.hasMore) return;
    state = const AsyncLoading<NotificationsState>();
    final next = await AsyncValue.guard(() async {
      final repo = await ref.read(
        accountNotificationRepositoryProvider(account).future,
      );
      final page = await repo.list(
        cursor: current.cursor,
        category: category,
        limit: notificationsPageLimit,
      );
      return NotificationsState(
        category: category,
        items: _appendDeduped(
          current.items,
          _filterAndDedupe(ref.read, page.items),
        ),
        cursor: page.cursor,
        renderToken: current.renderToken,
        owner: current.owner,
      );
    });
    if (ref.mounted &&
        generation == _generation &&
        ref.read(notificationFilterProvider) == category) {
      state = next;
    }
  }

  int suppressActor(String did) => _suppressActor(state, did, (next) {
    state = next;
  });
}

@Riverpod(keepAlive: true)
class Notifications extends _$Notifications {
  int _generation = 0;

  @override
  Future<NotificationsState> build() async {
    ++_generation;
    final owner = ref.read(sessionRegistryProvider).value?.activeLease?.session;
    final repo = ref.watch(notificationRepositoryProvider);
    final category = ref.watch(notificationFilterProvider);
    final page = await repo.list(
      limit: notificationsPageLimit,
      category: category,
    );
    return NotificationsState(
      items: _filterAndDedupe(ref.read, page.items),
      category: category,
      cursor: page.cursor,
      renderToken: ++_nextRenderToken,
      owner: owner,
    );
  }

  Future<void> loadMore() async {
    if (!state.hasValue || state.isLoading) return;
    final current = state.requireValue;
    final generation = _generation;
    final category = ref.read(notificationFilterProvider);
    if (!current.hasMore) return;

    state = const AsyncLoading<NotificationsState>();

    final next = await AsyncValue.guard(() async {
      final repo = ref.read(notificationRepositoryProvider);
      final page = await repo.list(
        cursor: current.cursor,
        category: category,
        limit: notificationsPageLimit,
      );
      return NotificationsState(
        category: category,
        items: _appendDeduped(
          current.items,
          _filterAndDedupe(ref.read, page.items),
        ),
        cursor: page.cursor,
        renderToken: current.renderToken,
        owner: current.owner,
      );
    });

    if (!ref.mounted ||
        generation != _generation ||
        ref.read(notificationFilterProvider) != category) {
      return;
    }
    state = next;
  }

  int suppressActor(String did) => _suppressActor(state, did, (next) {
    state = next;
  });
}

int _suppressActor(
  AsyncValue<NotificationsState> current,
  String did,
  void Function(AsyncValue<NotificationsState>) update,
) {
  final value = current.value;
  if (value == null) return 0;
  final retained = value.items
      .where(
        (item) =>
            item is! ActorNotification || item.actor.did.toString() != did,
      )
      .toList();
  final removed = value.items.length - retained.length;
  if (removed > 0) {
    update(AsyncData(value.copyWith(items: retained)));
  }
  return removed;
}

List<CraftskyNotification> _filterAndDedupe(
  BlockOverlayReader read,
  List<CraftskyNotification> items,
) {
  final seen = <String>{};
  return [
    for (final item in items)
      if ((item is! ActorNotification ||
              !isLogicallyBlocking(
                read,
                item.actor.did,
                authoritativeBlocking: item.actor.blocking ?? false,
              )) &&
          seen.add(item.id))
        item,
  ];
}

List<CraftskyNotification> _appendDeduped(
  List<CraftskyNotification> current,
  List<CraftskyNotification> next,
) {
  final seen = current.map((item) => item.id).toSet();
  return [
    ...current,
    for (final item in next)
      if (seen.add(item.id)) item,
  ];
}
