// Raw facet fixture readability is preferred over raw-string/null-aware lint noise.
// ignore_for_file: use_raw_strings, use_null_aware_elements

import 'dart:convert';

import 'package:craftsky_app/shared/rich_text/faceted_text_model.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('FacetedTextModel', () {
    test('UT-009 normalizes incoming ranges for rendering', () {
      const text = 'Hi @alice #SockKAL';
      final ranges = FacetedTextModel.fromRaw(
        text: text,
        rawFacets: [
          _rawFacet(
            text,
            '#SockKAL',
            {'\$type': 'app.bsky.richtext.facet#tag', 'tag': 'SockKAL'},
          ),
          _rawFacet(
            text,
            '@alice',
            {'\$type': 'app.bsky.richtext.facet#unknown'},
            {
              '\$type': 'app.bsky.richtext.facet#mention',
              'did': 'did:plc:alice',
            },
            {'\$type': 'app.bsky.richtext.facet#tag', 'tag': 'ignored'},
          ),
          _rawFacet(
            text,
            '@alice #SockKAL',
            {
              '\$type': 'app.bsky.richtext.facet#link',
              'uri': 'https://example.com',
            },
          ),
          {
            'index': {'byteStart': 0, 'byteEnd': 999},
            'features': [
              {
                '\$type': 'app.bsky.richtext.facet#link',
                'uri': 'https://bad.example',
              },
            ],
          },
          _rawFacet(
            text,
            'Hi',
            {'\$type': 'app.bsky.richtext.facet#unsupported'},
          ),
        ],
      );

      expect(ranges, hasLength(2));
      expect(ranges[0].charStart, text.indexOf('@alice'));
      expect(ranges[0].charEnd, text.indexOf('@alice') + '@alice'.length);
      expect(
        ranges[0].feature,
        isA<MentionFacetFeature>().having(
          (feature) => feature.did,
          'did',
          'did:plc:alice',
        ),
      );

      expect(ranges[1].charStart, text.indexOf('#SockKAL'));
      expect(ranges[1].charEnd, text.indexOf('#SockKAL') + '#SockKAL'.length);
      expect(
        ranges[1].feature,
        isA<TagFacetFeature>().having(
          (feature) => feature.tag,
          'tag',
          'SockKAL',
        ),
      );
    });

    test('UT-010 drops only ranges that split multibyte characters', () {
      const text = '🧶 @alice #SockKAL';
      final ranges = FacetedTextModel.fromRaw(
        text: text,
        rawFacets: [
          {
            'index': {'byteStart': 1, 'byteEnd': 2},
            'features': [
              {
                '\$type': 'app.bsky.richtext.facet#link',
                'uri': 'https://bad.example',
              },
            ],
          },
          _rawFacet(
            text,
            '#SockKAL',
            {'\$type': 'app.bsky.richtext.facet#tag', 'tag': 'SockKAL'},
          ),
        ],
      );

      expect(ranges, hasLength(1));
      expect(ranges.single.charStart, text.indexOf('#SockKAL'));
      expect(
        ranges.single.feature,
        isA<TagFacetFeature>().having(
          (feature) => feature.tag,
          'tag',
          'SockKAL',
        ),
      );
    });
  });
}

Map<String, dynamic> _rawFacet(
  String text,
  String visibleToken,
  Map<String, dynamic> firstFeature, [
  Map<String, dynamic>? secondFeature,
  Map<String, dynamic>? thirdFeature,
]) {
  final charStart = text.indexOf(visibleToken);
  expect(charStart, isNonNegative, reason: 'token must exist in test text');
  final charEnd = charStart + visibleToken.length;
  return {
    'index': {
      'byteStart': utf8.encode(text.substring(0, charStart)).length,
      'byteEnd': utf8.encode(text.substring(0, charEnd)).length,
    },
    'features': [
      firstFeature,
      if (secondFeature != null) secondFeature,
      if (thirdFeature != null) thirdFeature,
    ],
  };
}
