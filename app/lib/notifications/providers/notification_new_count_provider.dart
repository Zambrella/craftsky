import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'notification_new_count_provider.g.dart';

@Riverpod(keepAlive: true)
class AccountNotificationNewCount extends _$AccountNotificationNewCount {
  @override
  Future<int> build(AccountKey account) async {
    final repository = await ref.watch(
      accountNotificationNewnessRepositoryProvider(account).future,
    );
    return repository.count();
  }

  Future<void> refresh() async {
    final repository = await ref.read(
      accountNotificationNewnessRepositoryProvider(account).future,
    );
    final next = await AsyncValue.guard(repository.count);
    if (ref.mounted) state = next;
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
    final next = await AsyncValue.guard(
      ref.read(notificationNewnessRepositoryProvider).count,
    );
    if (ref.mounted) state = next;
  }

  void suppress(int count) {
    final current = state.value;
    if (current == null || count <= 0) return;
    state = AsyncData((current - count).clamp(0, current));
  }
}
