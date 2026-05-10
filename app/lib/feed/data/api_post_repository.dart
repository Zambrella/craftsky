import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/feed/data/post_repository.dart';
import 'package:craftsky_app/feed/models/interaction_write_response.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/models/post_thread.dart';

/// Production [PostRepository] backed by the AppView HTTP API.
class ApiPostRepository implements PostRepository {
  const ApiPostRepository(this._api);

  final PostApiClient _api;

  @override
  Future<Post> create({required String text}) => _api.createPost(text: text);

  @override
  Future<Post> fetch(String did, String rkey) => _api.getPost(did, rkey);

  @override
  Future<void> delete(String did, String rkey) => _api.deletePost(did, rkey);

  @override
  Future<PostPage> listDirectReplies(
    String did,
    String rkey, {
    String? cursor,
    int? limit,
  }) => _api.listDirectReplies(did, rkey, cursor: cursor, limit: limit);

  @override
  Future<PostThread> thread(String did, String rkey) =>
      _api.getThread(did, rkey);

  @override
  Future<InteractionWriteResponse> like(String did, String rkey) =>
      _api.likePost(did, rkey);

  @override
  Future<void> unlike(String did, String rkey) => _api.unlikePost(did, rkey);

  @override
  Future<InteractionWriteResponse> repost(String did, String rkey) =>
      _api.repostPost(did, rkey);

  @override
  Future<void> unrepost(String did, String rkey) =>
      _api.unrepostPost(did, rkey);

  @override
  Future<PostPage> listByAuthor(
    String handleOrDid, {
    String? cursor,
    int? limit,
  }) => _api.listPostsByAuthor(handleOrDid, cursor: cursor, limit: limit);
}
