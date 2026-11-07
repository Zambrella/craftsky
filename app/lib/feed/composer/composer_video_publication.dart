import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/feed/composer/video_publication_coordinator.dart';
import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/feed/data/video_service_client.dart';
import 'package:craftsky_app/feed/models/video_upload_limits.dart';
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
}) {
  VideoUploadAuthorization? uploadAuthorization;
  var usesBunny = false;
  String? bunnyJobId;
  return VideoPublicationCoordinator(
    operationAccountDid: owner.session.account.did.value,
    checkEligibility: api.getVideoUploadLimits,
    authorize: () async {
      uploadAuthorization = await api.authorizeVideoUpload();
      usesBunny = uploadAuthorization!.bunny != null;
      bunnyJobId = uploadAuthorization!.bunny?.jobId;
      return uploadAuthorization!;
    },
    upload:
        ({
          required authorizationHeader,
          required cancelToken,
          required bypassDeduplication,
          required onProgress,
        }) async {
          try {
            return await service.upload(
              source: VideoUploadSource(
                length: selection.byteLength,
                mimeType: selection.mimeType,
                openRead: selection.openRead,
              ),
              ownerDid: owner.session.account.did.value,
              authorizationHeader: authorizationHeader,
              bunny: uploadAuthorization?.bunny,
              bypassDeduplication: bypassDeduplication,
              cancelToken: cancelToken,
              onProgress: onProgress,
            );
          } finally {
            uploadAuthorization = null;
          }
        },
    poll: (jobId, cancelToken) => usesBunny
        ? api.getBunnyVideoStatus(jobId, cancelToken: cancelToken)
        : service.getJobStatus(jobId, cancelToken: cancelToken),
    wait: Future<void>.delayed,
    publish: publish,
    onProgress: onProgress,
    cancelRemote: () async {
      if (bunnyJobId case final jobId?) await api.cancelBunnyVideo(jobId);
    },
    onTerminal: () {
      uploadAuthorization = null;
      bunnyJobId = null;
    },
  );
}
