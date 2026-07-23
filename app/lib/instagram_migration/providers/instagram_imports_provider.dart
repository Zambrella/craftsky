import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/instagram_migration/data/instagram_migration_repository.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_migration_repository_provider.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_suggestions_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'instagram_imports_provider.g.dart';

@riverpod
class InstagramImports extends _$InstagramImports {
  @override
  Future<InstagramImportPage> build(ActiveAccountLease lease) async {
    final repository = await ref.watch(
      instagramMigrationRepositoryProvider(lease).future,
    );
    ensureInstagramOperationCurrent(ref, lease);
    final page = await repository.listImports();
    ensureInstagramOperationCurrent(ref, lease);
    return page;
  }

  Future<void> refresh() async {
    state = const AsyncLoading<InstagramImportPage>();
    try {
      final repository = await _repository();
      final page = await repository.listImports();
      ensureInstagramOperationCurrent(ref, lease);
      state = AsyncData(page);
    } on InstagramOperationDiscarded {
      return;
    } on Object catch (error, stackTrace) {
      if (!_isCurrent) return;
      state = AsyncError<InstagramImportPage>(error, stackTrace);
    }
  }

  Future<bool> loadMore() async {
    final current = state.value;
    if (current?.cursor == null) return false;
    try {
      final repository = await _repository();
      final next = await repository.listImports(cursor: current!.cursor);
      ensureInstagramOperationCurrent(ref, lease);
      final seen = current.items.map((item) => item.importId).toSet();
      state = AsyncData(
        InstagramImportPage(
          items: [
            ...current.items,
            ...next.items.where((item) => seen.add(item.importId)),
          ],
          cursor: next.cursor,
        ),
      );
      return true;
    } on InstagramOperationDiscarded {
      return false;
    } on Object {
      return false;
    }
  }

  Future<InstagramImportCreateResult?> create(
    InstagramImportRequest request,
  ) async {
    try {
      final repository = await _repository();
      final result = await repository.createImport(request);
      ensureInstagramOperationCurrent(ref, lease);
      final current = state.value;
      state = AsyncData(
        InstagramImportPage(
          items: [result.import, ...?current?.items],
          cursor: current?.cursor,
        ),
      );
      ref.invalidate(instagramSuggestionsProvider(lease));
      return result;
    } on InstagramOperationDiscarded {
      return null;
    } on Object {
      return null;
    }
  }

  Future<bool> reactivate(String importId) => _update(
    importId,
    const InstagramImportPatch(reactivate: true),
  );

  Future<bool> _update(String importId, InstagramImportPatch patch) async {
    try {
      final repository = await _repository();
      final updated = await repository.updateImport(importId, patch);
      ensureInstagramOperationCurrent(ref, lease);
      final current = state.value;
      if (current != null) {
        state = AsyncData(
          InstagramImportPage(
            items: [
              for (final item in current.items)
                if (item.importId == importId) updated else item,
            ],
            cursor: current.cursor,
          ),
        );
      }
      ref.invalidate(instagramSuggestionsProvider(lease));
      return true;
    } on InstagramOperationDiscarded {
      return false;
    } on Object {
      return false;
    }
  }

  Future<bool> delete(String importId) async {
    try {
      final repository = await _repository();
      await repository.deleteImport(importId);
      ensureInstagramOperationCurrent(ref, lease);
      final current = state.value;
      if (current != null) {
        state = AsyncData(
          InstagramImportPage(
            items: current.items
                .where((item) => item.importId != importId)
                .toList(growable: false),
            cursor: current.cursor,
          ),
        );
      }
      ref.invalidate(instagramSuggestionsProvider(lease));
      return true;
    } on InstagramOperationDiscarded {
      return false;
    } on Object {
      return false;
    }
  }

  Future<InstagramMigrationRepository> _repository() async {
    final repository = await ref.read(
      instagramMigrationRepositoryProvider(lease).future,
    );
    ensureInstagramOperationCurrent(ref, lease);
    return repository;
  }

  bool get _isCurrent {
    if (!ref.mounted) return false;
    try {
      ensureInstagramOperationCurrent(ref, lease);
      return true;
    } on InstagramOperationDiscarded {
      return false;
    }
  }
}
