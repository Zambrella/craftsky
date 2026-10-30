import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/feed/media/bounded_image_file_reader.dart';
import 'package:craftsky_app/feed/media/composer_image_media_service.dart';
import 'package:craftsky_app/feed/media/media_config.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/profile/models/profile_image_failure.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:craftsky_app/shared/image/basic_image_editor.dart';
import 'package:craftsky_app/shared/media/blob_api_client.dart';
import 'package:craftsky_app/shared/media/blob_api_client_provider.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'profile_image_picker_provider.g.dart';

// app.bsky.actor.profile limits both avatars and banners to 1 MB.
final profileImageMediaServiceProvider = Provider<ComposerImageMediaService>(
  (ref) => const ComposerImageMediaService(
    config: MediaConfig(
      maxImages: 1,
      maxImageBytes: 1000000,
      targetImageBytes: 950000,
      maxAltTextCharacters: 0,
    ),
  ),
);

final profileImagePickerProvider = Provider<ProfileImagePicker>((ref) {
  return ProfileImagePicker(
    picker: ref.watch(imagePickerProvider),
    media: ref.watch(profileImageMediaServiceProvider),
    blobApi: ref.watch(blobApiClientProvider),
  );
});

@riverpod
Future<ProfileImagePicker> accountProfileImagePicker(
  Ref ref,
  ActiveAccountLease lease,
) async {
  final dio = await ref.watch(accountDioProvider(lease.session.account).future);
  if (!ref.mounted) throw const ApiCanceled();
  return ProfileImagePicker(
    picker: ref.watch(imagePickerProvider),
    media: ref.watch(profileImageMediaServiceProvider),
    blobApi: managedBlobApiClient(ref, dio),
  );
}

class ProfileImagePicker {
  const new({
    required this._picker,
    required this._media,
    required this._blobApi,
  });

  final ImagePicker _picker;
  final ComposerImageMediaService _media;
  final BlobApiClient _blobApi;

  Future<ProfileImagePickResult?> pickAndUpload({
    required ImageSource source,
    required void Function(Uint8List bytes) onPreviewReady,
    BasicImageEditor? editImage,
    bool Function()? isCurrent,
  }) async {
    final prepared = await pickAndPrepare(
      source: source,
      onPreviewReady: onPreviewReady,
      editImage: editImage,
      isCurrent: isCurrent,
    );
    if (prepared == null || isCurrent?.call() == false) return null;

    final uploaded = await _blobApi.uploadImage(
      bytes: prepared.bytes,
      mimeType: prepared.mimeType,
    );
    if (isCurrent?.call() == false) return null;
    return ProfileImagePickResult(
      previewBytes: prepared.bytes,
      uploaded: uploaded,
    );
  }

  Future<PreparedProfileImage?> pickAndPrepare({
    required ImageSource source,
    required void Function(Uint8List bytes) onPreviewReady,
    BasicImageEditor? editImage,
    bool Function()? isCurrent,
  }) async {
    if (editImage != null && isCurrent == null) {
      throw ArgumentError('Avatar editing requires a currentness guard');
    }
    if (isCurrent?.call() == false) return null;
    final file = await _picker.pickImage(source: source);
    if (file == null || isCurrent?.call() == false) return null;

    final mimeType = file.mimeType ?? _media.mimeTypeForFileName(file.name);
    final length = await file.length();
    if (isCurrent?.call() == false) return null;
    final header = await _readHeaderBytes(file);
    if (isCurrent?.call() == false) return null;
    final originalCheck = _media.validateOriginalImage(
      sizeBytes: length,
      fileName: file.name,
      mimeType: mimeType,
      headerBytes: header,
    );
    if (!originalCheck.canPrepare) {
      throw ProfileImagePickException(
        expectedRejection: true,
        reason: originalCheck.rejectedReason == OriginalImageRejection.tooLarge
            ? ProfileImageFailure.fileTooLarge
            : ProfileImageFailure.unknown,
      );
    }

    final bytes = await readBoundedImageFile(
      file,
      maxBytes: _media.config.maxSourceImageBytes,
    );

    if (isCurrent?.call() == false) return null;
    late PreparedImagePayload prepared;
    try {
      prepared = await _media
          .prepareImage(bytes: bytes, fileName: file.name, mimeType: mimeType)
          .future;
    } on ImageSourceDimensionsTooLargeException catch (error) {
      if (error.width * error.height > error.maxPixels) {
        throw const ProfileImagePickException(
          reason: ProfileImageFailure.tooManyPixels,
        );
      }
      rethrow;
    }
    if (isCurrent?.call() == false) return null;
    final preparedCheck = _media.validatePreparedUploadBytes(
      originalBytes: bytes.length,
      preparedBytes: prepared.bytes.length,
      width: prepared.width,
      height: prepared.height,
    );
    if (!preparedCheck.canUpload) {
      throw const ProfileImagePickException();
    }

    if (editImage != null) {
      final output = await editImage(
        BasicImageEditRequest(
          bytes: prepared.bytes,
          mimeType: prepared.mimeType,
          mode: BasicImageEditMode.avatar,
        ),
      );
      if (output == null || isCurrent?.call() == false) return null;
      prepared = await _media
          .prepareImage(
            bytes: output.bytes,
            fileName: output.mimeType == 'image/png'
                ? 'avatar.png'
                : 'avatar.jpg',
            mimeType: output.mimeType,
          )
          .future;
      if (isCurrent?.call() == false) return null;
      final check = _media.validatePreparedUploadBytes(
        originalBytes: output.bytes.length,
        preparedBytes: prepared.bytes.length,
        width: prepared.width,
        height: prepared.height,
      );
      if (!check.canUpload || prepared.width != prepared.height) {
        throw const ProfileImagePickException(expectedRejection: true);
      }
    }
    if (isCurrent?.call() == false) return null;
    onPreviewReady(prepared.bytes);
    return PreparedProfileImage(
      bytes: prepared.bytes,
      mimeType: prepared.mimeType,
      width: prepared.width,
      height: prepared.height,
    );
  }

  Future<Uint8List> _readHeaderBytes(XFile file) async {
    final header = <int>[];
    final length = await file.length();
    await for (final chunk in file.openRead(0, length < 64 ? length : 64)) {
      header.addAll(chunk);
      if (header.length >= 64) break;
    }
    return Uint8List.fromList(
      header.length <= 64 ? header : header.sublist(0, 64),
    );
  }
}

class PreparedProfileImage {
  const new({
    required this.bytes,
    required this.mimeType,
    required this.width,
    required this.height,
  });

  final Uint8List bytes;
  final String mimeType;
  final int width;
  final int height;
}

class ProfileImagePickResult {
  const new({required this.previewBytes, required this.uploaded});

  final Uint8List previewBytes;
  final UploadedImageBlob uploaded;
}

class ProfileImagePickException implements Exception {
  const new({
    this.reason = ProfileImageFailure.unknown,
    this.expectedRejection = false,
  });

  final ProfileImageFailure reason;
  final bool expectedRejection;
}
