import 'package:craftsky_app/profile/widgets/profile_bio.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('ProfileBio', () {
    testWidgets(
      'AT-005 styles valid description facets with theme primary color',
      (
        tester,
      ) async {
        await tester.pumpWidget(
          ProviderScope(
            child: MaterialApp(
              theme: AppTheme.lightThemeData,
              home: Scaffold(
                body: ProfileBio(
                  description: 'Knitting #Lace',
                  descriptionFacets: [
                    _facet(9, 14, {
                      r'$type': 'app.bsky.richtext.facet#tag',
                      'tag': 'Lace',
                    }),
                  ],
                ),
              ),
            ),
          ),
        );

        final body = tester.widget<Text>(
          find.byWidgetPredicate(
            (widget) =>
                widget is Text &&
                widget.textSpan?.toPlainText() == 'Knitting #Lace',
          ),
        );
        final spans = _leafTextSpans(body.textSpan! as TextSpan);

        expect(spans.map((span) => span.text), ['Knitting ', '#Lace']);
        expect(spans[1].style?.color, BrandColors.cobalt);
      },
    );

    testWidgets('AT-005 safely renders plain bio when facets are invalid', (
      tester,
    ) async {
      await tester.pumpWidget(
        ProviderScope(
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            home: Scaffold(
              body: ProfileBio(
                description: 'Knitting #Lace',
                descriptionFacets: [
                  _facet(200, 220, {
                    r'$type': 'app.bsky.richtext.facet#tag',
                    'tag': 'bad',
                  }),
                ],
              ),
            ),
          ),
        ),
      );

      expect(find.text('Knitting #Lace'), findsOneWidget);
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
      if (child is TextSpan) visit(child);
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
