// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'saved_post_error.dart';

class SavedPostFailureMapper extends ClassMapperBase<SavedPostFailure> {
  SavedPostFailureMapper._();

  static SavedPostFailureMapper? _instance;
  static SavedPostFailureMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = SavedPostFailureMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'SavedPostFailure';

  static SavedPostFailureKind _$kind(SavedPostFailure v) => v.kind;
  static const Field<SavedPostFailure, SavedPostFailureKind> _f$kind = Field(
    'kind',
    _$kind,
  );
  static SavedPostOperation _$operation(SavedPostFailure v) => v.operation;
  static const Field<SavedPostFailure, SavedPostOperation> _f$operation = Field(
    'operation',
    _$operation,
  );

  @override
  final MappableFields<SavedPostFailure> fields = const {
    #kind: _f$kind,
    #operation: _f$operation,
  };

  static SavedPostFailure _instantiate(DecodingData data) {
    return SavedPostFailure(
      kind: data.dec(_f$kind),
      operation: data.dec(_f$operation),
    );
  }

  @override
  final Function instantiate = _instantiate;
}

mixin SavedPostFailureMappable {
  SavedPostFailureCopyWith<SavedPostFailure, SavedPostFailure, SavedPostFailure>
  get copyWith =>
      _SavedPostFailureCopyWithImpl<SavedPostFailure, SavedPostFailure>(
        this as SavedPostFailure,
        $identity,
        $identity,
      );
  @override
  bool operator ==(Object other) {
    return SavedPostFailureMapper.ensureInitialized().equalsValue(
      this as SavedPostFailure,
      other,
    );
  }

  @override
  int get hashCode {
    return SavedPostFailureMapper.ensureInitialized().hashValue(
      this as SavedPostFailure,
    );
  }
}

extension SavedPostFailureValueCopy<$R, $Out>
    on ObjectCopyWith<$R, SavedPostFailure, $Out> {
  SavedPostFailureCopyWith<$R, SavedPostFailure, $Out>
  get $asSavedPostFailure =>
      $base.as((v, t, t2) => _SavedPostFailureCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class SavedPostFailureCopyWith<$R, $In extends SavedPostFailure, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({SavedPostFailureKind? kind, SavedPostOperation? operation});
  SavedPostFailureCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _SavedPostFailureCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, SavedPostFailure, $Out>
    implements SavedPostFailureCopyWith<$R, SavedPostFailure, $Out> {
  _SavedPostFailureCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<SavedPostFailure> $mapper =
      SavedPostFailureMapper.ensureInitialized();
  @override
  $R call({SavedPostFailureKind? kind, SavedPostOperation? operation}) =>
      $apply(
        FieldCopyWithData({
          if (kind != null) #kind: kind,
          if (operation != null) #operation: operation,
        }),
      );
  @override
  SavedPostFailure $make(CopyWithData data) => SavedPostFailure(
    kind: data.get(#kind, or: $value.kind),
    operation: data.get(#operation, or: $value.operation),
  );

  @override
  SavedPostFailureCopyWith<$R2, SavedPostFailure, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _SavedPostFailureCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

