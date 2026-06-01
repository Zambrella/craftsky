import 'package:craftsky_app/shared/rich_text/widgets/faceted_text.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('FacetedText', () {
    testWidgets(
      'AT-005 renders valid facet ranges with the theme primary color',
      (
        tester,
      ) async {
        await tester.pumpWidget(
          ProviderScope(
            child: MaterialApp(
              theme: ThemeData(
                colorScheme: ColorScheme.fromSeed(seedColor: Colors.purple),
              ),
              home: Scaffold(
                body: FacetedText(
                  text: 'Hi @alice #Lace',
                  facets: [
                    _facet(3, 9, {
                      r'$type': 'app.bsky.richtext.facet#mention',
                      'did': 'did:plc:alice',
                    }),
                    _facet(10, 15, {
                      r'$type': 'app.bsky.richtext.facet#tag',
                      'tag': 'Lace',
                    }),
                  ],
                  style: const TextStyle(color: Colors.black),
                ),
              ),
            ),
          ),
        );

        final richText = tester.widget<RichText>(find.byType(RichText).last);
        final root = richText.text as TextSpan;
        final children = _leafTextSpans(root);

        expect(children.map((span) => span.text), [
          'Hi ',
          '@alice',
          ' ',
          '#Lace',
        ]);
        expect(children[0].style?.color, Colors.black);
        expect(
          children[1].style?.color,
          Theme.of(
            tester.element(find.byType(FacetedText)),
          ).colorScheme.primary,
        );
        expect(children[2].style?.color, Colors.black);
        expect(
          children[3].style?.color,
          Theme.of(
            tester.element(find.byType(FacetedText)),
          ).colorScheme.primary,
        );
      },
    );

    testWidgets('AT-005 drops invalid facets without throwing', (tester) async {
      await tester.pumpWidget(
        const ProviderScope(
          child: MaterialApp(
            home: Scaffold(
              body: FacetedText(
                text: 'Plain text',
                facets: [
                  {
                    'index': {'byteStart': 100, 'byteEnd': 120},
                    'features': [
                      {r'$type': 'app.bsky.richtext.facet#tag', 'tag': 'bad'},
                    ],
                  },
                ],
              ),
            ),
          ),
        ),
      );

      expect(find.text('Plain text'), findsOneWidget);
      expect(tester.takeException(), isNull);
    });
  });
}

List<TextSpan> _leafTextSpans(TextSpan root) {
  final leaves = <TextSpan>[];

  void visit(TextSpan span) {
    final children = span.children;
    if (children == null || children.isEmpty) {
      leaves.add(span);
      return;
    }
    for (final child in children) {
      if (child is TextSpan) {
        visit(child);
      }
    }
  }

  visit(root);
  return leaves;
}

Map<String, dynamic> _facet(
  int byteStart,
  int byteEnd,
  Map<String, dynamic> feature,
) {
  return {
    'index': {'byteStart': byteStart, 'byteEnd': byteEnd},
    'features': [feature],
  };
}
