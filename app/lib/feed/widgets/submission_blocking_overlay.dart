import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/material.dart';

final class SubmissionBlockingOverlay extends StatelessWidget {
  const SubmissionBlockingOverlay({required this.scheduling, super.key});

  final bool scheduling;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final copy = scheduling
        ? l10n.submissionSchedulingPost
        : l10n.submissionPublishingPost;
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
                    const StitchProgressIndicator(size: 48),
                    const SizedBox(height: 20),
                    Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 32),
                      child: Text(
                        copy,
                        textAlign: TextAlign.center,
                        style: Theme.of(context).textTheme.titleMedium,
                      ),
                    ),
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
