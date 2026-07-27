// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'saved_post_keys.dart';

class SavedPostKeyMapper extends ClassMapperBase<SavedPostKey> {
  SavedPostKeyMapper._();

  static SavedPostKeyMapper? _instance;
  static SavedPostKeyMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = SavedPostKeyMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'SavedPostKey';

  static AccountKey _$account(SavedPostKey v) => v.account;
  static const Field<SavedPostKey, AccountKey> _f$account = Field(
    'account',
    _$account,
  );
  static AtUri _$uri(SavedPostKey v) => v.uri;
  static const Field<SavedPostKey, AtUri> _f$uri = Field('uri', _$uri);

  @override
  final MappableFields<SavedPostKey> fields = const {
    #account: _f$account,
    #uri: _f$uri,
  };

  static SavedPostKey _instantiate(DecodingData data) {
    return SavedPostKey(account: data.dec(_f$account), uri: data.dec(_f$uri));
  }

  @override
  final Function instantiate = _instantiate;
}

mixin SavedPostKeyMappable {
  SavedPostKeyCopyWith<SavedPostKey, SavedPostKey, SavedPostKey> get copyWith =>
      _SavedPostKeyCopyWithImpl<SavedPostKey, SavedPostKey>(
        this as SavedPostKey,
        $identity,
        $identity,
      );
  @override
  bool operator ==(Object other) {
    return SavedPostKeyMapper.ensureInitialized().equalsValue(
      this as SavedPostKey,
      other,
    );
  }

  @override
  int get hashCode {
    return SavedPostKeyMapper.ensureInitialized().hashValue(
      this as SavedPostKey,
    );
  }
}

extension SavedPostKeyValueCopy<$R, $Out>
    on ObjectCopyWith<$R, SavedPostKey, $Out> {
  SavedPostKeyCopyWith<$R, SavedPostKey, $Out> get $asSavedPostKey =>
      $base.as((v, t, t2) => _SavedPostKeyCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class SavedPostKeyCopyWith<$R, $In extends SavedPostKey, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({AccountKey? account, AtUri? uri});
  SavedPostKeyCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _SavedPostKeyCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, SavedPostKey, $Out>
    implements SavedPostKeyCopyWith<$R, SavedPostKey, $Out> {
  _SavedPostKeyCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<SavedPostKey> $mapper =
      SavedPostKeyMapper.ensureInitialized();
  @override
  $R call({AccountKey? account, AtUri? uri}) => $apply(
    FieldCopyWithData({
      if (account != null) #account: account,
      if (uri != null) #uri: uri,
    }),
  );
  @override
  SavedPostKey $make(CopyWithData data) => SavedPostKey(
    account: data.get(#account, or: $value.account),
    uri: data.get(#uri, or: $value.uri),
  );

  @override
  SavedPostKeyCopyWith<$R2, SavedPostKey, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _SavedPostKeyCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class SavePostDialogKeyMapper extends ClassMapperBase<SavePostDialogKey> {
  SavePostDialogKeyMapper._();

  static SavePostDialogKeyMapper? _instance;
  static SavePostDialogKeyMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = SavePostDialogKeyMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'SavePostDialogKey';

  static AccountKey _$account(SavePostDialogKey v) => v.account;
  static const Field<SavePostDialogKey, AccountKey> _f$account = Field(
    'account',
    _$account,
  );
  static AtUri _$uri(SavePostDialogKey v) => v.uri;
  static const Field<SavePostDialogKey, AtUri> _f$uri = Field('uri', _$uri);
  static String? _$initialFolderId(SavePostDialogKey v) => v.initialFolderId;
  static const Field<SavePostDialogKey, String> _f$initialFolderId = Field(
    'initialFolderId',
    _$initialFolderId,
    opt: true,
  );

  @override
  final MappableFields<SavePostDialogKey> fields = const {
    #account: _f$account,
    #uri: _f$uri,
    #initialFolderId: _f$initialFolderId,
  };

  static SavePostDialogKey _instantiate(DecodingData data) {
    return SavePostDialogKey(
      account: data.dec(_f$account),
      uri: data.dec(_f$uri),
      initialFolderId: data.dec(_f$initialFolderId),
    );
  }

  @override
  final Function instantiate = _instantiate;
}

mixin SavePostDialogKeyMappable {
  SavePostDialogKeyCopyWith<
    SavePostDialogKey,
    SavePostDialogKey,
    SavePostDialogKey
  >
  get copyWith =>
      _SavePostDialogKeyCopyWithImpl<SavePostDialogKey, SavePostDialogKey>(
        this as SavePostDialogKey,
        $identity,
        $identity,
      );
  @override
  bool operator ==(Object other) {
    return SavePostDialogKeyMapper.ensureInitialized().equalsValue(
      this as SavePostDialogKey,
      other,
    );
  }

  @override
  int get hashCode {
    return SavePostDialogKeyMapper.ensureInitialized().hashValue(
      this as SavePostDialogKey,
    );
  }
}

extension SavePostDialogKeyValueCopy<$R, $Out>
    on ObjectCopyWith<$R, SavePostDialogKey, $Out> {
  SavePostDialogKeyCopyWith<$R, SavePostDialogKey, $Out>
  get $asSavePostDialogKey => $base.as(
    (v, t, t2) => _SavePostDialogKeyCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class SavePostDialogKeyCopyWith<
  $R,
  $In extends SavePostDialogKey,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({AccountKey? account, AtUri? uri, String? initialFolderId});
  SavePostDialogKeyCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _SavePostDialogKeyCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, SavePostDialogKey, $Out>
    implements SavePostDialogKeyCopyWith<$R, SavePostDialogKey, $Out> {
  _SavePostDialogKeyCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<SavePostDialogKey> $mapper =
      SavePostDialogKeyMapper.ensureInitialized();
  @override
  $R call({AccountKey? account, AtUri? uri, Object? initialFolderId = $none}) =>
      $apply(
        FieldCopyWithData({
          if (account != null) #account: account,
          if (uri != null) #uri: uri,
          if (initialFolderId != $none) #initialFolderId: initialFolderId,
        }),
      );
  @override
  SavePostDialogKey $make(CopyWithData data) => SavePostDialogKey(
    account: data.get(#account, or: $value.account),
    uri: data.get(#uri, or: $value.uri),
    initialFolderId: data.get(#initialFolderId, or: $value.initialFolderId),
  );

  @override
  SavePostDialogKeyCopyWith<$R2, SavePostDialogKey, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _SavePostDialogKeyCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class SavedPostListKeyMapper extends ClassMapperBase<SavedPostListKey> {
  SavedPostListKeyMapper._();

  static SavedPostListKeyMapper? _instance;
  static SavedPostListKeyMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = SavedPostListKeyMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'SavedPostListKey';

  static AccountKey _$account(SavedPostListKey v) => v.account;
  static const Field<SavedPostListKey, AccountKey> _f$account = Field(
    'account',
    _$account,
  );
  static SavedPostScope _$scope(SavedPostListKey v) => v.scope;
  static const Field<SavedPostListKey, SavedPostScope> _f$scope = Field(
    'scope',
    _$scope,
  );
  static SavedPostSort _$sort(SavedPostListKey v) => v.sort;
  static const Field<SavedPostListKey, SavedPostSort> _f$sort = Field(
    'sort',
    _$sort,
  );

  @override
  final MappableFields<SavedPostListKey> fields = const {
    #account: _f$account,
    #scope: _f$scope,
    #sort: _f$sort,
  };

  static SavedPostListKey _instantiate(DecodingData data) {
    return SavedPostListKey(
      account: data.dec(_f$account),
      scope: data.dec(_f$scope),
      sort: data.dec(_f$sort),
    );
  }

  @override
  final Function instantiate = _instantiate;
}

mixin SavedPostListKeyMappable {
  SavedPostListKeyCopyWith<SavedPostListKey, SavedPostListKey, SavedPostListKey>
  get copyWith =>
      _SavedPostListKeyCopyWithImpl<SavedPostListKey, SavedPostListKey>(
        this as SavedPostListKey,
        $identity,
        $identity,
      );
  @override
  bool operator ==(Object other) {
    return SavedPostListKeyMapper.ensureInitialized().equalsValue(
      this as SavedPostListKey,
      other,
    );
  }

  @override
  int get hashCode {
    return SavedPostListKeyMapper.ensureInitialized().hashValue(
      this as SavedPostListKey,
    );
  }
}

extension SavedPostListKeyValueCopy<$R, $Out>
    on ObjectCopyWith<$R, SavedPostListKey, $Out> {
  SavedPostListKeyCopyWith<$R, SavedPostListKey, $Out>
  get $asSavedPostListKey =>
      $base.as((v, t, t2) => _SavedPostListKeyCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class SavedPostListKeyCopyWith<$R, $In extends SavedPostListKey, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({AccountKey? account, SavedPostScope? scope, SavedPostSort? sort});
  SavedPostListKeyCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _SavedPostListKeyCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, SavedPostListKey, $Out>
    implements SavedPostListKeyCopyWith<$R, SavedPostListKey, $Out> {
  _SavedPostListKeyCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<SavedPostListKey> $mapper =
      SavedPostListKeyMapper.ensureInitialized();
  @override
  $R call({AccountKey? account, SavedPostScope? scope, SavedPostSort? sort}) =>
      $apply(
        FieldCopyWithData({
          if (account != null) #account: account,
          if (scope != null) #scope: scope,
          if (sort != null) #sort: sort,
        }),
      );
  @override
  SavedPostListKey $make(CopyWithData data) => SavedPostListKey(
    account: data.get(#account, or: $value.account),
    scope: data.get(#scope, or: $value.scope),
    sort: data.get(#sort, or: $value.sort),
  );

  @override
  SavedPostListKeyCopyWith<$R2, SavedPostListKey, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _SavedPostListKeyCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

