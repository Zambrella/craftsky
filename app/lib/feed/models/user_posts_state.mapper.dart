// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'user_posts_state.dart';

class UserPostsStateMapper extends ClassMapperBase<UserPostsState> {
  UserPostsStateMapper._();

  static UserPostsStateMapper? _instance;
  static UserPostsStateMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = UserPostsStateMapper._());
      PostMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'UserPostsState';

  static List<Post> _$items(UserPostsState v) => v.items;
  static const Field<UserPostsState, List<Post>> _f$items = Field(
    'items',
    _$items,
  );
  static String? _$cursor(UserPostsState v) => v.cursor;
  static const Field<UserPostsState, String> _f$cursor = Field(
    'cursor',
    _$cursor,
    opt: true,
  );

  @override
  final MappableFields<UserPostsState> fields = const {
    #items: _f$items,
    #cursor: _f$cursor,
  };

  static UserPostsState _instantiate(DecodingData data) {
    return UserPostsState(
      items: data.dec(_f$items),
      cursor: data.dec(_f$cursor),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static UserPostsState fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<UserPostsState>(map);
  }

  static UserPostsState fromJson(String json) {
    return ensureInitialized().decodeJson<UserPostsState>(json);
  }
}

mixin UserPostsStateMappable {
  String toJson() {
    return UserPostsStateMapper.ensureInitialized().encodeJson<UserPostsState>(
      this as UserPostsState,
    );
  }

  Map<String, dynamic> toMap() {
    return UserPostsStateMapper.ensureInitialized().encodeMap<UserPostsState>(
      this as UserPostsState,
    );
  }

  UserPostsStateCopyWith<UserPostsState, UserPostsState, UserPostsState>
  get copyWith => _UserPostsStateCopyWithImpl<UserPostsState, UserPostsState>(
    this as UserPostsState,
    $identity,
    $identity,
  );
  @override
  String toString() {
    return UserPostsStateMapper.ensureInitialized().stringifyValue(
      this as UserPostsState,
    );
  }

  @override
  bool operator ==(Object other) {
    return UserPostsStateMapper.ensureInitialized().equalsValue(
      this as UserPostsState,
      other,
    );
  }

  @override
  int get hashCode {
    return UserPostsStateMapper.ensureInitialized().hashValue(
      this as UserPostsState,
    );
  }
}

extension UserPostsStateValueCopy<$R, $Out>
    on ObjectCopyWith<$R, UserPostsState, $Out> {
  UserPostsStateCopyWith<$R, UserPostsState, $Out> get $asUserPostsState =>
      $base.as((v, t, t2) => _UserPostsStateCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class UserPostsStateCopyWith<$R, $In extends UserPostsState, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<$R, Post, PostCopyWith<$R, Post, Post>> get items;
  $R call({List<Post>? items, String? cursor});
  UserPostsStateCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _UserPostsStateCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, UserPostsState, $Out>
    implements UserPostsStateCopyWith<$R, UserPostsState, $Out> {
  _UserPostsStateCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<UserPostsState> $mapper =
      UserPostsStateMapper.ensureInitialized();
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
  UserPostsState $make(CopyWithData data) => UserPostsState(
    items: data.get(#items, or: $value.items),
    cursor: data.get(#cursor, or: $value.cursor),
  );

  @override
  UserPostsStateCopyWith<$R2, UserPostsState, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _UserPostsStateCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

