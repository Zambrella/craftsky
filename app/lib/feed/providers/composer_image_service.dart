import 'dart:typed_data';

import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/feed/media/image_dimensions.dart';
import 'package:craftsky_app/feed/media/image_metadata_stripper.dart';
import 'package:craftsky_app/feed/media/image_selection_validator.dart';
import 'package:craftsky_app/feed/media/image_upload_preparer.dart';
import 'package:craftsky_app/feed/media/media_config.dart';
import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/models/post_image_blob.dart';
import 'package:craftsky_app/feed/providers/image_draft_controller.dart';
import 'package:craftsky_app/feed/providers/post_api_client_provider.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';
import 'package:uuid/uuid.dart';

abstract interface class ComposerImageService {
  Future<void> addImages(ImageDraftController controller);
}

class ImageSelectionLimitExceededException implements Exception {
  const ImageSelectionLimitExceededException(this.maxImages);

  final int maxImages;
}

class SelectedComposerImage {
  const SelectedComposerImage({
    required this.id,
    required this.fileName,
    required this.mimeType,
    required this.bytes,
    required this.metadata,
  });

  final String id;
  final String fileName;
  final String mimeType;
  final Uint8List bytes;
  final Map<String, String> metadata;
}

class PreparedComposerImage {
  const PreparedComposerImage({
    required this.id,
    required this.fileName,
    required this.mimeType,
    required this.originalBytes,
    required this.preparedBytes,
    required this.aspectRatio,
    required this.strippedMetadata,
  });

  final String id;
  final String fileName;
  final String mimeType;
  final int originalBytes;
  final Uint8List preparedBytes;
  final CreatePostImageAspectRatio? aspectRatio;
  final Map<String, String> strippedMetadata;
}

abstract interface class ComposerImagePicker {
  Future<List<SelectedComposerImage>> pickImages({required int maxImages});
}

abstract interface class ComposerImagePreparer {
  Future<PreparedComposerImage> prepare(SelectedComposerImage image);
}

abstract interface class ComposerImageUploader {
  Future<UploadedImageBlob> upload(
    PreparedComposerImage image, {
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  });
}

class DefaultComposerImageService implements ComposerImageService {
  const DefaultComposerImageService({
    required ComposerImagePicker picker,
    required ComposerImagePreparer preparer,
    required ComposerImageUploader uploader,
    this.config = mediaConfig,
  }) : _picker = picker,
       _preparer = preparer,
       _uploader = uploader;

  final ComposerImagePicker _picker;
  final ComposerImagePreparer _preparer;
  final ComposerImageUploader _uploader;
  final MediaConfig config;

  @override
  Future<void> addImages(ImageDraftController controller) async {
    final remaining = config.maxImages - controller.images.length;
    if (remaining <= 0) return;

    final picked = await _picker.pickImages(maxImages: remaining);
    if (picked.isEmpty) return;

    final incoming = [
      for (final image in picked)
        _SelectionPair(
          image,
          LocalImageSelection(name: image.fileName, mimeType: image.mimeType),
        ),
    ];
    final existing = [
      for (final image in controller.images)
        LocalImageSelection(name: image.fileName, mimeType: image.mimeType),
    ];

    final selection = validateImageSelection(
      existing: existing,
      incoming: incoming.map((pair) => pair.selection).toList(),
      config: config,
    );

    var rejectedForLimit = false;

    _handleRejectedSelections(
      controller: controller,
      incoming: incoming,
      rejected: selection.rejected,
      onLimitExceeded: () => rejectedForLimit = true,
    );

    for (final accepted in selection.accepted) {
      final pair = incoming.firstWhere((p) => p.selection == accepted);
      await _prepareAndUploadAccepted(
        controller: controller,
        image: pair.image,
      );
    }

    if (rejectedForLimit) {
      throw ImageSelectionLimitExceededException(config.maxImages);
    }
  }

  void _handleRejectedSelections({
    required ImageDraftController controller,
    required List<_SelectionPair> incoming,
    required List<RejectedImageSelection> rejected,
    required void Function() onLimitExceeded,
  }) {
    for (final item in rejected) {
      if (item.reason == ImageSelectionRejection.imageLimitExceeded) {
        onLimitExceeded();
        continue;
      }

      final pair = incoming.firstWhere((p) => p.selection == item.image);
      controller.addDraftImage(
        DraftImageInput(
          id: pair.image.id,
          fileName: pair.image.fileName,
          mimeType: pair.image.mimeType,
          previewBytes: pair.image.bytes,
        ),
      );
      controller.markPreparationFailed(pair.image.id, 'Unsupported image type');
    }
  }

  Future<void> _prepareAndUploadAccepted({
    required ImageDraftController controller,
    required SelectedComposerImage image,
  }) async {
    controller.addDraftImage(
      DraftImageInput(
        id: image.id,
        fileName: image.fileName,
        mimeType: image.mimeType,
        previewBytes: image.bytes,
      ),
    );

    final prepared = await _prepare(controller: controller, image: image);
    if (prepared == null) return;

    if (!_validatePreparedSize(
      controller: controller,
      image: image,
      prepared: prepared,
    )) {
      return;
    }

    controller.markPrepared(image.id);
    await _uploadPrepared(
      controller: controller,
      image: image,
      prepared: prepared,
    );
  }

  Future<PreparedComposerImage?> _prepare({
    required ImageDraftController controller,
    required SelectedComposerImage image,
  }) async {
    try {
      return await _preparer.prepare(image);
    } on Exception {
      controller.markPreparationFailed(image.id, 'Could not prepare image');
      return null;
    }
  }

  bool _validatePreparedSize({
    required ImageDraftController controller,
    required SelectedComposerImage image,
    required PreparedComposerImage prepared,
  }) {
    final sizeCheck = validatePreparedUpload(
      originalBytes: prepared.originalBytes,
      preparedBytes: prepared.preparedBytes.length,
      config: config,
    );
    if (sizeCheck.canUpload) return true;

    controller.markPreparationFailed(
      image.id,
      'Image exceeds ${config.maxImageBytes ~/ (1024 * 1024)} MB limit',
    );
    return false;
  }

  Future<void> _uploadPrepared({
    required ImageDraftController controller,
    required SelectedComposerImage image,
    required PreparedComposerImage prepared,
  }) async {
    try {
      final uploaded = await _uploader.upload(
        prepared,
        onSendProgress: (sent, total) {
          if (total <= 0) return;
          controller.markUploadProgress(image.id, sent / total);
        },
      );
      controller.markUploaded(
        image.id,
        UploadedDraftImage(
          cid: uploaded.cid,
          mime: uploaded.mime,
          size: uploaded.size,
          aspectRatio: prepared.aspectRatio,
        ),
      );
    } on Exception {
      controller.markUploadFailed(image.id, 'Upload failed');
    }
  }
}

class DeviceComposerImagePicker implements ComposerImagePicker {
  DeviceComposerImagePicker({ImagePicker? picker, Uuid? uuid})
    : _picker = picker ?? ImagePicker(),
      _uuid = uuid ?? const Uuid();

  final ImagePicker _picker;
  final Uuid _uuid;

  @override
  Future<List<SelectedComposerImage>> pickImages({
    required int maxImages,
  }) async {
    if (maxImages <= 0) return const [];

    final files = await _picker.pickMultiImage();
    final selected = <SelectedComposerImage>[];
    for (final file in files) {
      final bytes = await file.readAsBytes();
      selected.add(
        SelectedComposerImage(
          id: _uuid.v4(),
          fileName: file.name,
          mimeType: _mimeTypeForFileName(file.name),
          bytes: Uint8List.fromList(bytes),
          metadata: const {},
        ),
      );
    }
    return selected;
  }
}

class DefaultComposerImagePreparer implements ComposerImagePreparer {
  const DefaultComposerImagePreparer();

  @override
  Future<PreparedComposerImage> prepare(SelectedComposerImage image) async {
    final prepared = prepareImageForUpload(
      originalBytes: image.bytes,
      fileName: image.fileName,
      mimeType: image.mimeType,
      metadata: image.metadata,
    );

    return PreparedComposerImage(
      id: image.id,
      fileName: image.fileName,
      mimeType: prepared.mimeType,
      originalBytes: image.bytes.length,
      preparedBytes: prepared.bytes,
      aspectRatio: toOptionalAspectRatio(
        width: prepared.width,
        height: prepared.height,
      ),
      strippedMetadata: prepared.metadata,
    );
  }
}

class AppViewComposerImageUploader implements ComposerImageUploader {
  const AppViewComposerImageUploader(this._api);

  final PostApiClient _api;

  @override
  Future<UploadedImageBlob> upload(
    PreparedComposerImage image, {
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) {
    return _api.uploadImage(
      bytes: image.preparedBytes,
      mimeType: image.mimeType,
      onSendProgress: onSendProgress,
      onReceiveProgress: onReceiveProgress,
    );
  }
}

final composerImagePickerProvider = Provider<ComposerImagePicker>(
  (ref) => DeviceComposerImagePicker(),
);

final composerImagePreparerProvider = Provider<ComposerImagePreparer>(
  (ref) => const DefaultComposerImagePreparer(),
);

final composerImageUploaderProvider = Provider<ComposerImageUploader>(
  (ref) => AppViewComposerImageUploader(ref.watch(postApiClientProvider)),
);

final composerImageServiceProvider = Provider<ComposerImageService>((ref) {
  return DefaultComposerImageService(
    picker: ref.watch(composerImagePickerProvider),
    preparer: ref.watch(composerImagePreparerProvider),
    uploader: ref.watch(composerImageUploaderProvider),
  );
});

class _SelectionPair {
  const _SelectionPair(this.image, this.selection);

  final SelectedComposerImage image;
  final LocalImageSelection selection;
}

String _mimeTypeForFileName(String fileName) {
  final lower = fileName.toLowerCase();
  if (lower.endsWith('.jpg') || lower.endsWith('.jpeg')) return 'image/jpeg';
  if (lower.endsWith('.png')) return 'image/png';
  if (lower.endsWith('.webp')) return 'image/webp';
  return '';
}
