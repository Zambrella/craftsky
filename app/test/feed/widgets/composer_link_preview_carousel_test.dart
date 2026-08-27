import 'dart:convert';

import 'package:craftsky_app/feed/composer/link_preview_candidate.dart';
import 'package:craftsky_app/feed/composer/link_preview_controller.dart';
import 'package:craftsky_app/feed/models/link_preview.dart';
import 'package:craftsky_app/feed/widgets/composer_link_preview_carousel.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('AT-001 renders loading and deterministic carousel controls', (
    tester,
  ) async {
    await tester.pumpWidget(_app(loading: true));
    expect(find.byKey(const Key('link-preview-loading')), findsOneWidget);

    var previous = 0;
    var next = 0;
    var dismiss = 0;
    final selected = SelectedLinkPreview(
      candidate: LinkPreviewCandidate.parse('https://source.example#section'),
      preview: LinkPreview(
        url: Uri.parse('https://final.example/pattern'),
        title: 'Pattern title',
        description: 'Pattern description',
      ),
    );
    await tester.pumpWidget(
      _app(
        selected: selected,
        current: 1,
        total: 2,
        onPrevious: () => previous++,
        onNext: () => next++,
        onDismiss: () => dismiss++,
      ),
    );

    expect(find.text('Pattern title'), findsOneWidget);
    expect(find.text('Link preview 1 of 2'), findsOneWidget);
    expect(find.text('final.example'), findsOneWidget);
    await tester.tap(find.byTooltip('Previous link preview'));
    await tester.tap(find.byTooltip('Next link preview'));
    await tester.tap(find.byTooltip('Dismiss link previews'));
    expect((previous, next, dismiss), (1, 1, 1));
  });

  testWidgets('renders thumbnail above copy on an outlined flat surface', (
    tester,
  ) async {
    final selected = SelectedLinkPreview(
      candidate: LinkPreviewCandidate.parse('https://source.example/pattern '),
      preview: LinkPreview(
        url: Uri.parse('https://final.example/pattern'),
        title: 'Pattern title',
        description: 'Pattern description',
        thumbnail: LinkPreviewThumbnail(
          bytes: base64Decode(
            'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwC'
            'AAAAC0lEQVR4nGNgYAAAAAMAASsJTYQAAAAASUVORK5CYII=',
          ),
          mimeType: 'image/png',
          width: 1,
          height: 1,
        ),
      ),
    );

    await tester.pumpWidget(_app(selected: selected, current: 1, total: 1));

    final image = find.byKey(const Key('link-preview-thumbnail'));
    final outline = tester.widget<DecoratedBox>(
      find.byKey(const Key('link-preview-carousel-outline')),
    );
    final decoration = outline.decoration as BoxDecoration;
    expect(image, findsOneWidget);
    expect(
      tester.getTopLeft(image).dy,
      lessThan(tester.getTopLeft(find.text('Pattern title')).dy),
    );
    expect(decoration.border, isNotNull);
    expect(decoration.boxShadow, isNull);
    expect(outline.position, DecorationPosition.foreground);
    expect(find.byType(Card), findsNothing);
    expect(tester.takeException(), isNull);
  });
}

Widget _app({
  SelectedLinkPreview? selected,
  int current = 0,
  int total = 0,
  bool loading = false,
  VoidCallback? onPrevious,
  VoidCallback? onNext,
  VoidCallback? onDismiss,
}) => MaterialApp(
  theme: AppTheme.lightThemeData,
  localizationsDelegates: const [
    AppLocalizations.delegate,
    GlobalMaterialLocalizations.delegate,
    GlobalWidgetsLocalizations.delegate,
    GlobalCupertinoLocalizations.delegate,
  ],
  supportedLocales: AppLocalizations.supportedLocales,
  home: Scaffold(
    body: ComposerLinkPreviewCarousel(
      selected: selected,
      current: current,
      total: total,
      loading: loading,
      onPrevious: onPrevious ?? () {},
      onNext: onNext ?? () {},
      onDismiss: onDismiss ?? () {},
    ),
  ),
);
