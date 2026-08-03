import 'dart:async';

import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:dio/dio.dart';

typedef PreparedImageUpload =
    Future<UploadedImageBlob> Function({
      required List<int> bytes,
      required String mimeType,
      required CancelToken cancelToken,
    });

final class ComposerMediaUploader {
  ComposerMediaUploader({
    required this._upload,
    this.transferBudget = const Duration(minutes: 1),
  });

  final PreparedImageUpload _upload;
  final Duration transferBudget;
  final Map<_UploadKey, UploadedImageBlob> _successfulUploads = {};

  Future<List<CreatePostImage>?> materializeImmediate({
    required String composerId,
    required List<ComposerImageDraft> images,
  }) async {
    if (images.isEmpty) return null;
    final currentKeys = <_UploadKey>{};
    final materialized = <CreatePostImage>[];
    for (final image in images) {
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
      final key = (
        composerId: composerId,
        mediaId: image.id,
        digest: phase.sha256,
      );
      currentKeys.add(key);
      var uploaded = _successfulUploads[key];
      if (uploaded == null) {
        final cancelToken = CancelToken();
        uploaded =
            await _upload(
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

  void disposeComposer(String composerId) {
    _successfulUploads.removeWhere((key, _) => key.composerId == composerId);
  }
}

typedef _UploadKey = ({String composerId, String mediaId, String digest});
