import 'package:craftsky_app/feed/media/media_config.dart';
import 'package:craftsky_app/feed/providers/image_draft_controller.dart';

const _maxPostCharacters = 2000;

bool canSubmitImagePostDraft({
  required String text,
  required List<DraftImageState> images,
  int maxPostCharacters = _maxPostCharacters,
  MediaConfig config = mediaConfig,
}) {
  final trimmedText = text.trim();
  if (trimmedText.isEmpty || text.length > maxPostCharacters) return false;

  for (final image in images) {
    if (image.lifecycle != DraftImageLifecycle.uploaded) {
      return false;
    }

    final alt = image.altText.trim();
    if (alt.isEmpty || alt.length > config.maxAltTextCharacters) {
      return false;
    }
  }

  return true;
}
