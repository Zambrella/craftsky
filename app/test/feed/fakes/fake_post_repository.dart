import 'package:craftsky_app/feed/data/post_repository.dart';
import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/models/interaction_write_response.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_comment_section.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/moderation/models/report_result.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';

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
    this.onReport,
    this.onListCommentBranchReplies,
    this.onCommentSection,
    this.onLike,
    this.onUnlike,
    this.onRepost,
    this.onUnrepost,
    this.onListByAuthor,
    this.onListTimeline,
    this.onListCommentsByAuthor,
  });

  final Future<Post> Function({
    required String text,
    PostReply? reply,
    List<CreatePostImage>? images,
  })?
  onCreate;
  final Future<Post> Function(Did did, RecordKey rkey)? onFetch;
  final Future<void> Function(Did did, RecordKey rkey)? onDelete;
  final Future<ReportResult> Function(
    Did did,
    RecordKey rkey,
    ReportSubmission submission,
  )?
  onReport;
  final Future<ReplyPage> Function(
    Did did,
    RecordKey rkey, {
    String? cursor,
    int? limit,
  })?
  onListCommentBranchReplies;
  final Future<PostCommentSection> Function(
    Did did,
    RecordKey rkey, {
    String? cursor,
    CommentSort? sort,
    AtUri? focus,
    int? limit,
  })?
  onCommentSection;
  final Future<InteractionWriteResponse> Function(Did did, RecordKey rkey)?
  onLike;
  final Future<void> Function(Did did, RecordKey rkey)? onUnlike;
  final Future<InteractionWriteResponse> Function(Did did, RecordKey rkey)?
  onRepost;
  final Future<void> Function(Did did, RecordKey rkey)? onUnrepost;
  final Future<PostPage> Function(
    String handleOrDid, {
    String? cursor,
    int? limit,
  })?
  onListByAuthor;
  final Future<PostPage> Function({String? cursor, int? limit})? onListTimeline;
  final Future<PostPage> Function(
    String handleOrDid, {
    String? cursor,
    int? limit,
  })?
  onListCommentsByAuthor;

  @override
  Future<Post> create({
    required String text,
    PostReply? reply,
    List<CreatePostImage>? images,
  }) =>
      onCreate?.call(text: text, reply: reply, images: images) ??
      Future<Post>.error(UnimplementedError('create not stubbed'));

  @override
  Future<Post> fetch(Did did, RecordKey rkey) =>
      onFetch?.call(did, rkey) ??
      Future<Post>.error(UnimplementedError('fetch not stubbed'));

  @override
  Future<void> delete(Did did, RecordKey rkey) =>
      onDelete?.call(did, rkey) ??
      Future<void>.error(UnimplementedError('delete not stubbed'));

  @override
  Future<ReportResult> report(
    Did did,
    RecordKey rkey,
    ReportSubmission submission,
  ) =>
      onReport?.call(did, rkey, submission) ??
      Future<ReportResult>.error(UnimplementedError('report not stubbed'));

  @override
  Future<ReplyPage> listCommentBranchReplies(
    Did did,
    RecordKey rkey, {
    String? cursor,
    int? limit,
  }) =>
      onListCommentBranchReplies?.call(
        did,
        rkey,
        cursor: cursor,
        limit: limit,
      ) ??
      Future<ReplyPage>.error(
        UnimplementedError('listCommentBranchReplies not stubbed'),
      );

  @override
  Future<PostCommentSection> commentSection(
    Did did,
    RecordKey rkey, {
    String? cursor,
    CommentSort? sort,
    AtUri? focus,
    int? limit,
  }) =>
      onCommentSection?.call(
        did,
        rkey,
        cursor: cursor,
        sort: sort,
        focus: focus,
        limit: limit,
      ) ??
      Future<PostCommentSection>.error(
        UnimplementedError('commentSection not stubbed'),
      );

  @override
  Future<InteractionWriteResponse> like(Did did, RecordKey rkey) =>
      onLike?.call(did, rkey) ??
      Future<InteractionWriteResponse>.error(
        UnimplementedError('like not stubbed'),
      );

  @override
  Future<void> unlike(Did did, RecordKey rkey) =>
      onUnlike?.call(did, rkey) ??
      Future<void>.error(UnimplementedError('unlike not stubbed'));

  @override
  Future<InteractionWriteResponse> repost(Did did, RecordKey rkey) =>
      onRepost?.call(did, rkey) ??
      Future<InteractionWriteResponse>.error(
        UnimplementedError('repost not stubbed'),
      );

  @override
  Future<void> unrepost(Did did, RecordKey rkey) =>
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

  @override
  Future<PostPage> listTimeline({String? cursor, int? limit}) =>
      onListTimeline?.call(cursor: cursor, limit: limit) ??
      Future<PostPage>.error(UnimplementedError('listTimeline not stubbed'));

  @override
  Future<PostPage> listCommentsByAuthor(
    String handleOrDid, {
    String? cursor,
    int? limit,
  }) =>
      onListCommentsByAuthor?.call(handleOrDid, cursor: cursor, limit: limit) ??
      Future<PostPage>.error(
        UnimplementedError('listCommentsByAuthor not stubbed'),
      );
}
