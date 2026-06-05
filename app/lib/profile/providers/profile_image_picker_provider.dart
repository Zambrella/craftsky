import 'dart:typed_data';

import 'package:craftsky_app/feed/media/composer_image_media_service.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/shared/media/blob_api_client.dart';
import 'package:craftsky_app/shared/media/blob_api_client_provider.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';

final profileImagePickerProvider = Provider<ProfileImagePicker>((ref) {
  return ProfileImagePicker(
    picker: ref.watch(imagePickerProvider),
    media: ref.watch(composerImageMediaServiceProvider),
    blobApi: ref.watch(blobApiClientProvider),
  );
});

class ProfileImagePicker {
  const ProfileImagePicker({
    required this._picker,
    required this._media,
    required this._blobApi,
  });

  final ImagePicker _picker;
  final ComposerImageMediaService _media;
  final BlobApiClient _blobApi;

  Future<ProfileImagePickResult?> pickAndUpload({
    required void Function(Uint8List bytes) onPreviewReady,
  }) async {
    final file = await _picker.pickImage(source: ImageSource.gallery);
    if (file == null) return null;

    final mimeType = file.mimeType ?? _media.mimeTypeForFileName(file.name);
    final length = await file.length();
    final header = await _readHeaderBytes(file);
    final originalCheck = _media.validateOriginalImage(
      sizeBytes: length,
      fileName: file.name,
      mimeType: mimeType,
      headerBytes: header,
    );
    if (!originalCheck.canPrepare) {
      throw const ProfileImagePickException();
    }

    final bytes = await file.readAsBytes();
    onPreviewReady(bytes);

    final prepared = await _media
        .prepareImage(bytes: bytes, fileName: file.name, mimeType: mimeType)
        .future;
    final preparedCheck = _media.validatePreparedUploadBytes(
      originalBytes: bytes.length,
      preparedBytes: prepared.bytes.length,
    );
    if (!preparedCheck.canUpload) {
      throw const ProfileImagePickException();
    }

    final uploaded = await _blobApi.uploadImage(
      bytes: prepared.bytes,
      mimeType: prepared.mimeType,
    );
    return ProfileImagePickResult(
      previewBytes: prepared.bytes,
      uploaded: uploaded,
    );
  }

  Future<Uint8List> _readHeaderBytes(XFile file) async {
    final header = <int>[];
    await for (final chunk in file.openRead(0, 16)) {
      header.addAll(chunk);
      if (header.length >= 16) break;
    }
    return Uint8List.fromList(
      header.length <= 16 ? header : header.sublist(0, 16),
    );
  }
}

class ProfileImagePickResult {
  const ProfileImagePickResult({
    required this.previewBytes,
    required this.uploaded,
  });

  final Uint8List previewBytes;
  final UploadedImageBlob uploaded;
}

class ProfileImagePickException implements Exception {
  const ProfileImagePickException();
}
