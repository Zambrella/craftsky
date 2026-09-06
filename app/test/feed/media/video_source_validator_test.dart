import 'dart:typed_data';

import 'package:craftsky_app/feed/media/video_source_validator.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final mp4Header = Uint8List.fromList(const [
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
  ]);

  test('validates source type, bytes, and known duration at exact bounds', () {
    final cases =
        <
          ({
            String name,
            int bytes,
            String mimeType,
            Uint8List header,
            Duration? duration,
            bool valid,
            VideoSourceRejection? rejection,
          })
        >[
          (
            name: 'exact bounds',
            bytes: 300000000,
            mimeType: 'video/mp4',
            header: mp4Header,
            duration: const Duration(seconds: 600),
            valid: true,
            rejection: null,
          ),
          (
            name: 'one byte over',
            bytes: 300000001,
            mimeType: 'video/mp4',
            header: mp4Header,
            duration: const Duration(seconds: 600),
            valid: false,
            rejection: VideoSourceRejection.tooLarge,
          ),
          (
            name: 'one millisecond over',
            bytes: 300000000,
            mimeType: 'video/mp4',
            header: mp4Header,
            duration: const Duration(milliseconds: 600001),
            valid: false,
            rejection: VideoSourceRejection.tooLong,
          ),
          (
            name: 'unknown duration',
            bytes: 300000000,
            mimeType: 'video/mp4',
            header: mp4Header,
            duration: null,
            valid: true,
            rejection: null,
          ),
          (
            name: 'MIME mismatch',
            bytes: 20,
            mimeType: 'video/quicktime',
            header: mp4Header,
            duration: const Duration(seconds: 1),
            valid: false,
            rejection: VideoSourceRejection.unsupportedType,
          ),
          (
            name: 'deceptive extension',
            bytes: 20,
            mimeType: 'video/mp4',
            header: Uint8List.fromList(List<int>.filled(12, 0)),
            duration: const Duration(seconds: 1),
            valid: false,
            rejection: VideoSourceRejection.unsupportedType,
          ),
        ];

    for (final testCase in cases) {
      final result = validateVideoSource(
        sizeBytes: testCase.bytes,
        fileName: 'source.mp4',
        mimeType: testCase.mimeType,
        headerBytes: testCase.header,
        duration: testCase.duration,
      );

      expect(result.canUpload, testCase.valid, reason: testCase.name);
      expect(result.rejectedReason, testCase.rejection, reason: testCase.name);
    }
  });
}
