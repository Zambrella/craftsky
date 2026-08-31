import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/widgets/product_editor.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('AT-005 failed replacement keeps the saved image', (
    tester,
  ) async {
    final initial = ProductDraft(
      id: 'saved',
      title: 'Yarn',
      destination: 'https://shop.example/yarn',
      image: ExistingBusinessImageDraft(
        BusinessImageView(
          cid: 'bafy-saved',
          mime: 'image/jpeg',
          size: 10,
          alt: 'Saved yarn',
          thumb: 'https://cdn.example/thumb',
          fullsize: 'https://cdn.example/full',
        ),
      ),
    );
    ProductDraft? saved;
    await tester.pumpWidget(
      _app(
        ProductEditor(
          initial: initial,
          pickImage: (_) async => throw Exception('upload failed'),
          onSave: (value) => saved = value,
        ),
      ),
    );

    await tester.tap(find.text('Replace image'));
    await tester.pumpAndSettle();
    expect(
      find.text('The image could not be uploaded. Try again.'),
      findsOneWidget,
    );

    await tester.tap(find.text('Save product'));
    await tester.pumpAndSettle();
    expect(saved?.image.toJson(), initial.image.toJson());
  });

  testWidgets('AT-005 validates required product fields', (tester) async {
    await tester.pumpWidget(
      _app(ProductEditor(onSave: (_) {}, pickImage: (_) async => null)),
    );

    await tester.tap(find.text('Save product'));
    await tester.pumpAndSettle();

    expect(find.text('Add a title.'), findsOneWidget);
    expect(find.text('Enter a credential-free HTTPS link.'), findsOneWidget);
    expect(find.text('Add an image.'), findsOneWidget);
  });
}

Widget _app(Widget child) => ProviderScope(
  child: MaterialApp(
    localizationsDelegates: AppLocalizations.localizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    home: Scaffold(body: child),
  ),
);
