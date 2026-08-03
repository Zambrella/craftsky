import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:flutter/material.dart';

final class DraftThumbnail extends StatelessWidget {
  const DraftThumbnail({
    required this.repository,
    required this.draftId,
    required this.mediaId,
    super.key,
  });

  final LocalPostDraftRepository repository;
  final String draftId;
  final String mediaId;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Semantics(
      image: true,
      label: l10n.draftsThumbnailSemantics,
      child: FutureBuilder(
        future: repository.readMedia(draftId, mediaId),
        builder: (context, snapshot) {
          if (snapshot.hasData) {
            return Image.memory(
              snapshot.requireData,
              width: 56,
              height: 56,
              fit: BoxFit.cover,
              gaplessPlayback: true,
            );
          }
          if (snapshot.hasError) {
            return const Icon(Icons.broken_image_outlined);
          }
          return const SizedBox.square(
            dimension: 56,
            child: Center(child: CircularProgressIndicator(strokeWidth: 2)),
          );
        },
      ),
    );
  }
}
