// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'router.dart';

class SavedPostFolderRouteDataMapper
    extends ClassMapperBase<SavedPostFolderRouteData> {
  SavedPostFolderRouteDataMapper._();

  static SavedPostFolderRouteDataMapper? _instance;
  static SavedPostFolderRouteDataMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = SavedPostFolderRouteDataMapper._(),
      );
      SavedPostFolderMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'SavedPostFolderRouteData';

  static SavedPostFolder _$folder(SavedPostFolderRouteData v) => v.folder;
  static const Field<SavedPostFolderRouteData, SavedPostFolder> _f$folder =
      Field('folder', _$folder);

  @override
  final MappableFields<SavedPostFolderRouteData> fields = const {
    #folder: _f$folder,
  };

  static SavedPostFolderRouteData _instantiate(DecodingData data) {
    return SavedPostFolderRouteData(folder: data.dec(_f$folder));
  }

  @override
  final Function instantiate = _instantiate;
}

mixin SavedPostFolderRouteDataMappable {
  SavedPostFolderRouteDataCopyWith<
    SavedPostFolderRouteData,
    SavedPostFolderRouteData,
    SavedPostFolderRouteData
  >
  get copyWith =>
      _SavedPostFolderRouteDataCopyWithImpl<
        SavedPostFolderRouteData,
        SavedPostFolderRouteData
      >(this as SavedPostFolderRouteData, $identity, $identity);
  @override
  bool operator ==(Object other) {
    return SavedPostFolderRouteDataMapper.ensureInitialized().equalsValue(
      this as SavedPostFolderRouteData,
      other,
    );
  }

  @override
  int get hashCode {
    return SavedPostFolderRouteDataMapper.ensureInitialized().hashValue(
      this as SavedPostFolderRouteData,
    );
  }
}

extension SavedPostFolderRouteDataValueCopy<$R, $Out>
    on ObjectCopyWith<$R, SavedPostFolderRouteData, $Out> {
  SavedPostFolderRouteDataCopyWith<$R, SavedPostFolderRouteData, $Out>
  get $asSavedPostFolderRouteData => $base.as(
    (v, t, t2) => _SavedPostFolderRouteDataCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class SavedPostFolderRouteDataCopyWith<
  $R,
  $In extends SavedPostFolderRouteData,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  SavedPostFolderCopyWith<$R, SavedPostFolder, SavedPostFolder> get folder;
  $R call({SavedPostFolder? folder});
  SavedPostFolderRouteDataCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _SavedPostFolderRouteDataCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, SavedPostFolderRouteData, $Out>
    implements
        SavedPostFolderRouteDataCopyWith<$R, SavedPostFolderRouteData, $Out> {
  _SavedPostFolderRouteDataCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<SavedPostFolderRouteData> $mapper =
      SavedPostFolderRouteDataMapper.ensureInitialized();
  @override
  SavedPostFolderCopyWith<$R, SavedPostFolder, SavedPostFolder> get folder =>
      $value.folder.copyWith.$chain((v) => call(folder: v));
  @override
  $R call({SavedPostFolder? folder}) =>
      $apply(FieldCopyWithData({if (folder != null) #folder: folder}));
  @override
  SavedPostFolderRouteData $make(CopyWithData data) =>
      SavedPostFolderRouteData(folder: data.get(#folder, or: $value.folder));

  @override
  SavedPostFolderRouteDataCopyWith<$R2, SavedPostFolderRouteData, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _SavedPostFolderRouteDataCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

