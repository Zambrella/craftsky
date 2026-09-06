import 'package:craftsky_app/feed/data/video_service_client.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'video_service_client_provider.g.dart';

@Riverpod(keepAlive: true)
VideoServiceConfiguration videoServiceConfiguration(Ref ref) =>
    VideoServiceConfiguration.bluesky();

@Riverpod(keepAlive: true)
VideoServiceClient videoServiceClient(Ref ref) =>
    VideoServiceClient.fromConfiguration(
      ref.watch(videoServiceConfigurationProvider),
    );
