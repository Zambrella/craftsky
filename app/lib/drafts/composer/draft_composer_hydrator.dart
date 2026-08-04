import 'dart:typed_data';

import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/draft_media_descriptor.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';

final class HydratedDraftMedia {
  const HydratedDraftMedia({required this.descriptor, required this.bytes});

  final DraftMediaDescriptor descriptor;
  final Uint8List? bytes;

  bool get isAvailable => bytes != null;

  @override
  String toString() => 'HydratedDraftMedia(<redacted>)';
}

final class LocalPostDraftSeed {
  LocalPostDraftSeed({
    required this.draft,
    required List<HydratedDraftMedia> media,
  }) : media = List.unmodifiable(media);

  final LocalPostDraft draft;
  final List<HydratedDraftMedia> media;

  bool get canSubmit => media.every((item) => item.isAvailable);

  @override
  String toString() => 'LocalPostDraftSeed(<redacted>)';
}

final class DraftComposerHydrator {
  const DraftComposerHydrator();

  Future<LocalPostDraftSeed> hydrate({
    required LocalPostDraftRepository repository,
    required LocalPostDraft draft,
  }) async {
    final hydrated = <HydratedDraftMedia>[];
    for (final descriptor in draft.media) {
      Uint8List? bytes;
      if (descriptor.availability == DraftMediaAvailability.available) {
        try {
          bytes = await repository.readMedia(draft.id, descriptor.mediaId);
        } on Object {
          bytes = null;
        }
      }
      hydrated.add(HydratedDraftMedia(descriptor: descriptor, bytes: bytes));
    }
    return LocalPostDraftSeed(draft: draft, media: hydrated);
  }
}
