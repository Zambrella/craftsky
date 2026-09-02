import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('detail edit serializes the complete known declaration', () {
    final current = BusinessProfile(
      cid: 'bafyreiaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
      businessTypes: const [
        BusinessOpenValue(value: 'teacher', known: true),
        BusinessOpenValue(value: 'future-type', known: false),
      ],
      offerings: const [
        BusinessOpenValue(value: 'classes', known: true),
        BusinessOpenValue(value: 'future-offering', known: false),
      ],
      tagline: 'Old tagline',
      hoursNote: 'Weekdays',
      serviceArea: 'South West',
      location: const BusinessLocation(country: 'GB', locality: 'Bristol'),
      primaryAction: const BusinessAction(
        type: 'book-class',
        destination: 'https://example.com/classes',
      ),
      products: [
        BusinessProductView(
          title: 'Class kit',
          uri: 'https://example.com/kit',
          image: BusinessImageView(
            cid: 'bafyreiaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
            mime: 'image/jpeg',
            size: 42,
            alt: 'A class kit',
            thumb: 'https://cdn.example/thumb',
            fullsize: 'https://cdn.example/full',
            aspectRatio: BusinessImageAspectRatio(width: 4, height: 3),
          ),
          price: const BusinessPrice(amount: '12.50', currency: 'GBP'),
        ),
      ],
    );

    final draft = BusinessDeclarationDraft.fromProfile(current).copyWith(
      tagline: 'New tagline',
    );

    expect(draft.expectedCid, current.cid);
    expect(draft.toJson(), {
      'businessTypes': ['teacher', 'future-type'],
      'offerings': ['classes', 'future-offering'],
      'tagline': 'New tagline',
      'hoursNote': 'Weekdays',
      'serviceArea': 'South West',
      'location': {'country': 'GB', 'locality': 'Bristol'},
      'primaryAction': {
        'type': 'book-class',
        'destination': 'https://example.com/classes',
      },
      'products': [
        {
          'title': 'Class kit',
          'uri': 'https://example.com/kit',
          'image': {
            'image': {
              r'$type': 'blob',
              'ref': {
                r'$link':
                    'bafyreiaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'
                    'aaaaaa',
              },
              'mimeType': 'image/jpeg',
              'size': 42,
            },
            'alt': 'A class kit',
            'aspectRatio': {'width': 4, 'height': 3},
          },
          'price': {'amount': '12.50', 'currency': 'GBP'},
        },
      ],
    });
  });

  test('empty draft has no CID and serializes every known collection', () {
    final draft = BusinessDeclarationDraft.empty();

    expect(draft.expectedCid, isNull);
    expect(draft.toJson(), {
      'businessTypes': <String>[],
      'offerings': <String>[],
      'products': <Map<String, dynamic>>[],
    });
  });
}
