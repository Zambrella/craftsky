// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'notifications_state.dart';

class NotificationsStateMapper extends ClassMapperBase<NotificationsState> {
  NotificationsStateMapper._();

  static NotificationsStateMapper? _instance;
  static NotificationsStateMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = NotificationsStateMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'NotificationsState';

  static List<CraftskyNotification> _$items(NotificationsState v) => v.items;
  static const Field<NotificationsState, List<CraftskyNotification>> _f$items =
      Field('items', _$items);
  static int _$renderToken(NotificationsState v) => v.renderToken;
  static const Field<NotificationsState, int> _f$renderToken = Field(
    'renderToken',
    _$renderToken,
  );
  static String? _$cursor(NotificationsState v) => v.cursor;
  static const Field<NotificationsState, String> _f$cursor = Field(
    'cursor',
    _$cursor,
    opt: true,
  );
  static AccountSessionLease? _$owner(NotificationsState v) => v.owner;
  static const Field<NotificationsState, AccountSessionLease> _f$owner = Field(
    'owner',
    _$owner,
    opt: true,
  );

  @override
  final MappableFields<NotificationsState> fields = const {
    #items: _f$items,
    #renderToken: _f$renderToken,
    #cursor: _f$cursor,
    #owner: _f$owner,
  };

  static NotificationsState _instantiate(DecodingData data) {
    return NotificationsState(
      items: data.dec(_f$items),
      renderToken: data.dec(_f$renderToken),
      cursor: data.dec(_f$cursor),
      owner: data.dec(_f$owner),
    );
  }

  @override
  final Function instantiate = _instantiate;
}

mixin NotificationsStateMappable {
  NotificationsStateCopyWith<
    NotificationsState,
    NotificationsState,
    NotificationsState
  >
  get copyWith =>
      _NotificationsStateCopyWithImpl<NotificationsState, NotificationsState>(
        this as NotificationsState,
        $identity,
        $identity,
      );
  @override
  bool operator ==(Object other) {
    return NotificationsStateMapper.ensureInitialized().equalsValue(
      this as NotificationsState,
      other,
    );
  }

  @override
  int get hashCode {
    return NotificationsStateMapper.ensureInitialized().hashValue(
      this as NotificationsState,
    );
  }
}

extension NotificationsStateValueCopy<$R, $Out>
    on ObjectCopyWith<$R, NotificationsState, $Out> {
  NotificationsStateCopyWith<$R, NotificationsState, $Out>
  get $asNotificationsState => $base.as(
    (v, t, t2) => _NotificationsStateCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class NotificationsStateCopyWith<
  $R,
  $In extends NotificationsState,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<
    $R,
    CraftskyNotification,
    ObjectCopyWith<$R, CraftskyNotification, CraftskyNotification>
  >
  get items;
  $R call({
    List<CraftskyNotification>? items,
    int? renderToken,
    String? cursor,
    AccountSessionLease? owner,
  });
  NotificationsStateCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _NotificationsStateCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, NotificationsState, $Out>
    implements NotificationsStateCopyWith<$R, NotificationsState, $Out> {
  _NotificationsStateCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<NotificationsState> $mapper =
      NotificationsStateMapper.ensureInitialized();
  @override
  ListCopyWith<
    $R,
    CraftskyNotification,
    ObjectCopyWith<$R, CraftskyNotification, CraftskyNotification>
  >
  get items => ListCopyWith(
    $value.items,
    (v, t) => ObjectCopyWith(v, $identity, t),
    (v) => call(items: v),
  );
  @override
  $R call({
    List<CraftskyNotification>? items,
    int? renderToken,
    Object? cursor = $none,
    Object? owner = $none,
  }) => $apply(
    FieldCopyWithData({
      if (items != null) #items: items,
      if (renderToken != null) #renderToken: renderToken,
      if (cursor != $none) #cursor: cursor,
      if (owner != $none) #owner: owner,
    }),
  );
  @override
  NotificationsState $make(CopyWithData data) => NotificationsState(
    items: data.get(#items, or: $value.items),
    renderToken: data.get(#renderToken, or: $value.renderToken),
    cursor: data.get(#cursor, or: $value.cursor),
    owner: data.get(#owner, or: $value.owner),
  );

  @override
  NotificationsStateCopyWith<$R2, NotificationsState, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _NotificationsStateCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

