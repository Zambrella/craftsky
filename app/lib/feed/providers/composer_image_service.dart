import 'package:craftsky_app/feed/providers/image_draft_controller.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

abstract interface class ComposerImageService {
  Future<void> addImages(ImageDraftController controller);
}

class NoopComposerImageService implements ComposerImageService {
  @override
  Future<void> addImages(ImageDraftController controller) async {}
}

final composerImageServiceProvider = Provider<ComposerImageService>(
  (ref) => NoopComposerImageService(),
);
