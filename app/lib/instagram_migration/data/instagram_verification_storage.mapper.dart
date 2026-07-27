// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'instagram_verification_storage.dart';

class InstagramVerificationSnapshotMapper
    extends ClassMapperBase<InstagramVerificationSnapshot> {
  InstagramVerificationSnapshotMapper._();

  static InstagramVerificationSnapshotMapper? _instance;
  static InstagramVerificationSnapshotMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = InstagramVerificationSnapshotMapper._(),
      );
    }
    return _instance!;
  }

  @override
  final String id = 'InstagramVerificationSnapshot';

  static String _$verificationId(InstagramVerificationSnapshot v) =>
      v.verificationId;
  static const Field<InstagramVerificationSnapshot, String> _f$verificationId =
      Field('verificationId', _$verificationId);
  static String _$challenge(InstagramVerificationSnapshot v) => v.challenge;
  static const Field<InstagramVerificationSnapshot, String> _f$challenge =
      Field('challenge', _$challenge);
  static Uri _$dmUrl(InstagramVerificationSnapshot v) => v.dmUrl;
  static const Field<InstagramVerificationSnapshot, Uri> _f$dmUrl = Field(
    'dmUrl',
    _$dmUrl,
  );
  static DateTime _$expiresAt(InstagramVerificationSnapshot v) => v.expiresAt;
  static const Field<InstagramVerificationSnapshot, DateTime> _f$expiresAt =
      Field('expiresAt', _$expiresAt);

  @override
  final MappableFields<InstagramVerificationSnapshot> fields = const {
    #verificationId: _f$verificationId,
    #challenge: _f$challenge,
    #dmUrl: _f$dmUrl,
    #expiresAt: _f$expiresAt,
  };

  static InstagramVerificationSnapshot _instantiate(DecodingData data) {
    return InstagramVerificationSnapshot(
      verificationId: data.dec(_f$verificationId),
      challenge: data.dec(_f$challenge),
      dmUrl: data.dec(_f$dmUrl),
      expiresAt: data.dec(_f$expiresAt),
    );
  }

  @override
  final Function instantiate = _instantiate;
}

mixin InstagramVerificationSnapshotMappable {
  InstagramVerificationSnapshotCopyWith<
    InstagramVerificationSnapshot,
    InstagramVerificationSnapshot,
    InstagramVerificationSnapshot
  >
  get copyWith =>
      _InstagramVerificationSnapshotCopyWithImpl<
        InstagramVerificationSnapshot,
        InstagramVerificationSnapshot
      >(this as InstagramVerificationSnapshot, $identity, $identity);
  @override
  bool operator ==(Object other) {
    return InstagramVerificationSnapshotMapper.ensureInitialized().equalsValue(
      this as InstagramVerificationSnapshot,
      other,
    );
  }

  @override
  int get hashCode {
    return InstagramVerificationSnapshotMapper.ensureInitialized().hashValue(
      this as InstagramVerificationSnapshot,
    );
  }
}

extension InstagramVerificationSnapshotValueCopy<$R, $Out>
    on ObjectCopyWith<$R, InstagramVerificationSnapshot, $Out> {
  InstagramVerificationSnapshotCopyWith<$R, InstagramVerificationSnapshot, $Out>
  get $asInstagramVerificationSnapshot => $base.as(
    (v, t, t2) =>
        _InstagramVerificationSnapshotCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class InstagramVerificationSnapshotCopyWith<
  $R,
  $In extends InstagramVerificationSnapshot,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({
    String? verificationId,
    String? challenge,
    Uri? dmUrl,
    DateTime? expiresAt,
  });
  InstagramVerificationSnapshotCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _InstagramVerificationSnapshotCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, InstagramVerificationSnapshot, $Out>
    implements
        InstagramVerificationSnapshotCopyWith<
          $R,
          InstagramVerificationSnapshot,
          $Out
        > {
  _InstagramVerificationSnapshotCopyWithImpl(
    super.value,
    super.then,
    super.then2,
  );

  @override
  late final ClassMapperBase<InstagramVerificationSnapshot> $mapper =
      InstagramVerificationSnapshotMapper.ensureInitialized();
  @override
  $R call({
    String? verificationId,
    String? challenge,
    Uri? dmUrl,
    DateTime? expiresAt,
  }) => $apply(
    FieldCopyWithData({
      if (verificationId != null) #verificationId: verificationId,
      if (challenge != null) #challenge: challenge,
      if (dmUrl != null) #dmUrl: dmUrl,
      if (expiresAt != null) #expiresAt: expiresAt,
    }),
  );
  @override
  InstagramVerificationSnapshot $make(CopyWithData data) =>
      InstagramVerificationSnapshot(
        verificationId: data.get(#verificationId, or: $value.verificationId),
        challenge: data.get(#challenge, or: $value.challenge),
        dmUrl: data.get(#dmUrl, or: $value.dmUrl),
        expiresAt: data.get(#expiresAt, or: $value.expiresAt),
      );

  @override
  InstagramVerificationSnapshotCopyWith<
    $R2,
    InstagramVerificationSnapshot,
    $Out2
  >
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _InstagramVerificationSnapshotCopyWithImpl<$R2, $Out2>($value, $cast, t);
}
