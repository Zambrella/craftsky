import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/drafts/providers/draft_save_controller.dart';
import 'package:craftsky_app/feed/composer/composer_submission_coordinator.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Retains the save provider while a composer persists its recovery snapshot.
Future<LocalPostDraft?> saveComposerSnapshot({
  required ProviderContainer container,
  required AccountKey owner,
  required DraftWriteRequest request,
}) async {
  final provider = draftSaveControllerProvider(owner);
  Object? publishedFailure;
  final subscription = container.listen(provider, (_, next) {
    if (next.hasError) publishedFailure = next.error;
  });
  try {
    return await container.read(provider.notifier).save(request);
  } on Object catch (error) {
    // This exact occurrence was published as AsyncError and belongs to the
    // provider observer. Pre-publication and late consumed failures still
    // propagate with their original cause to the submission owner.
    if (identical(publishedFailure, error)) {
      throw const ComposerSubmissionFailure();
    }
    rethrow;
  } finally {
    subscription.close();
  }
}
