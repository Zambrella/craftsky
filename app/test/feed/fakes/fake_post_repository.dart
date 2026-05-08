import 'package:craftsky_app/feed/data/post_repository.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';

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
    this.onListByAuthor,
  });

  final Future<Post> Function({required String text})? onCreate;
  final Future<Post> Function(String did, String rkey)? onFetch;
  final Future<void> Function(String did, String rkey)? onDelete;
  final Future<PostPage> Function(
    String handleOrDid, {
    String? cursor,
    int? limit,
  })?
  onListByAuthor;

  @override
  Future<Post> create({required String text}) =>
      onCreate?.call(text: text) ??
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
  Future<PostPage> listByAuthor(
    String handleOrDid, {
    String? cursor,
    int? limit,
  }) =>
      onListByAuthor?.call(handleOrDid, cursor: cursor, limit: limit) ??
      Future<PostPage>.error(UnimplementedError('listByAuthor not stubbed'));
}
