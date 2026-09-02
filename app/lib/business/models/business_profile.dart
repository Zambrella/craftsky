import 'dart:typed_data';

import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'business_profile.mapper.dart';

@MappableEnum()
enum AccountType { regular, business }

@MappableClass()
class BusinessOpenValue with BusinessOpenValueMappable {
  const BusinessOpenValue({required this.value, required this.known});

  final String value;
  final bool known;
}

@MappableClass()
class BusinessLocation with BusinessLocationMappable {
  const BusinessLocation({required this.country, this.locality});

  final String country;
  final String? locality;
}

@MappableClass()
class BusinessAction with BusinessActionMappable {
  const BusinessAction({required this.type, required this.destination});

  final String type;
  final String destination;
}

@MappableClass()
class BusinessPrice with BusinessPriceMappable {
  const BusinessPrice({required this.amount, required this.currency});

  final String amount;
  final String currency;
}

@MappableClass()
class BusinessImageAspectRatio with BusinessImageAspectRatioMappable {
  BusinessImageAspectRatio({required this.width, required this.height}) {
    if (width <= 0 || height <= 0) {
      throw const FormatException('Invalid business image aspect ratio');
    }
  }

  final int width;
  final int height;
}

@MappableClass(includeCustomMappers: [CidMapper()])
class BusinessImageView with BusinessImageViewMappable {
  BusinessImageView({
    required String cid,
    required this.mime,
    required this.size,
    required this.alt,
    required this.thumb,
    required this.fullsize,
    this.aspectRatio,
  }) : cid = Cid.parse(cid),
       previewBytes = null {
    if (size < 0 ||
        !const {'image/jpeg', 'image/png', 'image/webp'}.contains(mime) ||
        !_validDisplayUri(thumb) ||
        !_validDisplayUri(fullsize)) {
      throw const FormatException('Invalid business image');
    }
  }

  BusinessImageView.localPreview({
    required String cid,
    required this.mime,
    required this.size,
    required this.alt,
    required Uint8List previewBytes,
    this.aspectRatio,
  }) : cid = Cid.parse(cid),
       thumb = '',
       fullsize = '',
       previewBytes = previewBytes {
    if (size < 0 ||
        !const {'image/jpeg', 'image/png', 'image/webp'}.contains(mime) ||
        previewBytes.isEmpty) {
      throw const FormatException('Invalid business image');
    }
  }

  final Cid cid;
  final String mime;
  final int size;
  final String alt;
  final BusinessImageAspectRatio? aspectRatio;
  final String thumb;
  final String fullsize;
  final Uint8List? previewBytes;

  static bool _validDisplayUri(String value) {
    final uri = Uri.tryParse(value);
    return uri != null && uri.hasScheme && uri.host.isNotEmpty;
  }
}

@MappableClass()
class BusinessProductView with BusinessProductViewMappable {
  const BusinessProductView({
    required this.title,
    this.uri,
    this.image,
    this.price,
  });

  final String title;
  final String? uri;
  final BusinessImageView? image;
  final BusinessPrice? price;
}

@MappableClass(includeCustomMappers: [CidMapper()])
class BusinessProfile with BusinessProfileMappable {
  BusinessProfile({
    required String cid,
    this.businessTypes = const [],
    this.offerings = const [],
    this.tagline,
    this.hoursNote,
    this.serviceArea,
    this.location,
    this.primaryAction,
    this.products = const [],
  }) : cid = Cid.parse(cid);

  final Cid cid;
  final List<BusinessOpenValue> businessTypes;
  final List<BusinessOpenValue> offerings;
  final String? tagline;
  final String? hoursNote;
  final String? serviceArea;
  final BusinessLocation? location;
  final BusinessAction? primaryAction;
  final List<BusinessProductView> products;
}
