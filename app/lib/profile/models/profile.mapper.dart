// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'profile.dart';

class ProfileMapper extends ClassMapperBase<Profile> {
  ProfileMapper._();

  static ProfileMapper? _instance;
  static ProfileMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ProfileMapper._());
      MapperContainer.globals.useAll([DidMapper(), HandleMapper()]);
      ModerationMetadataMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'Profile';

  static Did _$did(Profile v) => v.did;
  static dynamic _arg$did(f) => f<Did>();
  static const Field<Profile, String> _f$did = Field(
    'did',
    _$did,
    arg: _arg$did,
  );
  static Handle _$handle(Profile v) => v.handle;
  static dynamic _arg$handle(f) => f<Handle>();
  static const Field<Profile, String> _f$handle = Field(
    'handle',
    _$handle,
    arg: _arg$handle,
  );
  static List<String> _$crafts(Profile v) => v.crafts;
  static const Field<Profile, List<String>> _f$crafts = Field(
    'crafts',
    _$crafts,
  );
  static String? _$displayName(Profile v) => v.displayName;
  static const Field<Profile, String> _f$displayName = Field(
    'displayName',
    _$displayName,
    opt: true,
  );
  static String? _$description(Profile v) => v.description;
  static const Field<Profile, String> _f$description = Field(
    'description',
    _$description,
    opt: true,
  );
  static String? _$avatar(Profile v) => v.avatar;
  static const Field<Profile, String> _f$avatar = Field(
    'avatar',
    _$avatar,
    opt: true,
  );
  static String? _$banner(Profile v) => v.banner;
  static const Field<Profile, String> _f$banner = Field(
    'banner',
    _$banner,
    opt: true,
  );
  static DateTime? _$createdAt(Profile v) => v.createdAt;
  static const Field<Profile, DateTime> _f$createdAt = Field(
    'createdAt',
    _$createdAt,
    opt: true,
  );
  static bool _$viewerIsFollowing(Profile v) => v.viewerIsFollowing;
  static const Field<Profile, bool> _f$viewerIsFollowing = Field(
    'viewerIsFollowing',
    _$viewerIsFollowing,
    opt: true,
    def: false,
  );
  static bool _$muted(Profile v) => v.muted;
  static const Field<Profile, bool> _f$muted = Field(
    'muted',
    _$muted,
    opt: true,
    def: false,
  );
  static bool _$blocking(Profile v) => v.blocking;
  static const Field<Profile, bool> _f$blocking = Field(
    'blocking',
    _$blocking,
    opt: true,
    def: false,
  );
  static bool _$blockedBy(Profile v) => v.blockedBy;
  static const Field<Profile, bool> _f$blockedBy = Field(
    'blockedBy',
    _$blockedBy,
    opt: true,
    def: false,
  );
  static bool _$isCraftskyProfile(Profile v) => v.isCraftskyProfile;
  static const Field<Profile, bool> _f$isCraftskyProfile = Field(
    'isCraftskyProfile',
    _$isCraftskyProfile,
    opt: true,
    def: true,
  );
  static int? _$followerCount(Profile v) => v.followerCount;
  static const Field<Profile, int> _f$followerCount = Field(
    'followerCount',
    _$followerCount,
    opt: true,
  );
  static int? _$followingCount(Profile v) => v.followingCount;
  static const Field<Profile, int> _f$followingCount = Field(
    'followingCount',
    _$followingCount,
    opt: true,
  );
  static int? _$mutualFollowerCount(Profile v) => v.mutualFollowerCount;
  static const Field<Profile, int> _f$mutualFollowerCount = Field(
    'mutualFollowerCount',
    _$mutualFollowerCount,
    opt: true,
  );
  static int? _$postCount(Profile v) => v.postCount;
  static const Field<Profile, int> _f$postCount = Field(
    'postCount',
    _$postCount,
    opt: true,
  );
  static int? _$postsLast7Days(Profile v) => v.postsLast7Days;
  static const Field<Profile, int> _f$postsLast7Days = Field(
    'postsLast7Days',
    _$postsLast7Days,
    opt: true,
  );
  static int? _$projectCount(Profile v) => v.projectCount;
  static const Field<Profile, int> _f$projectCount = Field(
    'projectCount',
    _$projectCount,
    opt: true,
  );
  static ModerationMetadata? _$moderation(Profile v) => v.moderation;
  static const Field<Profile, ModerationMetadata> _f$moderation = Field(
    'moderation',
    _$moderation,
    opt: true,
  );

  @override
  final MappableFields<Profile> fields = const {
    #did: _f$did,
    #handle: _f$handle,
    #crafts: _f$crafts,
    #displayName: _f$displayName,
    #description: _f$description,
    #avatar: _f$avatar,
    #banner: _f$banner,
    #createdAt: _f$createdAt,
    #viewerIsFollowing: _f$viewerIsFollowing,
    #muted: _f$muted,
    #blocking: _f$blocking,
    #blockedBy: _f$blockedBy,
    #isCraftskyProfile: _f$isCraftskyProfile,
    #followerCount: _f$followerCount,
    #followingCount: _f$followingCount,
    #mutualFollowerCount: _f$mutualFollowerCount,
    #postCount: _f$postCount,
    #postsLast7Days: _f$postsLast7Days,
    #projectCount: _f$projectCount,
    #moderation: _f$moderation,
  };

  static Profile _instantiate(DecodingData data) {
    return Profile(
      did: data.dec(_f$did),
      handle: data.dec(_f$handle),
      crafts: data.dec(_f$crafts),
      displayName: data.dec(_f$displayName),
      description: data.dec(_f$description),
      avatar: data.dec(_f$avatar),
      banner: data.dec(_f$banner),
      createdAt: data.dec(_f$createdAt),
      viewerIsFollowing: data.dec(_f$viewerIsFollowing),
      muted: data.dec(_f$muted),
      blocking: data.dec(_f$blocking),
      blockedBy: data.dec(_f$blockedBy),
      isCraftskyProfile: data.dec(_f$isCraftskyProfile),
      followerCount: data.dec(_f$followerCount),
      followingCount: data.dec(_f$followingCount),
      mutualFollowerCount: data.dec(_f$mutualFollowerCount),
      postCount: data.dec(_f$postCount),
      postsLast7Days: data.dec(_f$postsLast7Days),
      projectCount: data.dec(_f$projectCount),
      moderation: data.dec(_f$moderation),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static Profile fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<Profile>(map);
  }

  static Profile fromJson(String json) {
    return ensureInitialized().decodeJson<Profile>(json);
  }
}

mixin ProfileMappable {
  String toJson() {
    return ProfileMapper.ensureInitialized().encodeJson<Profile>(
      this as Profile,
    );
  }

  Map<String, dynamic> toMap() {
    return ProfileMapper.ensureInitialized().encodeMap<Profile>(
      this as Profile,
    );
  }

  ProfileCopyWith<Profile, Profile, Profile> get copyWith =>
      _ProfileCopyWithImpl<Profile, Profile>(
        this as Profile,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return ProfileMapper.ensureInitialized().stringifyValue(this as Profile);
  }

  @override
  bool operator ==(Object other) {
    return ProfileMapper.ensureInitialized().equalsValue(
      this as Profile,
      other,
    );
  }

  @override
  int get hashCode {
    return ProfileMapper.ensureInitialized().hashValue(this as Profile);
  }
}

extension ProfileValueCopy<$R, $Out> on ObjectCopyWith<$R, Profile, $Out> {
  ProfileCopyWith<$R, Profile, $Out> get $asProfile =>
      $base.as((v, t, t2) => _ProfileCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class ProfileCopyWith<$R, $In extends Profile, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>> get crafts;
  ModerationMetadataCopyWith<$R, ModerationMetadata, ModerationMetadata>?
  get moderation;
  $R call({
    String? did,
    String? handle,
    List<String>? crafts,
    String? displayName,
    String? description,
    String? avatar,
    String? banner,
    DateTime? createdAt,
    bool? viewerIsFollowing,
    bool? muted,
    bool? blocking,
    bool? blockedBy,
    bool? isCraftskyProfile,
    int? followerCount,
    int? followingCount,
    int? mutualFollowerCount,
    int? postCount,
    int? postsLast7Days,
    int? projectCount,
    ModerationMetadata? moderation,
  });
  ProfileCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _ProfileCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, Profile, $Out>
    implements ProfileCopyWith<$R, Profile, $Out> {
  _ProfileCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<Profile> $mapper =
      ProfileMapper.ensureInitialized();
  @override
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>> get crafts =>
      ListCopyWith(
        $value.crafts,
        (v, t) => ObjectCopyWith(v, $identity, t),
        (v) => call(crafts: v),
      );
  @override
  ModerationMetadataCopyWith<$R, ModerationMetadata, ModerationMetadata>?
  get moderation =>
      $value.moderation?.copyWith.$chain((v) => call(moderation: v));
  @override
  $R call({
    String? did,
    String? handle,
    List<String>? crafts,
    Object? displayName = $none,
    Object? description = $none,
    Object? avatar = $none,
    Object? banner = $none,
    Object? createdAt = $none,
    bool? viewerIsFollowing,
    bool? muted,
    bool? blocking,
    bool? blockedBy,
    bool? isCraftskyProfile,
    Object? followerCount = $none,
    Object? followingCount = $none,
    Object? mutualFollowerCount = $none,
    Object? postCount = $none,
    Object? postsLast7Days = $none,
    Object? projectCount = $none,
    Object? moderation = $none,
  }) => $apply(
    FieldCopyWithData({
      if (did != null) #did: did,
      if (handle != null) #handle: handle,
      if (crafts != null) #crafts: crafts,
      if (displayName != $none) #displayName: displayName,
      if (description != $none) #description: description,
      if (avatar != $none) #avatar: avatar,
      if (banner != $none) #banner: banner,
      if (createdAt != $none) #createdAt: createdAt,
      if (viewerIsFollowing != null) #viewerIsFollowing: viewerIsFollowing,
      if (muted != null) #muted: muted,
      if (blocking != null) #blocking: blocking,
      if (blockedBy != null) #blockedBy: blockedBy,
      if (isCraftskyProfile != null) #isCraftskyProfile: isCraftskyProfile,
      if (followerCount != $none) #followerCount: followerCount,
      if (followingCount != $none) #followingCount: followingCount,
      if (mutualFollowerCount != $none)
        #mutualFollowerCount: mutualFollowerCount,
      if (postCount != $none) #postCount: postCount,
      if (postsLast7Days != $none) #postsLast7Days: postsLast7Days,
      if (projectCount != $none) #projectCount: projectCount,
      if (moderation != $none) #moderation: moderation,
    }),
  );
  @override
  Profile $make(CopyWithData data) => Profile(
    did: data.get(#did, or: $value.did),
    handle: data.get(#handle, or: $value.handle),
    crafts: data.get(#crafts, or: $value.crafts),
    displayName: data.get(#displayName, or: $value.displayName),
    description: data.get(#description, or: $value.description),
    avatar: data.get(#avatar, or: $value.avatar),
    banner: data.get(#banner, or: $value.banner),
    createdAt: data.get(#createdAt, or: $value.createdAt),
    viewerIsFollowing: data.get(
      #viewerIsFollowing,
      or: $value.viewerIsFollowing,
    ),
    muted: data.get(#muted, or: $value.muted),
    blocking: data.get(#blocking, or: $value.blocking),
    blockedBy: data.get(#blockedBy, or: $value.blockedBy),
    isCraftskyProfile: data.get(
      #isCraftskyProfile,
      or: $value.isCraftskyProfile,
    ),
    followerCount: data.get(#followerCount, or: $value.followerCount),
    followingCount: data.get(#followingCount, or: $value.followingCount),
    mutualFollowerCount: data.get(
      #mutualFollowerCount,
      or: $value.mutualFollowerCount,
    ),
    postCount: data.get(#postCount, or: $value.postCount),
    postsLast7Days: data.get(#postsLast7Days, or: $value.postsLast7Days),
    projectCount: data.get(#projectCount, or: $value.projectCount),
    moderation: data.get(#moderation, or: $value.moderation),
  );

  @override
  ProfileCopyWith<$R2, Profile, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _ProfileCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

