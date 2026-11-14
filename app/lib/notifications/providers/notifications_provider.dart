import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_operation_canceled.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:craftsky_app/notifications/models/notification_category.dart';
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
  int _loadedPageCount = 1;
  NotificationCategory? _loadedCategory;

  @override
  Future<NotificationsState> build(AccountKey account) async {
    final buildRef = ref;
    final generation = ++_generation;
    final category = buildRef.watch(notificationFilterProvider);
    if (category != _loadedCategory) {
      _loadedPageCount = 1;
      _loadedCategory = category;
    }
    // This watch owns the disposable repository for the retained list.
    // ignore: riverpod_lint/only_use_keep_alive_inside_keep_alive
    final repo = await buildRef.watch(
      accountNotificationRepositoryProvider(account).future,
    );
    if (!buildRef.mounted) throw AccountOperationCanceled();
    final registry = buildRef.read(sessionRegistryProvider).requireValue;
    final owner = registry.leaseFor(account);
    final (items, cursor, pageCount) = await _reloadPages(
      repo,
      category,
      _loadedPageCount,
    );
    if (!buildRef.mounted || generation != _generation) {
      throw AccountOperationCanceled();
    }
    _loadedPageCount = pageCount;
    return NotificationsState(
      items: _filterAndDedupe(buildRef.read, items),
      category: category,
      cursor: cursor,
      renderToken: ++_nextRenderToken,
      owner: owner,
    );
  }

  Future<void> loadMore() async {
    final operationRef = ref;
    if (!state.hasValue || state.isLoading) return;
    final current = state.requireValue;
    final generation = _generation;
    final category = operationRef.read(notificationFilterProvider);
    if (!current.hasMore) return;
    state = const AsyncLoading<NotificationsState>();
    final next = await AsyncValue.guard(() async {
      // build's watch owns this repository throughout pagination.
      // ignore: riverpod_lint/only_use_keep_alive_inside_keep_alive
      final repo = await operationRef.read(
        accountNotificationRepositoryProvider(account).future,
      );
      if (!operationRef.mounted ||
          generation != _generation ||
          operationRef.read(notificationFilterProvider) != category) {
        throw AccountOperationCanceled();
      }
      final page = await repo.list(
        cursor: current.cursor,
        category: category,
        limit: notificationsPageLimit,
      );
      if (!operationRef.mounted ||
          generation != _generation ||
          operationRef.read(notificationFilterProvider) != category) {
        throw AccountOperationCanceled();
      }
      return NotificationsState(
        category: category,
        items: _appendDeduped(
          current.items,
          _filterAndDedupe(operationRef.read, page.items),
        ),
        cursor: page.cursor,
        renderToken: current.renderToken,
        owner: current.owner,
      );
    });
    if (operationRef.mounted &&
        generation == _generation &&
        operationRef.read(notificationFilterProvider) == category) {
      if (next.hasValue) _loadedPageCount++;
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
  int _loadedPageCount = 1;
  NotificationCategory? _loadedCategory;

  @override
  Future<NotificationsState> build() async {
    final buildRef = ref;
    final generation = ++_generation;
    final owner = buildRef
        .read(sessionRegistryProvider)
        .value
        ?.activeLease
        ?.session;
    final repo = buildRef.watch(notificationRepositoryProvider);
    final category = buildRef.watch(notificationFilterProvider);
    if (category != _loadedCategory) {
      _loadedPageCount = 1;
      _loadedCategory = category;
    }
    final (items, cursor, pageCount) = await _reloadPages(
      repo,
      category,
      _loadedPageCount,
    );
    if (!buildRef.mounted || generation != _generation) {
      throw AccountOperationCanceled();
    }
    _loadedPageCount = pageCount;
    return NotificationsState(
      items: _filterAndDedupe(buildRef.read, items),
      category: category,
      cursor: cursor,
      renderToken: ++_nextRenderToken,
      owner: owner,
    );
  }

  Future<void> loadMore() async {
    final operationRef = ref;
    if (!state.hasValue || state.isLoading) return;
    final current = state.requireValue;
    final generation = _generation;
    final category = operationRef.read(notificationFilterProvider);
    if (!current.hasMore) return;

    state = const AsyncLoading<NotificationsState>();

    final next = await AsyncValue.guard(() async {
      final repo = operationRef.read(notificationRepositoryProvider);
      final page = await repo.list(
        cursor: current.cursor,
        category: category,
        limit: notificationsPageLimit,
      );
      if (!operationRef.mounted ||
          generation != _generation ||
          operationRef.read(notificationFilterProvider) != category) {
        throw AccountOperationCanceled();
      }
      return NotificationsState(
        category: category,
        items: _appendDeduped(
          current.items,
          _filterAndDedupe(operationRef.read, page.items),
        ),
        cursor: page.cursor,
        renderToken: current.renderToken,
        owner: current.owner,
      );
    });

    if (!operationRef.mounted ||
        generation != _generation ||
        operationRef.read(notificationFilterProvider) != category) {
      return;
    }
    if (next.hasValue) _loadedPageCount++;
    state = next;
  }

  int suppressActor(String did) => _suppressActor(state, did, (next) {
    state = next;
  });
}

// A return visit refreshes the same depth the user already loaded, rather
// than dropping the tail and clamping a scrolled viewport to the first page.
Future<(List<CraftskyNotification>, String?, int)> _reloadPages(
  NotificationRepository repo,
  NotificationCategory? category,
  int pageCount,
) async {
  final items = <CraftskyNotification>[];
  String? cursor;
  var loaded = 0;
  do {
    final page = await repo.list(
      limit: notificationsPageLimit,
      category: category,
      cursor: cursor,
    );
    items.addAll(page.items);
    cursor = page.cursor;
    loaded++;
  } while (loaded < pageCount && cursor != null);
  return (items, cursor, loaded);
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
