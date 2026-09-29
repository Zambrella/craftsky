// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'timeline_page.dart';

class RepostReasonTypeMapper extends EnumMapper<RepostReasonType> {
  RepostReasonTypeMapper._();

  static RepostReasonTypeMapper? _instance;
  static RepostReasonTypeMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = RepostReasonTypeMapper._());
    }
    return _instance!;
  }

  static RepostReasonType fromValue(dynamic value) {
    ensureInitialized();
    return MapperContainer.globals.fromValue(value);
  }

  @override
  RepostReasonType decode(dynamic value) {
    switch (value) {
      case r'repost':
        return RepostReasonType.repost;
      default:
        throw MapperException.unknownEnumValue(value);
    }
  }

  @override
  dynamic encode(RepostReasonType self) {
    switch (self) {
      case RepostReasonType.repost:
        return r'repost';
    }
  }
}
extension RepostReasonTypeMapperExtension on RepostReasonType {
  String toValue() {
    RepostReasonTypeMapper.ensureInitialized();
    return MapperContainer.globals.toValue<RepostReasonType>(this) as String;
  }
}

class TimelinePageMapper extends ClassMapperBase<TimelinePage> {
  TimelinePageMapper._();

  static TimelinePageMapper? _instance;
  static TimelinePageMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = TimelinePageMapper._());
      TimelineItemMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'TimelinePage';

  static List<TimelineItem> _$items(TimelinePage v) => v.items;
  static const Field<TimelinePage, List<TimelineItem>> _f$items = Field(
    'items',
    _$items,
  );
  static String? _$cursor(TimelinePage v) => v.cursor;
  static const Field<TimelinePage, String> _f$cursor = Field(
    'cursor',
    _$cursor,
    opt: true,
  );

  @override
  final MappableFields<TimelinePage> fields = const {
    #items: _f$items,
    #cursor: _f$cursor,
  };
  @override
  final bool ignoreNull = true;

  static TimelinePage _instantiate(DecodingData data) {
    return TimelinePage(items: data.dec(_f$items), cursor: data.dec(_f$cursor));
  }

  @override
  final Function instantiate = _instantiate;

  static TimelinePage fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<TimelinePage>(map);
  }

  static TimelinePage fromJson(String json) {
    return ensureInitialized().decodeJson<TimelinePage>(json);
  }
}

mixin TimelinePageMappable {
  String toJson() {
    return TimelinePageMapper.ensureInitialized().encodeJson<TimelinePage>(
      this as TimelinePage,
    );
  }

  Map<String, dynamic> toMap() {
    return TimelinePageMapper.ensureInitialized().encodeMap<TimelinePage>(
      this as TimelinePage,
    );
  }

  TimelinePageCopyWith<TimelinePage, TimelinePage, TimelinePage> get copyWith =>
      _TimelinePageCopyWithImpl<TimelinePage, TimelinePage>(
        this as TimelinePage,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return TimelinePageMapper.ensureInitialized().stringifyValue(
      this as TimelinePage,
    );
  }

  @override
  bool operator ==(Object other) {
    return TimelinePageMapper.ensureInitialized().equalsValue(
      this as TimelinePage,
      other,
    );
  }

  @override
  int get hashCode {
    return TimelinePageMapper.ensureInitialized().hashValue(
      this as TimelinePage,
    );
  }
}

extension TimelinePageValueCopy<$R, $Out>
    on ObjectCopyWith<$R, TimelinePage, $Out> {
  TimelinePageCopyWith<$R, TimelinePage, $Out> get $asTimelinePage =>
      $base.as((v, t, t2) => _TimelinePageCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class TimelinePageCopyWith<$R, $In extends TimelinePage, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<
    $R,
    TimelineItem,
    TimelineItemCopyWith<$R, TimelineItem, TimelineItem>
  >
  get items;
  $R call({List<TimelineItem>? items, String? cursor});
  TimelinePageCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _TimelinePageCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, TimelinePage, $Out>
    implements TimelinePageCopyWith<$R, TimelinePage, $Out> {
  _TimelinePageCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<TimelinePage> $mapper =
      TimelinePageMapper.ensureInitialized();
  @override
  ListCopyWith<
    $R,
    TimelineItem,
    TimelineItemCopyWith<$R, TimelineItem, TimelineItem>
  >
  get items => ListCopyWith(
    $value.items,
    (v, t) => v.copyWith.$chain(t),
    (v) => call(items: v),
  );
  @override
  $R call({List<TimelineItem>? items, Object? cursor = $none}) => $apply(
    FieldCopyWithData({
      if (items != null) #items: items,
      if (cursor != $none) #cursor: cursor,
    }),
  );
  @override
  TimelinePage $make(CopyWithData data) => TimelinePage(
    items: data.get(#items, or: $value.items),
    cursor: data.get(#cursor, or: $value.cursor),
  );

  @override
  TimelinePageCopyWith<$R2, TimelinePage, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _TimelinePageCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class TimelineItemMapper extends ClassMapperBase<TimelineItem> {
  TimelineItemMapper._();

  static TimelineItemMapper? _instance;
  static TimelineItemMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = TimelineItemMapper._());
      PostMapper.ensureInitialized();
      RepostReasonMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'TimelineItem';

  static String _$itemKey(TimelineItem v) => v.itemKey;
  static const Field<TimelineItem, String> _f$itemKey = Field(
    'itemKey',
    _$itemKey,
  );
  static Post _$post(TimelineItem v) => v.post;
  static const Field<TimelineItem, Post> _f$post = Field('post', _$post);
  static RepostReason? _$reason(TimelineItem v) => v.reason;
  static const Field<TimelineItem, RepostReason> _f$reason = Field(
    'reason',
    _$reason,
    opt: true,
  );

  @override
  final MappableFields<TimelineItem> fields = const {
    #itemKey: _f$itemKey,
    #post: _f$post,
    #reason: _f$reason,
  };
  @override
  final bool ignoreNull = true;

  static TimelineItem _instantiate(DecodingData data) {
    return TimelineItem(
      itemKey: data.dec(_f$itemKey),
      post: data.dec(_f$post),
      reason: data.dec(_f$reason),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static TimelineItem fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<TimelineItem>(map);
  }

  static TimelineItem fromJson(String json) {
    return ensureInitialized().decodeJson<TimelineItem>(json);
  }
}

mixin TimelineItemMappable {
  String toJson() {
    return TimelineItemMapper.ensureInitialized().encodeJson<TimelineItem>(
      this as TimelineItem,
    );
  }

  Map<String, dynamic> toMap() {
    return TimelineItemMapper.ensureInitialized().encodeMap<TimelineItem>(
      this as TimelineItem,
    );
  }

  TimelineItemCopyWith<TimelineItem, TimelineItem, TimelineItem> get copyWith =>
      _TimelineItemCopyWithImpl<TimelineItem, TimelineItem>(
        this as TimelineItem,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return TimelineItemMapper.ensureInitialized().stringifyValue(
      this as TimelineItem,
    );
  }

  @override
  bool operator ==(Object other) {
    return TimelineItemMapper.ensureInitialized().equalsValue(
      this as TimelineItem,
      other,
    );
  }

  @override
  int get hashCode {
    return TimelineItemMapper.ensureInitialized().hashValue(
      this as TimelineItem,
    );
  }
}

extension TimelineItemValueCopy<$R, $Out>
    on ObjectCopyWith<$R, TimelineItem, $Out> {
  TimelineItemCopyWith<$R, TimelineItem, $Out> get $asTimelineItem =>
      $base.as((v, t, t2) => _TimelineItemCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class TimelineItemCopyWith<$R, $In extends TimelineItem, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  PostCopyWith<$R, Post, Post> get post;
  RepostReasonCopyWith<$R, RepostReason, RepostReason>? get reason;
  $R call({String? itemKey, Post? post, RepostReason? reason});
  TimelineItemCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _TimelineItemCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, TimelineItem, $Out>
    implements TimelineItemCopyWith<$R, TimelineItem, $Out> {
  _TimelineItemCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<TimelineItem> $mapper =
      TimelineItemMapper.ensureInitialized();
  @override
  PostCopyWith<$R, Post, Post> get post =>
      $value.post.copyWith.$chain((v) => call(post: v));
  @override
  RepostReasonCopyWith<$R, RepostReason, RepostReason>? get reason =>
      $value.reason?.copyWith.$chain((v) => call(reason: v));
  @override
  $R call({String? itemKey, Post? post, Object? reason = $none}) => $apply(
    FieldCopyWithData({
      if (itemKey != null) #itemKey: itemKey,
      if (post != null) #post: post,
      if (reason != $none) #reason: reason,
    }),
  );
  @override
  TimelineItem $make(CopyWithData data) => TimelineItem(
    itemKey: data.get(#itemKey, or: $value.itemKey),
    post: data.get(#post, or: $value.post),
    reason: data.get(#reason, or: $value.reason),
  );

  @override
  TimelineItemCopyWith<$R2, TimelineItem, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _TimelineItemCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class RepostReasonMapper extends ClassMapperBase<RepostReason> {
  RepostReasonMapper._();

  static RepostReasonMapper? _instance;
  static RepostReasonMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = RepostReasonMapper._());
      MapperContainer.globals.useAll([AtUriMapper(), CidMapper()]);
      RepostReasonTypeMapper.ensureInitialized();
      PostAuthorMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'RepostReason';

  static RepostReasonType _$type(RepostReason v) => v.type;
  static const Field<RepostReason, RepostReasonType> _f$type = Field(
    'type',
    _$type,
  );
  static PostAuthor _$by(RepostReason v) => v.by;
  static const Field<RepostReason, PostAuthor> _f$by = Field('by', _$by);
  static AtUri _$uri(RepostReason v) => v.uri;
  static dynamic _arg$uri(f) => f<AtUri>();
  static const Field<RepostReason, String> _f$uri = Field(
    'uri',
    _$uri,
    arg: _arg$uri,
  );
  static DateTime _$createdAt(RepostReason v) => v.createdAt;
  static const Field<RepostReason, DateTime> _f$createdAt = Field(
    'createdAt',
    _$createdAt,
  );
  static DateTime _$indexedAt(RepostReason v) => v.indexedAt;
  static const Field<RepostReason, DateTime> _f$indexedAt = Field(
    'indexedAt',
    _$indexedAt,
  );
  static Cid? _$cid(RepostReason v) => v.cid;
  static dynamic _arg$cid(f) => f<Cid>();
  static const Field<RepostReason, String> _f$cid = Field(
    'cid',
    _$cid,
    opt: true,
    arg: _arg$cid,
  );

  @override
  final MappableFields<RepostReason> fields = const {
    #type: _f$type,
    #by: _f$by,
    #uri: _f$uri,
    #createdAt: _f$createdAt,
    #indexedAt: _f$indexedAt,
    #cid: _f$cid,
  };
  @override
  final bool ignoreNull = true;

  static RepostReason _instantiate(DecodingData data) {
    return RepostReason(
      type: data.dec(_f$type),
      by: data.dec(_f$by),
      uri: data.dec(_f$uri),
      createdAt: data.dec(_f$createdAt),
      indexedAt: data.dec(_f$indexedAt),
      cid: data.dec(_f$cid),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static RepostReason fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<RepostReason>(map);
  }

  static RepostReason fromJson(String json) {
    return ensureInitialized().decodeJson<RepostReason>(json);
  }
}

mixin RepostReasonMappable {
  String toJson() {
    return RepostReasonMapper.ensureInitialized().encodeJson<RepostReason>(
      this as RepostReason,
    );
  }

  Map<String, dynamic> toMap() {
    return RepostReasonMapper.ensureInitialized().encodeMap<RepostReason>(
      this as RepostReason,
    );
  }

  RepostReasonCopyWith<RepostReason, RepostReason, RepostReason> get copyWith =>
      _RepostReasonCopyWithImpl<RepostReason, RepostReason>(
        this as RepostReason,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return RepostReasonMapper.ensureInitialized().stringifyValue(
      this as RepostReason,
    );
  }

  @override
  bool operator ==(Object other) {
    return RepostReasonMapper.ensureInitialized().equalsValue(
      this as RepostReason,
      other,
    );
  }

  @override
  int get hashCode {
    return RepostReasonMapper.ensureInitialized().hashValue(
      this as RepostReason,
    );
  }
}

extension RepostReasonValueCopy<$R, $Out>
    on ObjectCopyWith<$R, RepostReason, $Out> {
  RepostReasonCopyWith<$R, RepostReason, $Out> get $asRepostReason =>
      $base.as((v, t, t2) => _RepostReasonCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class RepostReasonCopyWith<$R, $In extends RepostReason, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  PostAuthorCopyWith<$R, PostAuthor, PostAuthor> get by;
  $R call({
    RepostReasonType? type,
    PostAuthor? by,
    String? uri,
    DateTime? createdAt,
    DateTime? indexedAt,
    String? cid,
  });
  RepostReasonCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _RepostReasonCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, RepostReason, $Out>
    implements RepostReasonCopyWith<$R, RepostReason, $Out> {
  _RepostReasonCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<RepostReason> $mapper =
      RepostReasonMapper.ensureInitialized();
  @override
  PostAuthorCopyWith<$R, PostAuthor, PostAuthor> get by =>
      $value.by.copyWith.$chain((v) => call(by: v));
  @override
  $R call({
    RepostReasonType? type,
    PostAuthor? by,
    String? uri,
    DateTime? createdAt,
    DateTime? indexedAt,
    Object? cid = $none,
  }) => $apply(
    FieldCopyWithData({
      if (type != null) #type: type,
      if (by != null) #by: by,
      if (uri != null) #uri: uri,
      if (createdAt != null) #createdAt: createdAt,
      if (indexedAt != null) #indexedAt: indexedAt,
      if (cid != $none) #cid: cid,
    }),
  );
  @override
  RepostReason $make(CopyWithData data) => RepostReason(
    type: data.get(#type, or: $value.type),
    by: data.get(#by, or: $value.by),
    uri: data.get(#uri, or: $value.uri),
    createdAt: data.get(#createdAt, or: $value.createdAt),
    indexedAt: data.get(#indexedAt, or: $value.indexedAt),
    cid: data.get(#cid, or: $value.cid),
  );

  @override
  RepostReasonCopyWith<$R2, RepostReason, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _RepostReasonCopyWithImpl<$R2, $Out2>($value, $cast, t);
}
