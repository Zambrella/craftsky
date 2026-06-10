// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'user_projects_state.dart';

class UserProjectsStateMapper extends ClassMapperBase<UserProjectsState> {
  UserProjectsStateMapper._();

  static UserProjectsStateMapper? _instance;
  static UserProjectsStateMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = UserProjectsStateMapper._());
      PostMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'UserProjectsState';

  static List<Post> _$items(UserProjectsState v) => v.items;
  static const Field<UserProjectsState, List<Post>> _f$items = Field(
    'items',
    _$items,
  );
  static String? _$cursor(UserProjectsState v) => v.cursor;
  static const Field<UserProjectsState, String> _f$cursor = Field(
    'cursor',
    _$cursor,
    opt: true,
  );

  @override
  final MappableFields<UserProjectsState> fields = const {
    #items: _f$items,
    #cursor: _f$cursor,
  };

  static UserProjectsState _instantiate(DecodingData data) {
    return UserProjectsState(
      items: data.dec(_f$items),
      cursor: data.dec(_f$cursor),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static UserProjectsState fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<UserProjectsState>(map);
  }

  static UserProjectsState fromJson(String json) {
    return ensureInitialized().decodeJson<UserProjectsState>(json);
  }
}

mixin UserProjectsStateMappable {
  String toJson() {
    return UserProjectsStateMapper.ensureInitialized()
        .encodeJson<UserProjectsState>(this as UserProjectsState);
  }

  Map<String, dynamic> toMap() {
    return UserProjectsStateMapper.ensureInitialized()
        .encodeMap<UserProjectsState>(this as UserProjectsState);
  }

  UserProjectsStateCopyWith<
    UserProjectsState,
    UserProjectsState,
    UserProjectsState
  >
  get copyWith =>
      _UserProjectsStateCopyWithImpl<UserProjectsState, UserProjectsState>(
        this as UserProjectsState,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return UserProjectsStateMapper.ensureInitialized().stringifyValue(
      this as UserProjectsState,
    );
  }

  @override
  bool operator ==(Object other) {
    return UserProjectsStateMapper.ensureInitialized().equalsValue(
      this as UserProjectsState,
      other,
    );
  }

  @override
  int get hashCode {
    return UserProjectsStateMapper.ensureInitialized().hashValue(
      this as UserProjectsState,
    );
  }
}

extension UserProjectsStateValueCopy<$R, $Out>
    on ObjectCopyWith<$R, UserProjectsState, $Out> {
  UserProjectsStateCopyWith<$R, UserProjectsState, $Out>
  get $asUserProjectsState => $base.as(
    (v, t, t2) => _UserProjectsStateCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class UserProjectsStateCopyWith<
  $R,
  $In extends UserProjectsState,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<$R, Post, PostCopyWith<$R, Post, Post>> get items;
  $R call({List<Post>? items, String? cursor});
  UserProjectsStateCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _UserProjectsStateCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, UserProjectsState, $Out>
    implements UserProjectsStateCopyWith<$R, UserProjectsState, $Out> {
  _UserProjectsStateCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<UserProjectsState> $mapper =
      UserProjectsStateMapper.ensureInitialized();
  @override
  ListCopyWith<$R, Post, PostCopyWith<$R, Post, Post>> get items =>
      ListCopyWith(
        $value.items,
        (v, t) => v.copyWith.$chain(t),
        (v) => call(items: v),
      );
  @override
  $R call({List<Post>? items, Object? cursor = $none}) => $apply(
    FieldCopyWithData({
      if (items != null) #items: items,
      if (cursor != $none) #cursor: cursor,
    }),
  );
  @override
  UserProjectsState $make(CopyWithData data) => UserProjectsState(
    items: data.get(#items, or: $value.items),
    cursor: data.get(#cursor, or: $value.cursor),
  );

  @override
  UserProjectsStateCopyWith<$R2, UserProjectsState, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _UserProjectsStateCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

