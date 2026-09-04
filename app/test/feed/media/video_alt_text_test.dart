import 'dart:typed_data';

import 'package:craftsky_app/feed/providers/composer_video_controller.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-002 video alt text counts Unicode grapheme clusters', () async {
    final controller = ComposerVideoController(picker: _Picker());
    await controller.selectExisting();

    controller
      ..setAltText('')
      ..setAltText('e\u0301' * 1000)
      ..setAltText('👩🏽‍🧶' * 1000);

    expect(
      () => controller.setAltText('👩🏽‍🧶' * 1001),
      throwsArgumentError,
    );
  });
}

final class _Picker implements ExistingVideoPicker {
  @override
  Future<LocalVideoSelection> pickExisting() async => LocalVideoSelection(
    displayName: 'video.mp4',
    mimeType: 'video/mp4',
    byteLength: 12,
    duration: const Duration(seconds: 1),
    headerBytes: Uint8List.fromList(const [
      0,
      0,
      0,
      24,
      0x66,
      0x74,
      0x79,
      0x70,
      0x69,
      0x73,
      0x6f,
      0x6d,
    ]),
    openRead: () => const Stream.empty(),
  );
}
