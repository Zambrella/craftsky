import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/drafts/composer/draft_composer_hydrator.dart';
import 'package:craftsky_app/drafts/composer/draft_submission_origin.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/feed/providers/composer_video_controller.dart';
import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';

/// Shared draft revision, recovery snapshot and media persistence rules.
/// Widgets supply account-scoped dependencies and retain presentation.
final class ComposerDraftSession {
  new(LocalPostDraft? draft) : _origin = DraftSubmissionOrigin(draft);

  final DraftSubmissionOrigin _origin;

  LocalPostDraft? get draft => _origin.draft;

  Future<void> saveOriginSnapshot({
    required ActiveAccountLease? expectedOwner,
    required ActiveAccountLease? activeOwner,
    required Future<LocalPostDraft?> Function(AccountKey owner) save,
  }) async {
    final original = draft;
    if (original == null) return;
    if (activeOwner == null ||
        activeOwner != expectedOwner ||
        activeOwner.session.account != original.owner) {
      throw DiagnosticStateError('local-draft account changed');
    }
    final saved = await save(original.owner);
    if (saved == null) {
      throw DiagnosticStateError('local-draft account changed');
    }
    _origin.acceptSnapshot(saved);
  }

  Future<void> deleteAfterSuccess({
    required Future<LocalPostDraftRepository> Function(AccountKey owner)
    repositoryFor,
    required Future<void> Function(AccountKey owner) refresh,
  }) async {
    final original = draft;
    if (original == null) return;
    final repository = await repositoryFor(original.owner);
    await repository.delete(original.id);
    await refresh(original.owner);
  }

  DraftVideoWrite? videoWrite({
    required LocalVideoSelection? selection,
    required bool useStoredVideo,
  }) {
    if (selection == null) return null;
    final stored = draft?.video;
    if (useStoredVideo && stored != null) {
      return ExistingStoredVideo(
        storageRevision: stored.storageRevision,
        expectedSourceSha256: stored.sourceSha256,
        expectedPosterSha256: stored.posterSha256,
        altText: selection.altText,
      );
    }
    final poster = selection.posterBytes;
    if (poster == null) {
      throw const DraftRepositoryException(
        DraftRepositoryFailureReason.invalidRequest,
      );
    }
    return PreparedDraftVideo(
      displayFileName: selection.displayName,
      mimeType: selection.mimeType,
      byteLength: selection.byteLength,
      openSource: selection.openRead,
      width: selection.width,
      height: selection.height,
      duration: selection.duration,
      altText: selection.altText,
      posterMimeType: 'image/jpeg',
      posterBytes: poster,
    );
  }

  Future<LocalVideoSelection?> restoreVideo(
    LocalPostDraftSeed seed,
    LocalPostDraftRepository repository,
  ) async {
    final descriptor = seed.draft.video;
    if (descriptor == null || repository is! LocalVideoDraftRepository) {
      return null;
    }
    final videoRepository = repository as LocalVideoDraftRepository;
    final poster = await videoRepository.readVideoPoster(seed.draft.id);
    return LocalVideoSelection(
      displayName: descriptor.displayFileName,
      mimeType: descriptor.mimeType,
      byteLength: descriptor.sourceByteLength,
      duration: descriptor.duration,
      width: descriptor.width,
      height: descriptor.height,
      headerBytes: Uint8List(0),
      openRead: () => videoRepository.openVideoSource(seed.draft.id),
      posterBytes: poster,
      altText: descriptor.altText,
    );
  }
}
