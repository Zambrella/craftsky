import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/timeline_page.dart';
import 'package:craftsky_app/feed/models/timeline_state.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'timeline_provider.g.dart';

const timelinePageLimit = 20;

/// Cursor-accumulating authenticated home timeline provider.
@riverpod
class Timeline extends _$Timeline {
  @override
  Future<TimelineState> build() async {
    final repo = ref.watch(postRepositoryProvider);
    final page = await repo.listTimeline(limit: timelinePageLimit);
    return TimelineState(items: _dedupe(page.items), cursor: page.cursor);
  }

  Future<void> loadMore() async {
    final current = state.value;
    if (current == null || !current.hasMore || state.isLoading) return;
    final ownership = captureActiveAccountOperation(ref);

    state = const AsyncLoading<TimelineState>();

    final next = await AsyncValue.guard(() async {
      final repo = ref.read(postRepositoryProvider);
      final page = await repo.listTimeline(
        cursor: current.cursor,
        limit: timelinePageLimit,
      );
      return TimelineState(
        items: _appendDeduped(current.items, page.items),
        cursor: page.cursor,
      );
    });

    if (!isActiveAccountOperationCurrent(ref, ownership)) return;
    state = next;
  }

  void prepend(Post post) {
    final current = state.value;
    if (current == null) return;
    final item = TimelineItem(itemKey: 'post:${post.uri}', post: post);
    if (current.items.any((existing) => existing.itemKey == item.itemKey)) {
      return;
    }
    state = AsyncData(current.copyWith(items: [item, ...current.items]));
  }

  void replace(Post post) {
    final current = state.value;
    if (current == null) return;
    state = AsyncData(
      current.copyWith(
        items: [
          for (final item in current.items)
            if (item.post.uri == post.uri) item.copyWith(post: post) else item,
        ],
      ),
    );
  }

  void removeByUri(AtUri uri) {
    final current = state.value;
    if (current == null) return;
    state = AsyncData(
      current.copyWith(
        items: current.items.where((item) => item.post.uri != uri).toList(),
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

void prependLiveTimelineCache(Ref ref, Post post) {
  if (ref.exists(timelineProvider)) {
    ref.read(timelineProvider.notifier).prepend(post);
  }
}

void updateLiveTimelineCache(Ref ref, Post post) {
  if (ref.exists(timelineProvider)) {
    ref.read(timelineProvider.notifier).replace(post);
  }
}

void removeFromLiveTimelineCache(Ref ref, AtUri uri) {
  if (ref.exists(timelineProvider)) {
    ref.read(timelineProvider.notifier).removeByUri(uri);
  }
}

List<TimelineItem> _dedupe(List<TimelineItem> items) {
  final seen = <String>{};
  return [
    for (final item in items)
      if (seen.add(item.itemKey)) item,
  ];
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
