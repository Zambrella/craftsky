import 'package:craftsky_app/feed/data/api_post_repository.dart';
import 'package:craftsky_app/feed/data/post_repository.dart';
import 'package:craftsky_app/feed/providers/post_api_client_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'post_repository_provider.g.dart';

@Riverpod(keepAlive: true)
PostRepository postRepository(Ref ref) =>
    ApiPostRepository(ref.watch(postApiClientProvider));
