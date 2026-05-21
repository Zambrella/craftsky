import 'package:craftsky_app/feed/models/create_post_image.dart';

import 'package:flutter/foundation.dart';

enum DraftImageLifecycle { preparing, uploading, uploaded, failed }

class DraftImageInput {
  const DraftImageInput({
    required this.id,
    required this.fileName,
    required this.mimeType,
    this.previewBytes,
  });

  final String id;
  final String fileName;
  final String mimeType;
  final Uint8List? previewBytes;
}

class UploadedDraftImage {
  const UploadedDraftImage({
    required this.cid,
    required this.mime,
    required this.size,
    this.aspectRatio,
  });

  final String cid;
  final String mime;
  final int size;
  final CreatePostImageAspectRatio? aspectRatio;
}

class DraftImageState {
  const DraftImageState({
    required this.id,
    required this.fileName,
    required this.mimeType,
    required this.lifecycle,
    required this.uploadProgress,
    this.altText = '',
    this.previewBytes,
    this.errorMessage,
    this.uploaded,
  });

  final String id;
  final String fileName;
  final String mimeType;
  final DraftImageLifecycle lifecycle;
  final double uploadProgress;
  final String altText;
  final Uint8List? previewBytes;
  final String? errorMessage;
  final UploadedDraftImage? uploaded;

  DraftImageState copyWith({
    DraftImageLifecycle? lifecycle,
    double? uploadProgress,
    String? altText,
    String? errorMessage,
    bool clearErrorMessage = false,
    UploadedDraftImage? uploaded,
    bool clearUploaded = false,
  }) {
    return DraftImageState(
      id: id,
      fileName: fileName,
      mimeType: mimeType,
      lifecycle: lifecycle ?? this.lifecycle,
      uploadProgress: uploadProgress ?? this.uploadProgress,
      altText: altText ?? this.altText,
      previewBytes: previewBytes,
      errorMessage: clearErrorMessage
          ? null
          : (errorMessage ?? this.errorMessage),
      uploaded: clearUploaded ? null : (uploaded ?? this.uploaded),
    );
  }
}

class ImageDraftController extends ChangeNotifier {
  final List<DraftImageState> _images = [];

  List<DraftImageState> get images => List.unmodifiable(_images);

  void addDraftImage(DraftImageInput input) {
    _images.add(
      DraftImageState(
        id: input.id,
        fileName: input.fileName,
        mimeType: input.mimeType,
        lifecycle: DraftImageLifecycle.preparing,
        uploadProgress: 0,
        altText: '',
        previewBytes: input.previewBytes,
      ),
    );
    notifyListeners();
  }

  void markPrepared(String id) {
    _update(
      id,
      (image) => image.copyWith(
        lifecycle: DraftImageLifecycle.uploading,
        uploadProgress: 0,
        clearErrorMessage: true,
      ),
    );
  }

  void markUploadProgress(String id, double progress) {
    _update(
      id,
      (image) => image.copyWith(
        lifecycle: DraftImageLifecycle.uploading,
        uploadProgress: progress.clamp(0, 1),
      ),
    );
  }

  void markUploaded(String id, UploadedDraftImage uploaded) {
    _update(
      id,
      (image) => image.copyWith(
        lifecycle: DraftImageLifecycle.uploaded,
        uploadProgress: 1,
        uploaded: uploaded,
        clearErrorMessage: true,
      ),
    );
  }

  void markPreparationFailed(String id, String message) {
    _markFailed(id, message);
  }

  void markUploadFailed(String id, String message) {
    _markFailed(id, message);
  }

  void retry(String id) {
    _update(
      id,
      (image) => image.copyWith(
        lifecycle: DraftImageLifecycle.preparing,
        uploadProgress: 0,
        clearErrorMessage: true,
        clearUploaded: true,
      ),
    );
  }

  void remove(String id) {
    _images.removeWhere((image) => image.id == id);
    notifyListeners();
  }

  void setAltText(String id, String value) {
    _update(id, (image) => image.copyWith(altText: value));
  }

  void reorder({required int fromIndex, required int toIndex}) {
    if (fromIndex < 0 || fromIndex >= _images.length) return;
    if (toIndex < 0 || toIndex >= _images.length) return;
    if (fromIndex == toIndex) return;

    final image = _images.removeAt(fromIndex);
    _images.insert(toIndex, image);
    notifyListeners();
  }

  void _markFailed(String id, String message) {
    _update(
      id,
      (image) => image.copyWith(
        lifecycle: DraftImageLifecycle.failed,
        errorMessage: message,
      ),
    );
  }

  void _update(String id, DraftImageState Function(DraftImageState) next) {
    final index = _images.indexWhere((image) => image.id == id);
    if (index < 0) return;
    _images[index] = next(_images[index]);
    notifyListeners();
  }
}
