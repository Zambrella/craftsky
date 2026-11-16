import 'dart:convert';
import 'dart:typed_data';

import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/widgets/product_card.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  testWidgets('IT-007 renders an accepted product local preview', (
    tester,
  ) async {
    final product = BusinessProductView(
      title: 'New image',
      images: [
        BusinessImageView.localPreview(
          cid: 'bafy-local',
          mime: 'image/png',
          size: _transparentPng.length,
          alt: 'Locally uploaded image',
          previewBytes: _transparentPng,
        ),
      ],
    );

    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: appLocalizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: ProviderScope(
          child: Scaffold(body: ProductCard(product: product)),
        ),
      ),
    );

    expect(find.byType(Image), findsOneWidget);
    expect(find.byType(CraftskyCard), findsOneWidget);
    expect(find.byType(CachedNetworkImage), findsNothing);
    expect(
      tester.widget<Image>(find.byType(Image)).image,
      isA<MemoryImage>(),
    );
  });

  testWidgets(
    'product photos and footer open its link once '
    'while swipes only select photos',
    (tester) async {
      var opened = 0;
      var confirmed = 0;
      var allowOpen = true;
      await tester.pumpWidget(
        ProviderScope(
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: Scaffold(
              body: ProductCard(
                product: BusinessProductView(
                  title: 'Wool',
                  uri: 'https://shop.example/wool',
                  images: [
                    for (var i = 0; i < 4; i++)
                      BusinessImageView.localPreview(
                        cid: 'bafy-photo-$i',
                        mime: 'image/png',
                        size: _transparentPng.length,
                        alt: 'Photo $i',
                        previewBytes: _transparentPng,
                      ),
                  ],
                ),
                confirmExternal: (_, _) async {
                  confirmed++;
                  return allowOpen;
                },
                launchExternal: (_) async {
                  opened++;
                  return true;
                },
              ),
            ),
          ),
        ),
      );
      final frame = find.byKey(const Key('post-image-frame'));
      expect(tester.getSize(frame).height, greaterThan(250));
      expect(find.text('1/4'), findsOneWidget);
      await tester.drag(find.byType(PageView), const Offset(-500, 0));
      await tester.pumpAndSettle();
      expect(find.text('2/4'), findsOneWidget);
      expect(opened, 0);
      expect(confirmed, 0);
      final outline = tester.widget<DecoratedBox>(
        find.byKey(const Key('post-image-carousel-outline')),
      );
      expect((outline.decoration as BoxDecoration).border, isNull);
      await tester.tap(frame);
      await tester.pumpAndSettle();
      await tester.pump(const Duration(milliseconds: 350));
      expect(opened, 1);
      expect(confirmed, 1);
      await tester.tap(find.text('Wool'));
      await tester.pumpAndSettle();
      expect(opened, 2);
      expect(confirmed, 2);
      final card = tester.getRect(find.byKey(const Key('product-open-link')));
      await tester.tapAt(Offset(card.left + 4, card.bottom - 4));
      await tester.pumpAndSettle();
      expect(opened, 3);
      expect(confirmed, 3);
      allowOpen = false;
      await tester.tap(frame);
      await tester.pumpAndSettle();
      expect(confirmed, 4);
      expect(opened, 3);
    },
  );

  testWidgets('AT-003 renders normalized product and launches exact URI', (
    tester,
  ) async {
    final launched = <Uri>[];
    const destination =
        'https://shop.example/yarn?colour=midnight%20blue#purchase';
    final product = BusinessProductView(
      title: 'Midnight skein',
      uri: destination,
      images: [
        BusinessImageView(
          cid: 'bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq',
          mime: 'image/jpeg',
          size: 1200,
          alt: 'A deep blue skein of yarn',
          thumb: 'https://cdn.example/thumb.jpg',
          fullsize: 'https://cdn.example/full.jpg',
        ),
      ],
      price: const BusinessPrice(amount: '12.5', currency: 'USD'),
    );

    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: appLocalizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: ProviderScope(
          child: Scaffold(
            body: ProductCard(
              product: product,
              launchExternal: (uri) async {
                launched.add(uri);
                return true;
              },
              confirmExternal: (_, _) async => true,
            ),
          ),
        ),
      ),
    );

    expect(find.text('Midnight skein'), findsOneWidget);
    expect(find.text(r'$12.50'), findsOneWidget);
    expect(find.byIcon(CraftskyIconsBold.externalLink), findsOneWidget);
    expect(
      tester
          .widget<CachedNetworkImage>(find.byType(CachedNetworkImage))
          .imageUrl,
      'https://cdn.example/thumb.jpg',
    );

    await tester.tap(find.byKey(const Key('product-open-link')));
    await tester.pump();
    expect(launched, [Uri.parse(destination)]);
    expect(find.textContaining('checkout'), findsNothing);
    expect(find.textContaining('available'), findsNothing);
  });
}

final Uint8List _transparentPng = base64Decode(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVQIHWP4z8DwHwAF'
  'gAI/ScL5WQAAAABJRU5ErkJggg==',
);
