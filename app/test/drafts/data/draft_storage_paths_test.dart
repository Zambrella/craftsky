import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/drafts/data/draft_storage_paths.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:path/path.dart' as p;

void main() {
  test('builds deterministic account-scoped paths from validated IDs', () {
    final paths = DraftStoragePaths(
      documentsRoot: '/documents',
      owner: AccountKey('did:plc:private-owner-canary'),
    );
    const draftId = '00000000-0000-4000-8000-000000000001';

    expect(paths.accountRoot, startsWith('/documents/CraftSky/drafts/v1/'));
    expect(paths.accountRoot, isNot(contains('did:plc:private-owner-canary')));
    expect(
      p.isWithin(paths.accountRoot, paths.draftDirectory(draftId)),
      isTrue,
    );
    expect(paths.manifestPath(draftId), endsWith('/manifest.json'));
    expect(
      paths.mediaFilePath(draftId, 'safe-image.jpg'),
      endsWith('/media/safe-image.jpg'),
    );
    expect(paths.toString(), 'DraftStoragePaths(<redacted>)');
  });

  test('rejects caller-controlled traversal components', () {
    final paths = DraftStoragePaths(
      documentsRoot: '/documents',
      owner: AccountKey('did:plc:alice'),
    );

    for (final invalidId in ['../outside', '/absolute', 'not-a-uuid']) {
      expect(
        () => paths.draftDirectory(invalidId),
        throwsA(isA<DraftPathException>()),
      );
    }
    for (final invalidFile in ['../outside.jpg', '/absolute.jpg', r'a\b.jpg']) {
      expect(
        () => paths.mediaFilePath(
          '00000000-0000-4000-8000-000000000001',
          invalidFile,
        ),
        throwsA(isA<DraftPathException>()),
      );
    }
  });
}
