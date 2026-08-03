import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/scheduled_posts/models/scheduled_post.dart';
import 'package:craftsky_app/scheduled_posts/providers/scheduled_post_repository_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'scheduled_posts_provider.g.dart';

final class ScheduledPostListState {
  ScheduledPostListState(List<ScheduledPostSummary> items)
    : items = List.unmodifiable(_sortAndDeduplicate(items));

  final List<ScheduledPostSummary> items;

  @override
  String toString() => 'ScheduledPostListState [REDACTED]';
}

@riverpod
class ScheduledPosts extends _$ScheduledPosts {
  @override
  Future<ScheduledPostListState> build(AccountKey account) async {
    final ownership = _captureOwnership(account);
    final result = await _load(account);
    _assertCurrent(ownership);
    return result;
  }

  Future<void> refresh() async {
    final ownership = _captureOwnership(account);
    final result = await _load(account);
    if (!isActiveAccountOperationCurrent(ref, ownership)) return;
    state = AsyncData(result);
  }

  ActiveAccountLease? _captureOwnership(AccountKey account) {
    final ownership = captureActiveAccountOperation(ref);
    if (ownership != null && ownership.session.account != account) {
      throw StateError('scheduled-post account changed');
    }
    return ownership;
  }

  void _assertCurrent(ActiveAccountLease? ownership) {
    if (!isActiveAccountOperationCurrent(ref, ownership)) {
      throw StateError('scheduled-post account changed');
    }
  }

  Future<ScheduledPostListState> _load(AccountKey account) async {
    final repository = await ref.read(
      accountScheduledPostRepositoryProvider(account).future,
    );
    return ScheduledPostListState(await repository.list());
  }
}

List<ScheduledPostSummary> _sortAndDeduplicate(
  List<ScheduledPostSummary> items,
) {
  final byID = <String, ScheduledPostSummary>{};
  for (final item in items) {
    byID[item.id] = item;
  }
  final result = byID.values.toList()
    ..sort((a, b) => a.scheduledAt.utc.compareTo(b.scheduledAt.utc));
  return result;
}
