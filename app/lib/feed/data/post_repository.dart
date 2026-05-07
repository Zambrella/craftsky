import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';

/// Read/write surface the post providers depend on. The production
/// binding is `ApiPostRepository`; the test suite swaps in
/// `FakePostRepository` (under `test/feed/fakes/`) for unit tests.
abstract interface class PostRepository {
  /// POST /v1/posts. AppView returns a synthetic [Post] populated from
  /// the PDS write response.
  Future<Post> create({required String text});

  /// GET /v1/posts/{did}/{rkey}
  Future<Post> fetch(String did, String rkey);

  /// DELETE /v1/posts/{did}/{rkey}. Idempotent.
  Future<void> delete(String did, String rkey);

  /// GET /v1/profiles/@{handleOrDid}/posts — newest-first, paginated.
  Future<PostPage> listByAuthor(
    String handleOrDid, {
    String? cursor,
    int? limit,
  });
}
