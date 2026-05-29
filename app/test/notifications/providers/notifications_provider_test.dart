import 'dart:async';

import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:craftsky_app/notifications/models/notification_page.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/notifications/providers/notifications_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  test('initial load retries after failure', () async {
    final repo = _FakeNotificationRepository()
      ..responses.add(Future<NotificationPage>.error(Exception('nope')))
      ..responses.add(Future.value(NotificationPage(items: [_follow('one')])));
    final container = ProviderContainer(
      overrides: [notificationRepositoryProvider.overrideWithValue(repo)],
    );
    addTearDown(container.dispose);
    final sub = container.listen(
      notificationsProvider,
      (_, _) {},
      fireImmediately: true,
    );
    addTearDown(sub.close);

    await Future<void>.delayed(Duration.zero);
    expect(container.read(notificationsProvider).hasError, isTrue);
    container.invalidate(notificationsProvider);
    final state = await container.read(notificationsProvider.future);

    expect(state.items.map((item) => item.rkey.toString()), ['one']);
    expect(repo.calls, hasLength(2));
  });

  test('appends next page and marks terminal cursor', () async {
    final repo = _FakeNotificationRepository()
      ..responses.add(
        Future.value(NotificationPage(items: [_follow('one')], cursor: 'next')),
      )
      ..responses.add(Future.value(NotificationPage(items: [_follow('two')])));
    final container = ProviderContainer(
      overrides: [notificationRepositoryProvider.overrideWithValue(repo)],
    );
    addTearDown(container.dispose);

    await container.read(notificationsProvider.future);
    await container.read(notificationsProvider.notifier).loadMore();

    final state = container.read(notificationsProvider).value!;
    expect(state.items.map((item) => item.rkey.toString()), ['one', 'two']);
    expect(state.hasMore, isFalse);
    expect(repo.calls.last.cursor, 'next');
  });

  test(
    'preserves rows and cursor on load-more failure and guards concurrency',
    () async {
      final gate = Completer<NotificationPage>();
      final repo = _FakeNotificationRepository()
        ..responses.add(
          Future.value(
            NotificationPage(items: [_follow('one')], cursor: 'next'),
          ),
        )
        ..responses.add(gate.future);
      final container = ProviderContainer(
        overrides: [notificationRepositoryProvider.overrideWithValue(repo)],
      );
      addTearDown(container.dispose);

      await container.read(notificationsProvider.future);
      final firstLoadMore = container
          .read(notificationsProvider.notifier)
          .loadMore();
      await container.read(notificationsProvider.notifier).loadMore();
      gate.completeError(Exception('boom'));
      await firstLoadMore;

      final state = container.read(notificationsProvider);
      expect(repo.calls, hasLength(2));
      expect(state.value!.items.map((item) => item.rkey.toString()), ['one']);
      expect(state.value!.cursor, 'next');
      expect(state.hasError, isTrue);
    },
  );
}

FollowNotification _follow(String rkey) =>
    CraftskyNotification.fromMap({
          'uri': 'at://did:plc:alice/app.bsky.graph.follow/$rkey',
          'cid': 'bafy$rkey',
          'rkey': rkey,
          'type': 'follow',
          'actor': {'did': 'did:plc:alice', 'handle': 'alice.craftsky.social'},
          'createdAt': '2026-05-28T13:00:00Z',
          'indexedAt': '2026-05-28T13:00:01Z',
        })
        as FollowNotification;

class _FakeNotificationRepository implements NotificationRepository {
  final responses = <Future<NotificationPage>>[];
  final calls = <_Call>[];

  @override
  Future<NotificationPage> list({String? cursor, int? limit}) {
    calls.add(_Call(cursor: cursor, limit: limit));
    return responses.removeAt(0);
  }
}

final class _Call {
  const _Call({this.cursor, this.limit});
  final String? cursor;
  final int? limit;
}
