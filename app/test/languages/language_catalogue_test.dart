import 'dart:convert';

import 'package:craftsky_app/languages/data/language_catalogue.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-008 pins the exact v1 base-language snapshot', () {
    expect(supportedLanguageTags.length, 184);
    expect(
      _catalogueFingerprint(supportedLanguageTags).toRadixString(16),
      '5a751f77a5ee754c',
    );
    expect(supportedLanguageTags, containsAll(['en', 'fr', 'cy', 'zh']));
    expect(
      supportedLanguageTags.every(
        (tag) => RegExp(r'^[a-z]{2}$').hasMatch(tag),
      ),
      isTrue,
    );
  });

  test('UT-009 every selectable language has a friendly English label', () {
    for (final tag in supportedLanguageTags) {
      expect(
        languageLabel(tag),
        isNot(tag),
        reason: '$tag must not render as its raw selectable code',
      );
    }
    expect(languageLabel('aa'), 'Afar');
    expect(languageLabel('zu'), 'Zulu');
  });

  test('UT-009 preserved external tags use a lossless fallback label', () {
    expect(languageLabel('fr'), 'French');
    expect(languageLabel('fr-CA'), isNotEmpty);
    expect(languageLabel('x-future'), 'x-future');
  });
}

BigInt _catalogueFingerprint(Set<String> tags) {
  var hash = BigInt.parse('cbf29ce484222325', radix: 16);
  final prime = BigInt.parse('100000001b3', radix: 16);
  final mask = BigInt.parse('ffffffffffffffff', radix: 16);
  for (final byte in utf8.encode((tags.toList()..sort()).join('\n'))) {
    hash ^= BigInt.from(byte);
    hash = (hash * prime) & mask;
  }
  return hash;
}
