import 'package:craftsky_app/feed/models/interaction_write_response.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_comment_section.dart';
import 'package:craftsky_app/feed/models/post_page.dart';

/// Read/write surface the post providers depend on. The production
/// binding is `ApiPostRepository`; the test suite swaps in
/// `FakePostRepository` (under `test/feed/fakes/`) for unit tests.
abstract interface class PostRepository {
  /// POST /v1/posts. AppView returns a synthetic [Post] populated from
  /// the PDS write response.
  Future<Post> create({required String text, PostReply? reply});

  /// GET /v1/posts/{did}/{rkey}
  Future<Post> fetch(String did, String rkey);

  /// DELETE /v1/posts/{did}/{rkey}. Idempotent.
  Future<void> delete(String did, String rkey);

  /// GET /v1/posts/{did}/{rkey}/replies — comment branch replies.
  Future<ReplyPage> listCommentBranchReplies(
    String did,
    String rkey, {
    String? cursor,
    int? limit,
  });

  /// GET /v1/posts/{did}/{rkey}/comments — root comment section.
  Future<PostCommentSection> commentSection(
    String did,
    String rkey, {
    String? cursor,
    CommentSort? sort,
    String? focus,
    int? limit,
  });

  /// POST /v1/posts/{did}/{rkey}/likes.
  Future<InteractionWriteResponse> like(String did, String rkey);

  /// DELETE /v1/posts/{did}/{rkey}/likes.
  Future<void> unlike(String did, String rkey);

  /// POST /v1/posts/{did}/{rkey}/reposts.
  Future<InteractionWriteResponse> repost(String did, String rkey);

  /// DELETE /v1/posts/{did}/{rkey}/reposts.
  Future<void> unrepost(String did, String rkey);

  /// GET /v1/profiles/@{handleOrDid}/posts — newest-first, paginated.
  Future<PostPage> listByAuthor(
    String handleOrDid, {
    String? cursor,
    int? limit,
  });
}
