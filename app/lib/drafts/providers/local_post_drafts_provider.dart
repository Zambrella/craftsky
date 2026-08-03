import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/drafts/providers/local_post_draft_repository_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'local_post_drafts_provider.g.dart';

final class LocalPostDraftListState {
  LocalPostDraftListState(List<LocalPostDraft> items)
    : items = List.unmodifiable(_sortAndDeduplicate(items));

  final List<LocalPostDraft> items;

  @override
  String toString() => 'LocalPostDraftListState(<redacted>)';
}

@riverpod
class LocalPostDrafts extends _$LocalPostDrafts {
  @override
  Future<LocalPostDraftListState> build(AccountKey account) async {
    final ownership = _captureOwnership();
    final result = await _load();
    _assertCurrent(ownership);
    return result;
  }

  Future<void> refresh() async {
    final ownership = _captureOwnership();
    final result = await _load();
    if (!isActiveAccountOperationCurrent(ref, ownership)) return;
    state = AsyncData(result);
  }

  Future<void> delete(String draftId) async {
    final ownership = _captureOwnership();
    final repository = await ref.read(
      accountLocalPostDraftRepositoryProvider(account).future,
    );
    _assertCurrent(ownership);
    await repository.delete(draftId);
    _assertCurrent(ownership);
    final result = await _load();
    if (!isActiveAccountOperationCurrent(ref, ownership)) return;
    state = AsyncData(result);
  }

  ActiveAccountLease? _captureOwnership() {
    final ownership = captureActiveAccountOperation(ref);
    if (ownership != null && ownership.session.account != account) {
      throw StateError('local-draft account changed');
    }
    return ownership;
  }

  void _assertCurrent(ActiveAccountLease? ownership) {
    if (!isActiveAccountOperationCurrent(ref, ownership)) {
      throw StateError('local-draft account changed');
    }
  }

  Future<LocalPostDraftListState> _load() async {
    final repository = await ref.read(
      accountLocalPostDraftRepositoryProvider(account).future,
    );
    return LocalPostDraftListState(await repository.list());
  }
}

List<LocalPostDraft> _sortAndDeduplicate(List<LocalPostDraft> items) {
  final byId = <String, LocalPostDraft>{};
  for (final item in items) {
    byId[item.id] = item;
  }
  final result = byId.values.toList()
    ..sort((left, right) {
      final updated = right.updatedAt.compareTo(left.updatedAt);
      return updated != 0 ? updated : left.id.compareTo(right.id);
    });
  return result;
}
