// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'saved_post_folder.dart';

class SavedPostFolderMapper extends ClassMapperBase<SavedPostFolder> {
  SavedPostFolderMapper._();

  static SavedPostFolderMapper? _instance;
  static SavedPostFolderMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = SavedPostFolderMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'SavedPostFolder';

  static String _$id(SavedPostFolder v) => v.id;
  static const Field<SavedPostFolder, String> _f$id = Field('id', _$id);
  static String _$name(SavedPostFolder v) => v.name;
  static const Field<SavedPostFolder, String> _f$name = Field('name', _$name);
  static DateTime _$createdAt(SavedPostFolder v) => v.createdAt;
  static const Field<SavedPostFolder, DateTime> _f$createdAt = Field(
    'createdAt',
    _$createdAt,
  );
  static DateTime _$updatedAt(SavedPostFolder v) => v.updatedAt;
  static const Field<SavedPostFolder, DateTime> _f$updatedAt = Field(
    'updatedAt',
    _$updatedAt,
  );

  @override
  final MappableFields<SavedPostFolder> fields = const {
    #id: _f$id,
    #name: _f$name,
    #createdAt: _f$createdAt,
    #updatedAt: _f$updatedAt,
  };

  static SavedPostFolder _instantiate(DecodingData data) {
    return SavedPostFolder(
      id: data.dec(_f$id),
      name: data.dec(_f$name),
      createdAt: data.dec(_f$createdAt),
      updatedAt: data.dec(_f$updatedAt),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static SavedPostFolder fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<SavedPostFolder>(map);
  }

  static SavedPostFolder fromJson(String json) {
    return ensureInitialized().decodeJson<SavedPostFolder>(json);
  }
}

mixin SavedPostFolderMappable {
  String toJson() {
    return SavedPostFolderMapper.ensureInitialized()
        .encodeJson<SavedPostFolder>(this as SavedPostFolder);
  }

  Map<String, dynamic> toMap() {
    return SavedPostFolderMapper.ensureInitialized().encodeMap<SavedPostFolder>(
      this as SavedPostFolder,
    );
  }

  SavedPostFolderCopyWith<SavedPostFolder, SavedPostFolder, SavedPostFolder>
  get copyWith =>
      _SavedPostFolderCopyWithImpl<SavedPostFolder, SavedPostFolder>(
        this as SavedPostFolder,
        $identity,
        $identity,
      );
}

extension SavedPostFolderValueCopy<$R, $Out>
    on ObjectCopyWith<$R, SavedPostFolder, $Out> {
  SavedPostFolderCopyWith<$R, SavedPostFolder, $Out> get $asSavedPostFolder =>
      $base.as((v, t, t2) => _SavedPostFolderCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class SavedPostFolderCopyWith<$R, $In extends SavedPostFolder, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? id, String? name, DateTime? createdAt, DateTime? updatedAt});
  SavedPostFolderCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _SavedPostFolderCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, SavedPostFolder, $Out>
    implements SavedPostFolderCopyWith<$R, SavedPostFolder, $Out> {
  _SavedPostFolderCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<SavedPostFolder> $mapper =
      SavedPostFolderMapper.ensureInitialized();
  @override
  $R call({
    String? id,
    String? name,
    DateTime? createdAt,
    DateTime? updatedAt,
  }) => $apply(
    FieldCopyWithData({
      if (id != null) #id: id,
      if (name != null) #name: name,
      if (createdAt != null) #createdAt: createdAt,
      if (updatedAt != null) #updatedAt: updatedAt,
    }),
  );
  @override
  SavedPostFolder $make(CopyWithData data) => SavedPostFolder(
    id: data.get(#id, or: $value.id),
    name: data.get(#name, or: $value.name),
    createdAt: data.get(#createdAt, or: $value.createdAt),
    updatedAt: data.get(#updatedAt, or: $value.updatedAt),
  );

  @override
  SavedPostFolderCopyWith<$R2, SavedPostFolder, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _SavedPostFolderCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class SavedPostFolderPageMapper extends ClassMapperBase<SavedPostFolderPage> {
  SavedPostFolderPageMapper._();

  static SavedPostFolderPageMapper? _instance;
  static SavedPostFolderPageMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = SavedPostFolderPageMapper._());
      SavedPostFolderMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'SavedPostFolderPage';

  static List<SavedPostFolder> _$items(SavedPostFolderPage v) => v.items;
  static const Field<SavedPostFolderPage, List<SavedPostFolder>> _f$items =
      Field('items', _$items);
  static String? _$cursor(SavedPostFolderPage v) => v.cursor;
  static const Field<SavedPostFolderPage, String> _f$cursor = Field(
    'cursor',
    _$cursor,
    opt: true,
  );

  @override
  final MappableFields<SavedPostFolderPage> fields = const {
    #items: _f$items,
    #cursor: _f$cursor,
  };
  @override
  final bool ignoreNull = true;

  static SavedPostFolderPage _instantiate(DecodingData data) {
    return SavedPostFolderPage(
      items: data.dec(_f$items),
      cursor: data.dec(_f$cursor),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static SavedPostFolderPage fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<SavedPostFolderPage>(map);
  }

  static SavedPostFolderPage fromJson(String json) {
    return ensureInitialized().decodeJson<SavedPostFolderPage>(json);
  }
}

mixin SavedPostFolderPageMappable {
  String toJson() {
    return SavedPostFolderPageMapper.ensureInitialized()
        .encodeJson<SavedPostFolderPage>(this as SavedPostFolderPage);
  }

  Map<String, dynamic> toMap() {
    return SavedPostFolderPageMapper.ensureInitialized()
        .encodeMap<SavedPostFolderPage>(this as SavedPostFolderPage);
  }

  SavedPostFolderPageCopyWith<
    SavedPostFolderPage,
    SavedPostFolderPage,
    SavedPostFolderPage
  >
  get copyWith =>
      _SavedPostFolderPageCopyWithImpl<
        SavedPostFolderPage,
        SavedPostFolderPage
      >(this as SavedPostFolderPage, $identity, $identity);
  @override
  bool operator ==(Object other) {
    return SavedPostFolderPageMapper.ensureInitialized().equalsValue(
      this as SavedPostFolderPage,
      other,
    );
  }

  @override
  int get hashCode {
    return SavedPostFolderPageMapper.ensureInitialized().hashValue(
      this as SavedPostFolderPage,
    );
  }
}

extension SavedPostFolderPageValueCopy<$R, $Out>
    on ObjectCopyWith<$R, SavedPostFolderPage, $Out> {
  SavedPostFolderPageCopyWith<$R, SavedPostFolderPage, $Out>
  get $asSavedPostFolderPage => $base.as(
    (v, t, t2) => _SavedPostFolderPageCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class SavedPostFolderPageCopyWith<
  $R,
  $In extends SavedPostFolderPage,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<
    $R,
    SavedPostFolder,
    SavedPostFolderCopyWith<$R, SavedPostFolder, SavedPostFolder>
  >
  get items;
  $R call({List<SavedPostFolder>? items, String? cursor});
  SavedPostFolderPageCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _SavedPostFolderPageCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, SavedPostFolderPage, $Out>
    implements SavedPostFolderPageCopyWith<$R, SavedPostFolderPage, $Out> {
  _SavedPostFolderPageCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<SavedPostFolderPage> $mapper =
      SavedPostFolderPageMapper.ensureInitialized();
  @override
  ListCopyWith<
    $R,
    SavedPostFolder,
    SavedPostFolderCopyWith<$R, SavedPostFolder, SavedPostFolder>
  >
  get items => ListCopyWith(
    $value.items,
    (v, t) => v.copyWith.$chain(t),
    (v) => call(items: v),
  );
  @override
  $R call({List<SavedPostFolder>? items, Object? cursor = $none}) => $apply(
    FieldCopyWithData({
      if (items != null) #items: items,
      if (cursor != $none) #cursor: cursor,
    }),
  );
  @override
  SavedPostFolderPage $make(CopyWithData data) => SavedPostFolderPage(
    items: data.get(#items, or: $value.items),
    cursor: data.get(#cursor, or: $value.cursor),
  );

  @override
  SavedPostFolderPageCopyWith<$R2, SavedPostFolderPage, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _SavedPostFolderPageCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

