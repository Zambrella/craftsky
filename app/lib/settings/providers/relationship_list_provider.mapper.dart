// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'relationship_list_provider.dart';

class RelationshipListStateMapper
    extends ClassMapperBase<RelationshipListState> {
  RelationshipListStateMapper._();

  static RelationshipListStateMapper? _instance;
  static RelationshipListStateMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = RelationshipListStateMapper._());
      ProfileAccountSummaryMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'RelationshipListState';

  static List<ProfileAccountSummary> _$items(RelationshipListState v) =>
      v.items;
  static const Field<RelationshipListState, List<ProfileAccountSummary>>
  _f$items = Field('items', _$items);
  static String? _$cursor(RelationshipListState v) => v.cursor;
  static const Field<RelationshipListState, String> _f$cursor = Field(
    'cursor',
    _$cursor,
    opt: true,
  );
  static Set<String> _$mutatingDids(RelationshipListState v) => v.mutatingDids;
  static const Field<RelationshipListState, Set<String>> _f$mutatingDids =
      Field('mutatingDids', _$mutatingDids, opt: true, def: const {});

  @override
  final MappableFields<RelationshipListState> fields = const {
    #items: _f$items,
    #cursor: _f$cursor,
    #mutatingDids: _f$mutatingDids,
  };

  static RelationshipListState _instantiate(DecodingData data) {
    return RelationshipListState(
      items: data.dec(_f$items),
      cursor: data.dec(_f$cursor),
      mutatingDids: data.dec(_f$mutatingDids),
    );
  }

  @override
  final Function instantiate = _instantiate;
}

mixin RelationshipListStateMappable {
  RelationshipListStateCopyWith<
    RelationshipListState,
    RelationshipListState,
    RelationshipListState
  >
  get copyWith =>
      _RelationshipListStateCopyWithImpl<
        RelationshipListState,
        RelationshipListState
      >(this as RelationshipListState, $identity, $identity);
  @override
  bool operator ==(Object other) {
    return RelationshipListStateMapper.ensureInitialized().equalsValue(
      this as RelationshipListState,
      other,
    );
  }

  @override
  int get hashCode {
    return RelationshipListStateMapper.ensureInitialized().hashValue(
      this as RelationshipListState,
    );
  }
}

extension RelationshipListStateValueCopy<$R, $Out>
    on ObjectCopyWith<$R, RelationshipListState, $Out> {
  RelationshipListStateCopyWith<$R, RelationshipListState, $Out>
  get $asRelationshipListState => $base.as(
    (v, t, t2) => _RelationshipListStateCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class RelationshipListStateCopyWith<
  $R,
  $In extends RelationshipListState,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<
    $R,
    ProfileAccountSummary,
    ProfileAccountSummaryCopyWith<
      $R,
      ProfileAccountSummary,
      ProfileAccountSummary
    >
  >
  get items;
  $R call({
    List<ProfileAccountSummary>? items,
    String? cursor,
    Set<String>? mutatingDids,
  });
  RelationshipListStateCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _RelationshipListStateCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, RelationshipListState, $Out>
    implements RelationshipListStateCopyWith<$R, RelationshipListState, $Out> {
  _RelationshipListStateCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<RelationshipListState> $mapper =
      RelationshipListStateMapper.ensureInitialized();
  @override
  ListCopyWith<
    $R,
    ProfileAccountSummary,
    ProfileAccountSummaryCopyWith<
      $R,
      ProfileAccountSummary,
      ProfileAccountSummary
    >
  >
  get items => ListCopyWith(
    $value.items,
    (v, t) => v.copyWith.$chain(t),
    (v) => call(items: v),
  );
  @override
  $R call({
    List<ProfileAccountSummary>? items,
    Object? cursor = $none,
    Set<String>? mutatingDids,
  }) => $apply(
    FieldCopyWithData({
      if (items != null) #items: items,
      if (cursor != $none) #cursor: cursor,
      if (mutatingDids != null) #mutatingDids: mutatingDids,
    }),
  );
  @override
  RelationshipListState $make(CopyWithData data) => RelationshipListState(
    items: data.get(#items, or: $value.items),
    cursor: data.get(#cursor, or: $value.cursor),
    mutatingDids: data.get(#mutatingDids, or: $value.mutatingDids),
  );

  @override
  RelationshipListStateCopyWith<$R2, RelationshipListState, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _RelationshipListStateCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

