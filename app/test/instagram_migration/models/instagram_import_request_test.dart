import 'dart:convert';
import 'dart:typed_data';

import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_import_parser.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-010 import requests contain only normalized graph data', () {
    const rawCanaries = [
      'synthetic_raw_filename.json',
      'synthetic_private_media_url',
      'synthetic_private_message',
      'synthetic_profile_value',
    ];
    final parserInput = Uint8List.fromList(
      utf8.encode(
        jsonEncode({
          'relationships_following': [
            {
              'title': rawCanaries[3],
              'string_list_data': [
                {
                  'value': 'Synthetic.Normalized',
                  'href': rawCanaries[1],
                },
              ],
              'message': rawCanaries[2],
            },
          ],
          'filename': rawCanaries[0],
        }),
      ),
    );
    final parsed = const InstagramImportParser().parseJson(parserInput);

    final request = InstagramImportRequest(
      sourceType: InstagramImportSourceType.instagramJson,
      entries: parsed.entries,
    );
    final encoded = jsonEncode(request.toMap());

    expect(request.toMap(), {
      'sourceType': 'instagramJson',
      'entries': [
        {'username': 'synthetic.normalized'},
      ],
    });
    for (final canary in rawCanaries) {
      expect(encoded, isNot(contains(canary)));
      expect(request.toString(), isNot(contains(canary)));
    }
    expect(request.toString(), isNot(contains('synthetic.normalized')));
  });
}
