import 'dart:async';

import 'package:craftsky_app/feed/composer/link_preview_controller.dart';
import 'package:craftsky_app/feed/composer/prepared_media_validation.dart';
import 'package:craftsky_app/feed/media/composer_image_media_service.dart';
import 'package:craftsky_app/feed/models/create_post_external.dart';
import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:crypto/crypto.dart';
import 'package:dio/dio.dart';

typedef PreparedImageUpload =
    Future<UploadedImageBlob> Function({
      required List<int> bytes,
      required String mimeType,
      required CancelToken cancelToken,
    });

final class ComposerMediaUploader {
  ComposerMediaUploader({
    this.transferBudget = const Duration(minutes: 1),
    this.mediaService = const ComposerImageMediaService(),
  });

  final Duration transferBudget;
  final ComposerImageMediaService mediaService;
  final Map<_UploadKey, UploadedImageBlob> _successfulUploads = {};
  final Map<_ExternalUploadKey, UploadedImageBlob> _successfulExternalUploads =
      {};

  Future<List<CreatePostImage>?> materializeImmediate({
    required String composerId,
    required List<ComposerImageDraft> images,
    required bool Function() ownershipIsCurrent,
    required PreparedImageUpload upload,
  }) async {
    if (images.isEmpty) return null;
    final currentKeys = <_UploadKey>{};
    final materialized = <CreatePostImage>[];
    for (final image in images) {
      _requireCurrentOwnership(ownershipIsCurrent);
      final phase = image.phase;
      if (phase case ImageUploaded(:final uploaded)) {
        materialized.add(
          CreatePostImage(
            blob: CreatePostBlob(
              ref: CreatePostBlobRef(link: uploaded.cid),
              mimeType: uploaded.mime,
              size: uploaded.size,
            ),
            alt: image.altText.trim(),
            aspectRatio: uploaded.aspectRatio,
          ),
        );
        continue;
      }
      if (phase is! ImageReady) {
        throw StateError('composer image is not ready');
      }
      final verifiedDigest = verifyPreparedMediaBytes(
        bytes: phase.bytes,
        mimeType: phase.mimeType,
        width: phase.width,
        height: phase.height,
        altText: image.altText,
        mediaService: mediaService,
        expectedSha256: phase.sha256,
      );
      final key = (
        composerId: composerId,
        mediaId: image.id,
        digest: verifiedDigest,
      );
      currentKeys.add(key);
      var uploaded = _successfulUploads[key];
      if (uploaded == null) {
        final cancelToken = CancelToken();
        uploaded =
            await upload(
              bytes: phase.bytes,
              mimeType: phase.mimeType,
              cancelToken: cancelToken,
            ).timeout(
              transferBudget,
              onTimeout: () {
                cancelToken.cancel('image transfer timed out');
                throw TimeoutException(
                  'image transfer timed out',
                  transferBudget,
                );
              },
            );
        _requireCurrentOwnership(ownershipIsCurrent);
        _successfulUploads[key] = uploaded;
      }
      materialized.add(
        CreatePostImage(
          blob: CreatePostBlob(
            ref: CreatePostBlobRef(link: uploaded.cid),
            mimeType: uploaded.mime,
            size: uploaded.size,
          ),
          alt: image.altText.trim(),
          aspectRatio: CreatePostImageAspectRatio(
            width: phase.width,
            height: phase.height,
          ),
        ),
      );
    }
    _successfulUploads.removeWhere(
      (key, _) => key.composerId == composerId && !currentKeys.contains(key),
    );
    return materialized;
  }

  Future<CreatePostExternal> materializeImmediateExternal({
    required String composerId,
    required SelectedLinkPreview selection,
    required bool Function() ownershipIsCurrent,
    required PreparedImageUpload upload,
  }) async {
    _requireCurrentOwnership(ownershipIsCurrent);
    final thumbnail = selection.preview.thumbnail;
    UploadedImageBlob? uploaded;
    if (thumbnail != null) {
      final key = (
        composerId: composerId,
        identity: selection.candidate.identity.toString(),
        digest: sha256.convert(thumbnail.bytes).toString(),
      );
      uploaded = _successfulExternalUploads[key];
      if (uploaded == null) {
        final cancelToken = CancelToken();
        uploaded =
            await upload(
              bytes: thumbnail.bytes,
              mimeType: thumbnail.mimeType,
              cancelToken: cancelToken,
            ).timeout(
              transferBudget,
              onTimeout: () {
                cancelToken.cancel('preview thumbnail transfer timed out');
                throw TimeoutException(
                  'preview thumbnail transfer timed out',
                  transferBudget,
                );
              },
            );
        _requireCurrentOwnership(ownershipIsCurrent);
        _successfulExternalUploads[key] = uploaded;
      }
    }
    return CreatePostExternal(
      uri: selection.navigationUri.toString(),
      title: selection.preview.title,
      description: selection.preview.description,
      thumb: uploaded == null
          ? null
          : CreatePostBlob(
              ref: CreatePostBlobRef(link: uploaded.cid),
              mimeType: uploaded.mime,
              size: uploaded.size,
            ),
    );
  }

  void disposeComposer(String composerId) {
    _successfulUploads.removeWhere((key, _) => key.composerId == composerId);
    _successfulExternalUploads.removeWhere(
      (key, _) => key.composerId == composerId,
    );
  }
}

typedef _UploadKey = ({String composerId, String mediaId, String digest});
typedef _ExternalUploadKey = ({
  String composerId,
  String identity,
  String digest,
});

void _requireCurrentOwnership(bool Function() ownershipIsCurrent) {
  if (!ownershipIsCurrent()) throw StateError('submission ownership changed');
}
