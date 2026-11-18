import 'dart:async';
import 'dart:io';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/drafts/composer/composer_draft_session.dart';
import 'package:craftsky_app/drafts/composer/draft_composer_hydrator.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/drafts/providers/local_post_draft_repository_provider.dart';
import 'package:craftsky_app/drafts/providers/local_post_drafts_provider.dart';
import 'package:craftsky_app/feed/composer/composer_ownership.dart';
import 'package:craftsky_app/feed/media/composer_image_media_service.dart';
import 'package:craftsky_app/feed/media/video_source_validator.dart';
import 'package:craftsky_app/feed/models/instagram_post_import_source.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/feed/providers/composer_video_controller.dart';
import 'package:craftsky_app/feed/providers/image_picker_existing_video.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/shared/observability/consumed_failure.dart';
import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';
import 'package:uuid/uuid.dart';

/// Reuses a local source draft or copies selected archive media into the normal
/// draft store before opening a composer. Only the archive caller owns the
/// extracted temporary files. The modal result is the confirmed AppView post.
Future<Post?> openInstagramPostComposer(
  BuildContext context,
  WidgetRef ref, {
  required ActiveAccountLease owner,
  required InstagramPostImportSource source,
  required String caption,
  required LocalPostDraftKind kind,
  required Future<({List<XFile> images, XFile? video})?> Function() loadMedia,
  Future<bool> Function(
    ({
      int readyPhotos,
      bool readyVideo,
      int failedPhotos,
      Object? videoFailure,
    })
    media,
  )?
  confirmMediaRecovery,
  Future<void> Function()? onMediaConsumed,
}) async {
  final account = owner.session.account;
  final operation = ActiveAccountOperation.fromWidget(ref, owner);
  void requireOwner() => requireComposerOwnership(() => operation.isCurrent);
  requireOwner();
  final repository = await ref.read(
    accountLocalPostDraftRepositoryProvider(account).future,
  );
  requireOwner();
  final existing = (await repository.list())
      .where((draft) => draft.instagramImport?.sourceKey == source.sourceKey)
      .firstOrNull;
  requireOwner();

  LocalPostDraft draft;
  if (existing != null) {
    draft = existing.instagramPublication != null
        ? existing
        : _draftForKind(existing, kind);
  } else {
    final selection = await loadMedia();
    requireOwner();
    if (selection == null) return null;
    final (:images, :video) = selection;
    final media = ref.read(composerImageMediaServiceProvider);
    if (images.length > media.maxImages ||
        (images.isNotEmpty && video != null)) {
      throw const DraftRepositoryException(
        DraftRepositoryFailureReason.invalidRequest,
      );
    }
    const uuid = Uuid();
    final preparedImages = <PreparedDraftMedia>[];
    var failedPhotoCount = 0;
    for (final image in images) {
      try {
        preparedImages.add(
          await _prepareImage(image, media, uuid.v4(), requireOwner),
        );
      } on Object catch (error, stack) {
        if (confirmMediaRecovery == null ||
            (error is! FormatException &&
                error is! ImageSourceDimensionsTooLargeException &&
                error is! FileSystemException &&
                error is! PlatformException)) {
          rethrow;
        }
        requireOwner();
        if (error is FileSystemException || error is PlatformException) {
          _reportMediaFailure(ref, owner, 'photo.prepare', error, stack);
        }
        failedPhotoCount++;
      }
      requireOwner();
    }

    DraftVideoWrite? preparedVideo;
    Object? videoFailure;
    if (video != null) {
      final controller = ComposerVideoController(
        picker: ImagePickerExistingVideo(pick: () async => video),
      );
      try {
        await controller.selectExisting();
      } on Object catch (error, stack) {
        if (confirmMediaRecovery == null ||
            (error is! VideoSourceException &&
                error is! FormatException &&
                error is! DiagnosticStateError &&
                error is! TimeoutException &&
                error is! FileSystemException &&
                error is! PlatformException)) {
          rethrow;
        }
        requireOwner();
        if (error is! VideoSourceException && error is! FormatException) {
          _reportMediaFailure(ref, owner, 'video.prepare', error, stack);
        }
        videoFailure = error;
      }
      requireOwner();
      preparedVideo = ComposerDraftSession(null).videoWrite(
        selection: controller.selection,
        useStoredVideo: false,
      );
    }
    if (confirmMediaRecovery != null) {
      final confirmed = await confirmMediaRecovery((
        readyPhotos: preparedImages.length,
        readyVideo: preparedVideo != null,
        failedPhotos: failedPhotoCount,
        videoFailure: videoFailure,
      ));
      requireOwner();
      if (!confirmed) return null;
    }
    final languages = [
      ref.read(activeLanguagePreferencesProvider).primaryLanguage,
    ];
    draft = await repository.save(
      DraftWriteRequest(
        id: uuid.v4(),
        owner: account,
        kind: kind,
        content: _draftContent(kind, caption, languages, sponsored: false),
        schedule: const DraftScheduleIntent.now(),
        orderedMedia: preparedImages,
        video: preparedVideo,
        instagramImport: source,
      ),
    );
    requireOwner();
    ref.invalidate(localPostDraftsProvider(account));
  }
  await onMediaConsumed?.call();
  requireOwner();

  final seed = await const DraftComposerHydrator().hydrate(
    repository: repository,
    draft: draft,
  );
  requireOwner();
  if (!context.mounted) return null;
  return await switch (draft.kind) {
    LocalPostDraftKind.standard => showPostComposerSheet(
      context,
      draftSeed: seed,
      draftOwner: owner,
    ),
    LocalPostDraftKind.project => showProjectComposerSheet(
      context,
      draftSeed: seed,
      draftOwner: owner,
    ),
  };
}

Future<PreparedDraftMedia> _prepareImage(
  XFile image,
  ComposerImageMediaService media,
  String mediaId,
  void Function() requireOwner,
) async {
  final size = await image.length();
  requireOwner();
  if (size <= 0 || size > media.config.maxSourceImageBytes) {
    throw const DiagnosticFormatException('Invalid source image size');
  }
  final bytes = await image.readAsBytes();
  requireOwner();
  final prepared = await media
      .prepareImage(
        bytes: bytes,
        fileName: image.name,
        mimeType: image.mimeType ?? '',
        allowWebp: true,
      )
      .future;
  return PreparedDraftMedia(
    mediaId: mediaId,
    displayFileName: prepared.mimeType == 'image/png'
        ? 'instagram-image.png'
        : 'instagram-image.jpg',
    mimeType: prepared.mimeType,
    bytes: prepared.bytes,
    width: prepared.width,
    height: prepared.height,
    altText: '',
  );
}

void _reportMediaFailure(
  WidgetRef ref,
  ActiveAccountLease owner,
  String operation,
  Object error,
  StackTrace stack,
) => reportConsumedFailure(
  error,
  stack,
  reporter: ref.read(errorReporterProvider),
  feature: 'InstagramPostImport',
  operation: operation,
  accountDid: owner.session.account.did,
);

/// Switching composer type is provisional. Closing the editor keeps the saved
/// draft and project fields. Saving or publishing commits the chosen type.
LocalPostDraft _draftForKind(LocalPostDraft draft, LocalPostDraftKind kind) {
  if (draft.kind == kind) return draft;
  final (text, languages, sponsored) = switch (draft.content) {
    StandardDraftContent(:final text, :final languages, :final sponsored) => (
      text,
      languages,
      sponsored,
    ),
    ProjectDraftContent(:final body, :final languages, :final sponsored) => (
      body,
      languages,
      sponsored,
    ),
  };
  return LocalPostDraft(
    id: draft.id,
    owner: draft.owner,
    kind: kind,
    createdAt: draft.createdAt,
    updatedAt: draft.updatedAt,
    content: _draftContent(kind, text, languages, sponsored: sponsored),
    schedule: draft.schedule,
    media: draft.media,
    video: draft.video,
    instagramImport: draft.instagramImport,
    availability: draft.availability,
    revision: draft.revision,
  );
}

LocalDraftContent _draftContent(
  LocalPostDraftKind kind,
  String text,
  List<String> languages, {
  required bool sponsored,
}) => switch (kind) {
  LocalPostDraftKind.standard => StandardDraftContent(
    text: text,
    languages: languages,
    sponsored: sponsored,
  ),
  LocalPostDraftKind.project => ProjectDraftContent(
    body: text,
    languages: languages,
    knownProjectFieldValues: const {},
    sponsored: sponsored,
  ),
};
