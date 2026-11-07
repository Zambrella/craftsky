import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/feed/models/timeline_page.dart';
import 'package:craftsky_app/feed/models/timeline_state.dart';
import 'package:craftsky_app/feed/providers/like_post_overlay.dart';
import 'package:craftsky_app/feed/providers/post_record_overlay.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/profile/providers/block_profile_overlay.dart';
import 'package:craftsky_app/profile/providers/follow_profile_overlay.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'timeline_provider.g.dart';

const timelinePageLimit = 20;

/// Cursor-accumulating authenticated home timeline provider.
@riverpod
class Timeline extends _$Timeline {
  @override
  Future<TimelineState> build() async {
    listenForPostEdits(ref, () {
      final current = state.value;
      if (current == null) return;
      state = AsyncData(
        current.copyWith(
          items: [
            for (final item in current.items)
              if (applyPostRecordOverlay(ref, item.post, reconcile: false)
                  case final post?)
                item.copyWith(post: post),
          ],
        ),
      );
    });
    ref.watch(activeContentLanguagePolicyProvider);
    final accountOperation = ActiveAccountOperation.capture(ref);
    final repo = ref.watch(postRepositoryProvider);
    final page = await repo.listTimeline(limit: timelinePageLimit);
    if (!accountOperation.isCurrent) {
      throw const ApiCanceled();
    }
    return TimelineState(
      items: _dedupe(_applyTimelineOverlays(ref, page.items)),
      cursor: page.cursor,
    );
  }

  Future<void> loadMore() async {
    final current = state.value;
    if (current == null || !current.hasMore || state.isLoading) return;
    final accountOperation = ActiveAccountOperation.capture(ref);

    state = const AsyncLoading<TimelineState>();

    final next = await AsyncValue.guard(() async {
      final repo = ref.read(postRepositoryProvider);
      final page = await repo.listTimeline(
        cursor: current.cursor,
        limit: timelinePageLimit,
      );
      if (!accountOperation.isCurrent) {
        throw const ApiCanceled();
      }
      return TimelineState(
        items: _appendDeduped(
          current.items,
          _applyTimelineOverlays(ref, page.items),
        ),
        cursor: page.cursor,
      );
    });

    accountOperation.publish(
      () => state = next.whenData(
        (value) => value.copyWith(
          items: [
            for (final item in value.items)
              if (applyPostRecordOverlay(ref, item.post, reconcile: false)
                  case final post?)
                item.copyWith(post: post),
          ],
        ),
      ),
    );
  }

  int suppressActor(String did) {
    final current = state.value;
    if (current == null) return 0;
    final retained = current.items
        .where(
          (item) =>
              item.post.author.did.toString() != did &&
              item.reason?.by.did.toString() != did,
        )
        .toList();
    final removed = current.items.length - retained.length;
    if (removed > 0) {
      state = AsyncData(current.copyWith(items: retained));
    }
    return removed;
  }
}

/// Retained chronological feed of indexed second-degree connections.
@riverpod
class ExtendedNetwork extends _$ExtendedNetwork {
  @override
  Future<TimelineState> build() async {
    listenForPostEdits(ref, () {
      final current = state.value;
      if (current == null) return;
      state = AsyncData(
        current.copyWith(
          items: [
            for (final item in current.items)
              if (applyPostRecordOverlay(ref, item.post, reconcile: false)
                  case final post?)
                item.copyWith(post: post),
          ],
        ),
      );
    });
    ref
      ..watch(activeContentLanguagePolicyProvider)
      ..watch(followProfileProjectionRevisionProvider);
    final accountOperation = ActiveAccountOperation.capture(ref);
    final repo = ref.watch(postRepositoryProvider);
    final page = await repo.listExtendedNetwork(limit: timelinePageLimit);
    if (!accountOperation.isCurrent) {
      throw const ApiCanceled();
    }
    return TimelineState(
      items: _dedupe(_applyExtendedNetworkOverlays(ref, page.items)),
      cursor: page.cursor,
    );
  }

  Future<void> loadMore() async {
    final current = state.value;
    if (current == null || !current.hasMore || state.isLoading) return;
    final accountOperation = ActiveAccountOperation.capture(ref);

    state = const AsyncLoading<TimelineState>();

    final next = await AsyncValue.guard(() async {
      final repo = ref.read(postRepositoryProvider);
      final page = await repo.listExtendedNetwork(
        cursor: current.cursor,
        limit: timelinePageLimit,
      );
      if (!accountOperation.isCurrent) {
        throw const ApiCanceled();
      }
      return TimelineState(
        items: _appendDeduped(
          current.items,
          _applyExtendedNetworkOverlays(ref, page.items),
        ),
        cursor: page.cursor,
      );
    });

    accountOperation.publish(
      () => state = next.whenData(
        (value) => value.copyWith(
          items: [
            for (final item in value.items)
              if (applyPostRecordOverlay(ref, item.post, reconcile: false)
                  case final post?)
                item.copyWith(post: post),
          ],
        ),
      ),
    );
  }

  int suppressActor(String did) {
    final current = state.value;
    if (current == null) return 0;
    final retained = current.items
        .where(
          (item) =>
              item.post.author.did.toString() != did &&
              item.reason?.by.did.toString() != did &&
              item.followedBy?.did.toString() != did,
        )
        .toList();
    final removed = current.items.length - retained.length;
    if (removed > 0) {
      state = AsyncData(current.copyWith(items: retained));
    }
    return removed;
  }
}

List<TimelineItem> _applyExtendedNetworkOverlays(
  Ref ref,
  List<TimelineItem> items,
) => _applyTimelineOverlays(
  ref,
  [
    for (final item in items)
      if (acceptedFollowProfileIntent(ref, item.post.author.did) != true &&
          (item.followedBy == null ||
              acceptedFollowProfileIntent(ref, item.followedBy!.did) != false))
        item,
  ],
  includePublications: false,
  listKey: 'extended-network',
);

List<TimelineItem> _dedupe(List<TimelineItem> items) {
  final seen = <String>{};
  return [
    for (final item in items)
      if (seen.add(item.itemKey)) item,
  ];
}

List<TimelineItem> _applyTimelineOverlays(
  Ref ref,
  List<TimelineItem> items, {
  bool includePublications = true,
  String listKey = 'timeline',
}) {
  final visible = [
    for (final item in items)
      if (!isLogicallyBlocking(
            ref.read,
            item.post.author.did,
            authoritativeBlocking: item.post.author.blocking ?? false,
          ) &&
          (item.reason == null ||
              !isLogicallyBlocking(
                ref.read,
                item.reason!.by.did,
                authoritativeBlocking: item.reason!.by.blocking ?? false,
              )) &&
          (item.followedBy == null ||
              !isLogicallyBlocking(
                ref.read,
                item.followedBy!.did,
                authoritativeBlocking: item.followedBy!.blocking ?? false,
              )))
        item,
  ];
  observePostListMembership(ref, items.map((item) => item.post), listKey);
  final result = <TimelineItem>[];
  final seen = <AtUri>{};
  for (final item in visible) {
    final post = applyPostRecordOverlay(ref, item.post);
    if (post == null) continue;
    seen.add(post.uri);
    result.add(item.copyWith(post: applyPostInteractionOverlays(ref, post)));
  }
  if (!includePublications) return result;
  for (final post in optimisticPostRecordOverlays(ref, listKey: listKey)) {
    if (post.reply == null && seen.add(post.uri)) {
      recordPublicationList(ref, post, listKey);
      result.insert(
        0,
        TimelineItem(
          itemKey: 'post:${post.uri}',
          post: applyPostInteractionOverlays(ref, post),
        ),
      );
    }
  }
  return result;
}

List<TimelineItem> _appendDeduped(
  List<TimelineItem> current,
  List<TimelineItem> next,
) {
  final seen = current.map((item) => item.itemKey).toSet();
  return [
    ...current,
    for (final item in next)
      if (seen.add(item.itemKey)) item,
  ];
}
