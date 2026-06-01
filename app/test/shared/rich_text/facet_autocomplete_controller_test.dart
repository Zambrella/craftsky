// Text editing fixtures are clearer without forcing every constructor const.
// ignore_for_file: prefer_const_constructors

import 'package:craftsky_app/shared/rich_text/facet_autocomplete_controller.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('FacetAutocompleteController', () {
    test('UT-011 detects active tokens only at valid boundaries', () {
      expect(
        _activeToken('@ali'),
        isA<ActiveFacetToken>()
            .having((token) => token.kind, 'kind', ActiveFacetTokenKind.mention)
            .having((token) => token.query, 'query', 'ali')
            .having((token) => token.start, 'start', 0)
            .having((token) => token.end, 'end', 4),
      );
      expect(_activeToken('Hi @ali')?.query, 'ali');
      expect(_activeToken('Meet (@ali')?.query, 'ali');
      expect(
        _activeToken('#sock'),
        isA<ActiveFacetToken>()
            .having((token) => token.kind, 'kind', ActiveFacetTokenKind.hashtag)
            .having((token) => token.query, 'query', 'sock'),
      );

      expect(_activeToken('hi@craftsky.social'), isNull);
      expect(_activeToken('https://example.com/#tag'), isNull);
      expect(_activeToken('abc#tag'), isNull);
      expect(_activeToken('@'), isNull);
      expect(_activeToken('#'), isNull);
    });

    test('UT-015 debounces lookups and ignores superseded tokens', () async {
      final debouncer = DebouncedFacetLookup<String>(
        debounce: const Duration(milliseconds: 5),
      );
      final queries = <String>[];

      final first = debouncer.schedule(() async {
        queries.add('a');
        return 'a';
      });
      final second = debouncer.schedule(() async {
        queries.add('al');
        return 'al';
      });

      await Future<void>.delayed(const Duration(milliseconds: 20));

      expect(await first, isNull);
      expect(await second, 'al');
      expect(queries, ['al']);
    });

    test('UT-020 replaces only the active token with one trailing space', () {
      const text = 'Meet (@ali) and #soc';
      final value = TextEditingValue(
        text: text,
        selection: const TextSelection.collapsed(offset: 10),
      );
      final token = FacetAutocompleteController.detectActiveToken(value)!;

      final replaced = FacetAutocompleteController.replaceActiveToken(
        current: value,
        token: token,
        replacementWithSingleTrailingSpace: '@alice.craftsky.social ',
      );

      expect(replaced.text, 'Meet (@alice.craftsky.social ) and #soc');
      expect(
        replaced.selection.baseOffset,
        'Meet (@alice.craftsky.social '.length,
      );

      final hashtagValue = _textEditingValue('Meet @ali and #soc');
      final hashtagToken = FacetAutocompleteController.detectActiveToken(
        hashtagValue,
      )!;
      final hashtagReplaced = FacetAutocompleteController.replaceActiveToken(
        current: hashtagValue,
        token: hashtagToken,
        replacementWithSingleTrailingSpace: '#SockKAL   ',
      );

      expect(hashtagReplaced.text, 'Meet @ali and #SockKAL ');
      expect(hashtagReplaced.selection.baseOffset, hashtagReplaced.text.length);
    });
  });
}

ActiveFacetToken? _activeToken(String text) {
  return FacetAutocompleteController.detectActiveToken(_textEditingValue(text));
}

TextEditingValue _textEditingValue(String text) {
  return TextEditingValue(
    text: text,
    selection: TextSelection.collapsed(offset: text.length),
  );
}
