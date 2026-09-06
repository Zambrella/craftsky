import 'package:craftsky_app/feed/data/video_service_client.dart';
import 'package:craftsky_app/feed/providers/video_service_client_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('IR-008 configuration accepts only the exact approved endpoint', () {
    expect(
      () => VideoServiceConfiguration(
        uploadEndpoint: Uri.parse(
          'https://video.bsky.app/xrpc/app.bsky.video.getJobStatus',
        ),
      ),
      throwsArgumentError,
    );

    final configuration = VideoServiceConfiguration.bluesky();
    expect(
      configuration.uploadEndpoint,
      Uri.parse('https://video.bsky.app/xrpc/app.bsky.video.uploadVideo'),
    );
  });

  test(
    'IR-008 isolated client is constructed through configuration provider',
    () {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      expect(
        container.read(videoServiceClientProvider),
        isA<VideoServiceClient>(),
      );
      expect(
        container.read(videoServiceConfigurationProvider).uploadEndpoint.path,
        '/xrpc/app.bsky.video.uploadVideo',
      );
    },
  );
}
