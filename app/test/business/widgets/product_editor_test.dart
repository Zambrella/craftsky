import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/widgets/product_editor.dart';
import 'package:craftsky_app/feed/widgets/composer_image_attachment_section.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/providers/profile_image_picker_provider.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/craftsky_select_inputs.dart';
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

    expect(find.byType(BrandTextField), findsWidgets);
    expect(find.byType(ComposerImageAttachmentSection), findsOneWidget);
    expect(find.widgetWithText(ChunkyButton, 'Save product'), findsOneWidget);
    expect(
      tester
          .widget<CachedNetworkImage>(find.byType(CachedNetworkImage))
          .imageUrl,
      'https://cdn.example/thumb',
    );

    await tester.scrollUntilVisible(
      find.text('Replace image'),
      250,
      scrollable: find.byType(Scrollable).last,
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

  testWidgets('duplicate product destinations are shown on the field', (
    tester,
  ) async {
    var persisted = false;
    final initial = ProductDraft(
      id: 'new',
      title: 'Another product',
      destination: 'https://google.com',
      image: ExistingBusinessImageDraft(
        BusinessImageView(
          cid: 'bafy-product',
          mime: 'image/jpeg',
          size: 10,
          alt: 'Product',
          thumb: 'https://cdn.example/thumb',
          fullsize: 'https://cdn.example/full',
        ),
      ),
    );
    await tester.pumpWidget(
      _app(
        ProductEditor(
          initial: initial,
          destinationExists: (destination) =>
              destination == 'https://google.com',
          persist: (_) async {
            persisted = true;
            return true;
          },
          onSave: (_) {},
        ),
      ),
    );

    await tester.tap(find.text('Save product'));
    await tester.pumpAndSettle();

    expect(
      find.text(
        'Use a different destination. Each product must link to a unique page.',
      ),
      findsOneWidget,
    );
    expect(persisted, isFalse);
  });

  testWidgets('dirty product editor confirms before closing', (tester) async {
    await tester.pumpWidget(
      _app(ProductEditor(onSave: (_) {}, pickImage: (_) async => null)),
    );

    await tester.enterText(
      find.byKey(const ValueKey('product-title')),
      'Changed product',
    );
    await tester.tap(find.byType(CloseButton));
    await tester.pumpAndSettle();

    expect(find.byType(CraftskyDialog), findsOneWidget);
    expect(find.text('Discard changes?'), findsOneWidget);
  });

  testWidgets('amount input accepts only canonical decimal editing states', (
    tester,
  ) async {
    await tester.pumpWidget(
      _app(ProductEditor(onSave: (_) {}, locale: const Locale('en', 'US'))),
    );

    final amount = find.byKey(const ValueKey('product-amount'));
    await tester.enterText(amount, '12.3456');
    expect(_editableText(tester, amount).controller.text, '12.3456');

    await tester.enterText(amount, '12.34.5');
    expect(_editableText(tester, amount).controller.text, '12.3456');
    await tester.enterText(amount, '12 dollars');
    expect(_editableText(tester, amount).controller.text, '12.3456');
  });

  testWidgets('new products default currency from the locale country', (
    tester,
  ) async {
    await tester.pumpWidget(
      _app(ProductEditor(onSave: (_) {}, locale: const Locale('en', 'GB'))),
    );

    final select = tester.widget<CraftskySingleSelectInput<String>>(
      find.byType(CraftskySingleSelectInput<String>),
    );
    expect(select.value, 'GBP');
  });

  testWidgets('editing preserves the product currency over the locale', (
    tester,
  ) async {
    const initial = ProductDraft(
      id: 'saved',
      title: 'Yarn',
      destination: 'https://shop.example/yarn',
      image: MissingBusinessImageDraft(),
      amount: '12',
      currency: 'EUR',
    );
    await tester.pumpWidget(
      _app(
        ProductEditor(
          initial: initial,
          onSave: (_) {},
          locale: const Locale('en', 'US'),
        ),
      ),
    );

    final select = tester.widget<CraftskySingleSelectInput<String>>(
      find.byType(CraftskySingleSelectInput<String>),
    );
    expect(select.value, 'EUR');
  });

  testWidgets('locale currency alone does not create an optional price', (
    tester,
  ) async {
    ProductDraft? saved;
    await tester.pumpWidget(
      _app(
        ProductEditor(
          pickImage: (onPreviewReady) async {
            final result = _uploadedPick('bafy-product');
            onPreviewReady(result.previewBytes);
            return result;
          },
          onSave: (value) => saved = value,
          locale: const Locale('en', 'GB'),
        ),
      ),
    );

    await tester.enterText(
      find.byKey(const ValueKey('product-title')),
      'Yarn',
    );
    await tester.enterText(
      find.byKey(const ValueKey('product-destination')),
      'https://shop.example/yarn',
    );
    final addImage = find.byKey(const Key('product-add-image'));
    await tester.ensureVisible(addImage);
    await tester.tap(addImage);
    await tester.pumpAndSettle();
    await tester.tap(find.text('Save product'));
    await tester.pumpAndSettle();

    expect(saved?.amount, isEmpty);
    expect(saved?.currency, isEmpty);
  });

  testWidgets('save canonicalizes trailing fractional zeroes', (tester) async {
    ProductDraft? saved;
    final initial = ProductDraft(
      id: 'saved',
      title: 'Yarn',
      destination: 'https://shop.example/yarn',
      image: ExistingBusinessImageDraft(
        BusinessImageView(
          cid: 'bafy-product',
          mime: 'image/jpeg',
          size: 10,
          alt: 'Yarn',
          thumb: 'https://cdn.example/thumb',
          fullsize: 'https://cdn.example/full',
        ),
      ),
      amount: '12.5',
      currency: 'GBP',
    );
    await tester.pumpWidget(
      _app(ProductEditor(initial: initial, onSave: (value) => saved = value)),
    );

    final amount = find.byKey(const ValueKey('product-amount'));
    expect(_editableText(tester, amount).controller.text, '12.50');
    await tester.enterText(amount, '12.50');
    await tester.tap(find.text('Save product'));
    await tester.pumpAndSettle();
    expect(saved?.amount, '12.5');

    saved = null;
    await tester.enterText(amount, '12.00');
    await tester.tap(find.text('Save product'));
    await tester.pumpAndSettle();
    expect(saved?.amount, '12');
  });

  testWidgets('save waits for persistence before applying the product', (
    tester,
  ) async {
    final persistence = Completer<bool>();
    ProductDraft? saved;
    final initial = ProductDraft(
      id: 'saved',
      title: 'Yarn',
      destination: 'https://shop.example/yarn',
      image: ExistingBusinessImageDraft(
        BusinessImageView(
          cid: 'bafy-product',
          mime: 'image/jpeg',
          size: 10,
          alt: 'Yarn',
          thumb: 'https://cdn.example/thumb',
          fullsize: 'https://cdn.example/full',
        ),
      ),
    );
    await tester.pumpWidget(
      _app(
        ProductEditor(
          initial: initial,
          persist: (_) => persistence.future,
          onSave: (value) => saved = value,
        ),
      ),
    );

    await tester.tap(find.text('Save product'));
    await tester.pump();

    expect(saved, isNull);
    expect(
      tester
          .widget<ChunkyButton>(find.byKey(const ValueKey('product-submit')))
          .onPressed,
      isNull,
    );

    persistence.complete(true);
    await tester.pumpAndSettle();
    expect(saved?.title, 'Yarn');
  });

  testWidgets('failed persistence keeps the editor open with an error', (
    tester,
  ) async {
    ProductDraft? saved;
    final initial = ProductDraft(
      id: 'saved',
      title: 'Yarn',
      destination: 'https://shop.example/yarn',
      image: ExistingBusinessImageDraft(
        BusinessImageView(
          cid: 'bafy-product',
          mime: 'image/jpeg',
          size: 10,
          alt: 'Yarn',
          thumb: 'https://cdn.example/thumb',
          fullsize: 'https://cdn.example/full',
        ),
      ),
    );
    await tester.pumpWidget(
      _app(
        ProductEditor(
          initial: initial,
          persist: (_) async => false,
          onSave: (value) => saved = value,
        ),
      ),
    );

    await tester.tap(find.text('Save product'));
    await tester.pumpAndSettle();

    expect(saved, isNull);
    expect(
      find.text('Products could not be saved. Check the fields and try again.'),
      findsOneWidget,
    );
  });

  testWidgets('currency selector searches localized ISO currency names', (
    tester,
  ) async {
    await tester.pumpWidget(
      _app(ProductEditor(onSave: (_) {}, locale: const Locale('en', 'US'))),
    );

    await tester.enterText(
      find.byKey(const Key('product-currency-search-input')),
      'yen',
    );
    await tester.pumpAndSettle();

    expect(find.textContaining('JPY -'), findsOneWidget);
    expect(find.byKey(const Key('product-currency-option-USD')), findsNothing);
  });
}

EditableText _editableText(WidgetTester tester, Finder field) =>
    tester.widget<EditableText>(
      find.descendant(of: field, matching: find.byType(EditableText)),
    );

ProfileImagePickResult _uploadedPick(String cid) {
  final bytes = Uint8List.fromList(
    base64Decode(
      'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwC'
      'AAAAC0lEQVR4nGNgYAAAAAMAASsJTYQAAAAASUVORK5CYII=',
    ),
  );
  return ProfileImagePickResult(
    previewBytes: bytes,
    uploaded: UploadedImageBlob(
      blob: UploadedBlob(
        type: 'blob',
        ref: UploadedBlobRef(link: cid),
        mimeType: 'image/jpeg',
        size: bytes.length,
      ),
      cid: cid,
      mime: 'image/jpeg',
      size: bytes.length,
    ),
  );
}

Widget _app(Widget child) => ProviderScope(
  child: MaterialApp(
    theme: AppTheme.lightThemeData,
    localizationsDelegates: AppLocalizations.localizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    home: Scaffold(body: child),
  ),
);
