import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/feed/data/post_repository.dart';
import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/models/interaction_write_response.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_comment_section.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/moderation/models/report_result.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';

/// Production [PostRepository] backed by the AppView HTTP API.
class ApiPostRepository implements PostRepository {
  const ApiPostRepository(this._api);

  final PostApiClient _api;

  @override
  Future<Post> create({
    required String text,
    PostReply? reply,
    Project? project,
    List<CreatePostImage>? images,
    List<Map<String, dynamic>>? facets,
  }) {
    assertProjectCreateIsTopLevel(project: project, reply: reply);
    return _api.createPost(
      text: text,
      reply: reply,
      project: project,
      images: images,
      facets: facets,
    );
  }

  @override
  Future<Post> fetch(Did did, RecordKey rkey) => _api.getPost(did, rkey);

  @override
  Future<void> delete(Did did, RecordKey rkey) => _api.deletePost(did, rkey);

  @override
  Future<ReportResult> report(
    Did did,
    RecordKey rkey,
    ReportSubmission submission,
  ) => _api.reportPost(did, rkey, submission);

  @override
  Future<ReplyPage> listCommentBranchReplies(
    Did did,
    RecordKey rkey, {
    String? cursor,
    int? limit,
  }) => _api.listCommentBranchReplies(did, rkey, cursor: cursor, limit: limit);

  @override
  Future<PostCommentSection> commentSection(
    Did did,
    RecordKey rkey, {
    String? cursor,
    CommentSort? sort,
    AtUri? focus,
    int? limit,
  }) => _api.getCommentSection(
    did,
    rkey,
    cursor: cursor,
    sort: sort,
    focus: focus,
    limit: limit,
  );

  @override
  Future<InteractionWriteResponse> like(Did did, RecordKey rkey) =>
      _api.likePost(did, rkey);

  @override
  Future<void> unlike(Did did, RecordKey rkey) => _api.unlikePost(did, rkey);

  @override
  Future<InteractionWriteResponse> repost(Did did, RecordKey rkey) =>
      _api.repostPost(did, rkey);

  @override
  Future<void> unrepost(Did did, RecordKey rkey) =>
      _api.unrepostPost(did, rkey);

  @override
  Future<PostPage> listByAuthor(
    String handleOrDid, {
    String? cursor,
    int? limit,
  }) => _api.listPostsByAuthor(handleOrDid, cursor: cursor, limit: limit);

  @override
  Future<PostPage> listProjectsByAuthor(
    String handleOrDid, {
    String? cursor,
    int? limit,
  }) => _api.listProjectsByAuthor(handleOrDid, cursor: cursor, limit: limit);

  @override
  Future<PostPage> listTimeline({String? cursor, int? limit}) =>
      _api.listTimeline(cursor: cursor, limit: limit);

  @override
  Future<PostPage> listCommentsByAuthor(
    String handleOrDid, {
    String? cursor,
    int? limit,
  }) => _api.listCommentsByAuthor(handleOrDid, cursor: cursor, limit: limit);
}
