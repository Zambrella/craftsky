import 'package:craftsky_app/feed/media/image_upload_preparer.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('validatePreparedUploadSize', () {
    test('accepts prepared bytes exactly at the limit', () {
      final result = validatePreparedUploadSize(
        preparedBytes: 15 * 1024 * 1024,
      );
      expect(result, isTrue);
    });

    test('rejects prepared bytes over the limit', () {
      final result = validatePreparedUploadSize(
        preparedBytes: (15 * 1024 * 1024) + 1,
      );
      expect(result, isFalse);
    });

    test(
      'decision is based on prepared bytes, not original metadata bytes',
      () {
        final result = validatePreparedUpload(
          originalBytes: 20 * 1024 * 1024,
          preparedBytes: 10 * 1024 * 1024,
        );
        expect(result.canUpload, isTrue);
        expect(result.rejectedReason, isNull);
      },
    );
  });
}
