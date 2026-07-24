// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'saved_post_destination.dart';

class SavedPostDestinationMapper extends ClassMapperBase<SavedPostDestination> {
  SavedPostDestinationMapper._();

  static SavedPostDestinationMapper? _instance;
  static SavedPostDestinationMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = SavedPostDestinationMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'SavedPostDestination';

  static AtUri _$threadUri(SavedPostDestination v) => v.threadUri;
  static const Field<SavedPostDestination, AtUri> _f$threadUri = Field(
    'threadUri',
    _$threadUri,
  );
  static AtUri? _$focusUri(SavedPostDestination v) => v.focusUri;
  static const Field<SavedPostDestination, AtUri> _f$focusUri = Field(
    'focusUri',
    _$focusUri,
    opt: true,
  );

  @override
  final MappableFields<SavedPostDestination> fields = const {
    #threadUri: _f$threadUri,
    #focusUri: _f$focusUri,
  };

  static SavedPostDestination _instantiate(DecodingData data) {
    return SavedPostDestination(
      threadUri: data.dec(_f$threadUri),
      focusUri: data.dec(_f$focusUri),
    );
  }

  @override
  final Function instantiate = _instantiate;
}

mixin SavedPostDestinationMappable {
  SavedPostDestinationCopyWith<
    SavedPostDestination,
    SavedPostDestination,
    SavedPostDestination
  >
  get copyWith =>
      _SavedPostDestinationCopyWithImpl<
        SavedPostDestination,
        SavedPostDestination
      >(this as SavedPostDestination, $identity, $identity);
  @override
  bool operator ==(Object other) {
    return SavedPostDestinationMapper.ensureInitialized().equalsValue(
      this as SavedPostDestination,
      other,
    );
  }

  @override
  int get hashCode {
    return SavedPostDestinationMapper.ensureInitialized().hashValue(
      this as SavedPostDestination,
    );
  }
}

extension SavedPostDestinationValueCopy<$R, $Out>
    on ObjectCopyWith<$R, SavedPostDestination, $Out> {
  SavedPostDestinationCopyWith<$R, SavedPostDestination, $Out>
  get $asSavedPostDestination => $base.as(
    (v, t, t2) => _SavedPostDestinationCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class SavedPostDestinationCopyWith<
  $R,
  $In extends SavedPostDestination,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({AtUri? threadUri, AtUri? focusUri});
  SavedPostDestinationCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _SavedPostDestinationCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, SavedPostDestination, $Out>
    implements SavedPostDestinationCopyWith<$R, SavedPostDestination, $Out> {
  _SavedPostDestinationCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<SavedPostDestination> $mapper =
      SavedPostDestinationMapper.ensureInitialized();
  @override
  $R call({AtUri? threadUri, Object? focusUri = $none}) => $apply(
    FieldCopyWithData({
      if (threadUri != null) #threadUri: threadUri,
      if (focusUri != $none) #focusUri: focusUri,
    }),
  );
  @override
  SavedPostDestination $make(CopyWithData data) => SavedPostDestination(
    threadUri: data.get(#threadUri, or: $value.threadUri),
    focusUri: data.get(#focusUri, or: $value.focusUri),
  );

  @override
  SavedPostDestinationCopyWith<$R2, SavedPostDestination, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _SavedPostDestinationCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

