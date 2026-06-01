// Test fixture readability is preferred over const/raw-string lint noise here.
// ignore_for_file: prefer_const_constructors, use_raw_strings

import 'dart:convert';

import 'package:craftsky_app/shared/rich_text/facet_generator.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('FacetGenerator', () {
    test(
      'UT-002 computes UTF-8 byte offsets after emoji and multibyte text',
      () async {
        const text = '🧶 café @alice.craftsky.social #Mending';
        final generator = FacetGenerator(
          mentionResolver: _MapMentionResolver({
            'alice.craftsky.social': 'did:plc:alice',
          }),
        );

        final facets = await generator.generate(text);

        final mention = _singleFacetWithType(
          facets,
          'app.bsky.richtext.facet#mention',
        );
        expect(
          mention['index'],
          _byteRangeForVisibleToken(text, '@alice.craftsky.social'),
        );
        expect(
          _firstFeature(mention),
          {'\$type': 'app.bsky.richtext.facet#mention', 'did': 'did:plc:alice'},
        );

        final tag = _singleFacetWithType(
          facets,
          'app.bsky.richtext.facet#tag',
        );
        expect(tag['index'], _byteRangeForVisibleToken(text, '#Mending'));
        expect(
          _firstFeature(tag),
          {'\$type': 'app.bsky.richtext.facet#tag', 'tag': 'Mending'},
        );

        final byteStarts = _byteStarts(facets);
        expect(byteStarts, orderedEquals([...byteStarts]..sort()));
      },
    );

    test(
      'UT-001 generates mention facets only for locally resolved handles',
      () async {
        const text = 'Hi @alice.craftsky.social @unknown.example';
        final generator = FacetGenerator(
          mentionResolver: _MapMentionResolver({
            'alice.craftsky.social': 'did:plc:alice',
          }),
        );

        final facets = await generator.generate(text);
        final mentions = _facetsWithType(
          facets,
          'app.bsky.richtext.facet#mention',
        );

        expect(mentions, hasLength(1));
        expect(
          mentions.single['index'],
          _byteRangeForVisibleToken(text, '@alice.craftsky.social'),
        );
        expect(
          _firstFeature(mentions.single),
          {'\$type': 'app.bsky.richtext.facet#mention', 'did': 'did:plc:alice'},
        );
        expect(text, contains('@unknown.example'));
      },
    );

    test('UT-004 recognizes HTTP, HTTPS, and bare-domain links', () async {
      const text = 'http://a.example https://b.example craftsky.social';
      final generator = FacetGenerator(
        mentionResolver: const _MapMentionResolver({}),
      );

      final facets = await generator.generate(text);
      final links = _facetsWithType(
        facets,
        'app.bsky.richtext.facet#link',
      );

      expect(links, hasLength(3));
      _expectLinkFacet(
        links[0],
        text: text,
        visibleToken: 'http://a.example',
        uri: 'http://a.example',
      );
      _expectLinkFacet(
        links[1],
        text: text,
        visibleToken: 'https://b.example',
        uri: 'https://b.example',
      );
      _expectLinkFacet(
        links[2],
        text: text,
        visibleToken: 'craftsky.social',
        uri: 'https://craftsky.social',
      );
    });

    test('UT-005 trims trailing punctuation from link facets', () async {
      const text = 'See craftsky.social, (https://example.com/path).';
      final generator = FacetGenerator(
        mentionResolver: const _MapMentionResolver({}),
      );

      final facets = await generator.generate(text);
      final links = _facetsWithType(
        facets,
        'app.bsky.richtext.facet#link',
      );

      expect(links, hasLength(2));
      _expectLinkFacet(
        links[0],
        text: text,
        visibleToken: 'craftsky.social',
        uri: 'https://craftsky.social',
      );
      _expectLinkFacet(
        links[1],
        text: text,
        visibleToken: 'https://example.com/path',
        uri: 'https://example.com/path',
      );
    });

    test('UT-018 parses hashtag characters per slice rules', () async {
      const text = '#SockKAL #café_2026 #sock-knit #🧶craft';
      final generator = FacetGenerator(
        mentionResolver: const _MapMentionResolver({}),
      );

      final facets = await generator.generate(text);
      final tags = _facetsWithType(facets, 'app.bsky.richtext.facet#tag');

      expect(tags, hasLength(3));
      _expectTagFacet(
        tags[0],
        text: text,
        visibleToken: '#SockKAL',
        tag: 'SockKAL',
      );
      _expectTagFacet(
        tags[1],
        text: text,
        visibleToken: '#café_2026',
        tag: 'café_2026',
      );
      _expectTagFacet(tags[2], text: text, visibleToken: '#sock', tag: 'sock');
    });

    test('UT-003 avoids overlapping tag facets inside URL fragments', () async {
      const text = 'https://craftsky.social/#SockKAL #SockKAL';
      final generator = FacetGenerator(
        mentionResolver: const _MapMentionResolver({}),
      );

      final facets = await generator.generate(text);
      final links = _facetsWithType(
        facets,
        'app.bsky.richtext.facet#link',
      );
      final tags = _facetsWithType(facets, 'app.bsky.richtext.facet#tag');

      expect(links, hasLength(1));
      _expectLinkFacet(
        links.single,
        text: text,
        visibleToken: 'https://craftsky.social/#SockKAL',
        uri: 'https://craftsky.social/#SockKAL',
      );
      expect(tags, hasLength(1));
      _expectTagFacet(
        tags.single,
        text: text,
        visibleToken: '#SockKAL',
        tag: 'SockKAL',
        occurrence: 2,
      );
    });
  });
}

class _MapMentionResolver implements MentionResolver {
  const _MapMentionResolver(this._didsByHandle);

  final Map<String, String> _didsByHandle;

  @override
  Future<String?> didForHandle(String handle) async => _didsByHandle[handle];
}

Map<String, dynamic> _singleFacetWithType(
  List<Map<String, dynamic>> facets,
  String type,
) {
  return _facetsWithType(facets, type).single;
}

List<Map<String, dynamic>> _facetsWithType(
  List<Map<String, dynamic>> facets,
  String type,
) {
  return facets
      .where((facet) => (_firstFeature(facet)['\$type'] as String?) == type)
      .toList();
}

void _expectLinkFacet(
  Map<String, dynamic> facet, {
  required String text,
  required String visibleToken,
  required String uri,
}) {
  expect(facet['index'], _byteRangeForVisibleToken(text, visibleToken));
  expect(
    _firstFeature(facet),
    {'\$type': 'app.bsky.richtext.facet#link', 'uri': uri},
  );
}

void _expectTagFacet(
  Map<String, dynamic> facet, {
  required String text,
  required String visibleToken,
  required String tag,
  int occurrence = 1,
}) {
  expect(
    facet['index'],
    _byteRangeForVisibleToken(text, visibleToken, occurrence: occurrence),
  );
  expect(
    _firstFeature(facet),
    {'\$type': 'app.bsky.richtext.facet#tag', 'tag': tag},
  );
}

Map<String, dynamic> _firstFeature(Map<String, dynamic> facet) {
  final features = facet['features']! as List<dynamic>;
  return features.single as Map<String, dynamic>;
}

Map<String, int> _byteRangeForVisibleToken(
  String text,
  String token, {
  int occurrence = 1,
}) {
  final charStart = _indexOfOccurrence(text, token, occurrence);
  expect(charStart, isNonNegative, reason: 'token must exist in test text');
  final charEnd = charStart + token.length;
  return {
    'byteStart': utf8.encode(text.substring(0, charStart)).length,
    'byteEnd': utf8.encode(text.substring(0, charEnd)).length,
  };
}

int _indexOfOccurrence(String text, String token, int occurrence) {
  var from = 0;
  for (var count = 1; count <= occurrence; count++) {
    final index = text.indexOf(token, from);
    if (index < 0 || count == occurrence) {
      return index;
    }
    from = index + token.length;
  }
  return -1;
}

List<int> _byteStarts(List<Map<String, dynamic>> facets) {
  return facets.map((facet) {
    final index = facet['index']! as Map<String, dynamic>;
    return index['byteStart']! as int;
  }).toList();
}
