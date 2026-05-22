import 'dart:async';

import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('PostComposerSheet discard confirmation', () {
    testWidgets('closes immediately when the composer is unchanged', (
      tester,
    ) async {
      await _openComposer(tester);

      await tester.tap(find.byType(CloseButton));
      await tester.pumpAndSettle();

      expect(find.text('Host'), findsOneWidget);
      expect(find.text('Discard draft?'), findsNothing);
    });

    testWidgets('close button confirms before discarding edits', (
      tester,
    ) async {
      await _openComposer(tester);
      await tester.enterText(find.byType(TextField).first, 'A cardigan WIP');
      await tester.pump();

      await tester.tap(find.byType(CloseButton));
      await tester.pumpAndSettle();

      expect(find.text('Discard draft?'), findsOneWidget);
      expect(find.text("Your draft won't be saved."), findsOneWidget);

      await tester.tap(find.text('Keep editing'));
      await tester.pumpAndSettle();

      expect(find.text('Discard draft?'), findsNothing);
      expect(find.text('New post'), findsOneWidget);

      await tester.tap(find.byType(CloseButton));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Discard'));
      await tester.pumpAndSettle();

      expect(find.text('Host'), findsOneWidget);
    });

    testWidgets('system back confirms before discarding edits', (tester) async {
      await _openComposer(tester);
      await tester.enterText(find.byType(TextField).first, 'A cardigan WIP');
      await tester.pump();

      await tester.binding.handlePopRoute();
      await tester.pumpAndSettle();

      expect(find.text('Discard draft?'), findsOneWidget);

      await tester.tap(find.text('Discard'));
      await tester.pumpAndSettle();

      expect(find.text('Host'), findsOneWidget);
    });
  });
}

Future<void> _openComposer(WidgetTester tester) async {
  await tester.pumpWidget(
    ProviderScope(
      child: MaterialApp(
        theme: _testTheme,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Builder(
          builder: (context) {
            return Scaffold(
              body: Center(
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    const Text('Host'),
                    ElevatedButton(
                      onPressed: () {
                        unawaited(
                          Navigator.of(context).push<Post?>(
                            MaterialPageRoute<Post?>(
                              fullscreenDialog: true,
                              builder: (_) => const PostComposerSheet(),
                            ),
                          ),
                        );
                      },
                      child: const Text('Open composer'),
                    ),
                  ],
                ),
              ),
            );
          },
        ),
      ),
    ),
  );

  await tester.tap(find.text('Open composer'));
  await tester.pumpAndSettle();
}

final ThemeData _testTheme =
    ThemeData.from(
      colorScheme: const ColorScheme.light(
        primary: BrandColors.cobalt,
        onSurface: BrandColors.ink,
        onSurfaceVariant: BrandColors.ink2,
        outline: BrandColors.ink3,
        outlineVariant: BrandColors.ink4,
        error: BrandColors.red,
      ),
    ).copyWith(
      scaffoldBackgroundColor: BrandColors.paper,
      extensions: const [
        SpacingTheme(),
        RadiusTheme(),
        DurationTheme(),
        BrandShadowTheme(),
        BrandSwatchTheme(),
        SemanticColorsTheme(
          error: BrandColors.red,
          warning: BrandColors.butter,
          success: BrandColors.moss,
          info: BrandColors.cobalt,
          errorSurface: BrandColors.redSoft,
          warningSurface: BrandColors.butter,
          successSurface: BrandColors.moss,
          infoSurface: BrandColors.cobaltSoft,
        ),
      ],
    );
