// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'business_event.dart';

class BusinessEventMapper extends ClassMapperBase<BusinessEvent> {
  BusinessEventMapper._();

  static BusinessEventMapper? _instance;
  static BusinessEventMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = BusinessEventMapper._());
      MapperContainer.globals.useAll([
        DidMapper(),
        RecordKeyMapper(),
        AtUriMapper(),
        CidMapper(),
      ]);
      BusinessOpenValueMapper.ensureInitialized();
      BusinessImageViewMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'BusinessEvent';

  static Did _$did(BusinessEvent v) => v.did;
  static dynamic _arg$did(f) => f<Did>();
  static const Field<BusinessEvent, String> _f$did = Field(
    'did',
    _$did,
    arg: _arg$did,
  );
  static RecordKey _$rkey(BusinessEvent v) => v.rkey;
  static dynamic _arg$rkey(f) => f<RecordKey>();
  static const Field<BusinessEvent, String> _f$rkey = Field(
    'rkey',
    _$rkey,
    arg: _arg$rkey,
  );
  static AtUri _$uri(BusinessEvent v) => v.uri;
  static dynamic _arg$uri(f) => f<AtUri>();
  static const Field<BusinessEvent, String> _f$uri = Field(
    'uri',
    _$uri,
    arg: _arg$uri,
  );
  static Cid _$cid(BusinessEvent v) => v.cid;
  static dynamic _arg$cid(f) => f<Cid>();
  static const Field<BusinessEvent, String> _f$cid = Field(
    'cid',
    _$cid,
    arg: _arg$cid,
  );
  static String _$name(BusinessEvent v) => v.name;
  static const Field<BusinessEvent, String> _f$name = Field('name', _$name);
  static DateTime _$startsAt(BusinessEvent v) => v.startsAt;
  static const Field<BusinessEvent, DateTime> _f$startsAt = Field(
    'startsAt',
    _$startsAt,
  );
  static DateTime _$endsAt(BusinessEvent v) => v.endsAt;
  static const Field<BusinessEvent, DateTime> _f$endsAt = Field(
    'endsAt',
    _$endsAt,
  );
  static List<BusinessOpenValue> _$roles(BusinessEvent v) => v.roles;
  static const Field<BusinessEvent, List<BusinessOpenValue>> _f$roles = Field(
    'roles',
    _$roles,
  );
  static BusinessOpenValue _$status(BusinessEvent v) => v.status;
  static const Field<BusinessEvent, BusinessOpenValue> _f$status = Field(
    'status',
    _$status,
  );
  static bool _$isAllDay(BusinessEvent v) => v.isAllDay;
  static const Field<BusinessEvent, bool> _f$isAllDay = Field(
    'isAllDay',
    _$isAllDay,
  );
  static DateTime _$createdAt(BusinessEvent v) => v.createdAt;
  static const Field<BusinessEvent, DateTime> _f$createdAt = Field(
    'createdAt',
    _$createdAt,
  );
  static bool _$past(BusinessEvent v) => v.past;
  static const Field<BusinessEvent, bool> _f$past = Field('past', _$past);
  static List<String> _$publicSuppressionReasons(BusinessEvent v) =>
      v.publicSuppressionReasons;
  static const Field<BusinessEvent, List<String>> _f$publicSuppressionReasons =
      Field('publicSuppressionReasons', _$publicSuppressionReasons);
  static List<String> _$upcomingExclusionReasons(BusinessEvent v) =>
      v.upcomingExclusionReasons;
  static const Field<BusinessEvent, List<String>> _f$upcomingExclusionReasons =
      Field('upcomingExclusionReasons', _$upcomingExclusionReasons);
  static BusinessOpenValue? _$mode(BusinessEvent v) => v.mode;
  static const Field<BusinessEvent, BusinessOpenValue> _f$mode = Field(
    'mode',
    _$mode,
    opt: true,
  );
  static String? _$timeZone(BusinessEvent v) => v.timeZone;
  static const Field<BusinessEvent, String> _f$timeZone = Field(
    'timeZone',
    _$timeZone,
    opt: true,
  );
  static String? _$summary(BusinessEvent v) => v.summary;
  static const Field<BusinessEvent, String> _f$summary = Field(
    'summary',
    _$summary,
    opt: true,
  );
  static String? _$venueName(BusinessEvent v) => v.venueName;
  static const Field<BusinessEvent, String> _f$venueName = Field(
    'venueName',
    _$venueName,
    opt: true,
  );
  static String? _$eventUri(BusinessEvent v) => v.eventUri;
  static const Field<BusinessEvent, String> _f$eventUri = Field(
    'eventUri',
    _$eventUri,
    opt: true,
  );
  static String? _$registrationUri(BusinessEvent v) => v.registrationUri;
  static const Field<BusinessEvent, String> _f$registrationUri = Field(
    'registrationUri',
    _$registrationUri,
    opt: true,
  );
  static BusinessImageView? _$image(BusinessEvent v) => v.image;
  static const Field<BusinessEvent, BusinessImageView> _f$image = Field(
    'image',
    _$image,
    opt: true,
  );

  @override
  final MappableFields<BusinessEvent> fields = const {
    #did: _f$did,
    #rkey: _f$rkey,
    #uri: _f$uri,
    #cid: _f$cid,
    #name: _f$name,
    #startsAt: _f$startsAt,
    #endsAt: _f$endsAt,
    #roles: _f$roles,
    #status: _f$status,
    #isAllDay: _f$isAllDay,
    #createdAt: _f$createdAt,
    #past: _f$past,
    #publicSuppressionReasons: _f$publicSuppressionReasons,
    #upcomingExclusionReasons: _f$upcomingExclusionReasons,
    #mode: _f$mode,
    #timeZone: _f$timeZone,
    #summary: _f$summary,
    #venueName: _f$venueName,
    #eventUri: _f$eventUri,
    #registrationUri: _f$registrationUri,
    #image: _f$image,
  };
  @override
  final bool ignoreNull = true;

  static BusinessEvent _instantiate(DecodingData data) {
    return BusinessEvent(
      did: data.dec(_f$did),
      rkey: data.dec(_f$rkey),
      uri: data.dec(_f$uri),
      cid: data.dec(_f$cid),
      name: data.dec(_f$name),
      startsAt: data.dec(_f$startsAt),
      endsAt: data.dec(_f$endsAt),
      roles: data.dec(_f$roles),
      status: data.dec(_f$status),
      isAllDay: data.dec(_f$isAllDay),
      createdAt: data.dec(_f$createdAt),
      past: data.dec(_f$past),
      publicSuppressionReasons: data.dec(_f$publicSuppressionReasons),
      upcomingExclusionReasons: data.dec(_f$upcomingExclusionReasons),
      mode: data.dec(_f$mode),
      timeZone: data.dec(_f$timeZone),
      summary: data.dec(_f$summary),
      venueName: data.dec(_f$venueName),
      eventUri: data.dec(_f$eventUri),
      registrationUri: data.dec(_f$registrationUri),
      image: data.dec(_f$image),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static BusinessEvent fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<BusinessEvent>(map);
  }

  static BusinessEvent fromJson(String json) {
    return ensureInitialized().decodeJson<BusinessEvent>(json);
  }
}
mixin BusinessEventMappable {
  String toJson() {
    return BusinessEventMapper.ensureInitialized().encodeJson<BusinessEvent>(
      this as BusinessEvent,
    );
  }

  Map<String, dynamic> toMap() {
    return BusinessEventMapper.ensureInitialized().encodeMap<BusinessEvent>(
      this as BusinessEvent,
    );
  }

  BusinessEventCopyWith<BusinessEvent, BusinessEvent, BusinessEvent>
  get copyWith => _BusinessEventCopyWithImpl<BusinessEvent, BusinessEvent>(
    this as BusinessEvent,
    $identity,
    $identity,
  );
  @override
  String toString() {
    return BusinessEventMapper.ensureInitialized().stringifyValue(
      this as BusinessEvent,
    );
  }

  @override
  bool operator ==(Object other) {
    return BusinessEventMapper.ensureInitialized().equalsValue(
      this as BusinessEvent,
      other,
    );
  }

  @override
  int get hashCode {
    return BusinessEventMapper.ensureInitialized().hashValue(
      this as BusinessEvent,
    );
  }
}

extension BusinessEventValueCopy<$R, $Out>
    on ObjectCopyWith<$R, BusinessEvent, $Out> {
  BusinessEventCopyWith<$R, BusinessEvent, $Out> get $asBusinessEvent =>
      $base.as((v, t, t2) => _BusinessEventCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class BusinessEventCopyWith<$R, $In extends BusinessEvent, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<
    $R,
    BusinessOpenValue,
    BusinessOpenValueCopyWith<$R, BusinessOpenValue, BusinessOpenValue>
  >
  get roles;
  BusinessOpenValueCopyWith<$R, BusinessOpenValue, BusinessOpenValue>
  get status;
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>>
  get publicSuppressionReasons;
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>>
  get upcomingExclusionReasons;
  BusinessOpenValueCopyWith<$R, BusinessOpenValue, BusinessOpenValue>? get mode;
  BusinessImageViewCopyWith<$R, BusinessImageView, BusinessImageView>?
  get image;
  $R call({
    String? did,
    String? rkey,
    String? uri,
    String? cid,
    String? name,
    DateTime? startsAt,
    DateTime? endsAt,
    List<BusinessOpenValue>? roles,
    BusinessOpenValue? status,
    bool? isAllDay,
    DateTime? createdAt,
    bool? past,
    List<String>? publicSuppressionReasons,
    List<String>? upcomingExclusionReasons,
    BusinessOpenValue? mode,
    String? timeZone,
    String? summary,
    String? venueName,
    String? eventUri,
    String? registrationUri,
    BusinessImageView? image,
  });
  BusinessEventCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _BusinessEventCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, BusinessEvent, $Out>
    implements BusinessEventCopyWith<$R, BusinessEvent, $Out> {
  _BusinessEventCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<BusinessEvent> $mapper =
      BusinessEventMapper.ensureInitialized();
  @override
  ListCopyWith<
    $R,
    BusinessOpenValue,
    BusinessOpenValueCopyWith<$R, BusinessOpenValue, BusinessOpenValue>
  >
  get roles => ListCopyWith(
    $value.roles,
    (v, t) => v.copyWith.$chain(t),
    (v) => call(roles: v),
  );
  @override
  BusinessOpenValueCopyWith<$R, BusinessOpenValue, BusinessOpenValue>
  get status => $value.status.copyWith.$chain((v) => call(status: v));
  @override
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>>
  get publicSuppressionReasons => ListCopyWith(
    $value.publicSuppressionReasons,
    (v, t) => ObjectCopyWith(v, $identity, t),
    (v) => call(publicSuppressionReasons: v),
  );
  @override
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>>
  get upcomingExclusionReasons => ListCopyWith(
    $value.upcomingExclusionReasons,
    (v, t) => ObjectCopyWith(v, $identity, t),
    (v) => call(upcomingExclusionReasons: v),
  );
  @override
  BusinessOpenValueCopyWith<$R, BusinessOpenValue, BusinessOpenValue>?
  get mode => $value.mode?.copyWith.$chain((v) => call(mode: v));
  @override
  BusinessImageViewCopyWith<$R, BusinessImageView, BusinessImageView>?
  get image => $value.image?.copyWith.$chain((v) => call(image: v));
  @override
  $R call({
    String? did,
    String? rkey,
    String? uri,
    String? cid,
    String? name,
    DateTime? startsAt,
    DateTime? endsAt,
    List<BusinessOpenValue>? roles,
    BusinessOpenValue? status,
    bool? isAllDay,
    DateTime? createdAt,
    bool? past,
    List<String>? publicSuppressionReasons,
    List<String>? upcomingExclusionReasons,
    Object? mode = $none,
    Object? timeZone = $none,
    Object? summary = $none,
    Object? venueName = $none,
    Object? eventUri = $none,
    Object? registrationUri = $none,
    Object? image = $none,
  }) => $apply(
    FieldCopyWithData({
      if (did != null) #did: did,
      if (rkey != null) #rkey: rkey,
      if (uri != null) #uri: uri,
      if (cid != null) #cid: cid,
      if (name != null) #name: name,
      if (startsAt != null) #startsAt: startsAt,
      if (endsAt != null) #endsAt: endsAt,
      if (roles != null) #roles: roles,
      if (status != null) #status: status,
      if (isAllDay != null) #isAllDay: isAllDay,
      if (createdAt != null) #createdAt: createdAt,
      if (past != null) #past: past,
      if (publicSuppressionReasons != null)
        #publicSuppressionReasons: publicSuppressionReasons,
      if (upcomingExclusionReasons != null)
        #upcomingExclusionReasons: upcomingExclusionReasons,
      if (mode != $none) #mode: mode,
      if (timeZone != $none) #timeZone: timeZone,
      if (summary != $none) #summary: summary,
      if (venueName != $none) #venueName: venueName,
      if (eventUri != $none) #eventUri: eventUri,
      if (registrationUri != $none) #registrationUri: registrationUri,
      if (image != $none) #image: image,
    }),
  );
  @override
  BusinessEvent $make(CopyWithData data) => BusinessEvent(
    did: data.get(#did, or: $value.did),
    rkey: data.get(#rkey, or: $value.rkey),
    uri: data.get(#uri, or: $value.uri),
    cid: data.get(#cid, or: $value.cid),
    name: data.get(#name, or: $value.name),
    startsAt: data.get(#startsAt, or: $value.startsAt),
    endsAt: data.get(#endsAt, or: $value.endsAt),
    roles: data.get(#roles, or: $value.roles),
    status: data.get(#status, or: $value.status),
    isAllDay: data.get(#isAllDay, or: $value.isAllDay),
    createdAt: data.get(#createdAt, or: $value.createdAt),
    past: data.get(#past, or: $value.past),
    publicSuppressionReasons: data.get(
      #publicSuppressionReasons,
      or: $value.publicSuppressionReasons,
    ),
    upcomingExclusionReasons: data.get(
      #upcomingExclusionReasons,
      or: $value.upcomingExclusionReasons,
    ),
    mode: data.get(#mode, or: $value.mode),
    timeZone: data.get(#timeZone, or: $value.timeZone),
    summary: data.get(#summary, or: $value.summary),
    venueName: data.get(#venueName, or: $value.venueName),
    eventUri: data.get(#eventUri, or: $value.eventUri),
    registrationUri: data.get(#registrationUri, or: $value.registrationUri),
    image: data.get(#image, or: $value.image),
  );

  @override
  BusinessEventCopyWith<$R2, BusinessEvent, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _BusinessEventCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class BusinessEventPageMapper extends ClassMapperBase<BusinessEventPage> {
  BusinessEventPageMapper._();

  static BusinessEventPageMapper? _instance;
  static BusinessEventPageMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = BusinessEventPageMapper._());
      BusinessEventMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'BusinessEventPage';

  static List<BusinessEvent> _$items(BusinessEventPage v) => v.items;
  static const Field<BusinessEventPage, List<BusinessEvent>> _f$items = Field(
    'items',
    _$items,
  );
  static String? _$cursor(BusinessEventPage v) => v.cursor;
  static const Field<BusinessEventPage, String> _f$cursor = Field(
    'cursor',
    _$cursor,
    opt: true,
  );

  @override
  final MappableFields<BusinessEventPage> fields = const {
    #items: _f$items,
    #cursor: _f$cursor,
  };
  @override
  final bool ignoreNull = true;

  static BusinessEventPage _instantiate(DecodingData data) {
    return BusinessEventPage(
      items: data.dec(_f$items),
      cursor: data.dec(_f$cursor),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static BusinessEventPage fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<BusinessEventPage>(map);
  }

  static BusinessEventPage fromJson(String json) {
    return ensureInitialized().decodeJson<BusinessEventPage>(json);
  }
}

mixin BusinessEventPageMappable {
  String toJson() {
    return BusinessEventPageMapper.ensureInitialized()
        .encodeJson<BusinessEventPage>(this as BusinessEventPage);
  }

  Map<String, dynamic> toMap() {
    return BusinessEventPageMapper.ensureInitialized()
        .encodeMap<BusinessEventPage>(this as BusinessEventPage);
  }

  BusinessEventPageCopyWith<
    BusinessEventPage,
    BusinessEventPage,
    BusinessEventPage
  >
  get copyWith =>
      _BusinessEventPageCopyWithImpl<BusinessEventPage, BusinessEventPage>(
        this as BusinessEventPage,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return BusinessEventPageMapper.ensureInitialized().stringifyValue(
      this as BusinessEventPage,
    );
  }

  @override
  bool operator ==(Object other) {
    return BusinessEventPageMapper.ensureInitialized().equalsValue(
      this as BusinessEventPage,
      other,
    );
  }

  @override
  int get hashCode {
    return BusinessEventPageMapper.ensureInitialized().hashValue(
      this as BusinessEventPage,
    );
  }
}

extension BusinessEventPageValueCopy<$R, $Out>
    on ObjectCopyWith<$R, BusinessEventPage, $Out> {
  BusinessEventPageCopyWith<$R, BusinessEventPage, $Out>
  get $asBusinessEventPage => $base.as(
    (v, t, t2) => _BusinessEventPageCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class BusinessEventPageCopyWith<
  $R,
  $In extends BusinessEventPage,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<
    $R,
    BusinessEvent,
    BusinessEventCopyWith<$R, BusinessEvent, BusinessEvent>
  >
  get items;
  $R call({List<BusinessEvent>? items, String? cursor});
  BusinessEventPageCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _BusinessEventPageCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, BusinessEventPage, $Out>
    implements BusinessEventPageCopyWith<$R, BusinessEventPage, $Out> {
  _BusinessEventPageCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<BusinessEventPage> $mapper =
      BusinessEventPageMapper.ensureInitialized();
  @override
  ListCopyWith<
    $R,
    BusinessEvent,
    BusinessEventCopyWith<$R, BusinessEvent, BusinessEvent>
  >
  get items => ListCopyWith(
    $value.items,
    (v, t) => v.copyWith.$chain(t),
    (v) => call(items: v),
  );
  @override
  $R call({List<BusinessEvent>? items, Object? cursor = $none}) => $apply(
    FieldCopyWithData({
      if (items != null) #items: items,
      if (cursor != $none) #cursor: cursor,
    }),
  );
  @override
  BusinessEventPage $make(CopyWithData data) => BusinessEventPage(
    items: data.get(#items, or: $value.items),
    cursor: data.get(#cursor, or: $value.cursor),
  );

  @override
  BusinessEventPageCopyWith<$R2, BusinessEventPage, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _BusinessEventPageCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class RecordMutationResultMapper extends ClassMapperBase<RecordMutationResult> {
  RecordMutationResultMapper._();

  static RecordMutationResultMapper? _instance;
  static RecordMutationResultMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = RecordMutationResultMapper._());
      MapperContainer.globals.useAll([
        DidMapper(),
        RecordKeyMapper(),
        AtUriMapper(),
        CidMapper(),
      ]);
    }
    return _instance!;
  }

  @override
  final String id = 'RecordMutationResult';

  static Cid _$cid(RecordMutationResult v) => v.cid;
  static dynamic _arg$cid(f) => f<Cid>();
  static const Field<RecordMutationResult, String> _f$cid = Field(
    'cid',
    _$cid,
    arg: _arg$cid,
  );
  static Did? _$did(RecordMutationResult v) => v.did;
  static dynamic _arg$did(f) => f<Did>();
  static const Field<RecordMutationResult, String> _f$did = Field(
    'did',
    _$did,
    opt: true,
    arg: _arg$did,
  );
  static RecordKey? _$rkey(RecordMutationResult v) => v.rkey;
  static dynamic _arg$rkey(f) => f<RecordKey>();
  static const Field<RecordMutationResult, String> _f$rkey = Field(
    'rkey',
    _$rkey,
    opt: true,
    arg: _arg$rkey,
  );
  static AtUri? _$uri(RecordMutationResult v) => v.uri;
  static dynamic _arg$uri(f) => f<AtUri>();
  static const Field<RecordMutationResult, String> _f$uri = Field(
    'uri',
    _$uri,
    opt: true,
    arg: _arg$uri,
  );

  @override
  final MappableFields<RecordMutationResult> fields = const {
    #cid: _f$cid,
    #did: _f$did,
    #rkey: _f$rkey,
    #uri: _f$uri,
  };
  @override
  final bool ignoreNull = true;

  static RecordMutationResult _instantiate(DecodingData data) {
    return RecordMutationResult(
      cid: data.dec(_f$cid),
      did: data.dec(_f$did),
      rkey: data.dec(_f$rkey),
      uri: data.dec(_f$uri),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static RecordMutationResult fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<RecordMutationResult>(map);
  }

  static RecordMutationResult fromJson(String json) {
    return ensureInitialized().decodeJson<RecordMutationResult>(json);
  }
}

mixin RecordMutationResultMappable {
  String toJson() {
    return RecordMutationResultMapper.ensureInitialized()
        .encodeJson<RecordMutationResult>(this as RecordMutationResult);
  }

  Map<String, dynamic> toMap() {
    return RecordMutationResultMapper.ensureInitialized()
        .encodeMap<RecordMutationResult>(this as RecordMutationResult);
  }

  RecordMutationResultCopyWith<
    RecordMutationResult,
    RecordMutationResult,
    RecordMutationResult
  >
  get copyWith =>
      _RecordMutationResultCopyWithImpl<
        RecordMutationResult,
        RecordMutationResult
      >(this as RecordMutationResult, $identity, $identity);
  @override
  String toString() {
    return RecordMutationResultMapper.ensureInitialized().stringifyValue(
      this as RecordMutationResult,
    );
  }

  @override
  bool operator ==(Object other) {
    return RecordMutationResultMapper.ensureInitialized().equalsValue(
      this as RecordMutationResult,
      other,
    );
  }

  @override
  int get hashCode {
    return RecordMutationResultMapper.ensureInitialized().hashValue(
      this as RecordMutationResult,
    );
  }
}

extension RecordMutationResultValueCopy<$R, $Out>
    on ObjectCopyWith<$R, RecordMutationResult, $Out> {
  RecordMutationResultCopyWith<$R, RecordMutationResult, $Out>
  get $asRecordMutationResult => $base.as(
    (v, t, t2) => _RecordMutationResultCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class RecordMutationResultCopyWith<
  $R,
  $In extends RecordMutationResult,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? cid, String? did, String? rkey, String? uri});
  RecordMutationResultCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _RecordMutationResultCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, RecordMutationResult, $Out>
    implements RecordMutationResultCopyWith<$R, RecordMutationResult, $Out> {
  _RecordMutationResultCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<RecordMutationResult> $mapper =
      RecordMutationResultMapper.ensureInitialized();
  @override
  $R call({
    String? cid,
    Object? did = $none,
    Object? rkey = $none,
    Object? uri = $none,
  }) => $apply(
    FieldCopyWithData({
      if (cid != null) #cid: cid,
      if (did != $none) #did: did,
      if (rkey != $none) #rkey: rkey,
      if (uri != $none) #uri: uri,
    }),
  );
  @override
  RecordMutationResult $make(CopyWithData data) => RecordMutationResult(
    cid: data.get(#cid, or: $value.cid),
    did: data.get(#did, or: $value.did),
    rkey: data.get(#rkey, or: $value.rkey),
    uri: data.get(#uri, or: $value.uri),
  );

  @override
  RecordMutationResultCopyWith<$R2, RecordMutationResult, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _RecordMutationResultCopyWithImpl<$R2, $Out2>($value, $cast, t);
}
