import 'package:craftsky_app/feed/models/post.dart';
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

    if (!ref.mounted) return;
    state = next;
  }

  void prepend(Post post) {
    final current = state.value;
    if (current == null) return;
    if (current.items.any((item) => item.uri == post.uri)) return;
    state = AsyncData(current.copyWith(items: [post, ...current.items]));
  }

  void replace(Post post) {
    final current = state.value;
    if (current == null) return;
    state = AsyncData(
      current.copyWith(
        items: [
          for (final item in current.items)
            if (item.uri == post.uri) post else item,
        ],
      ),
    );
  }

  void removeByUri(AtUri uri) {
    final current = state.value;
    if (current == null) return;
    state = AsyncData(
      current.copyWith(
        items: current.items.where((post) => post.uri != uri).toList(),
      ),
    );
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

List<Post> _dedupe(List<Post> posts) {
  final seen = <String>{};
  return [
    for (final post in posts)
      if (seen.add(post.uri)) post,
  ];
}

List<Post> _appendDeduped(List<Post> current, List<Post> next) {
  final seen = current.map((post) => post.uri).toSet();
  return [
    ...current,
    for (final post in next)
      if (seen.add(post.uri)) post,
  ];
}
