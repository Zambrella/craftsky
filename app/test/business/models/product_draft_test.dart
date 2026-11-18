import 'dart:typed_data';

import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  const image = UploadedBusinessImageDraft(
    cid: 'bafy-image',
    mime: 'image/jpeg',
    size: 42,
    alt: 'Blue yarn',
  );

  ProductDraft product(
    String uri, {
    String amount = '',
    String currency = '',
  }) => ProductDraft(
    id: uri,
    title: 'Yarn',
    destination: uri,
    images: const [image],
    amount: amount,
    currency: currency,
  );

  test(
    'reopening an accepted product retains local preview and alt changes',
    () {
      final bytes = Uint8List.fromList([1, 2, 3]);
      final draft = ProductDraft.fromView(
        BusinessProductView(
          title: 'Yarn',
          images: [
            BusinessImageView.localPreview(
              cid: 'bafy-image',
              mime: 'image/png',
              size: bytes.length,
              alt: 'Blue yarn',
              previewBytes: bytes,
            ),
          ],
        ),
        'saved',
      );
      expect(draft.image.previewBytes, same(bytes));
      final updated = (draft.image as ExistingBusinessImageDraft).withAlt(
        'Yarn',
      );
      expect(updated.previewBytes, same(bytes));
      expect(updated.toJson()['alt'], 'Yarn');
      expect(updated.toJson(), isNot(contains('previewBytes')));
    },
  );

  test('UT-006 validates product limits and retains authored order', () {
    final products = [
      product('https://shop.example/one'),
      product('https://shop.example/two'),
      product('https://shop.example/three'),
      product('https://shop.example/four'),
    ];

    expect(validateProductDrafts(products), isEmpty);
    expect(
      products.map((value) => value.toJson()['uri']),
      orderedEquals(products.map((value) => value.destination)),
    );
    expect(
      validateProductDrafts([
        ...products,
        product('https://shop.example/five'),
      ]),
      contains(ProductDraftError.tooManyProducts),
    );
  });

  test(
    'UT-006 requires title, credential-free HTTPS destination, and image',
    () {
      expect(
        const ProductDraft(
          id: 'invalid',
          title: ' ',
          destination: 'https://user:secret@shop.example/item',
          images: [MissingBusinessImageDraft()],
        ).validate(),
        containsAll([
          ProductDraftError.titleRequired,
          ProductDraftError.destinationInvalid,
          ProductDraftError.imageRequired,
        ]),
      );
    },
  );

  test(
    'UT-006 rejects exact duplicate URIs without normalizing similar URIs',
    () {
      expect(
        validateProductDrafts([
          product('https://shop.example/item'),
          product('https://shop.example/item'),
        ]),
        contains(ProductDraftError.duplicateDestination),
      );
      expect(
        validateProductDrafts([
          product('https://shop.example/item'),
          product('https://shop.example/item?variant=one'),
          product('https://SHOP.example/item'),
        ]),
        isEmpty,
      );
    },
  );

  test('UT-006 accepts only canonical amount and currency pairs', () {
    expect(
      product(
        'https://shop.example/a',
        amount: '1.23',
        currency: 'USD',
      ).validate(),
      isEmpty,
    );
    expect(
      product(
        'https://shop.example/b',
        amount: '1',
        currency: 'JPY',
      ).validate(),
      isEmpty,
    );
    expect(
      product(
        'https://shop.example/c',
        amount: '1.20',
        currency: 'USD',
      ).validate(),
      contains(ProductDraftError.priceInvalid),
    );
    expect(
      product(
        'https://shop.example/d',
        amount: '1.1',
        currency: 'JPY',
      ).validate(),
      contains(ProductDraftError.priceInvalid),
    );
    expect(
      product('https://shop.example/e', amount: '1').validate(),
      contains(ProductDraftError.priceIncomplete),
    );
  });
}
