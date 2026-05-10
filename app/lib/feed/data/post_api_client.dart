import 'package:craftsky_app/feed/models/interaction_write_response.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/models/post_thread.dart';
import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:dio/dio.dart';

/// Post-related AppView endpoints. Assumes the attached [Dio] has the
/// auth + error interceptors installed (see `dioProvider`); each call
/// is wrapped in `unwrapApi` so consumers see sealed `ApiException`
/// subtypes instead of raw `DioException`s.
class PostApiClient {
  const PostApiClient(this._dio);

  final Dio _dio;

  /// POST /v1/posts — text-only create.
  ///
  /// AppView returns a synthetic [Post] populated from the PDS write
  /// response; the firehose-driven indexer hasn't necessarily caught
  /// up yet by the time this returns.
  Future<Post> createPost({required String text}) => unwrapApi(() async {
    final res = await _dio.post<Map<String, dynamic>>(
      '/v1/posts',
      data: {'text': text},
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

  /// GET /v1/posts/{did}/{rkey}/replies — direct replies, oldest-first.
  Future<PostPage> listDirectReplies(
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
    return PostPageMapper.fromMap(res.data!);
  });

  /// GET /v1/posts/{did}/{rkey}/thread — nested reply thread.
  Future<PostThread> getThread(String did, String rkey) => unwrapApi(() async {
    final res = await _dio.get<Map<String, dynamic>>(
      '/v1/posts/$did/$rkey/thread',
    );
    return PostThreadMapper.fromMap(res.data!);
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
}
