import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/profile/data/profile_repository.dart';
import 'package:craftsky_app/profile/models/profile_account_page.dart';
import 'package:craftsky_app/profile/models/profile_account_summary.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:dart_mappable/dart_mappable.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'relationship_list_provider.g.dart';
part 'relationship_list_provider.mapper.dart';

enum RelationshipListKind { muted, blocked }

@MappableClass(
  generateMethods: GenerateMethods.copy | GenerateMethods.equals,
)
class RelationshipListState with RelationshipListStateMappable {
  const RelationshipListState({
    required this.items,
    this.cursor,
    this.mutatingDids = const {},
  });

  final List<ProfileAccountSummary> items;
  final String? cursor;
  final Set<String> mutatingDids;

  bool get hasMore => cursor != null;
}

@riverpod
class RelationshipList extends _$RelationshipList {
  @override
  Future<RelationshipListState> build(RelationshipListKind kind) async {
    final repository = ref.watch(profileRepositoryProvider);
    return _fromPage(await _fetch(repository));
  }

  Future<void> loadMore() async {
    if (!state.hasValue || state.isLoading) return;
    final current = state.requireValue;
    if (!current.hasMore) return;
    final ownership = captureActiveAccountOperation(ref);

    state = const AsyncLoading<RelationshipListState>();
    final next = await AsyncValue.guard(() async {
      final repository = ref.read(profileRepositoryProvider);
      final page = await _fetch(repository, cursor: current.cursor);
      return RelationshipListState(
        items: [...current.items, ...page.items],
        cursor: page.cursor,
        mutatingDids: current.mutatingDids,
      );
    });

    if (!isActiveAccountOperationCurrent(ref, ownership)) return;
    state = next;
  }

  Future<void> reverse(ProfileAccountSummary account) async {
    final current = state.value;
    if (current == null) return;
    final did = account.did.toString();
    if (current.mutatingDids.contains(did)) return;
    final ownership = captureActiveAccountOperation(ref);

    state = AsyncData(
      current.copyWith(mutatingDids: {...current.mutatingDids, did}),
    );
    try {
      final repository = ref.read(profileRepositoryProvider);
      switch (kind) {
        case RelationshipListKind.muted:
          await repository.unmute(did);
        case RelationshipListKind.blocked:
          await repository.unblock(did);
      }
      if (!isActiveAccountOperationCurrent(ref, ownership)) return;
      final latest = state.value;
      if (latest == null) return;
      state = AsyncData(
        latest.copyWith(
          items: latest.items.where((item) => item.did != account.did).toList(),
          mutatingDids: {...latest.mutatingDids}..remove(did),
        ),
      );
    } on Object {
      if (isActiveAccountOperationCurrent(ref, ownership)) {
        final latest = state.value;
        if (latest != null) {
          state = AsyncData(
            latest.copyWith(
              mutatingDids: {...latest.mutatingDids}..remove(did),
            ),
          );
        }
      }
      rethrow;
    }
  }

  Future<ProfileAccountPage> _fetch(
    ProfileRepository repository, {
    String? cursor,
  }) => switch (kind) {
    RelationshipListKind.muted => repository.listMutedProfiles(cursor: cursor),
    RelationshipListKind.blocked => repository.listBlockedProfiles(
      cursor: cursor,
    ),
  };

  RelationshipListState _fromPage(ProfileAccountPage page) =>
      RelationshipListState(items: page.items, cursor: page.cursor);
}
