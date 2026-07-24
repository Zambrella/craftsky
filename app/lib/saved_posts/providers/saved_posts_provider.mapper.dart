// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'saved_posts_provider.dart';

class SavedPostListStateMapper extends ClassMapperBase<SavedPostListState> {
  SavedPostListStateMapper._();

  static SavedPostListStateMapper? _instance;
  static SavedPostListStateMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = SavedPostListStateMapper._());
      SavedPostItemMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'SavedPostListState';

  static List<SavedPostItem> _$items(SavedPostListState v) => v.items;
  static const Field<SavedPostListState, List<SavedPostItem>> _f$items = Field(
    'items',
    _$items,
  );
  static String? _$cursor(SavedPostListState v) => v.cursor;
  static const Field<SavedPostListState, String> _f$cursor = Field(
    'cursor',
    _$cursor,
    opt: true,
  );
  static bool _$isLoadingMore(SavedPostListState v) => v.isLoadingMore;
  static const Field<SavedPostListState, bool> _f$isLoadingMore = Field(
    'isLoadingMore',
    _$isLoadingMore,
    opt: true,
    def: false,
  );
  static Object? _$incrementalError(SavedPostListState v) => v.incrementalError;
  static const Field<SavedPostListState, Object> _f$incrementalError = Field(
    'incrementalError',
    _$incrementalError,
    opt: true,
  );

  @override
  final MappableFields<SavedPostListState> fields = const {
    #items: _f$items,
    #cursor: _f$cursor,
    #isLoadingMore: _f$isLoadingMore,
    #incrementalError: _f$incrementalError,
  };

  static SavedPostListState _instantiate(DecodingData data) {
    return SavedPostListState(
      items: data.dec(_f$items),
      cursor: data.dec(_f$cursor),
      isLoadingMore: data.dec(_f$isLoadingMore),
      incrementalError: data.dec(_f$incrementalError),
    );
  }

  @override
  final Function instantiate = _instantiate;
}

mixin SavedPostListStateMappable {
  SavedPostListStateCopyWith<
    SavedPostListState,
    SavedPostListState,
    SavedPostListState
  >
  get copyWith =>
      _SavedPostListStateCopyWithImpl<SavedPostListState, SavedPostListState>(
        this as SavedPostListState,
        $identity,
        $identity,
      );
  @override
  bool operator ==(Object other) {
    return SavedPostListStateMapper.ensureInitialized().equalsValue(
      this as SavedPostListState,
      other,
    );
  }

  @override
  int get hashCode {
    return SavedPostListStateMapper.ensureInitialized().hashValue(
      this as SavedPostListState,
    );
  }
}

extension SavedPostListStateValueCopy<$R, $Out>
    on ObjectCopyWith<$R, SavedPostListState, $Out> {
  SavedPostListStateCopyWith<$R, SavedPostListState, $Out>
  get $asSavedPostListState => $base.as(
    (v, t, t2) => _SavedPostListStateCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class SavedPostListStateCopyWith<
  $R,
  $In extends SavedPostListState,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<
    $R,
    SavedPostItem,
    SavedPostItemCopyWith<$R, SavedPostItem, SavedPostItem>
  >
  get items;
  $R call({
    List<SavedPostItem>? items,
    String? cursor,
    bool? isLoadingMore,
    Object? incrementalError,
  });
  SavedPostListStateCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _SavedPostListStateCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, SavedPostListState, $Out>
    implements SavedPostListStateCopyWith<$R, SavedPostListState, $Out> {
  _SavedPostListStateCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<SavedPostListState> $mapper =
      SavedPostListStateMapper.ensureInitialized();
  @override
  ListCopyWith<
    $R,
    SavedPostItem,
    SavedPostItemCopyWith<$R, SavedPostItem, SavedPostItem>
  >
  get items => ListCopyWith(
    $value.items,
    (v, t) => v.copyWith.$chain(t),
    (v) => call(items: v),
  );
  @override
  $R call({
    List<SavedPostItem>? items,
    Object? cursor = $none,
    bool? isLoadingMore,
    Object? incrementalError = $none,
  }) => $apply(
    FieldCopyWithData({
      if (items != null) #items: items,
      if (cursor != $none) #cursor: cursor,
      if (isLoadingMore != null) #isLoadingMore: isLoadingMore,
      if (incrementalError != $none) #incrementalError: incrementalError,
    }),
  );
  @override
  SavedPostListState $make(CopyWithData data) => SavedPostListState(
    items: data.get(#items, or: $value.items),
    cursor: data.get(#cursor, or: $value.cursor),
    isLoadingMore: data.get(#isLoadingMore, or: $value.isLoadingMore),
    incrementalError: data.get(#incrementalError, or: $value.incrementalError),
  );

  @override
  SavedPostListStateCopyWith<$R2, SavedPostListState, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _SavedPostListStateCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

