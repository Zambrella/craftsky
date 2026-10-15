import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/feed/composer/video_publication_coordinator.dart';
import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/feed/data/video_service_client.dart';
import 'package:craftsky_app/feed/providers/composer_video_controller.dart';

/// Binds the existing publication state machine to the selected local source
/// and its captured owner. Both composer types share this transfer boundary.
VideoPublicationCoordinator composerVideoPublication({
  required ActiveAccountLease owner,
  required LocalVideoSelection selection,
  required PostApiClient api,
  required VideoServiceClient service,
  required VideoPublishOperation publish,
  required void Function(VideoPublicationProgress) onProgress,
}) => VideoPublicationCoordinator(
  operationAccountDid: owner.session.account.did.value,
  checkEligibility: api.getVideoUploadLimits,
  authorize: api.authorizeVideoUpload,
  upload:
      ({
        required authorizationHeader,
        required cancelToken,
        required bypassDeduplication,
        required onProgress,
      }) => service.upload(
        source: VideoUploadSource(
          length: selection.byteLength,
          openRead: selection.openRead,
        ),
        ownerDid: owner.session.account.did.value,
        authorizationHeader: authorizationHeader,
        bypassDeduplication: bypassDeduplication,
        cancelToken: cancelToken,
        onProgress: onProgress,
      ),
  poll: (jobId, cancelToken) =>
      service.getJobStatus(jobId, cancelToken: cancelToken),
  wait: Future<void>.delayed,
  publish: publish,
  onProgress: onProgress,
);
