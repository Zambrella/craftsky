// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'business_profile.dart';

class AccountTypeMapper extends EnumMapper<AccountType> {
  AccountTypeMapper._();

  static AccountTypeMapper? _instance;
  static AccountTypeMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = AccountTypeMapper._());
    }
    return _instance!;
  }

  static AccountType fromValue(dynamic value) {
    ensureInitialized();
    return MapperContainer.globals.fromValue(value);
  }

  @override
  AccountType decode(dynamic value) {
    switch (value) {
      case r'regular':
        return AccountType.regular;
      case r'business':
        return AccountType.business;
      default:
        throw MapperException.unknownEnumValue(value);
    }
  }

  @override
  dynamic encode(AccountType self) {
    switch (self) {
      case AccountType.regular:
        return r'regular';
      case AccountType.business:
        return r'business';
    }
  }
}
extension AccountTypeMapperExtension on AccountType {
  String toValue() {
    AccountTypeMapper.ensureInitialized();
    return MapperContainer.globals.toValue<AccountType>(this) as String;
  }
}

class BusinessOpenValueMapper extends ClassMapperBase<BusinessOpenValue> {
  BusinessOpenValueMapper._();

  static BusinessOpenValueMapper? _instance;
  static BusinessOpenValueMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = BusinessOpenValueMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'BusinessOpenValue';

  static String _$value(BusinessOpenValue v) => v.value;
  static const Field<BusinessOpenValue, String> _f$value = Field(
    'value',
    _$value,
  );
  static bool _$known(BusinessOpenValue v) => v.known;
  static const Field<BusinessOpenValue, bool> _f$known = Field(
    'known',
    _$known,
  );

  @override
  final MappableFields<BusinessOpenValue> fields = const {
    #value: _f$value,
    #known: _f$known,
  };

  static BusinessOpenValue _instantiate(DecodingData data) {
    return BusinessOpenValue(
      value: data.dec(_f$value),
      known: data.dec(_f$known),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static BusinessOpenValue fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<BusinessOpenValue>(map);
  }

  static BusinessOpenValue fromJson(String json) {
    return ensureInitialized().decodeJson<BusinessOpenValue>(json);
  }
}

mixin BusinessOpenValueMappable {
  String toJson() {
    return BusinessOpenValueMapper.ensureInitialized()
        .encodeJson<BusinessOpenValue>(this as BusinessOpenValue);
  }

  Map<String, dynamic> toMap() {
    return BusinessOpenValueMapper.ensureInitialized()
        .encodeMap<BusinessOpenValue>(this as BusinessOpenValue);
  }

  BusinessOpenValueCopyWith<
    BusinessOpenValue,
    BusinessOpenValue,
    BusinessOpenValue
  >
  get copyWith =>
      _BusinessOpenValueCopyWithImpl<BusinessOpenValue, BusinessOpenValue>(
        this as BusinessOpenValue,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return BusinessOpenValueMapper.ensureInitialized().stringifyValue(
      this as BusinessOpenValue,
    );
  }

  @override
  bool operator ==(Object other) {
    return BusinessOpenValueMapper.ensureInitialized().equalsValue(
      this as BusinessOpenValue,
      other,
    );
  }

  @override
  int get hashCode {
    return BusinessOpenValueMapper.ensureInitialized().hashValue(
      this as BusinessOpenValue,
    );
  }
}

extension BusinessOpenValueValueCopy<$R, $Out>
    on ObjectCopyWith<$R, BusinessOpenValue, $Out> {
  BusinessOpenValueCopyWith<$R, BusinessOpenValue, $Out>
  get $asBusinessOpenValue => $base.as(
    (v, t, t2) => _BusinessOpenValueCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class BusinessOpenValueCopyWith<
  $R,
  $In extends BusinessOpenValue,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? value, bool? known});
  BusinessOpenValueCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _BusinessOpenValueCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, BusinessOpenValue, $Out>
    implements BusinessOpenValueCopyWith<$R, BusinessOpenValue, $Out> {
  _BusinessOpenValueCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<BusinessOpenValue> $mapper =
      BusinessOpenValueMapper.ensureInitialized();
  @override
  $R call({String? value, bool? known}) => $apply(
    FieldCopyWithData({
      if (value != null) #value: value,
      if (known != null) #known: known,
    }),
  );
  @override
  BusinessOpenValue $make(CopyWithData data) => BusinessOpenValue(
    value: data.get(#value, or: $value.value),
    known: data.get(#known, or: $value.known),
  );

  @override
  BusinessOpenValueCopyWith<$R2, BusinessOpenValue, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _BusinessOpenValueCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class BusinessLocationMapper extends ClassMapperBase<BusinessLocation> {
  BusinessLocationMapper._();

  static BusinessLocationMapper? _instance;
  static BusinessLocationMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = BusinessLocationMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'BusinessLocation';

  static String _$country(BusinessLocation v) => v.country;
  static const Field<BusinessLocation, String> _f$country = Field(
    'country',
    _$country,
  );
  static String? _$locality(BusinessLocation v) => v.locality;
  static const Field<BusinessLocation, String> _f$locality = Field(
    'locality',
    _$locality,
    opt: true,
  );

  @override
  final MappableFields<BusinessLocation> fields = const {
    #country: _f$country,
    #locality: _f$locality,
  };

  static BusinessLocation _instantiate(DecodingData data) {
    return BusinessLocation(
      country: data.dec(_f$country),
      locality: data.dec(_f$locality),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static BusinessLocation fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<BusinessLocation>(map);
  }

  static BusinessLocation fromJson(String json) {
    return ensureInitialized().decodeJson<BusinessLocation>(json);
  }
}

mixin BusinessLocationMappable {
  String toJson() {
    return BusinessLocationMapper.ensureInitialized()
        .encodeJson<BusinessLocation>(this as BusinessLocation);
  }

  Map<String, dynamic> toMap() {
    return BusinessLocationMapper.ensureInitialized()
        .encodeMap<BusinessLocation>(this as BusinessLocation);
  }

  BusinessLocationCopyWith<BusinessLocation, BusinessLocation, BusinessLocation>
  get copyWith =>
      _BusinessLocationCopyWithImpl<BusinessLocation, BusinessLocation>(
        this as BusinessLocation,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return BusinessLocationMapper.ensureInitialized().stringifyValue(
      this as BusinessLocation,
    );
  }

  @override
  bool operator ==(Object other) {
    return BusinessLocationMapper.ensureInitialized().equalsValue(
      this as BusinessLocation,
      other,
    );
  }

  @override
  int get hashCode {
    return BusinessLocationMapper.ensureInitialized().hashValue(
      this as BusinessLocation,
    );
  }
}

extension BusinessLocationValueCopy<$R, $Out>
    on ObjectCopyWith<$R, BusinessLocation, $Out> {
  BusinessLocationCopyWith<$R, BusinessLocation, $Out>
  get $asBusinessLocation =>
      $base.as((v, t, t2) => _BusinessLocationCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class BusinessLocationCopyWith<$R, $In extends BusinessLocation, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? country, String? locality});
  BusinessLocationCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _BusinessLocationCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, BusinessLocation, $Out>
    implements BusinessLocationCopyWith<$R, BusinessLocation, $Out> {
  _BusinessLocationCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<BusinessLocation> $mapper =
      BusinessLocationMapper.ensureInitialized();
  @override
  $R call({String? country, Object? locality = $none}) => $apply(
    FieldCopyWithData({
      if (country != null) #country: country,
      if (locality != $none) #locality: locality,
    }),
  );
  @override
  BusinessLocation $make(CopyWithData data) => BusinessLocation(
    country: data.get(#country, or: $value.country),
    locality: data.get(#locality, or: $value.locality),
  );

  @override
  BusinessLocationCopyWith<$R2, BusinessLocation, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _BusinessLocationCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class BusinessActionMapper extends ClassMapperBase<BusinessAction> {
  BusinessActionMapper._();

  static BusinessActionMapper? _instance;
  static BusinessActionMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = BusinessActionMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'BusinessAction';

  static String _$type(BusinessAction v) => v.type;
  static const Field<BusinessAction, String> _f$type = Field('type', _$type);
  static String _$destination(BusinessAction v) => v.destination;
  static const Field<BusinessAction, String> _f$destination = Field(
    'destination',
    _$destination,
  );

  @override
  final MappableFields<BusinessAction> fields = const {
    #type: _f$type,
    #destination: _f$destination,
  };

  static BusinessAction _instantiate(DecodingData data) {
    return BusinessAction(
      type: data.dec(_f$type),
      destination: data.dec(_f$destination),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static BusinessAction fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<BusinessAction>(map);
  }

  static BusinessAction fromJson(String json) {
    return ensureInitialized().decodeJson<BusinessAction>(json);
  }
}

mixin BusinessActionMappable {
  String toJson() {
    return BusinessActionMapper.ensureInitialized().encodeJson<BusinessAction>(
      this as BusinessAction,
    );
  }

  Map<String, dynamic> toMap() {
    return BusinessActionMapper.ensureInitialized().encodeMap<BusinessAction>(
      this as BusinessAction,
    );
  }

  BusinessActionCopyWith<BusinessAction, BusinessAction, BusinessAction>
  get copyWith => _BusinessActionCopyWithImpl<BusinessAction, BusinessAction>(
    this as BusinessAction,
    $identity,
    $identity,
  );
  @override
  String toString() {
    return BusinessActionMapper.ensureInitialized().stringifyValue(
      this as BusinessAction,
    );
  }

  @override
  bool operator ==(Object other) {
    return BusinessActionMapper.ensureInitialized().equalsValue(
      this as BusinessAction,
      other,
    );
  }

  @override
  int get hashCode {
    return BusinessActionMapper.ensureInitialized().hashValue(
      this as BusinessAction,
    );
  }
}

extension BusinessActionValueCopy<$R, $Out>
    on ObjectCopyWith<$R, BusinessAction, $Out> {
  BusinessActionCopyWith<$R, BusinessAction, $Out> get $asBusinessAction =>
      $base.as((v, t, t2) => _BusinessActionCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class BusinessActionCopyWith<$R, $In extends BusinessAction, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? type, String? destination});
  BusinessActionCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _BusinessActionCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, BusinessAction, $Out>
    implements BusinessActionCopyWith<$R, BusinessAction, $Out> {
  _BusinessActionCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<BusinessAction> $mapper =
      BusinessActionMapper.ensureInitialized();
  @override
  $R call({String? type, String? destination}) => $apply(
    FieldCopyWithData({
      if (type != null) #type: type,
      if (destination != null) #destination: destination,
    }),
  );
  @override
  BusinessAction $make(CopyWithData data) => BusinessAction(
    type: data.get(#type, or: $value.type),
    destination: data.get(#destination, or: $value.destination),
  );

  @override
  BusinessActionCopyWith<$R2, BusinessAction, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _BusinessActionCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class BusinessPriceMapper extends ClassMapperBase<BusinessPrice> {
  BusinessPriceMapper._();

  static BusinessPriceMapper? _instance;
  static BusinessPriceMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = BusinessPriceMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'BusinessPrice';

  static String _$amount(BusinessPrice v) => v.amount;
  static const Field<BusinessPrice, String> _f$amount = Field(
    'amount',
    _$amount,
  );
  static String _$currency(BusinessPrice v) => v.currency;
  static const Field<BusinessPrice, String> _f$currency = Field(
    'currency',
    _$currency,
  );

  @override
  final MappableFields<BusinessPrice> fields = const {
    #amount: _f$amount,
    #currency: _f$currency,
  };

  static BusinessPrice _instantiate(DecodingData data) {
    return BusinessPrice(
      amount: data.dec(_f$amount),
      currency: data.dec(_f$currency),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static BusinessPrice fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<BusinessPrice>(map);
  }

  static BusinessPrice fromJson(String json) {
    return ensureInitialized().decodeJson<BusinessPrice>(json);
  }
}

mixin BusinessPriceMappable {
  String toJson() {
    return BusinessPriceMapper.ensureInitialized().encodeJson<BusinessPrice>(
      this as BusinessPrice,
    );
  }

  Map<String, dynamic> toMap() {
    return BusinessPriceMapper.ensureInitialized().encodeMap<BusinessPrice>(
      this as BusinessPrice,
    );
  }

  BusinessPriceCopyWith<BusinessPrice, BusinessPrice, BusinessPrice>
  get copyWith => _BusinessPriceCopyWithImpl<BusinessPrice, BusinessPrice>(
    this as BusinessPrice,
    $identity,
    $identity,
  );
  @override
  String toString() {
    return BusinessPriceMapper.ensureInitialized().stringifyValue(
      this as BusinessPrice,
    );
  }

  @override
  bool operator ==(Object other) {
    return BusinessPriceMapper.ensureInitialized().equalsValue(
      this as BusinessPrice,
      other,
    );
  }

  @override
  int get hashCode {
    return BusinessPriceMapper.ensureInitialized().hashValue(
      this as BusinessPrice,
    );
  }
}

extension BusinessPriceValueCopy<$R, $Out>
    on ObjectCopyWith<$R, BusinessPrice, $Out> {
  BusinessPriceCopyWith<$R, BusinessPrice, $Out> get $asBusinessPrice =>
      $base.as((v, t, t2) => _BusinessPriceCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class BusinessPriceCopyWith<$R, $In extends BusinessPrice, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? amount, String? currency});
  BusinessPriceCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _BusinessPriceCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, BusinessPrice, $Out>
    implements BusinessPriceCopyWith<$R, BusinessPrice, $Out> {
  _BusinessPriceCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<BusinessPrice> $mapper =
      BusinessPriceMapper.ensureInitialized();
  @override
  $R call({String? amount, String? currency}) => $apply(
    FieldCopyWithData({
      if (amount != null) #amount: amount,
      if (currency != null) #currency: currency,
    }),
  );
  @override
  BusinessPrice $make(CopyWithData data) => BusinessPrice(
    amount: data.get(#amount, or: $value.amount),
    currency: data.get(#currency, or: $value.currency),
  );

  @override
  BusinessPriceCopyWith<$R2, BusinessPrice, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _BusinessPriceCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class BusinessImageAspectRatioMapper
    extends ClassMapperBase<BusinessImageAspectRatio> {
  BusinessImageAspectRatioMapper._();

  static BusinessImageAspectRatioMapper? _instance;
  static BusinessImageAspectRatioMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = BusinessImageAspectRatioMapper._(),
      );
    }
    return _instance!;
  }

  @override
  final String id = 'BusinessImageAspectRatio';

  static int _$width(BusinessImageAspectRatio v) => v.width;
  static const Field<BusinessImageAspectRatio, int> _f$width = Field(
    'width',
    _$width,
  );
  static int _$height(BusinessImageAspectRatio v) => v.height;
  static const Field<BusinessImageAspectRatio, int> _f$height = Field(
    'height',
    _$height,
  );

  @override
  final MappableFields<BusinessImageAspectRatio> fields = const {
    #width: _f$width,
    #height: _f$height,
  };

  static BusinessImageAspectRatio _instantiate(DecodingData data) {
    return BusinessImageAspectRatio(
      width: data.dec(_f$width),
      height: data.dec(_f$height),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static BusinessImageAspectRatio fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<BusinessImageAspectRatio>(map);
  }

  static BusinessImageAspectRatio fromJson(String json) {
    return ensureInitialized().decodeJson<BusinessImageAspectRatio>(json);
  }
}

mixin BusinessImageAspectRatioMappable {
  String toJson() {
    return BusinessImageAspectRatioMapper.ensureInitialized()
        .encodeJson<BusinessImageAspectRatio>(this as BusinessImageAspectRatio);
  }

  Map<String, dynamic> toMap() {
    return BusinessImageAspectRatioMapper.ensureInitialized()
        .encodeMap<BusinessImageAspectRatio>(this as BusinessImageAspectRatio);
  }

  BusinessImageAspectRatioCopyWith<
    BusinessImageAspectRatio,
    BusinessImageAspectRatio,
    BusinessImageAspectRatio
  >
  get copyWith =>
      _BusinessImageAspectRatioCopyWithImpl<
        BusinessImageAspectRatio,
        BusinessImageAspectRatio
      >(this as BusinessImageAspectRatio, $identity, $identity);
  @override
  String toString() {
    return BusinessImageAspectRatioMapper.ensureInitialized().stringifyValue(
      this as BusinessImageAspectRatio,
    );
  }

  @override
  bool operator ==(Object other) {
    return BusinessImageAspectRatioMapper.ensureInitialized().equalsValue(
      this as BusinessImageAspectRatio,
      other,
    );
  }

  @override
  int get hashCode {
    return BusinessImageAspectRatioMapper.ensureInitialized().hashValue(
      this as BusinessImageAspectRatio,
    );
  }
}

extension BusinessImageAspectRatioValueCopy<$R, $Out>
    on ObjectCopyWith<$R, BusinessImageAspectRatio, $Out> {
  BusinessImageAspectRatioCopyWith<$R, BusinessImageAspectRatio, $Out>
  get $asBusinessImageAspectRatio => $base.as(
    (v, t, t2) => _BusinessImageAspectRatioCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class BusinessImageAspectRatioCopyWith<
  $R,
  $In extends BusinessImageAspectRatio,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({int? width, int? height});
  BusinessImageAspectRatioCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _BusinessImageAspectRatioCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, BusinessImageAspectRatio, $Out>
    implements
        BusinessImageAspectRatioCopyWith<$R, BusinessImageAspectRatio, $Out> {
  _BusinessImageAspectRatioCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<BusinessImageAspectRatio> $mapper =
      BusinessImageAspectRatioMapper.ensureInitialized();
  @override
  $R call({int? width, int? height}) => $apply(
    FieldCopyWithData({
      if (width != null) #width: width,
      if (height != null) #height: height,
    }),
  );
  @override
  BusinessImageAspectRatio $make(CopyWithData data) => BusinessImageAspectRatio(
    width: data.get(#width, or: $value.width),
    height: data.get(#height, or: $value.height),
  );

  @override
  BusinessImageAspectRatioCopyWith<$R2, BusinessImageAspectRatio, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _BusinessImageAspectRatioCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class BusinessImageViewMapper extends ClassMapperBase<BusinessImageView> {
  BusinessImageViewMapper._();

  static BusinessImageViewMapper? _instance;
  static BusinessImageViewMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = BusinessImageViewMapper._());
      MapperContainer.globals.useAll([CidMapper()]);
      BusinessImageAspectRatioMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'BusinessImageView';

  static Cid _$cid(BusinessImageView v) => v.cid;
  static dynamic _arg$cid(f) => f<Cid>();
  static const Field<BusinessImageView, String> _f$cid = Field(
    'cid',
    _$cid,
    arg: _arg$cid,
  );
  static String _$mime(BusinessImageView v) => v.mime;
  static const Field<BusinessImageView, String> _f$mime = Field('mime', _$mime);
  static int _$size(BusinessImageView v) => v.size;
  static const Field<BusinessImageView, int> _f$size = Field('size', _$size);
  static String _$alt(BusinessImageView v) => v.alt;
  static const Field<BusinessImageView, String> _f$alt = Field('alt', _$alt);
  static String _$thumb(BusinessImageView v) => v.thumb;
  static const Field<BusinessImageView, String> _f$thumb = Field(
    'thumb',
    _$thumb,
  );
  static String _$fullsize(BusinessImageView v) => v.fullsize;
  static const Field<BusinessImageView, String> _f$fullsize = Field(
    'fullsize',
    _$fullsize,
  );
  static BusinessImageAspectRatio? _$aspectRatio(BusinessImageView v) =>
      v.aspectRatio;
  static const Field<BusinessImageView, BusinessImageAspectRatio>
  _f$aspectRatio = Field('aspectRatio', _$aspectRatio, opt: true);
  static Uint8List? _$previewBytes(BusinessImageView v) => v.previewBytes;
  static const Field<BusinessImageView, Uint8List> _f$previewBytes = Field(
    'previewBytes',
    _$previewBytes,
    mode: FieldMode.member,
  );

  @override
  final MappableFields<BusinessImageView> fields = const {
    #cid: _f$cid,
    #mime: _f$mime,
    #size: _f$size,
    #alt: _f$alt,
    #thumb: _f$thumb,
    #fullsize: _f$fullsize,
    #aspectRatio: _f$aspectRatio,
    #previewBytes: _f$previewBytes,
  };

  static BusinessImageView _instantiate(DecodingData data) {
    return BusinessImageView(
      cid: data.dec(_f$cid),
      mime: data.dec(_f$mime),
      size: data.dec(_f$size),
      alt: data.dec(_f$alt),
      thumb: data.dec(_f$thumb),
      fullsize: data.dec(_f$fullsize),
      aspectRatio: data.dec(_f$aspectRatio),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static BusinessImageView fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<BusinessImageView>(map);
  }

  static BusinessImageView fromJson(String json) {
    return ensureInitialized().decodeJson<BusinessImageView>(json);
  }
}

mixin BusinessImageViewMappable {
  String toJson() {
    return BusinessImageViewMapper.ensureInitialized()
        .encodeJson<BusinessImageView>(this as BusinessImageView);
  }

  Map<String, dynamic> toMap() {
    return BusinessImageViewMapper.ensureInitialized()
        .encodeMap<BusinessImageView>(this as BusinessImageView);
  }

  BusinessImageViewCopyWith<
    BusinessImageView,
    BusinessImageView,
    BusinessImageView
  >
  get copyWith =>
      _BusinessImageViewCopyWithImpl<BusinessImageView, BusinessImageView>(
        this as BusinessImageView,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return BusinessImageViewMapper.ensureInitialized().stringifyValue(
      this as BusinessImageView,
    );
  }

  @override
  bool operator ==(Object other) {
    return BusinessImageViewMapper.ensureInitialized().equalsValue(
      this as BusinessImageView,
      other,
    );
  }

  @override
  int get hashCode {
    return BusinessImageViewMapper.ensureInitialized().hashValue(
      this as BusinessImageView,
    );
  }
}

extension BusinessImageViewValueCopy<$R, $Out>
    on ObjectCopyWith<$R, BusinessImageView, $Out> {
  BusinessImageViewCopyWith<$R, BusinessImageView, $Out>
  get $asBusinessImageView => $base.as(
    (v, t, t2) => _BusinessImageViewCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class BusinessImageViewCopyWith<
  $R,
  $In extends BusinessImageView,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  BusinessImageAspectRatioCopyWith<
    $R,
    BusinessImageAspectRatio,
    BusinessImageAspectRatio
  >?
  get aspectRatio;
  $R call({
    String? cid,
    String? mime,
    int? size,
    String? alt,
    String? thumb,
    String? fullsize,
    BusinessImageAspectRatio? aspectRatio,
  });
  BusinessImageViewCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _BusinessImageViewCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, BusinessImageView, $Out>
    implements BusinessImageViewCopyWith<$R, BusinessImageView, $Out> {
  _BusinessImageViewCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<BusinessImageView> $mapper =
      BusinessImageViewMapper.ensureInitialized();
  @override
  BusinessImageAspectRatioCopyWith<
    $R,
    BusinessImageAspectRatio,
    BusinessImageAspectRatio
  >?
  get aspectRatio =>
      $value.aspectRatio?.copyWith.$chain((v) => call(aspectRatio: v));
  @override
  $R call({
    String? cid,
    String? mime,
    int? size,
    String? alt,
    String? thumb,
    String? fullsize,
    Object? aspectRatio = $none,
  }) => $apply(
    FieldCopyWithData({
      if (cid != null) #cid: cid,
      if (mime != null) #mime: mime,
      if (size != null) #size: size,
      if (alt != null) #alt: alt,
      if (thumb != null) #thumb: thumb,
      if (fullsize != null) #fullsize: fullsize,
      if (aspectRatio != $none) #aspectRatio: aspectRatio,
    }),
  );
  @override
  BusinessImageView $make(CopyWithData data) => BusinessImageView(
    cid: data.get(#cid, or: $value.cid),
    mime: data.get(#mime, or: $value.mime),
    size: data.get(#size, or: $value.size),
    alt: data.get(#alt, or: $value.alt),
    thumb: data.get(#thumb, or: $value.thumb),
    fullsize: data.get(#fullsize, or: $value.fullsize),
    aspectRatio: data.get(#aspectRatio, or: $value.aspectRatio),
  );

  @override
  BusinessImageViewCopyWith<$R2, BusinessImageView, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _BusinessImageViewCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class BusinessProductViewMapper extends ClassMapperBase<BusinessProductView> {
  BusinessProductViewMapper._();

  static BusinessProductViewMapper? _instance;
  static BusinessProductViewMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = BusinessProductViewMapper._());
      BusinessImageViewMapper.ensureInitialized();
      BusinessPriceMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'BusinessProductView';

  static String _$title(BusinessProductView v) => v.title;
  static const Field<BusinessProductView, String> _f$title = Field(
    'title',
    _$title,
  );
  static String? _$uri(BusinessProductView v) => v.uri;
  static const Field<BusinessProductView, String> _f$uri = Field(
    'uri',
    _$uri,
    opt: true,
  );
  static BusinessImageView? _$image(BusinessProductView v) => v.image;
  static const Field<BusinessProductView, BusinessImageView> _f$image = Field(
    'image',
    _$image,
    opt: true,
  );
  static BusinessPrice? _$price(BusinessProductView v) => v.price;
  static const Field<BusinessProductView, BusinessPrice> _f$price = Field(
    'price',
    _$price,
    opt: true,
  );

  @override
  final MappableFields<BusinessProductView> fields = const {
    #title: _f$title,
    #uri: _f$uri,
    #image: _f$image,
    #price: _f$price,
  };

  static BusinessProductView _instantiate(DecodingData data) {
    return BusinessProductView(
      title: data.dec(_f$title),
      uri: data.dec(_f$uri),
      image: data.dec(_f$image),
      price: data.dec(_f$price),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static BusinessProductView fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<BusinessProductView>(map);
  }

  static BusinessProductView fromJson(String json) {
    return ensureInitialized().decodeJson<BusinessProductView>(json);
  }
}

mixin BusinessProductViewMappable {
  String toJson() {
    return BusinessProductViewMapper.ensureInitialized()
        .encodeJson<BusinessProductView>(this as BusinessProductView);
  }

  Map<String, dynamic> toMap() {
    return BusinessProductViewMapper.ensureInitialized()
        .encodeMap<BusinessProductView>(this as BusinessProductView);
  }

  BusinessProductViewCopyWith<
    BusinessProductView,
    BusinessProductView,
    BusinessProductView
  >
  get copyWith =>
      _BusinessProductViewCopyWithImpl<
        BusinessProductView,
        BusinessProductView
      >(this as BusinessProductView, $identity, $identity);
  @override
  String toString() {
    return BusinessProductViewMapper.ensureInitialized().stringifyValue(
      this as BusinessProductView,
    );
  }

  @override
  bool operator ==(Object other) {
    return BusinessProductViewMapper.ensureInitialized().equalsValue(
      this as BusinessProductView,
      other,
    );
  }

  @override
  int get hashCode {
    return BusinessProductViewMapper.ensureInitialized().hashValue(
      this as BusinessProductView,
    );
  }
}

extension BusinessProductViewValueCopy<$R, $Out>
    on ObjectCopyWith<$R, BusinessProductView, $Out> {
  BusinessProductViewCopyWith<$R, BusinessProductView, $Out>
  get $asBusinessProductView => $base.as(
    (v, t, t2) => _BusinessProductViewCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class BusinessProductViewCopyWith<
  $R,
  $In extends BusinessProductView,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  BusinessImageViewCopyWith<$R, BusinessImageView, BusinessImageView>?
  get image;
  BusinessPriceCopyWith<$R, BusinessPrice, BusinessPrice>? get price;
  $R call({
    String? title,
    String? uri,
    BusinessImageView? image,
    BusinessPrice? price,
  });
  BusinessProductViewCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _BusinessProductViewCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, BusinessProductView, $Out>
    implements BusinessProductViewCopyWith<$R, BusinessProductView, $Out> {
  _BusinessProductViewCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<BusinessProductView> $mapper =
      BusinessProductViewMapper.ensureInitialized();
  @override
  BusinessImageViewCopyWith<$R, BusinessImageView, BusinessImageView>?
  get image => $value.image?.copyWith.$chain((v) => call(image: v));
  @override
  BusinessPriceCopyWith<$R, BusinessPrice, BusinessPrice>? get price =>
      $value.price?.copyWith.$chain((v) => call(price: v));
  @override
  $R call({
    String? title,
    Object? uri = $none,
    Object? image = $none,
    Object? price = $none,
  }) => $apply(
    FieldCopyWithData({
      if (title != null) #title: title,
      if (uri != $none) #uri: uri,
      if (image != $none) #image: image,
      if (price != $none) #price: price,
    }),
  );
  @override
  BusinessProductView $make(CopyWithData data) => BusinessProductView(
    title: data.get(#title, or: $value.title),
    uri: data.get(#uri, or: $value.uri),
    image: data.get(#image, or: $value.image),
    price: data.get(#price, or: $value.price),
  );

  @override
  BusinessProductViewCopyWith<$R2, BusinessProductView, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _BusinessProductViewCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class BusinessProfileMapper extends ClassMapperBase<BusinessProfile> {
  BusinessProfileMapper._();

  static BusinessProfileMapper? _instance;
  static BusinessProfileMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = BusinessProfileMapper._());
      MapperContainer.globals.useAll([CidMapper()]);
      BusinessOpenValueMapper.ensureInitialized();
      BusinessLocationMapper.ensureInitialized();
      BusinessActionMapper.ensureInitialized();
      BusinessProductViewMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'BusinessProfile';

  static Cid _$cid(BusinessProfile v) => v.cid;
  static dynamic _arg$cid(f) => f<Cid>();
  static const Field<BusinessProfile, String> _f$cid = Field(
    'cid',
    _$cid,
    arg: _arg$cid,
  );
  static List<BusinessOpenValue> _$businessTypes(BusinessProfile v) =>
      v.businessTypes;
  static const Field<BusinessProfile, List<BusinessOpenValue>>
  _f$businessTypes = Field(
    'businessTypes',
    _$businessTypes,
    opt: true,
    def: const [],
  );
  static List<BusinessOpenValue> _$offerings(BusinessProfile v) => v.offerings;
  static const Field<BusinessProfile, List<BusinessOpenValue>> _f$offerings =
      Field('offerings', _$offerings, opt: true, def: const []);
  static String? _$tagline(BusinessProfile v) => v.tagline;
  static const Field<BusinessProfile, String> _f$tagline = Field(
    'tagline',
    _$tagline,
    opt: true,
  );
  static String? _$hoursNote(BusinessProfile v) => v.hoursNote;
  static const Field<BusinessProfile, String> _f$hoursNote = Field(
    'hoursNote',
    _$hoursNote,
    opt: true,
  );
  static String? _$serviceArea(BusinessProfile v) => v.serviceArea;
  static const Field<BusinessProfile, String> _f$serviceArea = Field(
    'serviceArea',
    _$serviceArea,
    opt: true,
  );
  static BusinessLocation? _$location(BusinessProfile v) => v.location;
  static const Field<BusinessProfile, BusinessLocation> _f$location = Field(
    'location',
    _$location,
    opt: true,
  );
  static BusinessAction? _$primaryAction(BusinessProfile v) => v.primaryAction;
  static const Field<BusinessProfile, BusinessAction> _f$primaryAction = Field(
    'primaryAction',
    _$primaryAction,
    opt: true,
  );
  static List<BusinessProductView> _$products(BusinessProfile v) => v.products;
  static const Field<BusinessProfile, List<BusinessProductView>> _f$products =
      Field('products', _$products, opt: true, def: const []);

  @override
  final MappableFields<BusinessProfile> fields = const {
    #cid: _f$cid,
    #businessTypes: _f$businessTypes,
    #offerings: _f$offerings,
    #tagline: _f$tagline,
    #hoursNote: _f$hoursNote,
    #serviceArea: _f$serviceArea,
    #location: _f$location,
    #primaryAction: _f$primaryAction,
    #products: _f$products,
  };

  static BusinessProfile _instantiate(DecodingData data) {
    return BusinessProfile(
      cid: data.dec(_f$cid),
      businessTypes: data.dec(_f$businessTypes),
      offerings: data.dec(_f$offerings),
      tagline: data.dec(_f$tagline),
      hoursNote: data.dec(_f$hoursNote),
      serviceArea: data.dec(_f$serviceArea),
      location: data.dec(_f$location),
      primaryAction: data.dec(_f$primaryAction),
      products: data.dec(_f$products),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static BusinessProfile fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<BusinessProfile>(map);
  }

  static BusinessProfile fromJson(String json) {
    return ensureInitialized().decodeJson<BusinessProfile>(json);
  }
}

mixin BusinessProfileMappable {
  String toJson() {
    return BusinessProfileMapper.ensureInitialized()
        .encodeJson<BusinessProfile>(this as BusinessProfile);
  }

  Map<String, dynamic> toMap() {
    return BusinessProfileMapper.ensureInitialized().encodeMap<BusinessProfile>(
      this as BusinessProfile,
    );
  }

  BusinessProfileCopyWith<BusinessProfile, BusinessProfile, BusinessProfile>
  get copyWith =>
      _BusinessProfileCopyWithImpl<BusinessProfile, BusinessProfile>(
        this as BusinessProfile,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return BusinessProfileMapper.ensureInitialized().stringifyValue(
      this as BusinessProfile,
    );
  }

  @override
  bool operator ==(Object other) {
    return BusinessProfileMapper.ensureInitialized().equalsValue(
      this as BusinessProfile,
      other,
    );
  }

  @override
  int get hashCode {
    return BusinessProfileMapper.ensureInitialized().hashValue(
      this as BusinessProfile,
    );
  }
}

extension BusinessProfileValueCopy<$R, $Out>
    on ObjectCopyWith<$R, BusinessProfile, $Out> {
  BusinessProfileCopyWith<$R, BusinessProfile, $Out> get $asBusinessProfile =>
      $base.as((v, t, t2) => _BusinessProfileCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class BusinessProfileCopyWith<$R, $In extends BusinessProfile, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<
    $R,
    BusinessOpenValue,
    BusinessOpenValueCopyWith<$R, BusinessOpenValue, BusinessOpenValue>
  >
  get businessTypes;
  ListCopyWith<
    $R,
    BusinessOpenValue,
    BusinessOpenValueCopyWith<$R, BusinessOpenValue, BusinessOpenValue>
  >
  get offerings;
  BusinessLocationCopyWith<$R, BusinessLocation, BusinessLocation>?
  get location;
  BusinessActionCopyWith<$R, BusinessAction, BusinessAction>? get primaryAction;
  ListCopyWith<
    $R,
    BusinessProductView,
    BusinessProductViewCopyWith<$R, BusinessProductView, BusinessProductView>
  >
  get products;
  $R call({
    String? cid,
    List<BusinessOpenValue>? businessTypes,
    List<BusinessOpenValue>? offerings,
    String? tagline,
    String? hoursNote,
    String? serviceArea,
    BusinessLocation? location,
    BusinessAction? primaryAction,
    List<BusinessProductView>? products,
  });
  BusinessProfileCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _BusinessProfileCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, BusinessProfile, $Out>
    implements BusinessProfileCopyWith<$R, BusinessProfile, $Out> {
  _BusinessProfileCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<BusinessProfile> $mapper =
      BusinessProfileMapper.ensureInitialized();
  @override
  ListCopyWith<
    $R,
    BusinessOpenValue,
    BusinessOpenValueCopyWith<$R, BusinessOpenValue, BusinessOpenValue>
  >
  get businessTypes => ListCopyWith(
    $value.businessTypes,
    (v, t) => v.copyWith.$chain(t),
    (v) => call(businessTypes: v),
  );
  @override
  ListCopyWith<
    $R,
    BusinessOpenValue,
    BusinessOpenValueCopyWith<$R, BusinessOpenValue, BusinessOpenValue>
  >
  get offerings => ListCopyWith(
    $value.offerings,
    (v, t) => v.copyWith.$chain(t),
    (v) => call(offerings: v),
  );
  @override
  BusinessLocationCopyWith<$R, BusinessLocation, BusinessLocation>?
  get location => $value.location?.copyWith.$chain((v) => call(location: v));
  @override
  BusinessActionCopyWith<$R, BusinessAction, BusinessAction>?
  get primaryAction =>
      $value.primaryAction?.copyWith.$chain((v) => call(primaryAction: v));
  @override
  ListCopyWith<
    $R,
    BusinessProductView,
    BusinessProductViewCopyWith<$R, BusinessProductView, BusinessProductView>
  >
  get products => ListCopyWith(
    $value.products,
    (v, t) => v.copyWith.$chain(t),
    (v) => call(products: v),
  );
  @override
  $R call({
    String? cid,
    List<BusinessOpenValue>? businessTypes,
    List<BusinessOpenValue>? offerings,
    Object? tagline = $none,
    Object? hoursNote = $none,
    Object? serviceArea = $none,
    Object? location = $none,
    Object? primaryAction = $none,
    List<BusinessProductView>? products,
  }) => $apply(
    FieldCopyWithData({
      if (cid != null) #cid: cid,
      if (businessTypes != null) #businessTypes: businessTypes,
      if (offerings != null) #offerings: offerings,
      if (tagline != $none) #tagline: tagline,
      if (hoursNote != $none) #hoursNote: hoursNote,
      if (serviceArea != $none) #serviceArea: serviceArea,
      if (location != $none) #location: location,
      if (primaryAction != $none) #primaryAction: primaryAction,
      if (products != null) #products: products,
    }),
  );
  @override
  BusinessProfile $make(CopyWithData data) => BusinessProfile(
    cid: data.get(#cid, or: $value.cid),
    businessTypes: data.get(#businessTypes, or: $value.businessTypes),
    offerings: data.get(#offerings, or: $value.offerings),
    tagline: data.get(#tagline, or: $value.tagline),
    hoursNote: data.get(#hoursNote, or: $value.hoursNote),
    serviceArea: data.get(#serviceArea, or: $value.serviceArea),
    location: data.get(#location, or: $value.location),
    primaryAction: data.get(#primaryAction, or: $value.primaryAction),
    products: data.get(#products, or: $value.products),
  );

  @override
  BusinessProfileCopyWith<$R2, BusinessProfile, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _BusinessProfileCopyWithImpl<$R2, $Out2>($value, $cast, t);
}
