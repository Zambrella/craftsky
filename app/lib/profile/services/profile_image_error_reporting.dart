import 'package:craftsky_app/feed/media/bounded_image_file_reader.dart';
import 'package:craftsky_app/feed/media/composer_image_media_service.dart';
import 'package:craftsky_app/profile/models/profile_image_failure.dart';
import 'package:craftsky_app/profile/providers/profile_image_picker_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/diagnostic_outcome.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/misc.dart' show ProviderException;
import 'package:logging/logging.dart';

enum ProfileImageOperation { onboarding, editor }

final _log = Logger('ProfileImages');

/// The UI consumes this failure, so it owns the issue. Context describes the
/// initiating account only: selected images, filenames and destinations stay
/// private even when the resulting profile will eventually be public.
Future<void> reportProfileImageFailure({
  required ErrorReporter reporter,
  required Object error,
  required StackTrace stackTrace,
  required ProfileImageOperation operation,
  required String accountDid,
}) async {
  // Acquisition failures already belong to Riverpod's error observer.
  if (error is ProviderException) return;
  final validation =
      (error is ProfileImagePickException &&
          (error.expectedRejection ||
              error.reason != ProfileImageFailure.unknown)) ||
      error is ImageSourceDimensionsTooLargeException ||
      error is ImageSourceTooLargeException;
  final denied =
      error is PlatformException &&
      const {
        'camera_access_denied',
        'camera_access_restricted',
        'photo_access_denied',
        'photo_access_restricted',
        'camera_access_denied_without_prompt',
        'photo_access_denied_without_prompt',
      }.contains(error.code);
  // These rejections describe the transfer contract, not invalid user input.
  // The UI consumes this failed attempt; reporting does not alter retry policy.
  final failedTransfer =
      error is ApiBadRequest &&
      const {
        'image_upload_failed',
        'invalid_image_upload_destination',
      }.contains(error.code);
  final context = ReportContext(
    feature: operation == ProfileImageOperation.onboarding
        ? 'Onboarding'
        : 'ProfileEditor',
    operation: 'profile.photo.pick_upload',
    classification: 'profile.photo.failed',
    outcome: validation || denied
        ? DiagnosticOutcome.expected
        : failedTransfer
        ? DiagnosticOutcome.terminal
        : DiagnosticOutcome.automatic,
    workflow: PrivateOperationalFailureContext(accountDid: accountDid),
  );
  final message = DiagnosticMessage('Profile photo failed', context: context);
  if (isExpectedDiagnostic(error, context)) {
    _log.warning(message, error, stackTrace);
    return;
  }
  _log.severe(message, error, stackTrace);
  await GuardedErrorReporter(reporter).captureException(
    error,
    stackTrace: stackTrace,
    context: context,
  );
}
