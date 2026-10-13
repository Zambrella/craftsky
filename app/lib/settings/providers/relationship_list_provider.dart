import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/profile/data/profile_repository.dart';
import 'package:craftsky_app/profile/models/profile_account_page.dart';
import 'package:craftsky_app/profile/models/profile_account_summary.dart';
import 'package:craftsky_app/profile/providers/block_profile_overlay.dart';
import 'package:craftsky_app/profile/providers/profile_relationship_provider.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/providers/toggle_block_profile_provider.dart';
import 'package:dart_mappable/dart_mappable.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'relationship_list_provider.g.dart';
part 'relationship_list_provider.mapper.dart';

enum RelationshipListKind { muted, blocked }

@MappableClass(
  generateMethods: GenerateMethods.copy | GenerateMethods.equals,
)
class RelationshipListState with RelationshipListStateMappable {
  const new({
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
    final accountOperation = ActiveAccountOperation.capture(ref);

    state = const AsyncLoading<RelationshipListState>();
    final next = await AsyncValue.guard(() async {
      final repository = ref.read(profileRepositoryProvider);
      final page = await _fetch(repository, cursor: current.cursor);
      return RelationshipListState(
        items: [...current.items, ..._visibleItems(page.items)],
        cursor: page.cursor,
        mutatingDids: current.mutatingDids,
      );
    });

    accountOperation.publish(() => state = next);
  }

  Future<void> reverse(ProfileAccountSummary account) async {
    final current = state.value;
    if (current == null) return;
    final did = account.did.toString();
    if (current.mutatingDids.contains(did)) return;
    final accountOperation = ActiveAccountOperation.capture(ref);
    final ownership = accountOperation.ownership;

    state = AsyncData(
      current.copyWith(mutatingDids: {...current.mutatingDids, did}),
    );
    try {
      final repository = ref.read(profileRepositoryProvider);
      switch (kind) {
        case RelationshipListKind.muted:
          final result = await repository.unmute(did);
          if (!accountOperation.isCurrent) return;
          if (ownership != null) {
            ref
                .read(
                  profileRelationshipProvider(
                    ownership.session.account,
                    account.did,
                  ).notifier,
                )
                .confirmExternalMutation(result);
          }
        case RelationshipListKind.blocked:
          await ref
              .read(toggleBlockProfileProvider.notifier)
              .toggle(targetDid: account.did, isBlocking: true);
          final result = ref.read(toggleBlockProfileProvider);
          if (result.hasError) throw Exception(result.error);
          if (result.value != false) return;
      }
      if (!accountOperation.isCurrent) return;
      final latest = state.value;
      if (latest == null) return;
      state = AsyncData(
        latest.copyWith(
          items: latest.items.where((item) => item.did != account.did).toList(),
          mutatingDids: {...latest.mutatingDids}..remove(did),
        ),
      );
    } on Object {
      if (accountOperation.isCurrent) {
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
      RelationshipListState(
        items: _visibleItems(page.items),
        cursor: page.cursor,
      );

  List<ProfileAccountSummary> _visibleItems(
    List<ProfileAccountSummary> items,
  ) {
    if (kind != RelationshipListKind.blocked) return items;
    return [
      for (final item in items)
        if (applyBlockAccountSummaryOverlay(ref.read, item) case final overlaid
            when overlaid.blocking)
          overlaid,
    ];
  }
}
