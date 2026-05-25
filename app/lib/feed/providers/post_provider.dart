import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'post_provider.g.dart';

/// Single-post read by `(did, rkey)`. No UI consumer in v1; exists for
/// future routes (deep-link share, thread page).
@riverpod
Future<Post> post(Ref ref, Did did, RecordKey rkey) =>
    ref.watch(postRepositoryProvider).fetch(did, rkey);
