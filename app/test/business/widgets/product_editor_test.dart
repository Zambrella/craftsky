import 'dart:async';
import 'dart:typed_data';

import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/widgets/product_editor.dart';
import 'package:craftsky_app/feed/widgets/composer_image_attachment_section.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile_image_failure.dart';
import 'package:craftsky_app/profile/providers/profile_image_picker_provider.dart';
import 'package:craftsky_app/shared/image/basic_image_editor.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/craftsky_select_inputs.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image/image.dart' as img;
import 'package:image_picker/image_picker.dart';

import 'package:material_ui/material_ui.dart';

import '../../test_support/session_registry_storage.dart';

void main() {
  for (final (error, message, reports) in <(Object, String, bool)>[
    (
      const ProfileImagePickException(
        reason: ProfileImageFailure.tooManyPixels,
      ),
      'That photo is over 48 megapixels. Choose a smaller photo or reduce its '
          'resolution and try again.',
      false,
    ),
    (
      const ProfileImagePickException(reason: ProfileImageFailure.fileTooLarge),
      'That photo is over 100 MB. Choose a smaller file and try again.',
      false,
    ),
    (
      StateError('PRIVATE_IMAGE_DEPENDENCY_PROSE'),
      'The image could not be uploaded. Try again.',
      true,
    ),
  ]) {
    testWidgets('product photo failure shows actionable copy: $message', (
      tester,
    ) async {
      final reporter = _ImageReporter();
      await tester.pumpWidget(
        _app(
          ProductEditor(
            onSave: (_) {},
            pickImage: (_, _) async =>
                Error.throwWithStackTrace(error, StackTrace.current),
          ),
          reporter: reporter,
        ),
      );
      final add = find.byKey(const Key('product-add-image'));
      await tester.ensureVisible(add);
      await tester.tap(add);
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const Key('product-choose-photos')));
      await tester.pumpAndSettle();
      expect(find.text(message), findsOneWidget);
      expect(find.textContaining('PRIVATE_IMAGE'), findsNothing);
      expect(reporter.errors, reports ? [same(error)] : isEmpty);
      if (reports) {
        expect(reporter.contexts.single.operation, 'business.product.image');
        expect(reporter.stacks.single, isNotNull);
      }
    });
  }

  testWidgets('Done dismisses product keyboard without saving', (tester) async {
    addTearDown(tester.view.resetViewInsets);
    var saved = false;
    await tester.pumpWidget(_app(ProductEditor(onSave: (_) => saved = true)));
    final field = find.byKey(const ValueKey('product-title'));
    await tester.enterText(field, 'Blue yarn');
    tester.view.viewInsets = FakeViewPadding(
      bottom: 300 * tester.view.devicePixelRatio,
    );
    await tester.pumpAndSettle();
    final done = find.byKey(const Key('product-composer-done'));
    expect(done, findsOneWidget);
    expect(find.byKey(const ValueKey('product-submit')), findsNothing);
    await tester.tap(done);
    await tester.pump();
    expect(tester.testTextInput.isVisible, isFalse);
    expect(_editableText(tester, field).controller.text, 'Blue yarn');
    expect(saved, isFalse);
    tester.view.resetViewInsets();
    await tester.pumpAndSettle();
    expect(done, findsNothing);
    expect(find.byKey(const ValueKey('product-submit')), findsOneWidget);
  });

  testWidgets('reopened local product photo renders in summary and details', (
    tester,
  ) async {
    final bytes = _preparedImage().bytes;
    final initial = ProductDraft.fromView(
      BusinessProductView(
        title: 'Yarn',
        uri: 'https://shop.example/yarn',
        images: [
          BusinessImageView.localPreview(
            cid: 'bafy-product',
            mime: 'image/png',
            size: bytes.length,
            alt: 'Blue yarn',
            previewBytes: bytes,
          ),
        ],
      ),
      'saved',
    );
    await tester.pumpWidget(
      _app(ProductEditor(initial: initial, onSave: (_) {})),
    );
    final summary = find.byKey(const Key('product-edit-images'));
    await tester.ensureVisible(summary);
    await tester.pumpAndSettle();
    final thumbnail = tester.widget<Image>(
      find.byKey(const Key('product-thumbnail-image')),
    );
    expect((thumbnail.image as MemoryImage).bytes, same(bytes));
    await tester.tap(summary);
    await tester.pumpAndSettle();
    final preview = tester.widget<Image>(
      find.byKey(const Key('product-preview-image')),
    );
    expect((preview.image as MemoryImage).bytes, same(bytes));
    expect(find.byType(CachedNetworkImage), findsNothing);
    expect(find.byKey(const Key('product-alt-image')), findsOneWidget);
  });

  for (final outcome in ['apply', 'cancel', 'fail']) {
    testWidgets(
      'product Photo details edit $outcome retains description '
      'and upload retry',
      (tester) async {
        final prepared = _preparedImage();
        var uploads = 0;
        var persists = 0;
        final products = <ProductDraft>[];
        await tester.pumpWidget(
          _app(
            ProductEditor(
              initial: const ProductDraft(
                id: 'saved',
                title: 'Yarn',
                destination: 'https://shop.example/yarn',
                images: [MissingBusinessImageDraft()],
              ),
              onSave: products.add,
              pickImage: (_, _) async => prepared,
              editImage: (request) async {
                expect(request.mode, BasicImageEditMode.composer);
                expect(request.bytes, same(prepared.bytes));
                if (outcome == 'cancel') return null;
                if (outcome == 'fail') {
                  throw StateError('PRIVATE_PRODUCT_EDIT_FAILURE');
                }
                return BasicImageEditOutput(
                  bytes: Uint8List.fromList(
                    img.encodePng(
                      img.Image(width: 20, height: 10, numChannels: 4),
                    ),
                  ),
                  mimeType: 'image/png',
                );
              },
              uploadImage: (image) async {
                uploads++;
                expect(image.width, outcome == 'apply' ? 20 : prepared.width);
                expect(image.height, outcome == 'apply' ? 10 : prepared.height);
                if (uploads == 1) {
                  throw StateError('PRIVATE_PRODUCT_UPLOAD_FAILURE');
                }
                return _uploadedPick('bafy-retried-product').uploaded;
              },
              persist: (product) async {
                persists++;
                return persists > 1;
              },
            ),
          ),
        );
        await tester.enterText(
          find.byKey(const ValueKey('product-title')),
          'Yarn',
        );
        final add = find.byKey(const Key('product-add-image'));
        await tester.ensureVisible(add);
        await tester.tap(add);
        await tester.pumpAndSettle();
        await tester.tap(find.byKey(const Key('product-choose-photos')));
        await tester.pumpAndSettle();
        expect(find.byKey(const Key('product-photo-details')), findsOneWidget);
        expect(tester.testTextInput.isVisible, isFalse);
        expect(find.byKey(const Key('product-add-image')), findsOneWidget);
        expect(find.byKey(const Key('product-move-up-image')), findsOneWidget);
        expect(
          find.byKey(const Key('product-move-down-image')),
          findsOneWidget,
        );
        final alt = find.byKey(const Key('product-alt-image'));
        await tester.ensureVisible(alt);
        await tester.enterText(alt, 'Blue yarn');
        tester.testTextInput.hide();
        final attachment = tester.widget<ComposerImageAttachmentSection>(
          find.byType(ComposerImageAttachmentSection, skipOffstage: false),
        );
        await tester.runAsync(() => attachment.onEditImage!('image'));
        await tester.pumpAndSettle();
        expect(uploads, 0);
        expect(find.textContaining('PRIVATE_PRODUCT_EDIT'), findsNothing);
        await tester.pageBack();
        await tester.pumpAndSettle();
        final summary = find.byKey(const Key('product-edit-images'));
        await tester.ensureVisible(summary);
        await tester.tap(summary);
        await tester.pumpAndSettle();
        expect(
          tester.widget<BrandTextField>(alt).controller!.text,
          'Blue yarn',
        );
        await tester.pageBack();
        await tester.pumpAndSettle();
        final save = find.byKey(const ValueKey('product-submit'));
        await tester.tap(save);
        await tester.pumpAndSettle();
        expect(uploads, 1);
        expect(products, isEmpty);
        expect(
          find.text('The image could not be uploaded. Try again.'),
          findsOneWidget,
        );
        await tester.tap(save);
        await tester.pumpAndSettle();
        expect(uploads, 2);
        expect(persists, 1);
        expect(products, isEmpty);
        await tester.tap(save);
        await tester.pumpAndSettle();
        expect(uploads, 2);
        expect(persists, 2);
        expect(products.single.image.alt, 'Blue yarn');
        expect(products.single.image.previewBytes, isNotNull);
      },
    );
  }

  testWidgets(
    'four product photos retain order, descriptions '
    'and partial uploads on retry',
    (tester) async {
      var picks = 0;
      final attempts = <int>[];
      var fail = true;
      ProductDraft? saved;
      await tester.pumpWidget(
        _app(
          ProductEditor(
            initial: const ProductDraft(
              id: 'four',
              title: 'Wool',
              destination: 'https://shop.example/four',
            ),
            pickImage: (_, _) async {
              picks++;
              return PreparedProfileImage(
                bytes: _preparedImage().bytes,
                mimeType: 'image/png',
                width: picks,
                height: 1,
              );
            },
            uploadImage: (image) async {
              attempts.add(image.width);
              if (attempts.length == 2 && fail) {
                fail = false;
                throw StateError('upload failed');
              }
              return _uploadedPick('bafy-photo-${image.width}').uploaded;
            },
            onSave: (value) => saved = value,
          ),
        ),
      );
      for (var i = 0; i < 4; i++) {
        final add = find.byKey(const Key('product-add-image'));
        await tester.ensureVisible(add);
        await tester.tap(add);
        await tester.pumpAndSettle();
        await tester.tap(find.byKey(const Key('product-choose-photos')));
        await tester.pumpAndSettle();
      }
      expect(picks, 4);
      expect(find.byKey(const Key('product-add-image')), findsNothing);
      var section = tester.widget<ComposerImageAttachmentSection>(
        find.byType(ComposerImageAttachmentSection, skipOffstage: false),
      );
      for (final image in section.imagesState.images) {
        section.onAltTextChanged(image.id, image.id);
      }
      section.onReorder(3, 0);
      await tester.pumpAndSettle();
      section = tester.widget<ComposerImageAttachmentSection>(
        find.byType(ComposerImageAttachmentSection, skipOffstage: false),
      );
      expect(section.imagesState.images.map((image) => image.id), [
        'image-3',
        'image',
        'image-1',
        'image-2',
      ]);
      section.onRemove('image-1');
      await tester.pumpAndSettle();
      expect(find.byKey(const Key('product-add-image')), findsOneWidget);
      await tester.pageBack();
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const ValueKey('product-submit')));
      await tester.pumpAndSettle();
      expect(saved, isNull);
      expect(attempts, [4, 1]);
      await tester.tap(find.byKey(const ValueKey('product-submit')));
      await tester.pumpAndSettle();
      expect(attempts, [4, 1, 1, 3]);
      expect(saved!.images.map((image) => image.alt), [
        'image-3',
        'image',
        'image-2',
      ]);
      expect((saved!.toJson()['images'] as List).length, 3);
    },
  );

  testWidgets('gallery adds batches in order using only remaining capacity', (
    tester,
  ) async {
    final limits = <int>[];
    await tester.pumpWidget(
      _app(
        ProductEditor(
          onSave: (_) {},
          pickImages: (limit) async {
            limits.add(limit);
            return [_preparedImage(), _preparedImage()];
          },
        ),
      ),
    );
    for (var batch = 0; batch < 2; batch++) {
      final add = find.byKey(const Key('product-add-image'));
      await tester.ensureVisible(add);
      await tester.tap(add);
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const Key('product-choose-photos')));
      await tester.pumpAndSettle();
    }
    expect(limits, [4, 2]);
    final section = tester.widget<ComposerImageAttachmentSection>(
      find.byType(ComposerImageAttachmentSection, skipOffstage: false),
    );
    expect(section.imagesState.images.map((image) => image.id), [
      'image',
      'image-1',
      'image-2',
      'image-3',
    ]);
    expect(find.byKey(const Key('product-add-image')), findsNothing);
  });
  testWidgets(
    'gallery cancellation and over-selection preserve attached photos',
    (tester) async {
      var picks = 0;
      await tester.pumpWidget(
        _app(
          ProductEditor(
            onSave: (_) {},
            pickImages: (limit) async {
              picks++;
              return picks == 1
                  ? [_preparedImage(), _preparedImage()]
                  : picks == 2
                  ? []
                  : [_preparedImage(), _preparedImage(), _preparedImage()];
            },
          ),
        ),
      );
      for (var batch = 0; batch < 3; batch++) {
        final add = find.byKey(const Key('product-add-image'));
        await tester.ensureVisible(add);
        await tester.tap(add);
        await tester.pumpAndSettle();
        await tester.tap(find.byKey(const Key('product-choose-photos')));
        await tester.pumpAndSettle();
      }
      final section = tester.widget<ComposerImageAttachmentSection>(
        find.byType(ComposerImageAttachmentSection, skipOffstage: false),
      );
      expect(section.imagesState.images.length, 2);
      expect(find.text('You can add up to 2 images'), findsOneWidget);
    },
  );

  testWidgets('AT-005 failed replacement keeps the saved image', (
    tester,
  ) async {
    final initial = ProductDraft(
      id: 'saved',
      title: 'Yarn',
      destination: 'https://shop.example/yarn',
      images: [
        ExistingBusinessImageDraft(
          BusinessImageView(
            cid: 'bafy-saved',
            mime: 'image/jpeg',
            size: 10,
            alt: 'Saved yarn',
            thumb: 'https://cdn.example/thumb',
            fullsize: 'https://cdn.example/full',
          ),
        ),
      ],
    );
    ProductDraft? saved;
    ImageSource? selectedSource;
    await tester.pumpWidget(
      _app(
        ProductEditor(
          initial: initial,
          pickImage: (source, _) async {
            selectedSource = source;
            throw Exception('upload failed');
          },
          onSave: (value) => saved = value,
        ),
      ),
    );

    expect(find.byType(BrandTextField), findsWidgets);
    expect(find.byType(ComposerImageAttachmentSection), findsOneWidget);
    expect(find.widgetWithText(ChunkyButton, 'Save product'), findsOneWidget);
    final summary = find.byKey(const Key('product-edit-images'));
    await tester.ensureVisible(summary);
    await tester.tap(summary);
    await tester.pumpAndSettle();
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
    await tester.tap(find.byKey(const Key('product-replace-take-photo')));
    await tester.pumpAndSettle();
    expect(selectedSource, ImageSource.camera);
    expect(
      find.text('The image could not be uploaded. Try again.'),
      findsOneWidget,
    );

    await tester.pageBack();
    await tester.pumpAndSettle();
    await tester.tap(find.text('Save product'));
    await tester.pumpAndSettle();
    expect(saved?.image.toJson(), initial.image.toJson());
  });

  testWidgets('AT-005 validates required product fields', (tester) async {
    await tester.pumpWidget(
      _app(ProductEditor(onSave: (_) {}, pickImage: (_, _) async => null)),
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
      images: [
        ExistingBusinessImageDraft(
          BusinessImageView(
            cid: 'bafy-product',
            mime: 'image/jpeg',
            size: 10,
            alt: 'Product',
            thumb: 'https://cdn.example/thumb',
            fullsize: 'https://cdn.example/full',
          ),
        ),
      ],
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
      _app(ProductEditor(onSave: (_) {}, pickImage: (_, _) async => null)),
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
      images: [MissingBusinessImageDraft()],
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
          pickImage: (_, onPreviewReady) async {
            final result = _preparedImage();
            onPreviewReady(result.bytes);
            return result;
          },
          uploadImage: (_) async => _uploadedPick('bafy-product').uploaded,
          onSave: (value) => saved = value,
          locale: const Locale('en', 'GB'),
        ),
      ),
    );

    await tester.enterText(find.byKey(const ValueKey('product-title')), 'Yarn');
    await tester.enterText(
      find.byKey(const ValueKey('product-destination')),
      'https://shop.example/yarn',
    );
    final addImage = find.byKey(const Key('product-add-image'));
    await tester.ensureVisible(addImage);
    await tester.tap(addImage);
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('product-choose-photos')));
    await tester.pumpAndSettle();
    await tester.pageBack();
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
      images: [
        ExistingBusinessImageDraft(
          BusinessImageView(
            cid: 'bafy-product',
            mime: 'image/jpeg',
            size: 10,
            alt: 'Yarn',
            thumb: 'https://cdn.example/thumb',
            fullsize: 'https://cdn.example/full',
          ),
        ),
      ],
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
      images: [
        ExistingBusinessImageDraft(
          BusinessImageView(
            cid: 'bafy-product',
            mime: 'image/jpeg',
            size: 10,
            alt: 'Yarn',
            thumb: 'https://cdn.example/thumb',
            fullsize: 'https://cdn.example/full',
          ),
        ),
      ],
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
          .widget<ChunkyButton>(
            find.descendant(
              of: find.byKey(const ValueKey('product-submit')),
              matching: find.byType(ChunkyButton),
            ),
          )
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
      images: [
        ExistingBusinessImageDraft(
          BusinessImageView(
            cid: 'bafy-product',
            mime: 'image/jpeg',
            size: 10,
            alt: 'Yarn',
            thumb: 'https://cdn.example/thumb',
            fullsize: 'https://cdn.example/full',
          ),
        ),
      ],
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

PreparedProfileImage _preparedImage() => PreparedProfileImage(
  bytes: _uploadedPick('bafy-image').previewBytes,
  mimeType: 'image/png',
  width: 1,
  height: 1,
);

ProfileImagePickResult _uploadedPick(String cid) {
  final bytes = Uint8List.fromList(
    img.encodePng(img.Image(width: 1, height: 1, numChannels: 4)),
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

Widget _app(Widget child, {ErrorReporter? reporter}) => ProviderScope(
  overrides: [
    // This override belongs to the test harness's root ProviderScope.
    // ignore: riverpod_lint/scoped_providers_should_specify_dependencies
    secureSessionRegistryStorageProvider.overrideWithValue(
      InMemorySessionRegistryStorage(),
    ),
    if (reporter != null) errorReporterProvider.overrideWithValue(reporter),
  ],
  child: MaterialApp(
    theme: AppTheme.lightThemeData,
    localizationsDelegates: appLocalizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    home: child,
  ),
);

final class _ImageReporter implements ErrorReporter {
  final errors = <Object>[];
  final stacks = <StackTrace?>[];
  final contexts = <ReportContext>[];
  @override
  bool get enabled => true;
  @override
  Future<String?> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) async {
    errors.add(error);
    stacks.add(stackTrace);
    contexts.add(context);
    return null;
  }
}
