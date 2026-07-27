// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'notification_destination.dart';

class NotificationDestinationMapper
    extends ClassMapperBase<NotificationDestination> {
  NotificationDestinationMapper._();

  static NotificationDestinationMapper? _instance;
  static NotificationDestinationMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = NotificationDestinationMapper._(),
      );
      MapperContainer.globals.useAll([DidMapper(), AtUriMapper()]);
      NotificationsDestinationMapper.ensureInitialized();
      InstagramMigrationDestinationMapper.ensureInitialized();
      ProfileDestinationMapper.ensureInitialized();
      PostDestinationMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'NotificationDestination';

  @override
  final MappableFields<NotificationDestination> fields = const {};

  static NotificationDestination _instantiate(DecodingData data) {
    throw MapperException.missingSubclass(
      'NotificationDestination',
      'type',
      '${data.value['type']}',
    );
  }

  @override
  final Function instantiate = _instantiate;

  static NotificationDestination fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<NotificationDestination>(map);
  }

  static NotificationDestination fromJson(String json) {
    return ensureInitialized().decodeJson<NotificationDestination>(json);
  }
}

mixin NotificationDestinationMappable {
  String toJson();
  Map<String, dynamic> toMap();
  NotificationDestinationCopyWith<
    NotificationDestination,
    NotificationDestination,
    NotificationDestination
  >
  get copyWith;
}

abstract class NotificationDestinationCopyWith<
  $R,
  $In extends NotificationDestination,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call();
  NotificationDestinationCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class NotificationsDestinationMapper
    extends SubClassMapperBase<NotificationsDestination> {
  NotificationsDestinationMapper._();

  static NotificationsDestinationMapper? _instance;
  static NotificationsDestinationMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = NotificationsDestinationMapper._(),
      );
      NotificationDestinationMapper.ensureInitialized().addSubMapper(
        _instance!,
      );
    }
    return _instance!;
  }

  @override
  final String id = 'NotificationsDestination';

  @override
  final MappableFields<NotificationsDestination> fields = const {};

  @override
  final String discriminatorKey = 'type';
  @override
  final dynamic discriminatorValue = 'NotificationsDestination';
  @override
  late final ClassMapperBase superMapper =
      NotificationDestinationMapper.ensureInitialized();

  static NotificationsDestination _instantiate(DecodingData data) {
    return NotificationsDestination();
  }

  @override
  final Function instantiate = _instantiate;

  static NotificationsDestination fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<NotificationsDestination>(map);
  }

  static NotificationsDestination fromJson(String json) {
    return ensureInitialized().decodeJson<NotificationsDestination>(json);
  }
}

mixin NotificationsDestinationMappable {
  String toJson() {
    return NotificationsDestinationMapper.ensureInitialized()
        .encodeJson<NotificationsDestination>(this as NotificationsDestination);
  }

  Map<String, dynamic> toMap() {
    return NotificationsDestinationMapper.ensureInitialized()
        .encodeMap<NotificationsDestination>(this as NotificationsDestination);
  }

  NotificationsDestinationCopyWith<
    NotificationsDestination,
    NotificationsDestination,
    NotificationsDestination
  >
  get copyWith =>
      _NotificationsDestinationCopyWithImpl<
        NotificationsDestination,
        NotificationsDestination
      >(this as NotificationsDestination, $identity, $identity);
  @override
  bool operator ==(Object other) {
    return NotificationsDestinationMapper.ensureInitialized().equalsValue(
      this as NotificationsDestination,
      other,
    );
  }

  @override
  int get hashCode {
    return NotificationsDestinationMapper.ensureInitialized().hashValue(
      this as NotificationsDestination,
    );
  }
}

extension NotificationsDestinationValueCopy<$R, $Out>
    on ObjectCopyWith<$R, NotificationsDestination, $Out> {
  NotificationsDestinationCopyWith<$R, NotificationsDestination, $Out>
  get $asNotificationsDestination => $base.as(
    (v, t, t2) => _NotificationsDestinationCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class NotificationsDestinationCopyWith<
  $R,
  $In extends NotificationsDestination,
  $Out
>
    implements NotificationDestinationCopyWith<$R, $In, $Out> {
  @override
  $R call();
  NotificationsDestinationCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _NotificationsDestinationCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, NotificationsDestination, $Out>
    implements
        NotificationsDestinationCopyWith<$R, NotificationsDestination, $Out> {
  _NotificationsDestinationCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<NotificationsDestination> $mapper =
      NotificationsDestinationMapper.ensureInitialized();
  @override
  $R call() => $apply(FieldCopyWithData({}));
  @override
  NotificationsDestination $make(CopyWithData data) =>
      NotificationsDestination();

  @override
  NotificationsDestinationCopyWith<$R2, NotificationsDestination, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _NotificationsDestinationCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class InstagramMigrationDestinationMapper
    extends SubClassMapperBase<InstagramMigrationDestination> {
  InstagramMigrationDestinationMapper._();

  static InstagramMigrationDestinationMapper? _instance;
  static InstagramMigrationDestinationMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = InstagramMigrationDestinationMapper._(),
      );
      NotificationDestinationMapper.ensureInitialized().addSubMapper(
        _instance!,
      );
    }
    return _instance!;
  }

  @override
  final String id = 'InstagramMigrationDestination';

  @override
  final MappableFields<InstagramMigrationDestination> fields = const {};

  @override
  final String discriminatorKey = 'type';
  @override
  final dynamic discriminatorValue = 'InstagramMigrationDestination';
  @override
  late final ClassMapperBase superMapper =
      NotificationDestinationMapper.ensureInitialized();

  static InstagramMigrationDestination _instantiate(DecodingData data) {
    return InstagramMigrationDestination();
  }

  @override
  final Function instantiate = _instantiate;

  static InstagramMigrationDestination fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<InstagramMigrationDestination>(map);
  }

  static InstagramMigrationDestination fromJson(String json) {
    return ensureInitialized().decodeJson<InstagramMigrationDestination>(json);
  }
}

mixin InstagramMigrationDestinationMappable {
  String toJson() {
    return InstagramMigrationDestinationMapper.ensureInitialized()
        .encodeJson<InstagramMigrationDestination>(
          this as InstagramMigrationDestination,
        );
  }

  Map<String, dynamic> toMap() {
    return InstagramMigrationDestinationMapper.ensureInitialized()
        .encodeMap<InstagramMigrationDestination>(
          this as InstagramMigrationDestination,
        );
  }

  InstagramMigrationDestinationCopyWith<
    InstagramMigrationDestination,
    InstagramMigrationDestination,
    InstagramMigrationDestination
  >
  get copyWith =>
      _InstagramMigrationDestinationCopyWithImpl<
        InstagramMigrationDestination,
        InstagramMigrationDestination
      >(this as InstagramMigrationDestination, $identity, $identity);
  @override
  bool operator ==(Object other) {
    return InstagramMigrationDestinationMapper.ensureInitialized().equalsValue(
      this as InstagramMigrationDestination,
      other,
    );
  }

  @override
  int get hashCode {
    return InstagramMigrationDestinationMapper.ensureInitialized().hashValue(
      this as InstagramMigrationDestination,
    );
  }
}

extension InstagramMigrationDestinationValueCopy<$R, $Out>
    on ObjectCopyWith<$R, InstagramMigrationDestination, $Out> {
  InstagramMigrationDestinationCopyWith<$R, InstagramMigrationDestination, $Out>
  get $asInstagramMigrationDestination => $base.as(
    (v, t, t2) =>
        _InstagramMigrationDestinationCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class InstagramMigrationDestinationCopyWith<
  $R,
  $In extends InstagramMigrationDestination,
  $Out
>
    implements NotificationDestinationCopyWith<$R, $In, $Out> {
  @override
  $R call();
  InstagramMigrationDestinationCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _InstagramMigrationDestinationCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, InstagramMigrationDestination, $Out>
    implements
        InstagramMigrationDestinationCopyWith<
          $R,
          InstagramMigrationDestination,
          $Out
        > {
  _InstagramMigrationDestinationCopyWithImpl(
    super.value,
    super.then,
    super.then2,
  );

  @override
  late final ClassMapperBase<InstagramMigrationDestination> $mapper =
      InstagramMigrationDestinationMapper.ensureInitialized();
  @override
  $R call() => $apply(FieldCopyWithData({}));
  @override
  InstagramMigrationDestination $make(CopyWithData data) =>
      InstagramMigrationDestination();

  @override
  InstagramMigrationDestinationCopyWith<
    $R2,
    InstagramMigrationDestination,
    $Out2
  >
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _InstagramMigrationDestinationCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ProfileDestinationMapper extends SubClassMapperBase<ProfileDestination> {
  ProfileDestinationMapper._();

  static ProfileDestinationMapper? _instance;
  static ProfileDestinationMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ProfileDestinationMapper._());
      NotificationDestinationMapper.ensureInitialized().addSubMapper(
        _instance!,
      );
    }
    return _instance!;
  }

  @override
  final String id = 'ProfileDestination';

  static Did _$did(ProfileDestination v) => v.did;
  static const Field<ProfileDestination, Did> _f$did = Field('did', _$did);

  @override
  final MappableFields<ProfileDestination> fields = const {#did: _f$did};

  @override
  final String discriminatorKey = 'type';
  @override
  final dynamic discriminatorValue = 'ProfileDestination';
  @override
  late final ClassMapperBase superMapper =
      NotificationDestinationMapper.ensureInitialized();

  static ProfileDestination _instantiate(DecodingData data) {
    return ProfileDestination(data.dec(_f$did));
  }

  @override
  final Function instantiate = _instantiate;

  static ProfileDestination fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ProfileDestination>(map);
  }

  static ProfileDestination fromJson(String json) {
    return ensureInitialized().decodeJson<ProfileDestination>(json);
  }
}

mixin ProfileDestinationMappable {
  String toJson() {
    return ProfileDestinationMapper.ensureInitialized()
        .encodeJson<ProfileDestination>(this as ProfileDestination);
  }

  Map<String, dynamic> toMap() {
    return ProfileDestinationMapper.ensureInitialized()
        .encodeMap<ProfileDestination>(this as ProfileDestination);
  }

  ProfileDestinationCopyWith<
    ProfileDestination,
    ProfileDestination,
    ProfileDestination
  >
  get copyWith =>
      _ProfileDestinationCopyWithImpl<ProfileDestination, ProfileDestination>(
        this as ProfileDestination,
        $identity,
        $identity,
      );
  @override
  bool operator ==(Object other) {
    return ProfileDestinationMapper.ensureInitialized().equalsValue(
      this as ProfileDestination,
      other,
    );
  }

  @override
  int get hashCode {
    return ProfileDestinationMapper.ensureInitialized().hashValue(
      this as ProfileDestination,
    );
  }
}

extension ProfileDestinationValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ProfileDestination, $Out> {
  ProfileDestinationCopyWith<$R, ProfileDestination, $Out>
  get $asProfileDestination => $base.as(
    (v, t, t2) => _ProfileDestinationCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class ProfileDestinationCopyWith<
  $R,
  $In extends ProfileDestination,
  $Out
>
    implements NotificationDestinationCopyWith<$R, $In, $Out> {
  @override
  $R call({Did? did});
  ProfileDestinationCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ProfileDestinationCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ProfileDestination, $Out>
    implements ProfileDestinationCopyWith<$R, ProfileDestination, $Out> {
  _ProfileDestinationCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ProfileDestination> $mapper =
      ProfileDestinationMapper.ensureInitialized();
  @override
  $R call({Did? did}) =>
      $apply(FieldCopyWithData({if (did != null) #did: did}));
  @override
  ProfileDestination $make(CopyWithData data) =>
      ProfileDestination(data.get(#did, or: $value.did));

  @override
  ProfileDestinationCopyWith<$R2, ProfileDestination, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _ProfileDestinationCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class PostDestinationMapper extends SubClassMapperBase<PostDestination> {
  PostDestinationMapper._();

  static PostDestinationMapper? _instance;
  static PostDestinationMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = PostDestinationMapper._());
      NotificationDestinationMapper.ensureInitialized().addSubMapper(
        _instance!,
      );
    }
    return _instance!;
  }

  @override
  final String id = 'PostDestination';

  static AtUri _$subjectUri(PostDestination v) => v.subjectUri;
  static const Field<PostDestination, AtUri> _f$subjectUri = Field(
    'subjectUri',
    _$subjectUri,
  );
  static AtUri? _$focusUri(PostDestination v) => v.focusUri;
  static const Field<PostDestination, AtUri> _f$focusUri = Field(
    'focusUri',
    _$focusUri,
    opt: true,
  );

  @override
  final MappableFields<PostDestination> fields = const {
    #subjectUri: _f$subjectUri,
    #focusUri: _f$focusUri,
  };

  @override
  final String discriminatorKey = 'type';
  @override
  final dynamic discriminatorValue = 'PostDestination';
  @override
  late final ClassMapperBase superMapper =
      NotificationDestinationMapper.ensureInitialized();

  static PostDestination _instantiate(DecodingData data) {
    return PostDestination(
      data.dec(_f$subjectUri),
      focusUri: data.dec(_f$focusUri),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static PostDestination fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<PostDestination>(map);
  }

  static PostDestination fromJson(String json) {
    return ensureInitialized().decodeJson<PostDestination>(json);
  }
}

mixin PostDestinationMappable {
  String toJson() {
    return PostDestinationMapper.ensureInitialized()
        .encodeJson<PostDestination>(this as PostDestination);
  }

  Map<String, dynamic> toMap() {
    return PostDestinationMapper.ensureInitialized().encodeMap<PostDestination>(
      this as PostDestination,
    );
  }

  PostDestinationCopyWith<PostDestination, PostDestination, PostDestination>
  get copyWith =>
      _PostDestinationCopyWithImpl<PostDestination, PostDestination>(
        this as PostDestination,
        $identity,
        $identity,
      );
  @override
  bool operator ==(Object other) {
    return PostDestinationMapper.ensureInitialized().equalsValue(
      this as PostDestination,
      other,
    );
  }

  @override
  int get hashCode {
    return PostDestinationMapper.ensureInitialized().hashValue(
      this as PostDestination,
    );
  }
}

extension PostDestinationValueCopy<$R, $Out>
    on ObjectCopyWith<$R, PostDestination, $Out> {
  PostDestinationCopyWith<$R, PostDestination, $Out> get $asPostDestination =>
      $base.as((v, t, t2) => _PostDestinationCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class PostDestinationCopyWith<$R, $In extends PostDestination, $Out>
    implements NotificationDestinationCopyWith<$R, $In, $Out> {
  @override
  $R call({AtUri? subjectUri, AtUri? focusUri});
  PostDestinationCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _PostDestinationCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, PostDestination, $Out>
    implements PostDestinationCopyWith<$R, PostDestination, $Out> {
  _PostDestinationCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<PostDestination> $mapper =
      PostDestinationMapper.ensureInitialized();
  @override
  $R call({AtUri? subjectUri, Object? focusUri = $none}) => $apply(
    FieldCopyWithData({
      if (subjectUri != null) #subjectUri: subjectUri,
      if (focusUri != $none) #focusUri: focusUri,
    }),
  );
  @override
  PostDestination $make(CopyWithData data) => PostDestination(
    data.get(#subjectUri, or: $value.subjectUri),
    focusUri: data.get(#focusUri, or: $value.focusUri),
  );

  @override
  PostDestinationCopyWith<$R2, PostDestination, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _PostDestinationCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

