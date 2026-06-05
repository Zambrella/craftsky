// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'facet_suggestion_repository.dart';

class AccountSuggestionMapper extends ClassMapperBase<AccountSuggestion> {
  AccountSuggestionMapper._();

  static AccountSuggestionMapper? _instance;
  static AccountSuggestionMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = AccountSuggestionMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'AccountSuggestion';

  static String _$did(AccountSuggestion v) => v.did;
  static const Field<AccountSuggestion, String> _f$did = Field('did', _$did);
  static String _$handle(AccountSuggestion v) => v.handle;
  static const Field<AccountSuggestion, String> _f$handle = Field(
    'handle',
    _$handle,
  );
  static String? _$displayName(AccountSuggestion v) => v.displayName;
  static const Field<AccountSuggestion, String> _f$displayName = Field(
    'displayName',
    _$displayName,
  );
  static String? _$avatar(AccountSuggestion v) => v.avatar;
  static const Field<AccountSuggestion, String> _f$avatar = Field(
    'avatar',
    _$avatar,
  );
  static bool _$isCraftskyProfile(AccountSuggestion v) => v.isCraftskyProfile;
  static const Field<AccountSuggestion, bool> _f$isCraftskyProfile = Field(
    'isCraftskyProfile',
    _$isCraftskyProfile,
    opt: true,
    def: false,
  );
  static bool _$viewerIsFollowing(AccountSuggestion v) => v.viewerIsFollowing;
  static const Field<AccountSuggestion, bool> _f$viewerIsFollowing = Field(
    'viewerIsFollowing',
    _$viewerIsFollowing,
    opt: true,
    def: false,
  );

  @override
  final MappableFields<AccountSuggestion> fields = const {
    #did: _f$did,
    #handle: _f$handle,
    #displayName: _f$displayName,
    #avatar: _f$avatar,
    #isCraftskyProfile: _f$isCraftskyProfile,
    #viewerIsFollowing: _f$viewerIsFollowing,
  };

  static AccountSuggestion _instantiate(DecodingData data) {
    return AccountSuggestion(
      did: data.dec(_f$did),
      handle: data.dec(_f$handle),
      displayName: data.dec(_f$displayName),
      avatar: data.dec(_f$avatar),
      isCraftskyProfile: data.dec(_f$isCraftskyProfile),
      viewerIsFollowing: data.dec(_f$viewerIsFollowing),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static AccountSuggestion fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<AccountSuggestion>(map);
  }

  static AccountSuggestion fromJson(String json) {
    return ensureInitialized().decodeJson<AccountSuggestion>(json);
  }
}

mixin AccountSuggestionMappable {
  String toJson() {
    return AccountSuggestionMapper.ensureInitialized()
        .encodeJson<AccountSuggestion>(this as AccountSuggestion);
  }

  Map<String, dynamic> toMap() {
    return AccountSuggestionMapper.ensureInitialized()
        .encodeMap<AccountSuggestion>(this as AccountSuggestion);
  }

  AccountSuggestionCopyWith<
    AccountSuggestion,
    AccountSuggestion,
    AccountSuggestion
  >
  get copyWith =>
      _AccountSuggestionCopyWithImpl<AccountSuggestion, AccountSuggestion>(
        this as AccountSuggestion,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return AccountSuggestionMapper.ensureInitialized().stringifyValue(
      this as AccountSuggestion,
    );
  }

  @override
  bool operator ==(Object other) {
    return AccountSuggestionMapper.ensureInitialized().equalsValue(
      this as AccountSuggestion,
      other,
    );
  }

  @override
  int get hashCode {
    return AccountSuggestionMapper.ensureInitialized().hashValue(
      this as AccountSuggestion,
    );
  }
}

extension AccountSuggestionValueCopy<$R, $Out>
    on ObjectCopyWith<$R, AccountSuggestion, $Out> {
  AccountSuggestionCopyWith<$R, AccountSuggestion, $Out>
  get $asAccountSuggestion => $base.as(
    (v, t, t2) => _AccountSuggestionCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class AccountSuggestionCopyWith<
  $R,
  $In extends AccountSuggestion,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({
    String? did,
    String? handle,
    String? displayName,
    String? avatar,
    bool? isCraftskyProfile,
    bool? viewerIsFollowing,
  });
  AccountSuggestionCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _AccountSuggestionCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, AccountSuggestion, $Out>
    implements AccountSuggestionCopyWith<$R, AccountSuggestion, $Out> {
  _AccountSuggestionCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<AccountSuggestion> $mapper =
      AccountSuggestionMapper.ensureInitialized();
  @override
  $R call({
    String? did,
    String? handle,
    Object? displayName = $none,
    Object? avatar = $none,
    bool? isCraftskyProfile,
    bool? viewerIsFollowing,
  }) => $apply(
    FieldCopyWithData({
      if (did != null) #did: did,
      if (handle != null) #handle: handle,
      if (displayName != $none) #displayName: displayName,
      if (avatar != $none) #avatar: avatar,
      if (isCraftskyProfile != null) #isCraftskyProfile: isCraftskyProfile,
      if (viewerIsFollowing != null) #viewerIsFollowing: viewerIsFollowing,
    }),
  );
  @override
  AccountSuggestion $make(CopyWithData data) => AccountSuggestion(
    did: data.get(#did, or: $value.did),
    handle: data.get(#handle, or: $value.handle),
    displayName: data.get(#displayName, or: $value.displayName),
    avatar: data.get(#avatar, or: $value.avatar),
    isCraftskyProfile: data.get(
      #isCraftskyProfile,
      or: $value.isCraftskyProfile,
    ),
    viewerIsFollowing: data.get(
      #viewerIsFollowing,
      or: $value.viewerIsFollowing,
    ),
  );

  @override
  AccountSuggestionCopyWith<$R2, AccountSuggestion, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _AccountSuggestionCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class HashtagSuggestionMapper extends ClassMapperBase<HashtagSuggestion> {
  HashtagSuggestionMapper._();

  static HashtagSuggestionMapper? _instance;
  static HashtagSuggestionMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = HashtagSuggestionMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'HashtagSuggestion';

  static String _$tag(HashtagSuggestion v) => v.tag;
  static const Field<HashtagSuggestion, String> _f$tag = Field('tag', _$tag);
  static int _$postsLast28Days(HashtagSuggestion v) => v.postsLast28Days;
  static const Field<HashtagSuggestion, int> _f$postsLast28Days = Field(
    'postsLast28Days',
    _$postsLast28Days,
    opt: true,
    def: 0,
  );

  @override
  final MappableFields<HashtagSuggestion> fields = const {
    #tag: _f$tag,
    #postsLast28Days: _f$postsLast28Days,
  };

  static HashtagSuggestion _instantiate(DecodingData data) {
    return HashtagSuggestion(
      tag: data.dec(_f$tag),
      postsLast28Days: data.dec(_f$postsLast28Days),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static HashtagSuggestion fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<HashtagSuggestion>(map);
  }

  static HashtagSuggestion fromJson(String json) {
    return ensureInitialized().decodeJson<HashtagSuggestion>(json);
  }
}

mixin HashtagSuggestionMappable {
  String toJson() {
    return HashtagSuggestionMapper.ensureInitialized()
        .encodeJson<HashtagSuggestion>(this as HashtagSuggestion);
  }

  Map<String, dynamic> toMap() {
    return HashtagSuggestionMapper.ensureInitialized()
        .encodeMap<HashtagSuggestion>(this as HashtagSuggestion);
  }

  HashtagSuggestionCopyWith<
    HashtagSuggestion,
    HashtagSuggestion,
    HashtagSuggestion
  >
  get copyWith =>
      _HashtagSuggestionCopyWithImpl<HashtagSuggestion, HashtagSuggestion>(
        this as HashtagSuggestion,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return HashtagSuggestionMapper.ensureInitialized().stringifyValue(
      this as HashtagSuggestion,
    );
  }

  @override
  bool operator ==(Object other) {
    return HashtagSuggestionMapper.ensureInitialized().equalsValue(
      this as HashtagSuggestion,
      other,
    );
  }

  @override
  int get hashCode {
    return HashtagSuggestionMapper.ensureInitialized().hashValue(
      this as HashtagSuggestion,
    );
  }
}

extension HashtagSuggestionValueCopy<$R, $Out>
    on ObjectCopyWith<$R, HashtagSuggestion, $Out> {
  HashtagSuggestionCopyWith<$R, HashtagSuggestion, $Out>
  get $asHashtagSuggestion => $base.as(
    (v, t, t2) => _HashtagSuggestionCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class HashtagSuggestionCopyWith<
  $R,
  $In extends HashtagSuggestion,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? tag, int? postsLast28Days});
  HashtagSuggestionCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _HashtagSuggestionCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, HashtagSuggestion, $Out>
    implements HashtagSuggestionCopyWith<$R, HashtagSuggestion, $Out> {
  _HashtagSuggestionCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<HashtagSuggestion> $mapper =
      HashtagSuggestionMapper.ensureInitialized();
  @override
  $R call({String? tag, int? postsLast28Days}) => $apply(
    FieldCopyWithData({
      if (tag != null) #tag: tag,
      if (postsLast28Days != null) #postsLast28Days: postsLast28Days,
    }),
  );
  @override
  HashtagSuggestion $make(CopyWithData data) => HashtagSuggestion(
    tag: data.get(#tag, or: $value.tag),
    postsLast28Days: data.get(#postsLast28Days, or: $value.postsLast28Days),
  );

  @override
  HashtagSuggestionCopyWith<$R2, HashtagSuggestion, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _HashtagSuggestionCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

