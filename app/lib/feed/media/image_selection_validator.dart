import 'package:craftsky_app/feed/media/media_config.dart';

const _supportedMimeTypes = <String>{'image/jpeg', 'image/png', 'image/webp'};
const _supportedExtensions = <String>{'.jpg', '.jpeg', '.png', '.webp'};

enum ImageSelectionRejection { unsupportedType, imageLimitExceeded }

class LocalImageSelection {
  const LocalImageSelection({required this.name, required this.mimeType});

  final String name;
  final String mimeType;
}

class RejectedImageSelection {
  const RejectedImageSelection({required this.image, required this.reason});

  final LocalImageSelection image;
  final ImageSelectionRejection reason;
}

class ImageSelectionValidationResult {
  const ImageSelectionValidationResult({
    required this.accepted,
    required this.rejected,
  });

  final List<LocalImageSelection> accepted;
  final List<RejectedImageSelection> rejected;
}

ImageSelectionValidationResult validateImageSelection({
  required List<LocalImageSelection> existing,
  required List<LocalImageSelection> incoming,
  MediaConfig config = mediaConfig,
}) {
  var remainingSlots = config.maxImages - existing.length;
  final accepted = <LocalImageSelection>[];
  final rejected = <RejectedImageSelection>[];

  for (final candidate in incoming) {
    if (!_isSupportedType(candidate)) {
      rejected.add(
        RejectedImageSelection(
          image: candidate,
          reason: ImageSelectionRejection.unsupportedType,
        ),
      );
      continue;
    }
    if (remainingSlots <= 0) {
      rejected.add(
        RejectedImageSelection(
          image: candidate,
          reason: ImageSelectionRejection.imageLimitExceeded,
        ),
      );
      continue;
    }

    accepted.add(candidate);
    remainingSlots -= 1;
  }

  return ImageSelectionValidationResult(accepted: accepted, rejected: rejected);
}

bool _isSupportedType(LocalImageSelection candidate) {
  final mime = candidate.mimeType.trim().toLowerCase();
  if (_supportedMimeTypes.contains(mime)) return true;

  final name = candidate.name.toLowerCase();
  return _supportedExtensions.any(name.endsWith);
}
