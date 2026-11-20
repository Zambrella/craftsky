import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_operation_canceled.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'notification_new_count_provider.g.dart';

@Riverpod(keepAlive: true)
class AccountNotificationNewCount extends _$AccountNotificationNewCount {
  int _countRevision = 0;

  @override
  Future<int> build(AccountKey account) async {
    // The count owns its disposable account repository through this watch.
    // ignore: riverpod_lint/only_use_keep_alive_inside_keep_alive
    final repository = await ref.watch(
      accountNotificationNewnessRepositoryProvider(account).future,
    );
    return await repository.count();
  }

  Future<void> refresh({int suppressCount = 0}) async {
    final operationRef = ref;
    final revision = ++_countRevision;
    final next = await AsyncValue.guard(() async {
      // build's watch keeps the same account repository alive during refresh.
      // ignore: riverpod_lint/only_use_keep_alive_inside_keep_alive
      final repository = await operationRef.read(
        accountNotificationNewnessRepositoryProvider(account).future,
      );
      if (!operationRef.mounted || revision != _countRevision) {
        throw AccountOperationCanceled();
      }
      return _subtractCount(await repository.count(), suppressCount);
    });
    if (operationRef.mounted && revision == _countRevision) state = next;
  }

  void suppress(int count) {
    final current = state.value;
    if (current == null || count <= 0) return;
    _countRevision++;
    state = AsyncData(_subtractCount(current, count));
  }
}

@Riverpod(keepAlive: true)
class NotificationNewCount extends _$NotificationNewCount {
  int _countRevision = 0;

  @override
  Future<int> build() =>
      ref.watch(notificationNewnessRepositoryProvider).count();

  Future<void> refresh({int suppressCount = 0}) async {
    final operationRef = ref;
    final revision = ++_countRevision;
    final next = await AsyncValue.guard(() async {
      final repository = operationRef.read(
        notificationNewnessRepositoryProvider,
      );
      return _subtractCount(await repository.count(), suppressCount);
    });
    if (operationRef.mounted && revision == _countRevision) state = next;
  }

  void suppress(int count) {
    final current = state.value;
    if (current == null || count <= 0) return;
    _countRevision++;
    state = AsyncData(_subtractCount(current, count));
  }
}

int _subtractCount(int current, int count) =>
    (current - count).clamp(0, current);
