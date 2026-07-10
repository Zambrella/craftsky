import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/models/interaction_write_response.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_comment_section.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/models/timeline_page.dart';
import 'package:craftsky_app/moderation/models/report_result.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';

/// Read/write surface the post providers depend on. The production
/// binding is `ApiPostRepository`; the test suite swaps in
/// `FakePostRepository` (under `test/feed/fakes/`) for unit tests.
abstract interface class PostRepository {
  /// POST /v1/posts. AppView returns a synthetic [Post] populated from
  /// the PDS write response.
  Future<Post> create({
    required String text,
    PostReply? reply,
    PostRef? quote,
    Project? project,
    List<CreatePostImage>? images,
    List<Map<String, dynamic>>? facets,
  });

  /// GET /v1/posts/{did}/{rkey}
  Future<Post> fetch(Did did, RecordKey rkey);

  /// DELETE /v1/posts/{did}/{rkey}. Idempotent.
  Future<void> delete(Did did, RecordKey rkey);

  /// POST /v1/posts/{did}/{rkey}/reports.
  Future<ReportResult> report(
    Did did,
    RecordKey rkey,
    ReportSubmission submission,
  );

  /// GET /v1/posts/{did}/{rkey}/replies — comment branch replies.
  Future<ReplyPage> listCommentBranchReplies(
    Did did,
    RecordKey rkey, {
    String? cursor,
    int? limit,
  });

  /// GET /v1/posts/{did}/{rkey}/comments — root comment section.
  Future<PostCommentSection> commentSection(
    Did did,
    RecordKey rkey, {
    String? cursor,
    CommentSort? sort,
    AtUri? focus,
    int? limit,
  });

  /// POST /v1/posts/{did}/{rkey}/likes.
  Future<InteractionWriteResponse> like(Did did, RecordKey rkey);

  /// DELETE /v1/posts/{did}/{rkey}/likes.
  Future<void> unlike(Did did, RecordKey rkey);

  /// POST /v1/posts/{did}/{rkey}/reposts.
  Future<InteractionWriteResponse> repost(Did did, RecordKey rkey);

  /// DELETE /v1/posts/{did}/{rkey}/reposts.
  Future<void> unrepost(Did did, RecordKey rkey);

  /// GET /v1/profiles/@{handleOrDid}/posts — newest-first, paginated.
  Future<PostPage> listByAuthor(
    String handleOrDid, {
    String? cursor,
    int? limit,
  });

  /// GET /v1/profiles/@{handleOrDid}/projects — newest-first, paginated.
  Future<PostPage> listProjectsByAuthor(
    String handleOrDid, {
    String? cursor,
    int? limit,
  });

  /// GET /v1/feed/timeline — authenticated home timeline, paginated.
  Future<TimelinePage> listTimeline({String? cursor, int? limit});

  /// GET /v1/profiles/@{handleOrDid}/comments — newest-first, paginated.
  Future<PostPage> listCommentsByAuthor(
    String handleOrDid, {
    String? cursor,
    int? limit,
  });
}
