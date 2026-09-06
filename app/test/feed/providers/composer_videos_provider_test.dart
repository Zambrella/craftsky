import 'dart:typed_data';

import 'package:craftsky_app/feed/models/video_upload_limits.dart';
import 'package:craftsky_app/feed/providers/composer_video_controller.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('AT-002 selection, replacement and removal stay local', () async {
    final picker = _FakePicker();
    final controller = ComposerVideoController(
      picker: picker,
    );

    await controller.selectExisting();
    expect(controller.selection?.displayName, 'local.mp4');
    await controller.replace();
    expect(picker.calls, 2);
    controller.remove();
    expect(controller.selection, isNull);
  });

  test(
    'AT-003 checks eligibility before picker and again at publish',
    () async {
      var eligibilityChecks = 0;
      final picker = _FakePicker();
      final controller = ComposerVideoController(
        picker: picker,
        checkEligibility: () async {
          eligibilityChecks++;
          return const VideoUploadLimits(
            canUpload: true,
            remainingDailyVideos: 1,
            remainingDailyBytes: 100,
          );
        },
      );

      await controller.selectExisting();
      controller.setAltText('A knitted blue scarf');
      final publishLimits = await controller.recheckEligibility();

      expect(eligibilityChecks, 2);
      expect(publishLimits?.canUpload, isTrue);
      expect(controller.selection?.altText, 'A knitted blue scarf');
    },
  );

  test('AT-003 disabled eligibility does not open picker', () async {
    final picker = _FakePicker();
    final controller = ComposerVideoController(
      picker: picker,
      checkEligibility: () async => const VideoUploadLimits(
        canUpload: false,
        remainingDailyVideos: 0,
        remainingDailyBytes: 0,
      ),
    );

    final limits = await controller.selectExisting();

    expect(limits?.canUpload, isFalse);
    expect(picker.calls, 0);
    expect(controller.selection, isNull);
  });
}

final class _FakePicker implements ExistingVideoPicker {
  int calls = 0;

  @override
  Future<LocalVideoSelection?> pickExisting() async {
    calls++;
    return LocalVideoSelection(
      displayName: 'local.mp4',
      mimeType: 'video/mp4',
      byteLength: 8,
      duration: null,
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
      openRead: () => Stream.value(const [
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
    );
  }
}
