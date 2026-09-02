import 'dart:convert';
import 'dart:typed_data';

import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/widgets/product_card.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('IT-007 renders an accepted product local preview', (
    tester,
  ) async {
    final product = BusinessProductView(
      title: 'New image',
      image: BusinessImageView.localPreview(
        cid: 'bafy-local',
        mime: 'image/png',
        size: _transparentPng.length,
        alt: 'Locally uploaded image',
        previewBytes: _transparentPng,
      ),
    );

    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(body: ProductCard(product: product)),
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

  testWidgets('AT-003 renders normalized product and launches exact URI', (
    tester,
  ) async {
    final launched = <Uri>[];
    const destination =
        'https://shop.example/yarn?colour=midnight%20blue#purchase';
    final product = BusinessProductView(
      title: 'Midnight skein',
      uri: destination,
      image: BusinessImageView(
        cid: 'bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq',
        mime: 'image/jpeg',
        size: 1200,
        alt: 'A deep blue skein of yarn',
        thumb: 'https://cdn.example/thumb.jpg',
        fullsize: 'https://cdn.example/full.jpg',
      ),
      price: const BusinessPrice(amount: '12.5', currency: 'USD'),
    );

    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
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
    );

    expect(find.text('Midnight skein'), findsOneWidget);
    expect(find.text(r'$12.50'), findsOneWidget);
    expect(find.byIcon(Icons.open_in_new), findsOneWidget);
    expect(
      tester
          .widget<CachedNetworkImage>(find.byType(CachedNetworkImage))
          .imageUrl,
      'https://cdn.example/thumb.jpg',
    );

    await tester.tap(find.byType(ProductCard));
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
