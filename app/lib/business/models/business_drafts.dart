import 'dart:convert';

import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/services/business_time_zone_service.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:flutter/foundation.dart';

const _unset = Object();

const businessEventNameLimit = 200;
const businessEventSummaryLimit = 1000;
const businessEventVenueLimit = 200;
const businessEventMaximumDuration = Duration(days: 31);
const businessEventRoles = {
  'organizer',
  'instructor',
  'vendor',
  'exhibitor',
  'speaker',
  'demonstrator',
};
const businessEventModes = {'in-person', 'online', 'hybrid'};
const businessEventStatuses = {'scheduled', 'cancelled', 'postponed'};

enum EventDraftError {
  nameRequired,
  nameTooLong,
  rolesInvalid,
  modeInvalid,
  statusInvalid,
  timeZoneInvalid,
  allDayBoundaryInvalid,
  endNotAfterStart,
  durationTooLong,
  summaryTooLong,
  venueNameTooLong,
  eventUriInvalid,
  registrationUriInvalid,
  destinationsDuplicate,
  imageInvalid,
}

@immutable
class BusinessEventUtcRange {
  const BusinessEventUtcRange({required this.startsAt, required this.endsAt});

  final String startsAt;
  final String endsAt;
}

@immutable
class BusinessEventDraft {
  const BusinessEventDraft({
    required this.name,
    required this.startsAt,
    required this.endsAt,
    required this.roles,
    required this.mode,
    required this.status,
    required this.timeZone,
    required this.isAllDay,
    this.summary,
    this.venueName,
    this.eventUri,
    this.registrationUri,
    this.image = const MissingBusinessImageDraft(),
  });

  factory BusinessEventDraft.fromEvent(
    BusinessEvent event, {
    BusinessTimeZoneService? timeZones,
  }) {
    final zones = timeZones ?? BusinessTimeZoneService.initialized();
    final zone = event.timeZone ?? 'UTC';
    return BusinessEventDraft(
      name: event.name,
      startsAt: zones.toLocal(zone, event.startsAt),
      endsAt: zones.toLocal(zone, event.endsAt),
      roles: event.roles.map((value) => value.value).toList(),
      mode: event.mode?.value ?? 'in-person',
      status: event.status.value,
      timeZone: zone,
      isAllDay: event.isAllDay,
      summary: event.summary,
      venueName: event.venueName,
      eventUri: event.eventUri,
      registrationUri: event.registrationUri,
      image: event.image == null
          ? const MissingBusinessImageDraft()
          : ExistingBusinessImageDraft(event.image!),
    );
  }

  final String name;
  final DateTime startsAt;
  final DateTime endsAt;
  final List<String> roles;
  final String mode;
  final String status;
  final String timeZone;
  final bool isAllDay;
  final String? summary;
  final String? venueName;
  final String? eventUri;
  final String? registrationUri;
  final BusinessImageDraft image;

  BusinessEventDraft copyWith({
    String? name,
    DateTime? startsAt,
    DateTime? endsAt,
    List<String>? roles,
    String? mode,
    String? status,
    String? timeZone,
    bool? isAllDay,
    Object? summary = _unset,
    Object? venueName = _unset,
    Object? eventUri = _unset,
    Object? registrationUri = _unset,
    BusinessImageDraft? image,
  }) => BusinessEventDraft(
    name: name ?? this.name,
    startsAt: startsAt ?? this.startsAt,
    endsAt: endsAt ?? this.endsAt,
    roles: roles ?? this.roles,
    mode: mode ?? this.mode,
    status: status ?? this.status,
    timeZone: timeZone ?? this.timeZone,
    isAllDay: isAllDay ?? this.isAllDay,
    summary: identical(summary, _unset) ? this.summary : summary as String?,
    venueName: identical(venueName, _unset)
        ? this.venueName
        : venueName as String?,
    eventUri: identical(eventUri, _unset) ? this.eventUri : eventUri as String?,
    registrationUri: identical(registrationUri, _unset)
        ? this.registrationUri
        : registrationUri as String?,
    image: image ?? this.image,
  );

  Set<EventDraftError> validate(BusinessTimeZoneService timeZones) {
    final errors = <EventDraftError>{};
    if (name.trim().isEmpty) errors.add(EventDraftError.nameRequired);
    if (!_validText(name, businessEventNameLimit, 2000)) {
      errors.add(EventDraftError.nameTooLong);
    }
    if (roles.isEmpty ||
        roles.length > 4 ||
        roles.toSet().length != roles.length ||
        roles.any((role) => !businessEventRoles.contains(role))) {
      errors.add(EventDraftError.rolesInvalid);
    }
    if (!businessEventModes.contains(mode)) {
      errors.add(EventDraftError.modeInvalid);
    }
    if (!businessEventStatuses.contains(status)) {
      errors.add(EventDraftError.statusInvalid);
    }
    if (timeZone.isEmpty || !timeZones.contains(timeZone)) {
      errors.add(EventDraftError.timeZoneInvalid);
    }
    if (isAllDay &&
        (startsAt.hour != 0 ||
            startsAt.minute != 0 ||
            startsAt.second != 0 ||
            endsAt.hour != 0 ||
            endsAt.minute != 0 ||
            endsAt.second != 0)) {
      errors.add(EventDraftError.allDayBoundaryInvalid);
    }
    if (timeZones.contains(timeZone)) {
      final start = timeZones.toUtc(timeZone, startsAt);
      final end = timeZones.toUtc(timeZone, endsAt);
      if (!end.isAfter(start)) {
        errors.add(EventDraftError.endNotAfterStart);
      } else if (end.difference(start) > businessEventMaximumDuration) {
        errors.add(EventDraftError.durationTooLong);
      }
    }
    if (summary case final value?
        when !_validText(value, businessEventSummaryLimit, 10000)) {
      errors.add(EventDraftError.summaryTooLong);
    }
    if (venueName case final value?
        when !_validText(value, businessEventVenueLimit, 2000)) {
      errors.add(EventDraftError.venueNameTooLong);
    }
    if (eventUri case final value? when !_validHttps(value)) {
      errors.add(EventDraftError.eventUriInvalid);
    }
    if (registrationUri case final value? when !_validHttps(value)) {
      errors.add(EventDraftError.registrationUriInvalid);
    }
    if (eventUri != null && eventUri == registrationUri) {
      errors.add(EventDraftError.destinationsDuplicate);
    }
    if (!image.isValid) errors.add(EventDraftError.imageInvalid);
    return errors;
  }

  BusinessEventUtcRange utcRange(BusinessTimeZoneService timeZones) {
    if (!timeZones.contains(timeZone)) {
      throw ArgumentError.value(timeZone, 'timeZone', 'Unknown IANA timezone');
    }
    final start = timeZones.toUtc(timeZone, startsAt);
    final end = timeZones.toUtc(timeZone, endsAt);
    return BusinessEventUtcRange(
      startsAt: _canonicalUtc(start),
      endsAt: _canonicalUtc(end),
    );
  }

  Map<String, dynamic> toCreateJson(BusinessTimeZoneService timeZones) =>
      _toJson(timeZones);

  Map<String, dynamic> toUpdateJson(BusinessTimeZoneService timeZones) =>
      _toJson(timeZones);

  Map<String, dynamic> _toJson(BusinessTimeZoneService timeZones) {
    final errors = validate(timeZones);
    if (errors.isNotEmpty) throw BusinessEventDraftValidationException(errors);
    final range = utcRange(timeZones);
    return {
      'name': name,
      'startsAt': range.startsAt,
      'endsAt': range.endsAt,
      'roles': List<String>.of(roles),
      'mode': mode,
      'status': status,
      'timeZone': timeZone,
      'isAllDay': isAllDay,
      if (summary != null) 'summary': summary,
      if (venueName != null) 'venueName': venueName,
      if (eventUri != null) 'eventUri': eventUri,
      if (registrationUri != null) 'registrationUri': registrationUri,
      if (image.hasImage) 'image': image.toJson(),
    };
  }
}

class BusinessEventDraftValidationException implements Exception {
  const BusinessEventDraftValidationException(this.errors);

  final Set<EventDraftError> errors;
}

bool _validText(String value, int runeLimit, int byteLimit) =>
    value.runes.length <= runeLimit && utf8.encode(value).length <= byteLimit;

bool _validHttps(String value) {
  final uri = Uri.tryParse(value);
  return uri != null &&
      uri.scheme == 'https' &&
      uri.host.isNotEmpty &&
      uri.userInfo.isEmpty;
}

String _canonicalUtc(DateTime value) {
  final utc = DateTime.fromMillisecondsSinceEpoch(
    value.toUtc().millisecondsSinceEpoch - value.millisecond,
    isUtc: true,
  );
  String two(int component) => component.toString().padLeft(2, '0');
  return '${utc.year.toString().padLeft(4, '0')}-'
      '${two(utc.month)}-${two(utc.day)}T'
      '${two(utc.hour)}:${two(utc.minute)}:${two(utc.second)}Z';
}

/// Complete known-field replacement for `social.craftsky.business.profile`.
///
/// Unknown top-level extensions remain AppView's merge responsibility. Open
/// catalog values and products are retained here so editing business details
/// cannot erase values authored by another client.
@immutable
class BusinessDeclarationDraft {
  const BusinessDeclarationDraft({
    required this.businessTypes,
    required this.offerings,
    required this.products,
    this.expectedCid,
    this.tagline,
    this.hoursNote,
    this.serviceArea,
    this.location,
    this.primaryAction,
  });

  factory BusinessDeclarationDraft.empty() => const BusinessDeclarationDraft(
    businessTypes: [],
    offerings: [],
    products: [],
  );

  factory BusinessDeclarationDraft.fromProfile(BusinessProfile? profile) {
    if (profile == null) return BusinessDeclarationDraft.empty();
    return BusinessDeclarationDraft(
      expectedCid: profile.cid,
      businessTypes: List.unmodifiable(profile.businessTypes),
      offerings: List.unmodifiable(profile.offerings),
      tagline: profile.tagline,
      hoursNote: profile.hoursNote,
      serviceArea: profile.serviceArea,
      location: profile.location,
      primaryAction: profile.primaryAction,
      products: List.unmodifiable(profile.products),
    );
  }

  final Cid? expectedCid;
  final List<BusinessOpenValue> businessTypes;
  final List<BusinessOpenValue> offerings;
  final String? tagline;
  final String? hoursNote;
  final String? serviceArea;
  final BusinessLocation? location;
  final BusinessAction? primaryAction;
  final List<BusinessProductView> products;

  BusinessDeclarationDraft copyWith({
    List<BusinessOpenValue>? businessTypes,
    List<BusinessOpenValue>? offerings,
    Object? tagline = _unset,
    Object? hoursNote = _unset,
    Object? serviceArea = _unset,
    Object? location = _unset,
    Object? primaryAction = _unset,
    List<BusinessProductView>? products,
  }) => BusinessDeclarationDraft(
    expectedCid: expectedCid,
    businessTypes: businessTypes ?? this.businessTypes,
    offerings: offerings ?? this.offerings,
    tagline: identical(tagline, _unset) ? this.tagline : tagline as String?,
    hoursNote: identical(hoursNote, _unset)
        ? this.hoursNote
        : hoursNote as String?,
    serviceArea: identical(serviceArea, _unset)
        ? this.serviceArea
        : serviceArea as String?,
    location: identical(location, _unset)
        ? this.location
        : location as BusinessLocation?,
    primaryAction: identical(primaryAction, _unset)
        ? this.primaryAction
        : primaryAction as BusinessAction?,
    products: products ?? this.products,
  );

  BusinessDeclarationDraft withExpectedCid(Cid cid) => BusinessDeclarationDraft(
    expectedCid: cid,
    businessTypes: businessTypes,
    offerings: offerings,
    tagline: tagline,
    hoursNote: hoursNote,
    serviceArea: serviceArea,
    location: location,
    primaryAction: primaryAction,
    products: products,
  );

  Map<String, dynamic> toJson({List<ProductDraft>? productDrafts}) {
    final json = <String, dynamic>{
      'businessTypes': businessTypes.map((value) => value.value).toList(),
      'offerings': offerings.map((value) => value.value).toList(),
      'products': productDrafts == null
          ? products.map(_productToJson).toList()
          : productDrafts.map((product) => product.toJson()).toList(),
    };
    if (tagline != null) json['tagline'] = tagline;
    if (hoursNote != null) json['hoursNote'] = hoursNote;
    if (serviceArea != null) json['serviceArea'] = serviceArea;
    if (location case final value?) {
      json['location'] = <String, dynamic>{
        'country': value.country,
        'locality': ?value.locality,
      };
    }
    if (primaryAction case final value?) {
      json['primaryAction'] = {
        'type': value.type,
        'destination': value.destination,
      };
    }
    return json;
  }

  BusinessProfile toProfile(Cid cid) => BusinessProfile(
    cid: cid.toString(),
    businessTypes: businessTypes,
    offerings: offerings,
    tagline: tagline,
    hoursNote: hoursNote,
    serviceArea: serviceArea,
    location: location,
    primaryAction: primaryAction,
    products: products,
  );

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is BusinessDeclarationDraft &&
          expectedCid == other.expectedCid &&
          _listEquals(businessTypes, other.businessTypes) &&
          _listEquals(offerings, other.offerings) &&
          tagline == other.tagline &&
          hoursNote == other.hoursNote &&
          serviceArea == other.serviceArea &&
          location == other.location &&
          primaryAction == other.primaryAction &&
          _listEquals(products, other.products);

  @override
  int get hashCode => Object.hash(
    expectedCid,
    Object.hashAll(businessTypes),
    Object.hashAll(offerings),
    tagline,
    hoursNote,
    serviceArea,
    location,
    primaryAction,
    Object.hashAll(products),
  );

  static Map<String, dynamic> _productToJson(BusinessProductView product) {
    final json = <String, dynamic>{'title': product.title};
    if (product.uri != null) json['uri'] = product.uri;
    if (product.image case final image?) {
      json['image'] = {
        'image': {
          r'$type': 'blob',
          'ref': {r'$link': image.cid.toString()},
          'mimeType': image.mime,
          'size': image.size,
        },
        'alt': image.alt,
        if (image.aspectRatio case final ratio?)
          'aspectRatio': {'width': ratio.width, 'height': ratio.height},
      };
    }
    if (product.price case final price?) {
      json['price'] = {'amount': price.amount, 'currency': price.currency};
    }
    return json;
  }

  static bool _listEquals<T>(List<T> first, List<T> second) {
    if (first.length != second.length) return false;
    for (var i = 0; i < first.length; i++) {
      if (first[i] != second[i]) return false;
    }
    return true;
  }
}

const businessProductLimit = 4;
const businessProductTitleLimit = 150;
const businessImageAltLimit = 1000;

enum ProductDraftError {
  tooManyProducts,
  duplicateDestination,
  titleRequired,
  titleTooLong,
  destinationInvalid,
  imageRequired,
  imageInvalid,
  priceIncomplete,
  priceInvalid,
}

@immutable
sealed class BusinessImageDraft {
  const BusinessImageDraft();

  String get alt;
  bool get hasImage;
  bool get isValid;
  Uint8List? get previewBytes => null;

  Map<String, dynamic>? toJson();
}

final class MissingBusinessImageDraft extends BusinessImageDraft {
  const MissingBusinessImageDraft();

  @override
  String get alt => '';

  @override
  bool get hasImage => false;

  @override
  bool get isValid => true;

  @override
  Map<String, dynamic>? toJson() => null;
}

final class RemovedBusinessImageDraft extends BusinessImageDraft {
  const RemovedBusinessImageDraft();

  @override
  String get alt => '';

  @override
  bool get hasImage => false;

  @override
  bool get isValid => true;

  @override
  Map<String, dynamic>? toJson() => null;
}

final class ExistingBusinessImageDraft extends BusinessImageDraft {
  ExistingBusinessImageDraft(BusinessImageView image, {String? alt})
    : cid = image.cid.toString(),
      mime = image.mime,
      size = image.size,
      alt = alt ?? image.alt,
      aspectRatio = image.aspectRatio;

  const ExistingBusinessImageDraft._({
    required this.cid,
    required this.mime,
    required this.size,
    required this.alt,
    required this.aspectRatio,
  });

  final String cid;
  final String mime;
  final int size;
  @override
  final String alt;
  final BusinessImageAspectRatio? aspectRatio;

  @override
  bool get hasImage => true;

  @override
  bool get isValid => _validImage(mime, size, alt, aspectRatio);

  ExistingBusinessImageDraft withAlt(String value) =>
      ExistingBusinessImageDraft._(
        cid: cid,
        mime: mime,
        size: size,
        alt: value,
        aspectRatio: aspectRatio,
      );

  @override
  Map<String, dynamic> toJson() => _imageToJson(
    cid: cid,
    mime: mime,
    size: size,
    alt: alt,
    aspectRatio: aspectRatio,
  );
}

final class UploadedBusinessImageDraft extends BusinessImageDraft {
  const UploadedBusinessImageDraft({
    required this.cid,
    required this.mime,
    required this.size,
    this.alt = '',
    this.aspectRatio,
    this.localPreviewBytes,
  });

  factory UploadedBusinessImageDraft.fromUpload(
    UploadedImageBlob upload, {
    String alt = '',
    BusinessImageAspectRatio? aspectRatio,
    Uint8List? previewBytes,
  }) => UploadedBusinessImageDraft(
    cid: upload.blob.ref.link,
    mime: upload.blob.mimeType,
    size: upload.blob.size,
    alt: alt,
    aspectRatio: aspectRatio,
    localPreviewBytes: previewBytes,
  );

  final String cid;
  final String mime;
  final int size;
  @override
  final String alt;
  final BusinessImageAspectRatio? aspectRatio;
  final Uint8List? localPreviewBytes;

  @override
  bool get hasImage => true;

  @override
  bool get isValid => _validImage(mime, size, alt, aspectRatio);

  @override
  Uint8List? get previewBytes => localPreviewBytes;

  UploadedBusinessImageDraft withAlt(String value) =>
      UploadedBusinessImageDraft(
        cid: cid,
        mime: mime,
        size: size,
        alt: value,
        aspectRatio: aspectRatio,
        localPreviewBytes: localPreviewBytes,
      );

  @override
  Map<String, dynamic> toJson() => _imageToJson(
    cid: cid,
    mime: mime,
    size: size,
    alt: alt,
    aspectRatio: aspectRatio,
  );
}

@immutable
class ProductDraft {
  const ProductDraft({
    required this.id,
    required this.title,
    required this.destination,
    required this.image,
    this.amount = '',
    this.currency = '',
  });

  factory ProductDraft.fromView(BusinessProductView product, String id) =>
      ProductDraft(
        id: id,
        title: product.title,
        destination: product.uri ?? '',
        image: product.image == null
            ? const MissingBusinessImageDraft()
            : ExistingBusinessImageDraft(product.image!),
        amount: product.price?.amount ?? '',
        currency: product.price?.currency ?? '',
      );

  final String id;
  final String title;
  final String destination;
  final BusinessImageDraft image;
  final String amount;
  final String currency;

  Set<ProductDraftError> validate() {
    final errors = <ProductDraftError>{};
    if (title.trim().isEmpty) errors.add(ProductDraftError.titleRequired);
    if (title.runes.length > businessProductTitleLimit ||
        utf8.encode(title).length > 1500) {
      errors.add(ProductDraftError.titleTooLong);
    }
    final uri = Uri.tryParse(destination);
    if (uri == null ||
        uri.scheme != 'https' ||
        uri.host.isEmpty ||
        uri.userInfo.isNotEmpty) {
      errors.add(ProductDraftError.destinationInvalid);
    }
    if (!image.hasImage) errors.add(ProductDraftError.imageRequired);
    if (!image.isValid) errors.add(ProductDraftError.imageInvalid);
    if (amount.isEmpty != currency.isEmpty) {
      errors.add(ProductDraftError.priceIncomplete);
    } else if (amount.isNotEmpty && !_validPrice(amount, currency)) {
      errors.add(ProductDraftError.priceInvalid);
    }
    return errors;
  }

  Map<String, dynamic> toJson() => {
    'title': title,
    'uri': destination,
    'image': ?image.toJson(),
    if (amount.isNotEmpty) 'price': {'amount': amount, 'currency': currency},
  };
}

Set<ProductDraftError> validateProductDrafts(List<ProductDraft> products) {
  final errors = <ProductDraftError>{};
  if (products.length > businessProductLimit) {
    errors.add(ProductDraftError.tooManyProducts);
  }
  final destinations = <String>{};
  for (final product in products) {
    errors.addAll(product.validate());
    if (!destinations.add(product.destination)) {
      errors.add(ProductDraftError.duplicateDestination);
    }
  }
  return errors;
}

Map<String, dynamic> _imageToJson({
  required String cid,
  required String mime,
  required int size,
  required String alt,
  required BusinessImageAspectRatio? aspectRatio,
}) => {
  'image': {
    r'$type': 'blob',
    'ref': {r'$link': cid},
    'mimeType': mime,
    'size': size,
  },
  'alt': alt,
  if (aspectRatio case final value?)
    'aspectRatio': {'width': value.width, 'height': value.height},
};

bool _validImage(
  String mime,
  int size,
  String alt,
  BusinessImageAspectRatio? aspectRatio,
) =>
    const {'image/jpeg', 'image/png', 'image/webp'}.contains(mime) &&
    size >= 0 &&
    size <= 15 * 1024 * 1024 &&
    alt.runes.length <= businessImageAltLimit &&
    utf8.encode(alt).length <= businessImageAltLimit &&
    (aspectRatio == null || (aspectRatio.width > 0 && aspectRatio.height > 0));

bool _validPrice(String amount, String currency) {
  if (!_isoCurrencies.contains(currency) ||
      const {'XXX', 'XTS'}.contains(currency)) {
    return false;
  }
  final parts = amount.split('.');
  if (parts.length > 2 ||
      !RegExp(r'^(0|[1-9][0-9]{0,11})$').hasMatch(parts[0])) {
    return false;
  }
  if (parts.length == 1) return true;
  final fraction = parts[1];
  final scale = _currencyScale(currency);
  return scale > 0 &&
      fraction.isNotEmpty &&
      fraction.length <= scale &&
      !fraction.endsWith('0') &&
      RegExp(r'^[0-9]+$').hasMatch(fraction);
}

int _currencyScale(String currency) {
  if (_zeroMinorUnitCurrencies.contains(currency)) return 0;
  if (currency == 'CLF') return 4;
  if (_threeMinorUnitCurrencies.contains(currency)) return 3;
  return 2;
}

final Set<String> _isoCurrencies =
    '''
AED AFN ALL AMD ANG AOA ARS AUD AWG AZN BAM BBD BDT BGN BHD BIF BMD BND BOB
BOV BRL BSD BTN BWP BYN BZD CAD CDF CHE CHF CHW CLF CLP CNY COP COU CRC CUC
CUP CVE CZK DJF DKK DOP DZD EGP ERN ETB EUR FJD FKP GBP GEL GHS GIP GMD GNF
GTQ GYD HKD HNL HRK HTG HUF IDR ILS INR IQD IRR ISK JMD JOD JPY KES KGS
KHR KMF KPW KRW KWD KYD KZT LAK LBP LKR LRD LSL LYD MAD MDL MGA MKD MMK
MNT MOP MRU MUR MVR MWK MXN MXV MYR MZN NAD NGN NIO NOK NPR NZD OMR PAB
PEN PGK PHP PKR PLN PYG QAR RON RSD RUB RWF SAR SBD SCR SDG SEK SGD SHP SLE
SLL SOS SRD SSP STN SVC SYP SZL THB TJS TMT TND TOP TRY TTD TWD TZS UAH
UGX USD USN UYI UYU UYW UZS VED VES VND VUV WST XAF XAG XAU XBA XBB XBC
XBD XCD XCG XDR XOF XPD XPF XPT XSU XTS XUA XXX YER ZAR ZMW ZWG
'''
        .split(RegExp(r'\s+'))
        .where((value) => value.isNotEmpty)
        .toSet();

const _zeroMinorUnitCurrencies = {
  'BIF',
  'CLP',
  'DJF',
  'GNF',
  'ISK',
  'JPY',
  'KMF',
  'KRW',
  'PYG',
  'RWF',
  'UGX',
  'UYI',
  'VND',
  'VUV',
  'XAF',
  'XOF',
  'XPF',
};

const _threeMinorUnitCurrencies = {
  'BHD',
  'IQD',
  'JOD',
  'KWD',
  'LYD',
  'OMR',
  'TND',
};
