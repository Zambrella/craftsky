import 'package:craftsky_app/shared/rich_text/faceted_text_model.dart';
import 'package:craftsky_app/shared/rich_text/faceted_text_span_builder.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('FacetedTextSpanBuilder', () {
    test('UT-016 applies primary color only to facet spans', () {
      const text = 'Hi @alice and #SockKAL';
      const primary = Color(0xff6750a4);
      const baseStyle = TextStyle(color: Colors.black87, fontSize: 16);

      final span = FacetedTextSpanBuilder.build(
        text: text,
        ranges: const [
          NormalizedFacetRange(
            charStart: 3,
            charEnd: 9,
            feature: FacetFeature.mention('did:plc:alice'),
          ),
          NormalizedFacetRange(
            charStart: 14,
            charEnd: 22,
            feature: FacetFeature.tag('SockKAL'),
          ),
        ],
        baseStyle: baseStyle,
        facetColor: primary,
      );

      final children = span.children!.cast<TextSpan>();
      expect(children.map((child) => child.text), [
        'Hi ',
        '@alice',
        ' and ',
        '#SockKAL',
      ]);
      expect(children[0].style!.color, Colors.black87);
      expect(children[1].style!.color, primary);
      expect(children[2].style!.color, Colors.black87);
      expect(children[3].style!.color, primary);
    });
  });
}
