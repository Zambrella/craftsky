import 'package:craftsky_app/feed/data/post_repository.dart';
import 'package:craftsky_app/feed/models/interaction_write_response.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/models/post_thread.dart';

/// Programmable [PostRepository] for unit tests. Each method delegates
/// to an optional callback the test sets up; unstubbed methods complete
/// with `UnimplementedError` so a test that misses a dependency fails
/// loudly instead of silently no-op'ing.
///
/// Mirrors `FakeProfileRepository`.
///
/// Usage:
///
/// ```dart
/// final repo = FakePostRepository(
///   onListByAuthor: (id, {cursor, limit}) async => PostPage(items: [...]),
/// );
/// final container = ProviderContainer.test(
///   overrides: [postRepositoryProvider.overrideWithValue(repo)],
/// );
/// ```
class FakePostRepository implements PostRepository {
  FakePostRepository({
    this.onCreate,
    this.onFetch,
    this.onDelete,
    this.onListDirectReplies,
    this.onThread,
    this.onLike,
    this.onUnlike,
    this.onRepost,
    this.onUnrepost,
    this.onListByAuthor,
  });

  final Future<Post> Function({required String text, PostReply? reply})?
  onCreate;
  final Future<Post> Function(String did, String rkey)? onFetch;
  final Future<void> Function(String did, String rkey)? onDelete;
  final Future<PostPage> Function(
    String did,
    String rkey, {
    String? cursor,
    int? limit,
  })?
  onListDirectReplies;
  final Future<PostThread> Function(String did, String rkey)? onThread;
  final Future<InteractionWriteResponse> Function(String did, String rkey)?
  onLike;
  final Future<void> Function(String did, String rkey)? onUnlike;
  final Future<InteractionWriteResponse> Function(String did, String rkey)?
  onRepost;
  final Future<void> Function(String did, String rkey)? onUnrepost;
  final Future<PostPage> Function(
    String handleOrDid, {
    String? cursor,
    int? limit,
  })?
  onListByAuthor;

  @override
  Future<Post> create({required String text, PostReply? reply}) =>
      onCreate?.call(text: text, reply: reply) ??
      Future<Post>.error(UnimplementedError('create not stubbed'));

  @override
  Future<Post> fetch(String did, String rkey) =>
      onFetch?.call(did, rkey) ??
      Future<Post>.error(UnimplementedError('fetch not stubbed'));

  @override
  Future<void> delete(String did, String rkey) =>
      onDelete?.call(did, rkey) ??
      Future<void>.error(UnimplementedError('delete not stubbed'));

  @override
  Future<PostPage> listDirectReplies(
    String did,
    String rkey, {
    String? cursor,
    int? limit,
  }) =>
      onListDirectReplies?.call(did, rkey, cursor: cursor, limit: limit) ??
      Future<PostPage>.error(
        UnimplementedError('listDirectReplies not stubbed'),
      );

  @override
  Future<PostThread> thread(String did, String rkey) =>
      onThread?.call(did, rkey) ??
      Future<PostThread>.error(UnimplementedError('thread not stubbed'));

  @override
  Future<InteractionWriteResponse> like(String did, String rkey) =>
      onLike?.call(did, rkey) ??
      Future<InteractionWriteResponse>.error(
        UnimplementedError('like not stubbed'),
      );

  @override
  Future<void> unlike(String did, String rkey) =>
      onUnlike?.call(did, rkey) ??
      Future<void>.error(UnimplementedError('unlike not stubbed'));

  @override
  Future<InteractionWriteResponse> repost(String did, String rkey) =>
      onRepost?.call(did, rkey) ??
      Future<InteractionWriteResponse>.error(
        UnimplementedError('repost not stubbed'),
      );

  @override
  Future<void> unrepost(String did, String rkey) =>
      onUnrepost?.call(did, rkey) ??
      Future<void>.error(UnimplementedError('unrepost not stubbed'));

  @override
  Future<PostPage> listByAuthor(
    String handleOrDid, {
    String? cursor,
    int? limit,
  }) =>
      onListByAuthor?.call(handleOrDid, cursor: cursor, limit: limit) ??
      Future<PostPage>.error(UnimplementedError('listByAuthor not stubbed'));
}
