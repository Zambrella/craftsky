import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'notification_new_count_provider.g.dart';

@Riverpod(keepAlive: true)
class AccountNotificationNewCount extends _$AccountNotificationNewCount {
  @override
  Future<int> build(AccountKey account) async {
    // The count owns its disposable account repository through this watch.
    // ignore: riverpod_lint/only_use_keep_alive_inside_keep_alive
    final repository = await ref.watch(
      accountNotificationNewnessRepositoryProvider(account).future,
    );
    return await repository.count();
  }

  Future<void> refresh() async {
    final operationRef = ref;
    // build's watch keeps the same account repository alive during refresh.
    // ignore: riverpod_lint/only_use_keep_alive_inside_keep_alive
    final repository = await operationRef.read(
      accountNotificationNewnessRepositoryProvider(account).future,
    );
    if (!operationRef.mounted) return;
    final next = await AsyncValue.guard(repository.count);
    if (operationRef.mounted) state = next;
  }

  void suppress(int count) {
    final current = state.value;
    if (current == null || count <= 0) return;
    state = AsyncData((current - count).clamp(0, current));
  }
}

@Riverpod(keepAlive: true)
class NotificationNewCount extends _$NotificationNewCount {
  @override
  Future<int> build() =>
      ref.watch(notificationNewnessRepositoryProvider).count();

  Future<void> refresh() async {
    final operationRef = ref;
    final next = await AsyncValue.guard(
      operationRef.read(notificationNewnessRepositoryProvider).count,
    );
    if (operationRef.mounted) state = next;
  }

  void suppress(int count) {
    final current = state.value;
    if (current == null || count <= 0) return;
    state = AsyncData((current - count).clamp(0, current));
  }
}
