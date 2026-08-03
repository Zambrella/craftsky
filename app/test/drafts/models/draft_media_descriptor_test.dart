import 'package:craftsky_app/drafts/models/draft_manifest_codec.dart';
import 'package:craftsky_app/drafts/models/draft_media_descriptor.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('rejects unsafe or inconsistent persisted media metadata', () {
    final invalid = [
      _descriptor(storageFileName: '../outside.jpg'),
      _descriptor(storageFileName: '/private/outside.jpg'),
      _descriptor(storageFileName: r'nested\outside.jpg'),
      _descriptor(mimeType: 'image/gif'),
      _descriptor(byteLength: 0),
      _descriptor(sha256: 'not-a-sha256'),
      _descriptor(width: 0),
      _descriptor(height: 0),
      _descriptor(order: -1),
      _descriptor(mediaId: 'not-a-uuid'),
      _descriptor(storageRevision: 'not-a-uuid'),
    ];

    for (final descriptor in invalid) {
      expect(
        descriptor.validate,
        throwsA(
          isA<DraftManifestException>().having(
            (error) => error.reason,
            'reason',
            DraftManifestFailureReason.invalidMedia,
          ),
        ),
      );
    }
  });
}

DraftMediaDescriptor _descriptor({
  String mediaId = '00000000-0000-4000-8000-000000000002',
  String storageRevision = '00000000-0000-4000-8000-000000000003',
  String storageFileName =
      '00000000-0000-4000-8000-000000000002-'
      '00000000-0000-4000-8000-000000000003.jpg',
  String mimeType = 'image/jpeg',
  int byteLength = 1234,
  String sha256 =
      '0123456789abcdef0123456789abcdef'
      '0123456789abcdef0123456789abcdef',
  int width = 800,
  int height = 600,
  int order = 0,
}) => DraftMediaDescriptor(
  mediaId: mediaId,
  storageRevision: storageRevision,
  storageFileName: storageFileName,
  displayFileName: 'swatch.jpg',
  mimeType: mimeType,
  byteLength: byteLength,
  sha256: sha256,
  width: width,
  height: height,
  altText: 'Blue knitted swatch',
  order: order,
);
