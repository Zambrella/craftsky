// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'saved_post_folders_provider.dart';

class SavedPostFolderListStateMapper
    extends ClassMapperBase<SavedPostFolderListState> {
  SavedPostFolderListStateMapper._();

  static SavedPostFolderListStateMapper? _instance;
  static SavedPostFolderListStateMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = SavedPostFolderListStateMapper._(),
      );
      SavedPostFolderMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'SavedPostFolderListState';

  static List<SavedPostFolder> _$items(SavedPostFolderListState v) => v.items;
  static const Field<SavedPostFolderListState, List<SavedPostFolder>> _f$items =
      Field('items', _$items);
  static String? _$cursor(SavedPostFolderListState v) => v.cursor;
  static const Field<SavedPostFolderListState, String> _f$cursor = Field(
    'cursor',
    _$cursor,
    opt: true,
  );
  static bool _$isLoadingMore(SavedPostFolderListState v) => v.isLoadingMore;
  static const Field<SavedPostFolderListState, bool> _f$isLoadingMore = Field(
    'isLoadingMore',
    _$isLoadingMore,
    opt: true,
    def: false,
  );
  static Object? _$incrementalError(SavedPostFolderListState v) =>
      v.incrementalError;
  static const Field<SavedPostFolderListState, Object> _f$incrementalError =
      Field('incrementalError', _$incrementalError, opt: true);
  static SavedPostFailure? _$mutationFailure(SavedPostFolderListState v) =>
      v.mutationFailure;
  static const Field<SavedPostFolderListState, SavedPostFailure>
  _f$mutationFailure = Field('mutationFailure', _$mutationFailure, opt: true);
  static String? _$deletedFolderId(SavedPostFolderListState v) =>
      v.deletedFolderId;
  static const Field<SavedPostFolderListState, String> _f$deletedFolderId =
      Field('deletedFolderId', _$deletedFolderId, opt: true);
  static Map<String, SavedPostFolder> _$retainedFolders(
    SavedPostFolderListState v,
  ) => v.retainedFolders;
  static const Field<SavedPostFolderListState, Map<String, SavedPostFolder>>
  _f$retainedFolders = Field(
    'retainedFolders',
    _$retainedFolders,
    opt: true,
    def: const {},
  );

  @override
  final MappableFields<SavedPostFolderListState> fields = const {
    #items: _f$items,
    #cursor: _f$cursor,
    #isLoadingMore: _f$isLoadingMore,
    #incrementalError: _f$incrementalError,
    #mutationFailure: _f$mutationFailure,
    #deletedFolderId: _f$deletedFolderId,
    #retainedFolders: _f$retainedFolders,
  };

  static SavedPostFolderListState _instantiate(DecodingData data) {
    return SavedPostFolderListState(
      items: data.dec(_f$items),
      cursor: data.dec(_f$cursor),
      isLoadingMore: data.dec(_f$isLoadingMore),
      incrementalError: data.dec(_f$incrementalError),
      mutationFailure: data.dec(_f$mutationFailure),
      deletedFolderId: data.dec(_f$deletedFolderId),
      retainedFolders: data.dec(_f$retainedFolders),
    );
  }

  @override
  final Function instantiate = _instantiate;
}

mixin SavedPostFolderListStateMappable {
  SavedPostFolderListStateCopyWith<
    SavedPostFolderListState,
    SavedPostFolderListState,
    SavedPostFolderListState
  >
  get copyWith =>
      _SavedPostFolderListStateCopyWithImpl<
        SavedPostFolderListState,
        SavedPostFolderListState
      >(this as SavedPostFolderListState, $identity, $identity);
  @override
  bool operator ==(Object other) {
    return SavedPostFolderListStateMapper.ensureInitialized().equalsValue(
      this as SavedPostFolderListState,
      other,
    );
  }

  @override
  int get hashCode {
    return SavedPostFolderListStateMapper.ensureInitialized().hashValue(
      this as SavedPostFolderListState,
    );
  }
}

extension SavedPostFolderListStateValueCopy<$R, $Out>
    on ObjectCopyWith<$R, SavedPostFolderListState, $Out> {
  SavedPostFolderListStateCopyWith<$R, SavedPostFolderListState, $Out>
  get $asSavedPostFolderListState => $base.as(
    (v, t, t2) => _SavedPostFolderListStateCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class SavedPostFolderListStateCopyWith<
  $R,
  $In extends SavedPostFolderListState,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<
    $R,
    SavedPostFolder,
    SavedPostFolderCopyWith<$R, SavedPostFolder, SavedPostFolder>
  >
  get items;
  MapCopyWith<
    $R,
    String,
    SavedPostFolder,
    SavedPostFolderCopyWith<$R, SavedPostFolder, SavedPostFolder>
  >
  get retainedFolders;
  $R call({
    List<SavedPostFolder>? items,
    String? cursor,
    bool? isLoadingMore,
    Object? incrementalError,
    SavedPostFailure? mutationFailure,
    String? deletedFolderId,
    Map<String, SavedPostFolder>? retainedFolders,
  });
  SavedPostFolderListStateCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _SavedPostFolderListStateCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, SavedPostFolderListState, $Out>
    implements
        SavedPostFolderListStateCopyWith<$R, SavedPostFolderListState, $Out> {
  _SavedPostFolderListStateCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<SavedPostFolderListState> $mapper =
      SavedPostFolderListStateMapper.ensureInitialized();
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
  MapCopyWith<
    $R,
    String,
    SavedPostFolder,
    SavedPostFolderCopyWith<$R, SavedPostFolder, SavedPostFolder>
  >
  get retainedFolders => MapCopyWith(
    $value.retainedFolders,
    (v, t) => v.copyWith.$chain(t),
    (v) => call(retainedFolders: v),
  );
  @override
  $R call({
    List<SavedPostFolder>? items,
    Object? cursor = $none,
    bool? isLoadingMore,
    Object? incrementalError = $none,
    Object? mutationFailure = $none,
    Object? deletedFolderId = $none,
    Map<String, SavedPostFolder>? retainedFolders,
  }) => $apply(
    FieldCopyWithData({
      if (items != null) #items: items,
      if (cursor != $none) #cursor: cursor,
      if (isLoadingMore != null) #isLoadingMore: isLoadingMore,
      if (incrementalError != $none) #incrementalError: incrementalError,
      if (mutationFailure != $none) #mutationFailure: mutationFailure,
      if (deletedFolderId != $none) #deletedFolderId: deletedFolderId,
      if (retainedFolders != null) #retainedFolders: retainedFolders,
    }),
  );
  @override
  SavedPostFolderListState $make(CopyWithData data) => SavedPostFolderListState(
    items: data.get(#items, or: $value.items),
    cursor: data.get(#cursor, or: $value.cursor),
    isLoadingMore: data.get(#isLoadingMore, or: $value.isLoadingMore),
    incrementalError: data.get(#incrementalError, or: $value.incrementalError),
    mutationFailure: data.get(#mutationFailure, or: $value.mutationFailure),
    deletedFolderId: data.get(#deletedFolderId, or: $value.deletedFolderId),
    retainedFolders: data.get(#retainedFolders, or: $value.retainedFolders),
  );

  @override
  SavedPostFolderListStateCopyWith<$R2, SavedPostFolderListState, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _SavedPostFolderListStateCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

