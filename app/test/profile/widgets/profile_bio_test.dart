import 'package:craftsky_app/profile/widgets/profile_bio.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('ProfileBio', () {
    testWidgets(
      'AT-005 detects and styles plain bio tokens with theme primary color',
      (
        tester,
      ) async {
        await tester.pumpWidget(
          ProviderScope(
            child: MaterialApp(
              theme: AppTheme.lightThemeData,
              home: Scaffold(
                body: ProfileBio(
                  description:
                      'Visit craftsky.social @alice.craftsky.social #Lace',
                ),
              ),
            ),
          ),
        );

        final body = tester.widget<Text>(
          find.byWidgetPredicate(
            (widget) =>
                widget is Text &&
                widget.textSpan?.toPlainText() ==
                    'Visit craftsky.social @alice.craftsky.social #Lace',
          ),
        );
        final spans = _leafTextSpans(body.textSpan! as TextSpan);

        expect(spans.map((span) => span.text), [
          'Visit ',
          'craftsky.social',
          ' ',
          '@alice.craftsky.social',
          ' ',
          '#Lace',
        ]);
        expect(spans[1].style?.color, BrandColors.cobalt);
        expect(spans[3].style?.color, BrandColors.cobalt);
        expect(spans[5].style?.color, BrandColors.cobalt);
      },
    );

    testWidgets('UT-009 leaves unsupported schemes and URL fragments safe', (
      tester,
    ) async {
      await tester.pumpWidget(
        ProviderScope(
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            home: Scaffold(
              body: ProfileBio(
                description:
                    'mailto:x@y.example https://craftsky.social/#lace #lace',
              ),
            ),
          ),
        ),
      );

      final body = tester.widget<Text>(
        find.byWidgetPredicate(
          (widget) =>
              widget is Text &&
              widget.textSpan?.toPlainText() ==
                  'mailto:x@y.example https://craftsky.social/#lace #lace',
        ),
      );
      final spans = _leafTextSpans(body.textSpan! as TextSpan);
      expect(spans.map((span) => span.text), [
        'mailto:x@y.example ',
        'https://craftsky.social/#lace',
        ' ',
        '#lace',
      ]);
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
