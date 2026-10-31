import 'package:craftsky_app/chat/models/text_validation.dart';
import 'package:flutter_test/flutter_test.dart';

// UT-002 / FR-005 / AC-006. These independently exercise the Flutter path.
void main() {
  test('message text preserves independent grapheme and UTF-8 limits', () {
    final byteBoundary = 'a${'\u0301' * 4999}';
    final cases = <(String, String, bool)>[
      ('empty', '', false),
      ('whitespace', ' \n\t\u2003', false),
      ('byte order mark only', '\ufeff', false),
      ('ordinary', ' knitting plans ', true),
      ('999 graphemes', 'x' * 999, true),
      ('1000 graphemes', 'x' * 1000, true),
      ('1001 graphemes', 'x' * 1001, false),
      ('1000 emoji', '🧶' * 1000, true),
      ('1001 emoji', '🧶' * 1001, false),
      ('combining clusters', 'e\u0301' * 1000, true),
      ('ZWJ clusters', '👩‍👩‍👧‍👦' * 400, true),
      ('9999 bytes', byteBoundary, true),
      ('10000 bytes', '${byteBoundary}b', true),
      ('10001 bytes', '${byteBoundary}bc', false),
      ('unpaired surrogate', String.fromCharCode(0xd800), false),
    ];
    for (final (name, text, valid) in cases) {
      expect(validateMessageText(text) == null, valid, reason: name);
    }
  });
}
