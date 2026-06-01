// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'moderation_metadata.dart';

class ModerationMetadataMapper extends ClassMapperBase<ModerationMetadata> {
  ModerationMetadataMapper._();

  static ModerationMetadataMapper? _instance;
  static ModerationMetadataMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ModerationMetadataMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'ModerationMetadata';

  static String _$warningKind(ModerationMetadata v) => v.warningKind;
  static const Field<ModerationMetadata, String> _f$warningKind = Field(
    'warningKind',
    _$warningKind,
  );

  @override
  final MappableFields<ModerationMetadata> fields = const {
    #warningKind: _f$warningKind,
  };

  static ModerationMetadata _instantiate(DecodingData data) {
    return ModerationMetadata(warningKind: data.dec(_f$warningKind));
  }

  @override
  final Function instantiate = _instantiate;

  static ModerationMetadata fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ModerationMetadata>(map);
  }

  static ModerationMetadata fromJson(String json) {
    return ensureInitialized().decodeJson<ModerationMetadata>(json);
  }
}

mixin ModerationMetadataMappable {
  String toJson() {
    return ModerationMetadataMapper.ensureInitialized()
        .encodeJson<ModerationMetadata>(this as ModerationMetadata);
  }

  Map<String, dynamic> toMap() {
    return ModerationMetadataMapper.ensureInitialized()
        .encodeMap<ModerationMetadata>(this as ModerationMetadata);
  }

  ModerationMetadataCopyWith<
    ModerationMetadata,
    ModerationMetadata,
    ModerationMetadata
  >
  get copyWith =>
      _ModerationMetadataCopyWithImpl<ModerationMetadata, ModerationMetadata>(
        this as ModerationMetadata,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return ModerationMetadataMapper.ensureInitialized().stringifyValue(
      this as ModerationMetadata,
    );
  }

  @override
  bool operator ==(Object other) {
    return ModerationMetadataMapper.ensureInitialized().equalsValue(
      this as ModerationMetadata,
      other,
    );
  }

  @override
  int get hashCode {
    return ModerationMetadataMapper.ensureInitialized().hashValue(
      this as ModerationMetadata,
    );
  }
}

extension ModerationMetadataValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ModerationMetadata, $Out> {
  ModerationMetadataCopyWith<$R, ModerationMetadata, $Out>
  get $asModerationMetadata => $base.as(
    (v, t, t2) => _ModerationMetadataCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class ModerationMetadataCopyWith<
  $R,
  $In extends ModerationMetadata,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? warningKind});
  ModerationMetadataCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ModerationMetadataCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ModerationMetadata, $Out>
    implements ModerationMetadataCopyWith<$R, ModerationMetadata, $Out> {
  _ModerationMetadataCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ModerationMetadata> $mapper =
      ModerationMetadataMapper.ensureInitialized();
  @override
  $R call({String? warningKind}) => $apply(
    FieldCopyWithData({if (warningKind != null) #warningKind: warningKind}),
  );
  @override
  ModerationMetadata $make(CopyWithData data) => ModerationMetadata(
    warningKind: data.get(#warningKind, or: $value.warningKind),
  );

  @override
  ModerationMetadataCopyWith<$R2, ModerationMetadata, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _ModerationMetadataCopyWithImpl<$R2, $Out2>($value, $cast, t);
}
