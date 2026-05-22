import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/models/interaction_write_response.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_comment_section.dart';
import 'package:craftsky_app/feed/models/post_image_blob.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:dio/dio.dart';

/// Post-related AppView endpoints. Assumes the attached [Dio] has the
/// auth + error interceptors installed (see `dioProvider`); each call
/// is wrapped in `unwrapApi` so consumers see sealed `ApiException`
/// subtypes instead of raw `DioException`s.
class PostApiClient {
  const PostApiClient(this._dio);

  final Dio _dio;

  /// POST /v1/blobs/images — uploads prepared image bytes.
  Future<UploadedImageBlob> uploadImage({
    required List<int> bytes,
    required String mimeType,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
    CancelToken? cancelToken,
  }) => unwrapApi(() async {
    final res = await _dio.post<Map<String, dynamic>>(
      '/v1/blobs/images',
      data: bytes,
      cancelToken: cancelToken,
      onSendProgress: onSendProgress,
      onReceiveProgress: onReceiveProgress,
      options: Options(contentType: mimeType),
    );
    return UploadedImageBlob.fromMap(res.data!);
  });

  /// POST /v1/posts — text-only create, optionally as a reply.
  ///
  /// AppView returns a synthetic [Post] populated from the PDS write
  /// response; the firehose-driven indexer hasn't necessarily caught
  /// up yet by the time this returns.
  Future<Post> createPost({
    required String text,
    PostReply? reply,
    List<CreatePostImage>? images,
  }) => unwrapApi(() async {
    final res = await _dio.post<Map<String, dynamic>>(
      '/v1/posts',
      data: {
        'text': text,
        if (reply != null) 'reply': reply.toMap(),
        if (images != null)
          'images': images.map((image) => image.toMap()).toList(),
      },
    );
    return PostMapper.fromMap(res.data!);
  });

  /// GET /v1/posts/{did}/{rkey}
  Future<Post> getPost(String did, String rkey) => unwrapApi(() async {
    final res = await _dio.get<Map<String, dynamic>>('/v1/posts/$did/$rkey');
    return PostMapper.fromMap(res.data!);
  });

  /// DELETE /v1/posts/{did}/{rkey} — idempotent per AppView spec.
  Future<void> deletePost(String did, String rkey) => unwrapApi(() async {
    await _dio.delete<void>('/v1/posts/$did/$rkey');
  });

  /// GET /v1/posts/{did}/{rkey}/replies — flattened comment branch replies.
  Future<ReplyPage> listCommentBranchReplies(
    String did,
    String rkey, {
    String? cursor,
    int? limit,
  }) => unwrapApi(() async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/v1/posts/$did/$rkey/replies',
      queryParameters: {
        if (cursor != null) 'cursor': cursor,
        if (limit != null) 'limit': limit.toString(),
      },
    );
    return ReplyPageMapper.fromMap(res.data!);
  });

  /// GET /v1/posts/{did}/{rkey}/comments — root comment section.
  Future<PostCommentSection> getCommentSection(
    String did,
    String rkey, {
    String? cursor,
    CommentSort? sort,
    String? focus,
    int? limit,
  }) => unwrapApi(() async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/v1/posts/$did/$rkey/comments',
      queryParameters: {
        if (cursor != null) 'cursor': cursor,
        if (sort != null) 'sort': sort.name,
        if (focus != null) 'focus': focus,
        if (limit != null) 'limit': limit.toString(),
      },
    );
    return PostCommentSectionMapper.fromMap(res.data!);
  });

  /// POST /v1/posts/{did}/{rkey}/likes.
  Future<InteractionWriteResponse> likePost(String did, String rkey) =>
      unwrapApi(() async {
        final res = await _dio.post<Map<String, dynamic>>(
          '/v1/posts/$did/$rkey/likes',
        );
        return InteractionWriteResponseMapper.fromMap(res.data!);
      });

  /// DELETE /v1/posts/{did}/{rkey}/likes.
  Future<void> unlikePost(String did, String rkey) => unwrapApi(() async {
    await _dio.delete<void>('/v1/posts/$did/$rkey/likes');
  });

  /// POST /v1/posts/{did}/{rkey}/reposts.
  Future<InteractionWriteResponse> repostPost(String did, String rkey) =>
      unwrapApi(() async {
        final res = await _dio.post<Map<String, dynamic>>(
          '/v1/posts/$did/$rkey/reposts',
        );
        return InteractionWriteResponseMapper.fromMap(res.data!);
      });

  /// DELETE /v1/posts/{did}/{rkey}/reposts.
  Future<void> unrepostPost(String did, String rkey) => unwrapApi(() async {
    await _dio.delete<void>('/v1/posts/$did/$rkey/reposts');
  });

  /// GET /v1/profiles/@{handleOrDid}/posts — newest-first.
  ///
  /// The `@`-prefix matches the existing convention used by
  /// `ProfileApiClient.getProfile`; the AppView strips it before
  /// resolving. [limit] caps server-side at 100.
  Future<PostPage> listPostsByAuthor(
    String handleOrDid, {
    String? cursor,
    int? limit,
  }) => unwrapApi(() async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/v1/profiles/@$handleOrDid/posts',
      queryParameters: {
        if (cursor != null) 'cursor': cursor,
        if (limit != null) 'limit': limit.toString(),
      },
    );
    return PostPageMapper.fromMap(res.data!);
  });

  /// GET /v1/profiles/@{handleOrDid}/comments — newest-first.
  Future<PostPage> listCommentsByAuthor(
    String handleOrDid, {
    String? cursor,
    int? limit,
  }) => unwrapApi(() async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/v1/profiles/@$handleOrDid/comments',
      queryParameters: {
        if (cursor != null) 'cursor': cursor,
        if (limit != null) 'limit': limit.toString(),
      },
    );
    return PostPageMapper.fromMap(res.data!);
  });
}
