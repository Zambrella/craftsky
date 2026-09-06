import 'dart:typed_data';

import 'package:characters/characters.dart';
import 'package:craftsky_app/feed/media/video_source_validator.dart';
import 'package:craftsky_app/feed/models/video_upload_limits.dart';

final class LocalVideoSelection {
  const LocalVideoSelection({
    required this.displayName,
    required this.mimeType,
    required this.byteLength,
    required this.duration,
    required this.headerBytes,
    required this.openRead,
    this.width = 1,
    this.height = 1,
    this.posterBytes,
    this.altText = '',
  });

  final String displayName;
  final String mimeType;
  final int byteLength;
  final Duration? duration;
  final int width;
  final int height;
  final Uint8List headerBytes;
  final Stream<List<int>> Function() openRead;
  final Uint8List? posterBytes;
  final String altText;

  LocalVideoSelection withAltText(String value) => LocalVideoSelection(
    displayName: displayName,
    mimeType: mimeType,
    byteLength: byteLength,
    duration: duration,
    width: width,
    height: height,
    headerBytes: headerBytes,
    openRead: openRead,
    posterBytes: posterBytes,
    altText: value,
  );

  @override
  String toString() => 'LocalVideoSelection(<redacted>)';
}

// The interface gives platform pickers a replaceable test seam.
// ignore: one_member_abstracts
abstract interface class ExistingVideoPicker {
  Future<LocalVideoSelection?> pickExisting();
}

final class ComposerVideoController {
  factory ComposerVideoController({
    required ExistingVideoPicker picker,
    Future<VideoUploadLimits> Function()? checkEligibility,
  }) => ComposerVideoController._(picker, checkEligibility);

  ComposerVideoController._(
    this._picker,
    this._checkEligibility,
  );

  final ExistingVideoPicker _picker;
  final Future<VideoUploadLimits> Function()? _checkEligibility;
  LocalVideoSelection? _selection;

  LocalVideoSelection? get selection => _selection;

  Future<VideoUploadLimits?> selectExisting() async {
    final limits = await _checkEligibility?.call();
    if (limits != null && !limits.canUpload) return limits;
    final selected = await _picker.pickExisting();
    if (selected == null) return limits;
    final validation = validateVideoSource(
      sizeBytes: selected.byteLength,
      fileName: selected.displayName,
      mimeType: selected.mimeType,
      headerBytes: selected.headerBytes,
      duration: selected.duration,
    );
    if (!validation.canUpload) throw ArgumentError('Invalid video source');
    _selection = selected;
    return limits;
  }

  Future<VideoUploadLimits?> replace() => selectExisting();

  LocalVideoSelection? get restoredSelection => _selection;
  set restoredSelection(LocalVideoSelection value) => _selection = value;

  void remove() => _selection = null;

  void setAltText(String value) {
    if (value.characters.length > 1000) {
      throw ArgumentError('Video alt text is too long');
    }
    _selection = _selection?.withAltText(value);
  }

  Future<VideoUploadLimits?> recheckEligibility() async =>
      _checkEligibility?.call();
}
