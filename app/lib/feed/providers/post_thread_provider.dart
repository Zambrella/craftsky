import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/models/post_thread.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'post_thread_provider.g.dart';

@riverpod
Future<PostPage> directReplies(
  Ref ref,
  String did,
  String rkey, {
  String? cursor,
  int? limit,
}) => ref
    .watch(postRepositoryProvider)
    .listDirectReplies(did, rkey, cursor: cursor, limit: limit);

@riverpod
Future<PostThread> postThread(Ref ref, String did, String rkey) =>
    ref.watch(postRepositoryProvider).thread(did, rkey);
