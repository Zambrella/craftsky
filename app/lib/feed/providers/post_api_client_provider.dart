import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'post_api_client_provider.g.dart';

@Riverpod(keepAlive: true)
PostApiClient postApiClient(Ref ref) =>
    PostApiClient(ref.watch(dioProvider));
