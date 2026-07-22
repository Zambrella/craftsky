import 'dart:convert';
import 'dart:typed_data';

import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_import_parser.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('InstagramImportParser', () {
    const parser = InstagramImportParser();

    test(
      'UT-009 parses the known following export using only value fields',
      () {
        final bytes = Uint8List.fromList(
          utf8.encode(
            jsonEncode({
              'relationships_following': [
                {
                  'title': 'ignored_display_name',
                  'string_list_data': [
                    {
                      'href': 'https://synthetic.invalid/private-profile',
                      'value': '  @Synthetic.User_2  ',
                      'timestamp': 1,
                    },
                  ],
                },
              ],
              'relationships_followers': [
                {
                  'string_list_data': [
                    {'value': 'synthetic.private.follower'},
                  ],
                },
              ],
              'private_message': 'ignored_private_message',
            }),
          ),
        );

        final result = parser.parseJson(bytes);

        expect(result.entries, [
          const InstagramImportEntry(username: 'synthetic.user_2'),
        ]);
        expect(result.ignoredEntryCount, 0);
        expect(result.duplicateEntryCount, 0);
      },
    );

    test('UT-009 rejects a follower-only export locally', () {
      final bytes = Uint8List.fromList(
        utf8.encode(
          jsonEncode([
            {
              'title': '',
              'media_list_data': [
                {'uri': 'synthetic-private-media'},
              ],
              'string_list_data': [
                {'value': 'SyntheticFollower'},
              ],
            },
          ]),
        ),
      );

      expect(
        () => parser.parseJson(bytes),
        throwsA(
          isA<InstagramImportParseException>().having(
            (error) => error.code,
            'code',
            InstagramImportParseErrorCode.unsupportedShape,
          ),
        ),
      );
    });

    test('UT-009 normalizes, deduplicates, and counts invalid values', () {
      final bytes = Uint8List.fromList(
        utf8.encode(
          jsonEncode({
            'relationships_following': [
              {
                'string_list_data': [
                  {'value': ' @One.User '},
                  {'value': 'one.user'},
                  {'value': 'valid_name2'},
                  {'value': '@@double'},
                  {'value': 'unicode_é'},
                  {'value': 'synthetic_Kelvin'},
                  {'value': 'https://synthetic.invalid/profile'},
                  {'value': ''},
                  {'value': 42},
                ],
              },
            ],
          }),
        ),
      );

      final result = parser.parseJson(bytes);

      expect(
        result.entries.map((entry) => entry.username),
        ['one.user', 'valid_name2'],
      );
      expect(result.duplicateEntryCount, 1);
      expect(result.ignoredEntryCount, 6);
    });

    test('UT-009 categorizes malformed JSON without retaining excerpts', () {
      const privateCanary = 'synthetic_private_archive_canary';
      final bytes = Uint8List.fromList(
        utf8.encode('{"unrelated":"$privateCanary"'),
      );

      expect(
        () => parser.parseJson(bytes),
        throwsA(
          isA<InstagramImportParseException>()
              .having(
                (error) => error.code,
                'code',
                InstagramImportParseErrorCode.invalidJson,
              )
              .having(
                (error) => error.toString(),
                'safe message',
                isNot(contains(privateCanary)),
              ),
        ),
      );
    });

    test('UT-009 rejects changed archive nesting as unsupported', () {
      final bytes = Uint8List.fromList(
        utf8.encode(
          jsonEncode({
            'following': {
              'relationships_following': <Object?>[],
            },
          }),
        ),
      );

      expect(
        () => parser.parseJson(bytes),
        throwsA(
          isA<InstagramImportParseException>().having(
            (error) => error.code,
            'code',
            InstagramImportParseErrorCode.unsupportedShape,
          ),
        ),
      );
    });

    test('UT-009 rejects ZIP input before JSON decoding', () {
      final bytes = Uint8List.fromList([0x50, 0x4b, 0x03, 0x04, 0x00]);

      expect(
        () => parser.parseJson(bytes),
        throwsA(
          isA<InstagramImportParseException>().having(
            (error) => error.code,
            'code',
            InstagramImportParseErrorCode.unsupportedFormat,
          ),
        ),
      );
    });

    test('UT-009 accepts 20 MiB and rejects one byte more', () {
      const maximumBytes = 20 * 1024 * 1024;
      final minimal = utf8.encode('{"relationships_following":[]}');

      Uint8List paddedTo(int length) => Uint8List(length)
        ..fillRange(0, length, 0x20)
        ..setRange(0, minimal.length, minimal);

      final result = parser.parseJson(paddedTo(maximumBytes));
      expect(result.entries, isEmpty);

      expect(
        () => parser.parseJson(paddedTo(maximumBytes + 1)),
        throwsA(
          isA<InstagramImportParseException>().having(
            (error) => error.code,
            'code',
            InstagramImportParseErrorCode.fileTooLarge,
          ),
        ),
      );
    });

    test('UT-009 accepts 10,000 unique entries and rejects 10,001', () {
      Uint8List exportWith(int count) => Uint8List.fromList(
        utf8.encode(
          jsonEncode({
            'relationships_following': List.generate(count, (index) {
              return {
                'string_list_data': [
                  {'value': 'synthetic${index.toString().padLeft(5, '0')}'},
                ],
              };
            }),
          }),
        ),
      );

      final result = parser.parseJson(exportWith(10000));
      expect(result.entries, hasLength(10000));

      expect(
        () => parser.parseJson(exportWith(10001)),
        throwsA(
          isA<InstagramImportParseException>().having(
            (error) => error.code,
            'code',
            InstagramImportParseErrorCode.tooManyEntries,
          ),
        ),
      );
    });

    test('UT-009 parses manual lines as accounts followed', () {
      final result = parser.parseManual(
        ' @Synthetic.One \nsynthetic.two\nsynthetic.one\ninvalid value',
      );

      expect(result.entries, [
        const InstagramImportEntry(username: 'synthetic.one'),
        const InstagramImportEntry(username: 'synthetic.two'),
      ]);
      expect(result.duplicateEntryCount, 1);
      expect(result.ignoredEntryCount, 1);
    });
  });
}
