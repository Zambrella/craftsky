import 'dart:io';

import 'package:craftsky_app/chat/data/chat_draft_store.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';

// IT-017 / NFR-002: production storage configuration excludes device transfer.
void main() {
  test('draft namespace is device-local and excluded from Android backup', () {
    const storage = ChatDraftStore.localStorage;
    expect(storage.iOptions.synchronizable, isFalse);
    expect(
      storage.iOptions.accessibility,
      KeychainAccessibility.unlocked_this_device,
    );
    expect(storage.aOptions.storageNamespace, 'craftsky_chat_drafts');
    final manifest = File('android/app/src/main/AndroidManifest.xml')
        .readAsStringSync();
    expect(
      manifest,
      contains('android:fullBackupContent="@xml/chat_backup_rules"'),
    );
    expect(
      manifest,
      contains('android:dataExtractionRules="@xml/chat_data_extraction_rules"'),
    );
    for (final file in ['chat_backup_rules', 'chat_data_extraction_rules']) {
      final rules = File('android/app/src/main/res/xml/$file.xml')
          .readAsStringSync();
      for (final name in [
        'craftsky_chat_drafts',
        'FlutterSecureKeyStorage:craftsky_chat_drafts',
        'FlutterSecureStorageConfiguration:craftsky_chat_drafts',
      ]) {
        expect(rules, contains('domain="sharedpref" path="$name.xml"'));
      }
      if (file == 'chat_data_extraction_rules') {
        expect(rules, contains('<cloud-backup>'));
        expect(rules, contains('<device-transfer>'));
        expect(
          RegExp('path="craftsky_chat_drafts.xml"').allMatches(rules),
          hasLength(2),
        );
      }
    }
  });
}
