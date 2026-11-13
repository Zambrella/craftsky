import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:craftsky_app/shared/browsing/browsing_provider.dart';
import 'package:craftsky_app/shared/media/blob_api_client_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'post_api_client_provider.g.dart';

@Riverpod(keepAlive: true)
PostApiClient postApiClient(Ref ref) => PostApiClient(
  ref.watch(dioProvider),
  blobClient: ref.watch(isGuestBrowsingProvider)
      ? null
      // Own the disposable blob client while member access is active.
      // ignore: riverpod_lint/only_use_keep_alive_inside_keep_alive
      : ref.watch(blobApiClientProvider),
);
