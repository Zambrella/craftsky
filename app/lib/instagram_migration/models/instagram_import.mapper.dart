// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'instagram_import.dart';

class InstagramImportSourceTypeMapper
    extends EnumMapper<InstagramImportSourceType> {
  InstagramImportSourceTypeMapper._();

  static InstagramImportSourceTypeMapper? _instance;
  static InstagramImportSourceTypeMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = InstagramImportSourceTypeMapper._(),
      );
    }
    return _instance!;
  }

  static InstagramImportSourceType fromValue(dynamic value) {
    ensureInitialized();
    return MapperContainer.globals.fromValue(value);
  }

  @override
  InstagramImportSourceType decode(dynamic value) {
    switch (value) {
      case r'manual':
        return InstagramImportSourceType.manual;
      case r'instagramJson':
        return InstagramImportSourceType.instagramJson;
      case r'unknown':
        return InstagramImportSourceType.unknown;
      default:
        return InstagramImportSourceType.values[2];
    }
  }

  @override
  dynamic encode(InstagramImportSourceType self) {
    switch (self) {
      case InstagramImportSourceType.manual:
        return r'manual';
      case InstagramImportSourceType.instagramJson:
        return r'instagramJson';
      case InstagramImportSourceType.unknown:
        return r'unknown';
    }
  }
}
extension InstagramImportSourceTypeMapperExtension
    on InstagramImportSourceType {
  String toValue() {
    InstagramImportSourceTypeMapper.ensureInitialized();
    return MapperContainer.globals.toValue<InstagramImportSourceType>(this)
        as String;
  }
}

class InstagramImportStateMapper extends EnumMapper<InstagramImportState> {
  InstagramImportStateMapper._();

  static InstagramImportStateMapper? _instance;
  static InstagramImportStateMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = InstagramImportStateMapper._());
    }
    return _instance!;
  }

  static InstagramImportState fromValue(dynamic value) {
    ensureInitialized();
    return MapperContainer.globals.fromValue(value);
  }

  @override
  InstagramImportState decode(dynamic value) {
    switch (value) {
      case r'active':
        return InstagramImportState.active;
      case r'membershipInactive':
        return InstagramImportState.membershipInactive;
      case r'unknown':
        return InstagramImportState.unknown;
      default:
        return InstagramImportState.values[2];
    }
  }

  @override
  dynamic encode(InstagramImportState self) {
    switch (self) {
      case InstagramImportState.active:
        return r'active';
      case InstagramImportState.membershipInactive:
        return r'membershipInactive';
      case InstagramImportState.unknown:
        return r'unknown';
    }
  }
}

extension InstagramImportStateMapperExtension on InstagramImportState {
  String toValue() {
    InstagramImportStateMapper.ensureInitialized();
    return MapperContainer.globals.toValue<InstagramImportState>(this)
        as String;
  }
}

class InstagramImportEntryMapper extends ClassMapperBase<InstagramImportEntry> {
  InstagramImportEntryMapper._();

  static InstagramImportEntryMapper? _instance;
  static InstagramImportEntryMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = InstagramImportEntryMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'InstagramImportEntry';

  static String _$username(InstagramImportEntry v) => v.username;
  static const Field<InstagramImportEntry, String> _f$username = Field(
    'username',
    _$username,
  );

  @override
  final MappableFields<InstagramImportEntry> fields = const {
    #username: _f$username,
  };

  static InstagramImportEntry _instantiate(DecodingData data) {
    return InstagramImportEntry(username: data.dec(_f$username));
  }

  @override
  final Function instantiate = _instantiate;
}

mixin InstagramImportEntryMappable {
  String toJson() {
    return InstagramImportEntryMapper.ensureInitialized()
        .encodeJson<InstagramImportEntry>(this as InstagramImportEntry);
  }

  Map<String, dynamic> toMap() {
    return InstagramImportEntryMapper.ensureInitialized()
        .encodeMap<InstagramImportEntry>(this as InstagramImportEntry);
  }

  InstagramImportEntryCopyWith<
    InstagramImportEntry,
    InstagramImportEntry,
    InstagramImportEntry
  >
  get copyWith =>
      _InstagramImportEntryCopyWithImpl<
        InstagramImportEntry,
        InstagramImportEntry
      >(this as InstagramImportEntry, $identity, $identity);
  @override
  bool operator ==(Object other) {
    return InstagramImportEntryMapper.ensureInitialized().equalsValue(
      this as InstagramImportEntry,
      other,
    );
  }

  @override
  int get hashCode {
    return InstagramImportEntryMapper.ensureInitialized().hashValue(
      this as InstagramImportEntry,
    );
  }
}

extension InstagramImportEntryValueCopy<$R, $Out>
    on ObjectCopyWith<$R, InstagramImportEntry, $Out> {
  InstagramImportEntryCopyWith<$R, InstagramImportEntry, $Out>
  get $asInstagramImportEntry => $base.as(
    (v, t, t2) => _InstagramImportEntryCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class InstagramImportEntryCopyWith<
  $R,
  $In extends InstagramImportEntry,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? username});
  InstagramImportEntryCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _InstagramImportEntryCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, InstagramImportEntry, $Out>
    implements InstagramImportEntryCopyWith<$R, InstagramImportEntry, $Out> {
  _InstagramImportEntryCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<InstagramImportEntry> $mapper =
      InstagramImportEntryMapper.ensureInitialized();
  @override
  $R call({String? username}) =>
      $apply(FieldCopyWithData({if (username != null) #username: username}));
  @override
  InstagramImportEntry $make(CopyWithData data) =>
      InstagramImportEntry(username: data.get(#username, or: $value.username));

  @override
  InstagramImportEntryCopyWith<$R2, InstagramImportEntry, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _InstagramImportEntryCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class InstagramImportParseResultMapper
    extends ClassMapperBase<InstagramImportParseResult> {
  InstagramImportParseResultMapper._();

  static InstagramImportParseResultMapper? _instance;
  static InstagramImportParseResultMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = InstagramImportParseResultMapper._(),
      );
      InstagramImportEntryMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'InstagramImportParseResult';

  static List<InstagramImportEntry> _$entries(InstagramImportParseResult v) =>
      v.entries;
  static const Field<InstagramImportParseResult, List<InstagramImportEntry>>
  _f$entries = Field('entries', _$entries);
  static int _$ignoredEntryCount(InstagramImportParseResult v) =>
      v.ignoredEntryCount;
  static const Field<InstagramImportParseResult, int> _f$ignoredEntryCount =
      Field('ignoredEntryCount', _$ignoredEntryCount, opt: true, def: 0);
  static int _$duplicateEntryCount(InstagramImportParseResult v) =>
      v.duplicateEntryCount;
  static const Field<InstagramImportParseResult, int> _f$duplicateEntryCount =
      Field('duplicateEntryCount', _$duplicateEntryCount, opt: true, def: 0);

  @override
  final MappableFields<InstagramImportParseResult> fields = const {
    #entries: _f$entries,
    #ignoredEntryCount: _f$ignoredEntryCount,
    #duplicateEntryCount: _f$duplicateEntryCount,
  };

  static InstagramImportParseResult _instantiate(DecodingData data) {
    return InstagramImportParseResult(
      entries: data.dec(_f$entries),
      ignoredEntryCount: data.dec(_f$ignoredEntryCount),
      duplicateEntryCount: data.dec(_f$duplicateEntryCount),
    );
  }

  @override
  final Function instantiate = _instantiate;
}

mixin InstagramImportParseResultMappable {
  InstagramImportParseResultCopyWith<
    InstagramImportParseResult,
    InstagramImportParseResult,
    InstagramImportParseResult
  >
  get copyWith =>
      _InstagramImportParseResultCopyWithImpl<
        InstagramImportParseResult,
        InstagramImportParseResult
      >(this as InstagramImportParseResult, $identity, $identity);
  @override
  bool operator ==(Object other) {
    return InstagramImportParseResultMapper.ensureInitialized().equalsValue(
      this as InstagramImportParseResult,
      other,
    );
  }

  @override
  int get hashCode {
    return InstagramImportParseResultMapper.ensureInitialized().hashValue(
      this as InstagramImportParseResult,
    );
  }
}

extension InstagramImportParseResultValueCopy<$R, $Out>
    on ObjectCopyWith<$R, InstagramImportParseResult, $Out> {
  InstagramImportParseResultCopyWith<$R, InstagramImportParseResult, $Out>
  get $asInstagramImportParseResult => $base.as(
    (v, t, t2) => _InstagramImportParseResultCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class InstagramImportParseResultCopyWith<
  $R,
  $In extends InstagramImportParseResult,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<
    $R,
    InstagramImportEntry,
    InstagramImportEntryCopyWith<$R, InstagramImportEntry, InstagramImportEntry>
  >
  get entries;
  $R call({
    List<InstagramImportEntry>? entries,
    int? ignoredEntryCount,
    int? duplicateEntryCount,
  });
  InstagramImportParseResultCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _InstagramImportParseResultCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, InstagramImportParseResult, $Out>
    implements
        InstagramImportParseResultCopyWith<
          $R,
          InstagramImportParseResult,
          $Out
        > {
  _InstagramImportParseResultCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<InstagramImportParseResult> $mapper =
      InstagramImportParseResultMapper.ensureInitialized();
  @override
  ListCopyWith<
    $R,
    InstagramImportEntry,
    InstagramImportEntryCopyWith<$R, InstagramImportEntry, InstagramImportEntry>
  >
  get entries => ListCopyWith(
    $value.entries,
    (v, t) => v.copyWith.$chain(t),
    (v) => call(entries: v),
  );
  @override
  $R call({
    List<InstagramImportEntry>? entries,
    int? ignoredEntryCount,
    int? duplicateEntryCount,
  }) => $apply(
    FieldCopyWithData({
      if (entries != null) #entries: entries,
      if (ignoredEntryCount != null) #ignoredEntryCount: ignoredEntryCount,
      if (duplicateEntryCount != null)
        #duplicateEntryCount: duplicateEntryCount,
    }),
  );
  @override
  InstagramImportParseResult $make(CopyWithData data) =>
      InstagramImportParseResult(
        entries: data.get(#entries, or: $value.entries),
        ignoredEntryCount: data.get(
          #ignoredEntryCount,
          or: $value.ignoredEntryCount,
        ),
        duplicateEntryCount: data.get(
          #duplicateEntryCount,
          or: $value.duplicateEntryCount,
        ),
      );

  @override
  InstagramImportParseResultCopyWith<$R2, InstagramImportParseResult, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _InstagramImportParseResultCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class InstagramImportRequestMapper
    extends ClassMapperBase<InstagramImportRequest> {
  InstagramImportRequestMapper._();

  static InstagramImportRequestMapper? _instance;
  static InstagramImportRequestMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = InstagramImportRequestMapper._());
      InstagramImportSourceTypeMapper.ensureInitialized();
      InstagramImportEntryMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'InstagramImportRequest';

  static InstagramImportSourceType _$sourceType(InstagramImportRequest v) =>
      v.sourceType;
  static const Field<InstagramImportRequest, InstagramImportSourceType>
  _f$sourceType = Field('sourceType', _$sourceType);
  static List<InstagramImportEntry> _$entries(InstagramImportRequest v) =>
      v.entries;
  static const Field<InstagramImportRequest, List<InstagramImportEntry>>
  _f$entries = Field('entries', _$entries);

  @override
  final MappableFields<InstagramImportRequest> fields = const {
    #sourceType: _f$sourceType,
    #entries: _f$entries,
  };

  static InstagramImportRequest _instantiate(DecodingData data) {
    return InstagramImportRequest(
      sourceType: data.dec(_f$sourceType),
      entries: data.dec(_f$entries),
    );
  }

  @override
  final Function instantiate = _instantiate;
}

mixin InstagramImportRequestMappable {
  InstagramImportRequestCopyWith<
    InstagramImportRequest,
    InstagramImportRequest,
    InstagramImportRequest
  >
  get copyWith =>
      _InstagramImportRequestCopyWithImpl<
        InstagramImportRequest,
        InstagramImportRequest
      >(this as InstagramImportRequest, $identity, $identity);
  @override
  bool operator ==(Object other) {
    return InstagramImportRequestMapper.ensureInitialized().equalsValue(
      this as InstagramImportRequest,
      other,
    );
  }

  @override
  int get hashCode {
    return InstagramImportRequestMapper.ensureInitialized().hashValue(
      this as InstagramImportRequest,
    );
  }
}

extension InstagramImportRequestValueCopy<$R, $Out>
    on ObjectCopyWith<$R, InstagramImportRequest, $Out> {
  InstagramImportRequestCopyWith<$R, InstagramImportRequest, $Out>
  get $asInstagramImportRequest => $base.as(
    (v, t, t2) => _InstagramImportRequestCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class InstagramImportRequestCopyWith<
  $R,
  $In extends InstagramImportRequest,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<
    $R,
    InstagramImportEntry,
    InstagramImportEntryCopyWith<$R, InstagramImportEntry, InstagramImportEntry>
  >
  get entries;
  $R call({
    InstagramImportSourceType? sourceType,
    List<InstagramImportEntry>? entries,
  });
  InstagramImportRequestCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _InstagramImportRequestCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, InstagramImportRequest, $Out>
    implements
        InstagramImportRequestCopyWith<$R, InstagramImportRequest, $Out> {
  _InstagramImportRequestCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<InstagramImportRequest> $mapper =
      InstagramImportRequestMapper.ensureInitialized();
  @override
  ListCopyWith<
    $R,
    InstagramImportEntry,
    InstagramImportEntryCopyWith<$R, InstagramImportEntry, InstagramImportEntry>
  >
  get entries => ListCopyWith(
    $value.entries,
    (v, t) => v.copyWith.$chain(t),
    (v) => call(entries: v),
  );
  @override
  $R call({
    InstagramImportSourceType? sourceType,
    List<InstagramImportEntry>? entries,
  }) => $apply(
    FieldCopyWithData({
      if (sourceType != null) #sourceType: sourceType,
      if (entries != null) #entries: entries,
    }),
  );
  @override
  InstagramImportRequest $make(CopyWithData data) => InstagramImportRequest(
    sourceType: data.get(#sourceType, or: $value.sourceType),
    entries: data.get(#entries, or: $value.entries),
  );

  @override
  InstagramImportRequestCopyWith<$R2, InstagramImportRequest, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _InstagramImportRequestCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class InstagramImportSummaryMapper
    extends ClassMapperBase<InstagramImportSummary> {
  InstagramImportSummaryMapper._();

  static InstagramImportSummaryMapper? _instance;
  static InstagramImportSummaryMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = InstagramImportSummaryMapper._());
      InstagramImportStateMapper.ensureInitialized();
      InstagramImportSourceTypeMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'InstagramImportSummary';

  static String _$importId(InstagramImportSummary v) => v.importId;
  static const Field<InstagramImportSummary, String> _f$importId = Field(
    'importId',
    _$importId,
  );
  static InstagramImportState _$state(InstagramImportSummary v) => v.state;
  static const Field<InstagramImportSummary, InstagramImportState> _f$state =
      Field('state', _$state);
  static InstagramImportSourceType _$sourceType(InstagramImportSummary v) =>
      v.sourceType;
  static const Field<InstagramImportSummary, InstagramImportSourceType>
  _f$sourceType = Field('sourceType', _$sourceType);
  static int _$followingCount(InstagramImportSummary v) => v.followingCount;
  static const Field<InstagramImportSummary, int> _f$followingCount = Field(
    'followingCount',
    _$followingCount,
  );
  static DateTime _$createdAt(InstagramImportSummary v) => v.createdAt;
  static const Field<InstagramImportSummary, DateTime> _f$createdAt = Field(
    'createdAt',
    _$createdAt,
  );

  @override
  final MappableFields<InstagramImportSummary> fields = const {
    #importId: _f$importId,
    #state: _f$state,
    #sourceType: _f$sourceType,
    #followingCount: _f$followingCount,
    #createdAt: _f$createdAt,
  };

  static InstagramImportSummary _instantiate(DecodingData data) {
    return InstagramImportSummary(
      importId: data.dec(_f$importId),
      state: data.dec(_f$state),
      sourceType: data.dec(_f$sourceType),
      followingCount: data.dec(_f$followingCount),
      createdAt: data.dec(_f$createdAt),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static InstagramImportSummary fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<InstagramImportSummary>(map);
  }

  static InstagramImportSummary fromJson(String json) {
    return ensureInitialized().decodeJson<InstagramImportSummary>(json);
  }
}

mixin InstagramImportSummaryMappable {
  InstagramImportSummaryCopyWith<
    InstagramImportSummary,
    InstagramImportSummary,
    InstagramImportSummary
  >
  get copyWith =>
      _InstagramImportSummaryCopyWithImpl<
        InstagramImportSummary,
        InstagramImportSummary
      >(this as InstagramImportSummary, $identity, $identity);
  @override
  bool operator ==(Object other) {
    return InstagramImportSummaryMapper.ensureInitialized().equalsValue(
      this as InstagramImportSummary,
      other,
    );
  }

  @override
  int get hashCode {
    return InstagramImportSummaryMapper.ensureInitialized().hashValue(
      this as InstagramImportSummary,
    );
  }
}

extension InstagramImportSummaryValueCopy<$R, $Out>
    on ObjectCopyWith<$R, InstagramImportSummary, $Out> {
  InstagramImportSummaryCopyWith<$R, InstagramImportSummary, $Out>
  get $asInstagramImportSummary => $base.as(
    (v, t, t2) => _InstagramImportSummaryCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class InstagramImportSummaryCopyWith<
  $R,
  $In extends InstagramImportSummary,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({
    String? importId,
    InstagramImportState? state,
    InstagramImportSourceType? sourceType,
    int? followingCount,
    DateTime? createdAt,
  });
  InstagramImportSummaryCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _InstagramImportSummaryCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, InstagramImportSummary, $Out>
    implements
        InstagramImportSummaryCopyWith<$R, InstagramImportSummary, $Out> {
  _InstagramImportSummaryCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<InstagramImportSummary> $mapper =
      InstagramImportSummaryMapper.ensureInitialized();
  @override
  $R call({
    String? importId,
    InstagramImportState? state,
    InstagramImportSourceType? sourceType,
    int? followingCount,
    DateTime? createdAt,
  }) => $apply(
    FieldCopyWithData({
      if (importId != null) #importId: importId,
      if (state != null) #state: state,
      if (sourceType != null) #sourceType: sourceType,
      if (followingCount != null) #followingCount: followingCount,
      if (createdAt != null) #createdAt: createdAt,
    }),
  );
  @override
  InstagramImportSummary $make(CopyWithData data) => InstagramImportSummary(
    importId: data.get(#importId, or: $value.importId),
    state: data.get(#state, or: $value.state),
    sourceType: data.get(#sourceType, or: $value.sourceType),
    followingCount: data.get(#followingCount, or: $value.followingCount),
    createdAt: data.get(#createdAt, or: $value.createdAt),
  );

  @override
  InstagramImportSummaryCopyWith<$R2, InstagramImportSummary, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _InstagramImportSummaryCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class InstagramImportCreateResultMapper
    extends ClassMapperBase<InstagramImportCreateResult> {
  InstagramImportCreateResultMapper._();

  static InstagramImportCreateResultMapper? _instance;
  static InstagramImportCreateResultMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = InstagramImportCreateResultMapper._(),
      );
      InstagramImportSummaryMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'InstagramImportCreateResult';

  static InstagramImportSummary _$import(InstagramImportCreateResult v) =>
      v.import;
  static const Field<InstagramImportCreateResult, InstagramImportSummary>
  _f$import = Field('import', _$import);
  static int _$followingCount(InstagramImportCreateResult v) =>
      v.followingCount;
  static const Field<InstagramImportCreateResult, int> _f$followingCount =
      Field('followingCount', _$followingCount);

  @override
  final MappableFields<InstagramImportCreateResult> fields = const {
    #import: _f$import,
    #followingCount: _f$followingCount,
  };

  static InstagramImportCreateResult _instantiate(DecodingData data) {
    return InstagramImportCreateResult(
      import: data.dec(_f$import),
      followingCount: data.dec(_f$followingCount),
    );
  }

  @override
  final Function instantiate = _instantiate;
}

mixin InstagramImportCreateResultMappable {
  InstagramImportCreateResultCopyWith<
    InstagramImportCreateResult,
    InstagramImportCreateResult,
    InstagramImportCreateResult
  >
  get copyWith =>
      _InstagramImportCreateResultCopyWithImpl<
        InstagramImportCreateResult,
        InstagramImportCreateResult
      >(this as InstagramImportCreateResult, $identity, $identity);
  @override
  bool operator ==(Object other) {
    return InstagramImportCreateResultMapper.ensureInitialized().equalsValue(
      this as InstagramImportCreateResult,
      other,
    );
  }

  @override
  int get hashCode {
    return InstagramImportCreateResultMapper.ensureInitialized().hashValue(
      this as InstagramImportCreateResult,
    );
  }
}

extension InstagramImportCreateResultValueCopy<$R, $Out>
    on ObjectCopyWith<$R, InstagramImportCreateResult, $Out> {
  InstagramImportCreateResultCopyWith<$R, InstagramImportCreateResult, $Out>
  get $asInstagramImportCreateResult => $base.as(
    (v, t, t2) => _InstagramImportCreateResultCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class InstagramImportCreateResultCopyWith<
  $R,
  $In extends InstagramImportCreateResult,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  InstagramImportSummaryCopyWith<
    $R,
    InstagramImportSummary,
    InstagramImportSummary
  >
  get import;
  $R call({InstagramImportSummary? import, int? followingCount});
  InstagramImportCreateResultCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _InstagramImportCreateResultCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, InstagramImportCreateResult, $Out>
    implements
        InstagramImportCreateResultCopyWith<
          $R,
          InstagramImportCreateResult,
          $Out
        > {
  _InstagramImportCreateResultCopyWithImpl(
    super.value,
    super.then,
    super.then2,
  );

  @override
  late final ClassMapperBase<InstagramImportCreateResult> $mapper =
      InstagramImportCreateResultMapper.ensureInitialized();
  @override
  InstagramImportSummaryCopyWith<
    $R,
    InstagramImportSummary,
    InstagramImportSummary
  >
  get import => $value.import.copyWith.$chain((v) => call(import: v));
  @override
  $R call({InstagramImportSummary? import, int? followingCount}) => $apply(
    FieldCopyWithData({
      if (import != null) #import: import,
      if (followingCount != null) #followingCount: followingCount,
    }),
  );
  @override
  InstagramImportCreateResult $make(CopyWithData data) =>
      InstagramImportCreateResult(
        import: data.get(#import, or: $value.import),
        followingCount: data.get(#followingCount, or: $value.followingCount),
      );

  @override
  InstagramImportCreateResultCopyWith<$R2, InstagramImportCreateResult, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _InstagramImportCreateResultCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class InstagramImportPageMapper extends ClassMapperBase<InstagramImportPage> {
  InstagramImportPageMapper._();

  static InstagramImportPageMapper? _instance;
  static InstagramImportPageMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = InstagramImportPageMapper._());
      InstagramImportSummaryMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'InstagramImportPage';

  static List<InstagramImportSummary> _$items(InstagramImportPage v) => v.items;
  static const Field<InstagramImportPage, List<InstagramImportSummary>>
  _f$items = Field('items', _$items);
  static String? _$cursor(InstagramImportPage v) => v.cursor;
  static const Field<InstagramImportPage, String> _f$cursor = Field(
    'cursor',
    _$cursor,
  );

  @override
  final MappableFields<InstagramImportPage> fields = const {
    #items: _f$items,
    #cursor: _f$cursor,
  };

  static InstagramImportPage _instantiate(DecodingData data) {
    return InstagramImportPage(
      items: data.dec(_f$items),
      cursor: data.dec(_f$cursor),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static InstagramImportPage fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<InstagramImportPage>(map);
  }

  static InstagramImportPage fromJson(String json) {
    return ensureInitialized().decodeJson<InstagramImportPage>(json);
  }
}

mixin InstagramImportPageMappable {
  InstagramImportPageCopyWith<
    InstagramImportPage,
    InstagramImportPage,
    InstagramImportPage
  >
  get copyWith =>
      _InstagramImportPageCopyWithImpl<
        InstagramImportPage,
        InstagramImportPage
      >(this as InstagramImportPage, $identity, $identity);
  @override
  bool operator ==(Object other) {
    return InstagramImportPageMapper.ensureInitialized().equalsValue(
      this as InstagramImportPage,
      other,
    );
  }

  @override
  int get hashCode {
    return InstagramImportPageMapper.ensureInitialized().hashValue(
      this as InstagramImportPage,
    );
  }
}

extension InstagramImportPageValueCopy<$R, $Out>
    on ObjectCopyWith<$R, InstagramImportPage, $Out> {
  InstagramImportPageCopyWith<$R, InstagramImportPage, $Out>
  get $asInstagramImportPage => $base.as(
    (v, t, t2) => _InstagramImportPageCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class InstagramImportPageCopyWith<
  $R,
  $In extends InstagramImportPage,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<
    $R,
    InstagramImportSummary,
    InstagramImportSummaryCopyWith<
      $R,
      InstagramImportSummary,
      InstagramImportSummary
    >
  >
  get items;
  $R call({List<InstagramImportSummary>? items, String? cursor});
  InstagramImportPageCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _InstagramImportPageCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, InstagramImportPage, $Out>
    implements InstagramImportPageCopyWith<$R, InstagramImportPage, $Out> {
  _InstagramImportPageCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<InstagramImportPage> $mapper =
      InstagramImportPageMapper.ensureInitialized();
  @override
  ListCopyWith<
    $R,
    InstagramImportSummary,
    InstagramImportSummaryCopyWith<
      $R,
      InstagramImportSummary,
      InstagramImportSummary
    >
  >
  get items => ListCopyWith(
    $value.items,
    (v, t) => v.copyWith.$chain(t),
    (v) => call(items: v),
  );
  @override
  $R call({List<InstagramImportSummary>? items, Object? cursor = $none}) =>
      $apply(
        FieldCopyWithData({
          if (items != null) #items: items,
          if (cursor != $none) #cursor: cursor,
        }),
      );
  @override
  InstagramImportPage $make(CopyWithData data) => InstagramImportPage(
    items: data.get(#items, or: $value.items),
    cursor: data.get(#cursor, or: $value.cursor),
  );

  @override
  InstagramImportPageCopyWith<$R2, InstagramImportPage, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _InstagramImportPageCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class InstagramImportPatchMapper extends ClassMapperBase<InstagramImportPatch> {
  InstagramImportPatchMapper._();

  static InstagramImportPatchMapper? _instance;
  static InstagramImportPatchMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = InstagramImportPatchMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'InstagramImportPatch';

  static bool _$reactivate(InstagramImportPatch v) => v.reactivate;
  static const Field<InstagramImportPatch, bool> _f$reactivate = Field(
    'reactivate',
    _$reactivate,
  );

  @override
  final MappableFields<InstagramImportPatch> fields = const {
    #reactivate: _f$reactivate,
  };

  static InstagramImportPatch _instantiate(DecodingData data) {
    return InstagramImportPatch(reactivate: data.dec(_f$reactivate));
  }

  @override
  final Function instantiate = _instantiate;
}

mixin InstagramImportPatchMappable {
  String toJson() {
    return InstagramImportPatchMapper.ensureInitialized()
        .encodeJson<InstagramImportPatch>(this as InstagramImportPatch);
  }

  Map<String, dynamic> toMap() {
    return InstagramImportPatchMapper.ensureInitialized()
        .encodeMap<InstagramImportPatch>(this as InstagramImportPatch);
  }

  InstagramImportPatchCopyWith<
    InstagramImportPatch,
    InstagramImportPatch,
    InstagramImportPatch
  >
  get copyWith =>
      _InstagramImportPatchCopyWithImpl<
        InstagramImportPatch,
        InstagramImportPatch
      >(this as InstagramImportPatch, $identity, $identity);
  @override
  bool operator ==(Object other) {
    return InstagramImportPatchMapper.ensureInitialized().equalsValue(
      this as InstagramImportPatch,
      other,
    );
  }

  @override
  int get hashCode {
    return InstagramImportPatchMapper.ensureInitialized().hashValue(
      this as InstagramImportPatch,
    );
  }
}

extension InstagramImportPatchValueCopy<$R, $Out>
    on ObjectCopyWith<$R, InstagramImportPatch, $Out> {
  InstagramImportPatchCopyWith<$R, InstagramImportPatch, $Out>
  get $asInstagramImportPatch => $base.as(
    (v, t, t2) => _InstagramImportPatchCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class InstagramImportPatchCopyWith<
  $R,
  $In extends InstagramImportPatch,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({bool? reactivate});
  InstagramImportPatchCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _InstagramImportPatchCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, InstagramImportPatch, $Out>
    implements InstagramImportPatchCopyWith<$R, InstagramImportPatch, $Out> {
  _InstagramImportPatchCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<InstagramImportPatch> $mapper =
      InstagramImportPatchMapper.ensureInitialized();
  @override
  $R call({bool? reactivate}) => $apply(
    FieldCopyWithData({if (reactivate != null) #reactivate: reactivate}),
  );
  @override
  InstagramImportPatch $make(CopyWithData data) => InstagramImportPatch(
    reactivate: data.get(#reactivate, or: $value.reactivate),
  );

  @override
  InstagramImportPatchCopyWith<$R2, InstagramImportPatch, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _InstagramImportPatchCopyWithImpl<$R2, $Out2>($value, $cast, t);
}
