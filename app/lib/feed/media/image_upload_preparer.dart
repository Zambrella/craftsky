import 'package:craftsky_app/feed/media/media_config.dart';

enum PreparedUploadRejection { tooLarge }

class PreparedUploadValidationResult {
  const PreparedUploadValidationResult({
    required this.canUpload,
    required this.rejectedReason,
  });

  final bool canUpload;
  final PreparedUploadRejection? rejectedReason;
}

bool validatePreparedUploadSize({
  required int preparedBytes,
  MediaConfig config = mediaConfig,
}) => preparedBytes <= config.maxImageBytes;

PreparedUploadValidationResult validatePreparedUpload({
  required int originalBytes,
  required int preparedBytes,
  MediaConfig config = mediaConfig,
}) {
  if (!validatePreparedUploadSize(
    preparedBytes: preparedBytes,
    config: config,
  )) {
    return const PreparedUploadValidationResult(
      canUpload: false,
      rejectedReason: PreparedUploadRejection.tooLarge,
    );
  }
  return const PreparedUploadValidationResult(
    canUpload: true,
    rejectedReason: null,
  );
}
