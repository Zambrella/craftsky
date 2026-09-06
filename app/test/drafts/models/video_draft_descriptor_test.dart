import 'package:craftsky_app/drafts/models/video_draft_descriptor.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'UT-015 source and poster descriptors round-trip without private paths',
    () {
      const descriptor = VideoDraftDescriptor(
        storageRevision: '00000000-0000-4000-8000-000000000001',
        sourceStorageFileName: 'source-revision.mp4',
        sourceByteLength: 123,
        sourceSha256:
            'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
        posterStorageFileName: 'poster-revision.jpg',
        posterByteLength: 12,
        posterSha256:
            'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
        displayFileName: 'private-name.mp4',
        mimeType: 'video/mp4',
        posterMimeType: 'image/jpeg',
        width: 1920,
        height: 1080,
        duration: Duration(seconds: 12),
        altText: 'Spinning wool',
      );

      expect(descriptor..validate(), same(descriptor));
      expect(descriptor.sourceByteLength, 123);
      expect(descriptor.sourceStorageFileName, isNot(contains('/Users/alice')));
      expect(descriptor.toString(), 'VideoDraftDescriptor(<redacted>)');
    },
  );
}
