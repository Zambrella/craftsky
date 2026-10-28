import 'package:craftsky_app/feed/composer/video_publication_coordinator.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/theme/publishing_progress_indicator.dart';
import 'package:material_ui/material_ui.dart';

enum SubmissionKind { post, comment, reply, edit }

final class SubmissionBlockingOverlay extends StatelessWidget {
  const new({
    required this.scheduling,
    super.key,
    this.kind = SubmissionKind.post,
    this.videoProgress,
    this.onCancelVideo,
  });

  final bool scheduling;
  final SubmissionKind kind;
  final VideoPublicationProgress? videoProgress;
  final VoidCallback? onCancelVideo;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final copy = switch (videoProgress?.stage) {
      VideoPublicationStage.uploading => l10n.postVideoUploading,
      VideoPublicationStage.processing => l10n.postVideoProcessing,
      VideoPublicationStage.publishing =>
        kind == SubmissionKind.edit
            ? l10n.postEditSaving
            : l10n.postVideoPublishing,
      _ => switch (kind) {
        SubmissionKind.post =>
          scheduling
              ? l10n.submissionSchedulingPost
              : l10n.submissionPublishingPost,
        SubmissionKind.comment => l10n.submissionPublishingComment,
        SubmissionKind.reply => l10n.submissionPublishingReply,
        SubmissionKind.edit => l10n.postEditSaving,
      },
    };
    final canCancel =
        onCancelVideo != null &&
        canCancelVideoPublication(videoProgress?.stage);
    return Positioned.fill(
      child: Material(
        type: MaterialType.transparency,
        child: Semantics(
          container: true,
          liveRegion: true,
          label: copy,
          child: Stack(
            fit: StackFit.expand,
            children: [
              ModalBarrier(
                key: const Key('submission-modal-barrier'),
                dismissible: false,
                color: Theme.of(context).colorScheme.surface.withValues(
                  alpha: 0.94,
                ),
              ),
              Center(
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    const PublishingProgressIndicator(),
                    const SizedBox(height: 20),
                    Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 32),
                      child: Text(
                        copy,
                        textAlign: TextAlign.center,
                        style: Theme.of(context).textTheme.titleMedium,
                      ),
                    ),
                    if (videoProgress case final progress?) ...[
                      const SizedBox(height: 16),
                      SizedBox(
                        width: 240,
                        child: LinearProgressIndicator(
                          value: progress.fraction,
                        ),
                      ),
                    ],
                    if (canCancel) ...[
                      const SizedBox(height: 12),
                      TextButton(
                        key: const Key('cancel-video-publication'),
                        onPressed: onCancelVideo,
                        child: Text(l10n.postVideoCancel),
                      ),
                    ],
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
