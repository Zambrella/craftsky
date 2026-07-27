import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:archive/archive.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_export_file_parser.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_selected_export_parser_native.dart';
import 'package:file_selector/file_selector.dart' show XFile;
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'IT-023 ZIP crosses the repository boundary as normalized JSON evidence',
    () async {
      const pathCanary = 'private-path-canary';
      const unrelatedCanary = 'private-unrelated-message-canary';
      const titleCanary = 'Synthetic.Private_User';
      final directory = Directory.systemTemp.createTempSync(
        'craftsky-$pathCanary-',
      );
      addTearDown(() => directory.deleteSync(recursive: true));
      final following = Uint8List.fromList(
        utf8.encode(
          jsonEncode({
            'relationships_following': [
              {
                'title': titleCanary,
                'string_list_data': [
                  {
                    'href':
                        'https://www.instagram.com/_u/synthetic.private_user',
                    'timestamp': 1,
                  },
                ],
              },
            ],
          }),
        ),
      );
      final archive = Archive()
        ..addFile(
          ArchiveFile.bytes(
            InstagramExportFileParser.followingEntryPath,
            following,
          ),
        )
        ..addFile(
          ArchiveFile.string(
            'messages/inbox/$unrelatedCanary.json',
            unrelatedCanary,
          ),
        );
      final file = File('${directory.path}/$pathCanary.zip')
        ..writeAsBytesSync(ZipEncoder().encodeBytes(archive));

      final parsed = await parseSelectedInstagramExport(XFile(file.path));
      final request = InstagramImportRequest(
        sourceType: InstagramImportSourceType.instagramJson,
        entries: parsed.entries,
      );
      final wire = jsonEncode(request.toMap());

      expect(jsonDecode(wire), {
        'sourceType': 'instagramJson',
        'entries': [
          {'username': 'synthetic.private_user'},
        ],
      });
      expect(wire, isNot(contains('instagramZip')));
      expect(wire, isNot(contains(pathCanary)));
      expect(wire, isNot(contains(unrelatedCanary)));
      expect(wire, isNot(contains(titleCanary)));
      expect(wire, isNot(contains('instagram.com')));
    },
  );
}
