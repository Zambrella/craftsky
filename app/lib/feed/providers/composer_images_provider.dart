import 'dart:async';
import 'dart:typed_data';

import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/feed/media/composer_image_media_service.dart';
import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/models/post_image_blob.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/post_api_client_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/pipeline/item_pipeline.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:uuid/uuid.dart';

part 'composer_images_provider.g.dart';

const _uploadProgressUpdateInterval = Duration(milliseconds: 100);

final imagePickerProvider = Provider<ImagePicker>((ref) => ImagePicker());

final composerImageMediaServiceProvider = Provider<ComposerImageMediaService>(
  (ref) => const ComposerImageMediaService(),
);

@riverpod
class ComposerImages extends _$ComposerImages {
  final _operations = <String, _ImageOperation>{};
  var _noticeId = 0;

  late PostApiClient _api;
  late ComposerImageMediaService _media;
  late ImagePicker _picker;
  late Uuid _uuid;

  @override
  ComposerImagesState build(String composerId) {
    _api = ref.watch(postApiClientProvider);
    _media = ref.watch(composerImageMediaServiceProvider);
    _picker = ref.watch(imagePickerProvider);
    _uuid = const Uuid();
    ref.onDispose(_cancelAll);
    return const ComposerImagesState(images: []);
  }

  Future<void> addImages() async {
    final remaining = _media.maxImages - state.images.length;
    if (remaining <= 0) {
      _setNotice(
        ImageSelectionLimitNotice(
          id: _nextNoticeId(),
          maxImages: _media.maxImages,
          acceptedCount: 0,
        ),
      );
      return;
    }

    final List<XFile> files;
    try {
      files = await _picker.pickMultiImage();
    } on Object {
      _setNotice(ImagePickerFailedNotice(id: _nextNoticeId()));
      return;
    }
    if (files.isEmpty) return;

    final incoming = [
      for (final file in files)
        _SelectedComposerImage(
          id: _uuid.v4(),
          file: file,
          fileName: file.name,
          mimeType: file.mimeType ?? _media.mimeTypeForFileName(file.name),
        ),
    ];
    final pairs = [
      for (final image in incoming)
        _SelectionPair(
          image,
          LocalImageSelection(name: image.fileName, mimeType: image.mimeType),
        ),
    ];

    final selection = _media.validateSelection(
      existing: [
        for (final image in state.images)
          LocalImageSelection(name: image.fileName, mimeType: image.mimeType),
      ],
      incoming: pairs.map((pair) => pair.selection).toList(),
    );

    _handleRejectedSelections(pairs: pairs, rejected: selection.rejected);

    final accepted = [
      for (final item in selection.accepted)
        pairs.firstWhere((pair) => pair.selection == item).image,
    ];
    if (accepted.isEmpty) return;

    state = state.copyWith(
      images: [
        ...state.images,
        for (final image in accepted)
          ComposerImageDraft(
            id: image.id,
            fileName: image.fileName,
            mimeType: image.mimeType,
            altText: '',
            phase: const ImageQueued(),
          ),
      ],
    );

    final jobs = <_ComposerImagePipelineItem>[];
    for (final image in accepted) {
      final operation = _ImageOperation(uploadCancelToken: CancelToken());
      _operations[image.id] = operation;
      jobs.add(
        _ComposerImagePipelineItem(
          id: image.id,
          file: image.file,
          fileName: image.fileName,
          mimeType: image.mimeType,
          operation: operation,
        ),
      );
    }
    _startPipeline(jobs);
  }

  void remove(String imageId) {
    _operations.remove(imageId)?.cancel();
    state = state.copyWith(
      images: state.images.where((image) => image.id != imageId).toList(),
    );
  }

  void setAltText(String imageId, String value) {
    _updateImage(imageId, (image) => image.copyWith(altText: value));
  }

  void reorder({required int fromIndex, required int toIndex}) {
    if (fromIndex < 0 || fromIndex >= state.images.length) return;
    if (toIndex < 0 || toIndex >= state.images.length) return;
    if (fromIndex == toIndex) return;

    final images = [...state.images];
    final image = images.removeAt(fromIndex);
    images.insert(toIndex, image);
    state = state.copyWith(images: images);
  }

  void retry(String imageId) {
    final image = state.images.firstWhere((image) => image.id == imageId);
    if (image.phase case ImageFailed(:final failure) when failure.canRetry) {
      final operation = _ImageOperation(uploadCancelToken: CancelToken());
      _operations[image.id] = operation;
      _updateImage(
        imageId,
        (image) => image.copyWith(phase: const ImageQueued()),
      );
      final job = _ComposerImagePipelineItem(
        id: image.id,
        file: null,
        fileName: image.fileName,
        mimeType: image.mimeType,
        operation: operation,
        originalBytes: image.previewBytes,
      );
      _startPipeline([job]);
    }
  }

  void clearNotice(int noticeId) {
    if (state.notice?.id != noticeId) return;
    state = state.copyWith(notice: null);
  }

  void _startPipeline(List<_ComposerImagePipelineItem> jobs) {
    unawaited(
      _consumePipeline(
        runPipeline<_ComposerImagePipelineItem>(
          items: jobs,
          steps: _imagePipelineSteps(),
          concurrency: _media.maxImages,
        ),
      ),
    );
  }

  List<PipelineStep<_ComposerImagePipelineItem>> _imagePipelineSteps() {
    return [
      PipelineStep(
        name: _ImagePipelineStepNames.read,
        run: _readImageBytes,
      ),
      PipelineStep(
        name: _ImagePipelineStepNames.prepare,
        run: _prepareImage,
      ),
      PipelineStep(
        name: _ImagePipelineStepNames.validatePrepared,
        run: _validatePreparedImage,
      ),
      PipelineStep(
        name: _ImagePipelineStepNames.upload,
        run: _uploadImage,
      ),
    ];
  }

  Future<void> _consumePipeline(
    Stream<PipelineEvent<_ComposerImagePipelineItem>> events,
  ) async {
    await for (final event in events) {
      if (!ref.mounted) return;
      _handlePipelineEvent(event);
    }
  }

  void _handlePipelineEvent(PipelineEvent<_ComposerImagePipelineItem> event) {
    if (!_isActive(event.item.id, event.item.operation)) return;
    switch (event) {
      case StepStarted(:final stepName):
        _handleStepStarted(event.item.id, stepName);
      case StepProgress(:final stepName, :final progress):
        _handleStepProgress(event.item.id, stepName, progress);
      case StepCompleted(:final stepName, :final result):
        _handleStepCompleted(event.item.id, stepName, result);
      case StepFailed(:final stepName, :final error):
        _handleStepFailed(event.item.id, stepName, error);
      case ItemCompleted(:final result):
        _handleItemCompleted(result);
    }
  }

  void _handleStepStarted(String imageId, String stepName) {
    switch (stepName) {
      case _ImagePipelineStepNames.read:
        _setPhase(imageId, const ImageReading());
      case _ImagePipelineStepNames.prepare:
        _setPhase(imageId, const ImagePreparing());
      case _ImagePipelineStepNames.upload:
        _setPhase(imageId, const ImageUploading(TransferStarting()));
      case _ImagePipelineStepNames.validatePrepared:
        break;
    }
  }

  void _handleStepProgress(String imageId, String stepName, Object progress) {
    if (stepName != _ImagePipelineStepNames.upload) return;
    if (progress case final ImageTransferProgress transferProgress) {
      _setPhase(imageId, ImageUploading(transferProgress));
    }
  }

  void _handleStepCompleted(
    String imageId,
    String stepName,
    _ComposerImagePipelineItem result,
  ) {
    switch (stepName) {
      case _ImagePipelineStepNames.read:
        final bytes = result.originalBytes;
        if (bytes == null) return;
        _updateImage(
          imageId,
          (draft) => draft.copyWith(
            previewBytes: Uint8List.fromList(bytes),
            previewAspectRatio: result.previewAspectRatio,
          ),
        );
      case _ImagePipelineStepNames.prepare:
        final prepared = result.prepared;
        if (prepared == null) return;
        _updateImage(
          imageId,
          (draft) => draft.copyWith(
            previewAspectRatio: _media.aspectRatioFor(
              width: prepared.width,
              height: prepared.height,
            ),
          ),
        );
      case _ImagePipelineStepNames.validatePrepared ||
          _ImagePipelineStepNames.upload:
        break;
    }
  }

  void _handleStepFailed(String imageId, String stepName, Object error) {
    if (error is ApiCanceled) return;
    if (error is _OriginalImageTooLarge || error is _PreparedImageTooLarge) {
      _fail(imageId, ImageTooLarge(_media.maxImageBytes));
      return;
    }
    if (error is _UnsupportedOriginalImage) {
      _fail(imageId, const UnsupportedImageType());
      return;
    }

    switch (stepName) {
      case _ImagePipelineStepNames.read || _ImagePipelineStepNames.prepare:
        _fail(imageId, const ImagePreparationFailed());
      case _ImagePipelineStepNames.validatePrepared:
        _fail(imageId, ImageTooLarge(_media.maxImageBytes));
      case _ImagePipelineStepNames.upload:
        _fail(imageId, const ImageUploadFailed());
    }
  }

  void _handleItemCompleted(_ComposerImagePipelineItem result) {
    final prepared = result.prepared;
    final uploaded = result.uploaded;
    if (prepared == null || uploaded == null) return;

    _operations.remove(result.id);
    _setPhase(
      result.id,
      ImageUploaded(
        UploadedDraftImage(
          cid: uploaded.cid,
          mime: uploaded.mime,
          size: uploaded.size,
          aspectRatio: _media.aspectRatioFor(
            width: prepared.width,
            height: prepared.height,
          ),
        ),
      ),
    );
  }

  Future<_ComposerImagePipelineItem> _readImageBytes(
    _ComposerImagePipelineItem item,
    PipelineStepContext<_ComposerImagePipelineItem> context,
  ) async {
    final bytes = item.originalBytes ?? await _readValidatedFileBytes(item);
    if (item.originalBytes != null) {
      _throwIfInvalidOriginal(
        item: item,
        sizeBytes: bytes.length,
        headerBytes: _headerFromBytes(bytes),
      );
    }
    final job = _media.inspectImage(
      bytes: Uint8List.fromList(bytes),
      fileName: item.fileName,
      mimeType: item.mimeType,
    );
    item.operation.inspectionJob = job;
    final inspected = await job.future;
    item.operation.inspectionJob = null;
    return item.copyWith(
      originalBytes: Uint8List.fromList(bytes),
      previewAspectRatio: _media.aspectRatioFor(
        width: inspected.width,
        height: inspected.height,
      ),
    );
  }

  Future<Uint8List> _readValidatedFileBytes(
    _ComposerImagePipelineItem item,
  ) async {
    final file = item.file!;
    final sizeBytes = await file.length();
    _throwIfInvalidOriginal(
      item: item,
      sizeBytes: sizeBytes,
      headerBytes: Uint8List(0),
    );

    final headerBytes = await _readHeaderBytes(file);
    _throwIfInvalidOriginal(
      item: item,
      sizeBytes: sizeBytes,
      headerBytes: headerBytes,
    );

    final bytes = await file.readAsBytes();
    _throwIfInvalidOriginal(
      item: item,
      sizeBytes: bytes.length,
      headerBytes: _headerFromBytes(bytes),
    );
    return bytes;
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

  Uint8List _headerFromBytes(Uint8List bytes) {
    return Uint8List.sublistView(
      bytes,
      0,
      bytes.length < 16 ? bytes.length : 16,
    );
  }

  void _throwIfInvalidOriginal({
    required _ComposerImagePipelineItem item,
    required int sizeBytes,
    required Uint8List headerBytes,
  }) {
    final validation = _media.validateOriginalImage(
      sizeBytes: sizeBytes,
      fileName: item.fileName,
      mimeType: item.mimeType,
      headerBytes: headerBytes,
    );
    if (validation.canPrepare) return;

    switch (validation.rejectedReason) {
      case OriginalImageRejection.tooLarge:
        throw const _OriginalImageTooLarge();
      case OriginalImageRejection.unsupportedType:
        throw const _UnsupportedOriginalImage();
      case null:
        throw const _UnsupportedOriginalImage();
    }
  }

  Future<_ComposerImagePipelineItem> _prepareImage(
    _ComposerImagePipelineItem item,
    PipelineStepContext<_ComposerImagePipelineItem> context,
  ) async {
    final bytes = item.originalBytes!;
    final job = _media.prepareImage(
      bytes: Uint8List.fromList(bytes),
      fileName: item.fileName,
      mimeType: item.mimeType,
    );
    item.operation.preparationJob = job;
    final prepared = await job.future;
    item.operation.preparationJob = null;
    return item.copyWith(prepared: prepared);
  }

  Future<_ComposerImagePipelineItem> _validatePreparedImage(
    _ComposerImagePipelineItem item,
    PipelineStepContext<_ComposerImagePipelineItem> context,
  ) async {
    final sizeCheck = _media.validatePreparedUploadBytes(
      originalBytes: item.originalBytes!.length,
      preparedBytes: item.prepared!.bytes.length,
    );
    if (!sizeCheck.canUpload) throw const _PreparedImageTooLarge();
    return item;
  }

  Future<_ComposerImagePipelineItem> _uploadImage(
    _ComposerImagePipelineItem item,
    PipelineStepContext<_ComposerImagePipelineItem> context,
  ) async {
    final prepared = item.prepared!;
    final uploaded = await _api.uploadImage(
      bytes: prepared.bytes,
      mimeType: prepared.mimeType,
      cancelToken: item.operation.uploadCancelToken,
      onSendProgress: (sent, total) {
        if (item.operation.updateSendProgress(sent: sent, total: total)) {
          context.reportProgress(item.operation.toTransferProgress());
        }
      },
      onReceiveProgress: (received, total) {
        if (item.operation.updateReceiveProgress(
          received: received,
          total: total,
        )) {
          context.reportProgress(item.operation.toTransferProgress());
        }
      },
    );
    return item.copyWith(uploaded: uploaded);
  }

  void _handleRejectedSelections({
    required List<_SelectionPair> pairs,
    required List<RejectedImageSelection> rejected,
  }) {
    var unsupported = 0;
    var limitExceeded = false;
    for (final item in rejected) {
      switch (item.reason) {
        case ImageSelectionRejection.imageLimitExceeded:
          limitExceeded = true;
        case ImageSelectionRejection.unsupportedType:
          unsupported += 1;
      }
    }

    if (limitExceeded) {
      _setNotice(
        ImageSelectionLimitNotice(
          id: _nextNoticeId(),
          maxImages: _media.maxImages,
          acceptedCount: pairs.length - rejected.length,
        ),
      );
    } else if (unsupported > 0) {
      _setNotice(
        UnsupportedImagesNotice(id: _nextNoticeId(), count: unsupported),
      );
    }
  }

  void _setNotice(ComposerImageNotice notice) {
    state = state.copyWith(notice: notice);
  }

  int _nextNoticeId() => ++_noticeId;

  void _setPhase(String imageId, ComposerImagePhase phase) {
    _updateImage(imageId, (image) => image.copyWith(phase: phase));
  }

  void _fail(String imageId, ComposerImageFailure failure) {
    _operations.remove(imageId);
    _setPhase(imageId, ImageFailed(failure));
  }

  void _updateImage(
    String imageId,
    ComposerImageDraft Function(ComposerImageDraft image) update,
  ) {
    final index = state.images.indexWhere((image) => image.id == imageId);
    if (index < 0) return;
    final images = [...state.images];
    images[index] = update(images[index]);
    state = state.copyWith(images: images);
  }

  bool _isActive(String imageId, _ImageOperation operation) {
    return !operation.canceled && _operations[imageId] == operation;
  }

  void _cancelAll() {
    for (final operation in _operations.values) {
      operation.cancel();
    }
    _operations.clear();
  }
}

final class _SelectedComposerImage {
  const _SelectedComposerImage({
    required this.id,
    required this.file,
    required this.fileName,
    required this.mimeType,
  });

  final String id;
  final XFile? file;
  final String fileName;
  final String mimeType;
}

final class _ComposerImagePipelineItem {
  const _ComposerImagePipelineItem({
    required this.id,
    required this.file,
    required this.fileName,
    required this.mimeType,
    required this.operation,
    this.originalBytes,
    this.previewAspectRatio,
    this.prepared,
    this.uploaded,
  });

  final String id;
  final XFile? file;
  final String fileName;
  final String mimeType;
  final _ImageOperation operation;
  final Uint8List? originalBytes;
  final CreatePostImageAspectRatio? previewAspectRatio;
  final PreparedImagePayload? prepared;
  final UploadedImageBlob? uploaded;

  _ComposerImagePipelineItem copyWith({
    Uint8List? originalBytes,
    CreatePostImageAspectRatio? previewAspectRatio,
    PreparedImagePayload? prepared,
    UploadedImageBlob? uploaded,
  }) {
    return _ComposerImagePipelineItem(
      id: id,
      file: file,
      fileName: fileName,
      mimeType: mimeType,
      operation: operation,
      originalBytes: originalBytes ?? this.originalBytes,
      previewAspectRatio: previewAspectRatio ?? this.previewAspectRatio,
      prepared: prepared ?? this.prepared,
      uploaded: uploaded ?? this.uploaded,
    );
  }
}

abstract final class _ImagePipelineStepNames {
  static const read = 'read';
  static const prepare = 'prepare';
  static const validatePrepared = 'validatePrepared';
  static const upload = 'upload';
}

final class _PreparedImageTooLarge implements Exception {
  const _PreparedImageTooLarge();
}

final class _OriginalImageTooLarge implements Exception {
  const _OriginalImageTooLarge();
}

final class _UnsupportedOriginalImage implements Exception {
  const _UnsupportedOriginalImage();
}

final class _SelectionPair {
  const _SelectionPair(this.image, this.selection);

  final _SelectedComposerImage image;
  final LocalImageSelection selection;
}

final class _ImageOperation {
  _ImageOperation({required this.uploadCancelToken});

  final CancelToken uploadCancelToken;
  ImageInspectionJob? inspectionJob;
  ImagePreparationJob? preparationJob;
  bool canceled = false;
  int sent = 0;
  int sendTotal = 0;
  int received = 0;
  int receiveTotal = 0;
  DateTime? _lastProgressReportedAt;
  bool _hasReportedProgress = false;
  bool _sendFinishedReported = false;
  bool _receiveStartedReported = false;
  bool _receiveFinishedReported = false;

  void cancel() {
    canceled = true;
    inspectionJob?.cancel();
    preparationJob?.cancel();
    uploadCancelToken.cancel('image removed');
  }

  ImageTransferProgress toTransferProgress() {
    return TransferBytes(
      sent: sent,
      sendTotal: sendTotal,
      received: received,
      receiveTotal: receiveTotal,
    );
  }

  bool updateSendProgress({required int sent, required int total}) {
    this.sent = sent;
    sendTotal = total;

    final sendFinished = total > 0 && sent >= total;
    final force = sendFinished && !_sendFinishedReported;
    if (sendFinished) _sendFinishedReported = true;

    return _shouldReportProgress(force: force);
  }

  bool updateReceiveProgress({required int received, required int total}) {
    this.received = received;
    receiveTotal = total;

    final receiveStarted = received > 0;
    final receiveFinished = total > 0 && received >= total;
    final force =
        (receiveStarted && !_receiveStartedReported) ||
        (receiveFinished && !_receiveFinishedReported);
    if (receiveStarted) _receiveStartedReported = true;
    if (receiveFinished) _receiveFinishedReported = true;

    return _shouldReportProgress(force: force);
  }

  bool _shouldReportProgress({required bool force}) {
    final now = DateTime.now();
    final lastReportedAt = _lastProgressReportedAt;
    if (!_hasReportedProgress ||
        force ||
        lastReportedAt == null ||
        now.difference(lastReportedAt) >= _uploadProgressUpdateInterval) {
      _hasReportedProgress = true;
      _lastProgressReportedAt = now;
      return true;
    }
    return false;
  }
}
