// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'billing_state.dart';

class BillingSubscriptionMapper extends ClassMapperBase<BillingSubscription> {
  BillingSubscriptionMapper._();

  static BillingSubscriptionMapper? _instance;
  static BillingSubscriptionMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = BillingSubscriptionMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'BillingSubscription';

  static String _$id(BillingSubscription v) => v.id;
  static const Field<BillingSubscription, String> _f$id = Field('id', _$id);
  static String _$productId(BillingSubscription v) => v.productId;
  static const Field<BillingSubscription, String> _f$productId = Field(
    'productId',
    _$productId,
  );
  static String _$store(BillingSubscription v) => v.store;
  static const Field<BillingSubscription, String> _f$store = Field(
    'store',
    _$store,
  );
  static String _$status(BillingSubscription v) => v.status;
  static const Field<BillingSubscription, String> _f$status = Field(
    'status',
    _$status,
  );
  static bool _$givesAccess(BillingSubscription v) => v.givesAccess;
  static const Field<BillingSubscription, bool> _f$givesAccess = Field(
    'givesAccess',
    _$givesAccess,
  );
  static bool _$pendingPayment(BillingSubscription v) => v.pendingPayment;
  static const Field<BillingSubscription, bool> _f$pendingPayment = Field(
    'pendingPayment',
    _$pendingPayment,
  );
  static String _$anomaly(BillingSubscription v) => v.anomaly;
  static const Field<BillingSubscription, String> _f$anomaly = Field(
    'anomaly',
    _$anomaly,
  );
  static String? _$autoRenewalStatus(BillingSubscription v) =>
      v.autoRenewalStatus;
  static const Field<BillingSubscription, String> _f$autoRenewalStatus = Field(
    'autoRenewalStatus',
    _$autoRenewalStatus,
    opt: true,
  );
  static DateTime? _$currentPeriodStartsAt(BillingSubscription v) =>
      v.currentPeriodStartsAt;
  static const Field<BillingSubscription, DateTime> _f$currentPeriodStartsAt =
      Field('currentPeriodStartsAt', _$currentPeriodStartsAt, opt: true);
  static DateTime? _$currentPeriodEndsAt(BillingSubscription v) =>
      v.currentPeriodEndsAt;
  static const Field<BillingSubscription, DateTime> _f$currentPeriodEndsAt =
      Field('currentPeriodEndsAt', _$currentPeriodEndsAt, opt: true);
  static DateTime? _$endsAt(BillingSubscription v) => v.endsAt;
  static const Field<BillingSubscription, DateTime> _f$endsAt = Field(
    'endsAt',
    _$endsAt,
    opt: true,
  );

  @override
  final MappableFields<BillingSubscription> fields = const {
    #id: _f$id,
    #productId: _f$productId,
    #store: _f$store,
    #status: _f$status,
    #givesAccess: _f$givesAccess,
    #pendingPayment: _f$pendingPayment,
    #anomaly: _f$anomaly,
    #autoRenewalStatus: _f$autoRenewalStatus,
    #currentPeriodStartsAt: _f$currentPeriodStartsAt,
    #currentPeriodEndsAt: _f$currentPeriodEndsAt,
    #endsAt: _f$endsAt,
  };
  @override
  final bool ignoreNull = true;

  static BillingSubscription _instantiate(DecodingData data) {
    return BillingSubscription(
      id: data.dec(_f$id),
      productId: data.dec(_f$productId),
      store: data.dec(_f$store),
      status: data.dec(_f$status),
      givesAccess: data.dec(_f$givesAccess),
      pendingPayment: data.dec(_f$pendingPayment),
      anomaly: data.dec(_f$anomaly),
      autoRenewalStatus: data.dec(_f$autoRenewalStatus),
      currentPeriodStartsAt: data.dec(_f$currentPeriodStartsAt),
      currentPeriodEndsAt: data.dec(_f$currentPeriodEndsAt),
      endsAt: data.dec(_f$endsAt),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static BillingSubscription fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<BillingSubscription>(map);
  }

  static BillingSubscription fromJson(String json) {
    return ensureInitialized().decodeJson<BillingSubscription>(json);
  }
}

mixin BillingSubscriptionMappable {
  BillingSubscriptionCopyWith<
    BillingSubscription,
    BillingSubscription,
    BillingSubscription
  >
  get copyWith =>
      _BillingSubscriptionCopyWithImpl<
        BillingSubscription,
        BillingSubscription
      >(this as BillingSubscription, $identity, $identity);
  @override
  bool operator ==(Object other) {
    return BillingSubscriptionMapper.ensureInitialized().equalsValue(
      this as BillingSubscription,
      other,
    );
  }

  @override
  int get hashCode {
    return BillingSubscriptionMapper.ensureInitialized().hashValue(
      this as BillingSubscription,
    );
  }
}

extension BillingSubscriptionValueCopy<$R, $Out>
    on ObjectCopyWith<$R, BillingSubscription, $Out> {
  BillingSubscriptionCopyWith<$R, BillingSubscription, $Out>
  get $asBillingSubscription => $base.as(
    (v, t, t2) => _BillingSubscriptionCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class BillingSubscriptionCopyWith<
  $R,
  $In extends BillingSubscription,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({
    String? id,
    String? productId,
    String? store,
    String? status,
    bool? givesAccess,
    bool? pendingPayment,
    String? anomaly,
    String? autoRenewalStatus,
    DateTime? currentPeriodStartsAt,
    DateTime? currentPeriodEndsAt,
    DateTime? endsAt,
  });
  BillingSubscriptionCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _BillingSubscriptionCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, BillingSubscription, $Out>
    implements BillingSubscriptionCopyWith<$R, BillingSubscription, $Out> {
  _BillingSubscriptionCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<BillingSubscription> $mapper =
      BillingSubscriptionMapper.ensureInitialized();
  @override
  $R call({
    String? id,
    String? productId,
    String? store,
    String? status,
    bool? givesAccess,
    bool? pendingPayment,
    String? anomaly,
    Object? autoRenewalStatus = $none,
    Object? currentPeriodStartsAt = $none,
    Object? currentPeriodEndsAt = $none,
    Object? endsAt = $none,
  }) => $apply(
    FieldCopyWithData({
      if (id != null) #id: id,
      if (productId != null) #productId: productId,
      if (store != null) #store: store,
      if (status != null) #status: status,
      if (givesAccess != null) #givesAccess: givesAccess,
      if (pendingPayment != null) #pendingPayment: pendingPayment,
      if (anomaly != null) #anomaly: anomaly,
      if (autoRenewalStatus != $none) #autoRenewalStatus: autoRenewalStatus,
      if (currentPeriodStartsAt != $none)
        #currentPeriodStartsAt: currentPeriodStartsAt,
      if (currentPeriodEndsAt != $none)
        #currentPeriodEndsAt: currentPeriodEndsAt,
      if (endsAt != $none) #endsAt: endsAt,
    }),
  );
  @override
  BillingSubscription $make(CopyWithData data) => BillingSubscription(
    id: data.get(#id, or: $value.id),
    productId: data.get(#productId, or: $value.productId),
    store: data.get(#store, or: $value.store),
    status: data.get(#status, or: $value.status),
    givesAccess: data.get(#givesAccess, or: $value.givesAccess),
    pendingPayment: data.get(#pendingPayment, or: $value.pendingPayment),
    anomaly: data.get(#anomaly, or: $value.anomaly),
    autoRenewalStatus: data.get(
      #autoRenewalStatus,
      or: $value.autoRenewalStatus,
    ),
    currentPeriodStartsAt: data.get(
      #currentPeriodStartsAt,
      or: $value.currentPeriodStartsAt,
    ),
    currentPeriodEndsAt: data.get(
      #currentPeriodEndsAt,
      or: $value.currentPeriodEndsAt,
    ),
    endsAt: data.get(#endsAt, or: $value.endsAt),
  );

  @override
  BillingSubscriptionCopyWith<$R2, BillingSubscription, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _BillingSubscriptionCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class BillingLicenseMapper extends ClassMapperBase<BillingLicense> {
  BillingLicenseMapper._();

  static BillingLicenseMapper? _instance;
  static BillingLicenseMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = BillingLicenseMapper._());
      MapperContainer.globals.useAll([DidMapper()]);
      SubscriptionTierMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'BillingLicense';

  static String _$id(BillingLicense v) => v.id;
  static const Field<BillingLicense, String> _f$id = Field('id', _$id);
  static String _$subscriptionId(BillingLicense v) => v.subscriptionId;
  static const Field<BillingLicense, String> _f$subscriptionId = Field(
    'subscriptionId',
    _$subscriptionId,
  );
  static SubscriptionTier _$tier(BillingLicense v) => v.tier;
  static const Field<BillingLicense, SubscriptionTier> _f$tier = Field(
    'tier',
    _$tier,
  );
  static bool _$assignable(BillingLicense v) => v.assignable;
  static const Field<BillingLicense, bool> _f$assignable = Field(
    'assignable',
    _$assignable,
  );
  static String _$anomaly(BillingLicense v) => v.anomaly;
  static const Field<BillingLicense, String> _f$anomaly = Field(
    'anomaly',
    _$anomaly,
  );
  static Did? _$assignedDid(BillingLicense v) => v.assignedDid;
  static const Field<BillingLicense, Did> _f$assignedDid = Field(
    'assignedDid',
    _$assignedDid,
    opt: true,
  );
  static DateTime? _$assignedAt(BillingLicense v) => v.assignedAt;
  static const Field<BillingLicense, DateTime> _f$assignedAt = Field(
    'assignedAt',
    _$assignedAt,
    opt: true,
  );

  @override
  final MappableFields<BillingLicense> fields = const {
    #id: _f$id,
    #subscriptionId: _f$subscriptionId,
    #tier: _f$tier,
    #assignable: _f$assignable,
    #anomaly: _f$anomaly,
    #assignedDid: _f$assignedDid,
    #assignedAt: _f$assignedAt,
  };
  @override
  final bool ignoreNull = true;

  static BillingLicense _instantiate(DecodingData data) {
    return BillingLicense(
      id: data.dec(_f$id),
      subscriptionId: data.dec(_f$subscriptionId),
      tier: data.dec(_f$tier),
      assignable: data.dec(_f$assignable),
      anomaly: data.dec(_f$anomaly),
      assignedDid: data.dec(_f$assignedDid),
      assignedAt: data.dec(_f$assignedAt),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static BillingLicense fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<BillingLicense>(map);
  }

  static BillingLicense fromJson(String json) {
    return ensureInitialized().decodeJson<BillingLicense>(json);
  }
}

mixin BillingLicenseMappable {
  BillingLicenseCopyWith<BillingLicense, BillingLicense, BillingLicense>
  get copyWith => _BillingLicenseCopyWithImpl<BillingLicense, BillingLicense>(
    this as BillingLicense,
    $identity,
    $identity,
  );
  @override
  bool operator ==(Object other) {
    return BillingLicenseMapper.ensureInitialized().equalsValue(
      this as BillingLicense,
      other,
    );
  }

  @override
  int get hashCode {
    return BillingLicenseMapper.ensureInitialized().hashValue(
      this as BillingLicense,
    );
  }
}

extension BillingLicenseValueCopy<$R, $Out>
    on ObjectCopyWith<$R, BillingLicense, $Out> {
  BillingLicenseCopyWith<$R, BillingLicense, $Out> get $asBillingLicense =>
      $base.as((v, t, t2) => _BillingLicenseCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class BillingLicenseCopyWith<$R, $In extends BillingLicense, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({
    String? id,
    String? subscriptionId,
    SubscriptionTier? tier,
    bool? assignable,
    String? anomaly,
    Did? assignedDid,
    DateTime? assignedAt,
  });
  BillingLicenseCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _BillingLicenseCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, BillingLicense, $Out>
    implements BillingLicenseCopyWith<$R, BillingLicense, $Out> {
  _BillingLicenseCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<BillingLicense> $mapper =
      BillingLicenseMapper.ensureInitialized();
  @override
  $R call({
    String? id,
    String? subscriptionId,
    SubscriptionTier? tier,
    bool? assignable,
    String? anomaly,
    Object? assignedDid = $none,
    Object? assignedAt = $none,
  }) => $apply(
    FieldCopyWithData({
      if (id != null) #id: id,
      if (subscriptionId != null) #subscriptionId: subscriptionId,
      if (tier != null) #tier: tier,
      if (assignable != null) #assignable: assignable,
      if (anomaly != null) #anomaly: anomaly,
      if (assignedDid != $none) #assignedDid: assignedDid,
      if (assignedAt != $none) #assignedAt: assignedAt,
    }),
  );
  @override
  BillingLicense $make(CopyWithData data) => BillingLicense(
    id: data.get(#id, or: $value.id),
    subscriptionId: data.get(#subscriptionId, or: $value.subscriptionId),
    tier: data.get(#tier, or: $value.tier),
    assignable: data.get(#assignable, or: $value.assignable),
    anomaly: data.get(#anomaly, or: $value.anomaly),
    assignedDid: data.get(#assignedDid, or: $value.assignedDid),
    assignedAt: data.get(#assignedAt, or: $value.assignedAt),
  );

  @override
  BillingLicenseCopyWith<$R2, BillingLicense, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _BillingLicenseCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class BillingStateMapper extends ClassMapperBase<BillingState> {
  BillingStateMapper._();

  static BillingStateMapper? _instance;
  static BillingStateMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = BillingStateMapper._());
      BillingSubscriptionMapper.ensureInitialized();
      BillingLicenseMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'BillingState';

  static String _$billingAccountId(BillingState v) => v.billingAccountId;
  static const Field<BillingState, String> _f$billingAccountId = Field(
    'billingAccountId',
    _$billingAccountId,
  );
  static String _$revenueCatAppUserId(BillingState v) => v.revenueCatAppUserId;
  static const Field<BillingState, String> _f$revenueCatAppUserId = Field(
    'revenueCatAppUserId',
    _$revenueCatAppUserId,
  );
  static int _$requestedGeneration(BillingState v) => v.requestedGeneration;
  static const Field<BillingState, int> _f$requestedGeneration = Field(
    'requestedGeneration',
    _$requestedGeneration,
  );
  static int _$reconciledGeneration(BillingState v) => v.reconciledGeneration;
  static const Field<BillingState, int> _f$reconciledGeneration = Field(
    'reconciledGeneration',
    _$reconciledGeneration,
  );
  static bool _$reconciliationStale(BillingState v) => v.reconciliationStale;
  static const Field<BillingState, bool> _f$reconciliationStale = Field(
    'reconciliationStale',
    _$reconciliationStale,
  );
  static List<BillingSubscription> _$subscriptions(BillingState v) =>
      v.subscriptions;
  static const Field<BillingState, List<BillingSubscription>> _f$subscriptions =
      Field('subscriptions', _$subscriptions);
  static List<BillingLicense> _$licenses(BillingState v) => v.licenses;
  static const Field<BillingState, List<BillingLicense>> _f$licenses = Field(
    'licenses',
    _$licenses,
  );
  static DateTime? _$reconciliationRequestedAt(BillingState v) =>
      v.reconciliationRequestedAt;
  static const Field<BillingState, DateTime> _f$reconciliationRequestedAt =
      Field(
        'reconciliationRequestedAt',
        _$reconciliationRequestedAt,
        opt: true,
      );
  static DateTime? _$reconciledAt(BillingState v) => v.reconciledAt;
  static const Field<BillingState, DateTime> _f$reconciledAt = Field(
    'reconciledAt',
    _$reconciledAt,
    opt: true,
  );

  @override
  final MappableFields<BillingState> fields = const {
    #billingAccountId: _f$billingAccountId,
    #revenueCatAppUserId: _f$revenueCatAppUserId,
    #requestedGeneration: _f$requestedGeneration,
    #reconciledGeneration: _f$reconciledGeneration,
    #reconciliationStale: _f$reconciliationStale,
    #subscriptions: _f$subscriptions,
    #licenses: _f$licenses,
    #reconciliationRequestedAt: _f$reconciliationRequestedAt,
    #reconciledAt: _f$reconciledAt,
  };
  @override
  final bool ignoreNull = true;

  static BillingState _instantiate(DecodingData data) {
    return BillingState(
      billingAccountId: data.dec(_f$billingAccountId),
      revenueCatAppUserId: data.dec(_f$revenueCatAppUserId),
      requestedGeneration: data.dec(_f$requestedGeneration),
      reconciledGeneration: data.dec(_f$reconciledGeneration),
      reconciliationStale: data.dec(_f$reconciliationStale),
      subscriptions: data.dec(_f$subscriptions),
      licenses: data.dec(_f$licenses),
      reconciliationRequestedAt: data.dec(_f$reconciliationRequestedAt),
      reconciledAt: data.dec(_f$reconciledAt),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static BillingState fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<BillingState>(map);
  }

  static BillingState fromJson(String json) {
    return ensureInitialized().decodeJson<BillingState>(json);
  }
}

mixin BillingStateMappable {
  BillingStateCopyWith<BillingState, BillingState, BillingState> get copyWith =>
      _BillingStateCopyWithImpl<BillingState, BillingState>(
        this as BillingState,
        $identity,
        $identity,
      );
  @override
  bool operator ==(Object other) {
    return BillingStateMapper.ensureInitialized().equalsValue(
      this as BillingState,
      other,
    );
  }

  @override
  int get hashCode {
    return BillingStateMapper.ensureInitialized().hashValue(
      this as BillingState,
    );
  }
}

extension BillingStateValueCopy<$R, $Out>
    on ObjectCopyWith<$R, BillingState, $Out> {
  BillingStateCopyWith<$R, BillingState, $Out> get $asBillingState =>
      $base.as((v, t, t2) => _BillingStateCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class BillingStateCopyWith<$R, $In extends BillingState, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<
    $R,
    BillingSubscription,
    BillingSubscriptionCopyWith<$R, BillingSubscription, BillingSubscription>
  >
  get subscriptions;
  ListCopyWith<
    $R,
    BillingLicense,
    BillingLicenseCopyWith<$R, BillingLicense, BillingLicense>
  >
  get licenses;
  $R call({
    String? billingAccountId,
    String? revenueCatAppUserId,
    int? requestedGeneration,
    int? reconciledGeneration,
    bool? reconciliationStale,
    List<BillingSubscription>? subscriptions,
    List<BillingLicense>? licenses,
    DateTime? reconciliationRequestedAt,
    DateTime? reconciledAt,
  });
  BillingStateCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _BillingStateCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, BillingState, $Out>
    implements BillingStateCopyWith<$R, BillingState, $Out> {
  _BillingStateCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<BillingState> $mapper =
      BillingStateMapper.ensureInitialized();
  @override
  ListCopyWith<
    $R,
    BillingSubscription,
    BillingSubscriptionCopyWith<$R, BillingSubscription, BillingSubscription>
  >
  get subscriptions => ListCopyWith(
    $value.subscriptions,
    (v, t) => v.copyWith.$chain(t),
    (v) => call(subscriptions: v),
  );
  @override
  ListCopyWith<
    $R,
    BillingLicense,
    BillingLicenseCopyWith<$R, BillingLicense, BillingLicense>
  >
  get licenses => ListCopyWith(
    $value.licenses,
    (v, t) => v.copyWith.$chain(t),
    (v) => call(licenses: v),
  );
  @override
  $R call({
    String? billingAccountId,
    String? revenueCatAppUserId,
    int? requestedGeneration,
    int? reconciledGeneration,
    bool? reconciliationStale,
    List<BillingSubscription>? subscriptions,
    List<BillingLicense>? licenses,
    Object? reconciliationRequestedAt = $none,
    Object? reconciledAt = $none,
  }) => $apply(
    FieldCopyWithData({
      if (billingAccountId != null) #billingAccountId: billingAccountId,
      if (revenueCatAppUserId != null)
        #revenueCatAppUserId: revenueCatAppUserId,
      if (requestedGeneration != null)
        #requestedGeneration: requestedGeneration,
      if (reconciledGeneration != null)
        #reconciledGeneration: reconciledGeneration,
      if (reconciliationStale != null)
        #reconciliationStale: reconciliationStale,
      if (subscriptions != null) #subscriptions: subscriptions,
      if (licenses != null) #licenses: licenses,
      if (reconciliationRequestedAt != $none)
        #reconciliationRequestedAt: reconciliationRequestedAt,
      if (reconciledAt != $none) #reconciledAt: reconciledAt,
    }),
  );
  @override
  BillingState $make(CopyWithData data) => BillingState(
    billingAccountId: data.get(#billingAccountId, or: $value.billingAccountId),
    revenueCatAppUserId: data.get(
      #revenueCatAppUserId,
      or: $value.revenueCatAppUserId,
    ),
    requestedGeneration: data.get(
      #requestedGeneration,
      or: $value.requestedGeneration,
    ),
    reconciledGeneration: data.get(
      #reconciledGeneration,
      or: $value.reconciledGeneration,
    ),
    reconciliationStale: data.get(
      #reconciliationStale,
      or: $value.reconciliationStale,
    ),
    subscriptions: data.get(#subscriptions, or: $value.subscriptions),
    licenses: data.get(#licenses, or: $value.licenses),
    reconciliationRequestedAt: data.get(
      #reconciliationRequestedAt,
      or: $value.reconciliationRequestedAt,
    ),
    reconciledAt: data.get(#reconciledAt, or: $value.reconciledAt),
  );

  @override
  BillingStateCopyWith<$R2, BillingState, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _BillingStateCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class BillingAssignmentMapper extends ClassMapperBase<BillingAssignment> {
  BillingAssignmentMapper._();

  static BillingAssignmentMapper? _instance;
  static BillingAssignmentMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = BillingAssignmentMapper._());
      MapperContainer.globals.useAll([DidMapper()]);
    }
    return _instance!;
  }

  @override
  final String id = 'BillingAssignment';

  static String _$licenseId(BillingAssignment v) => v.licenseId;
  static const Field<BillingAssignment, String> _f$licenseId = Field(
    'licenseId',
    _$licenseId,
  );
  static Did _$targetDid(BillingAssignment v) => v.targetDid;
  static const Field<BillingAssignment, Did> _f$targetDid = Field(
    'targetDid',
    _$targetDid,
  );
  static DateTime _$assignedAt(BillingAssignment v) => v.assignedAt;
  static const Field<BillingAssignment, DateTime> _f$assignedAt = Field(
    'assignedAt',
    _$assignedAt,
  );

  @override
  final MappableFields<BillingAssignment> fields = const {
    #licenseId: _f$licenseId,
    #targetDid: _f$targetDid,
    #assignedAt: _f$assignedAt,
  };

  static BillingAssignment _instantiate(DecodingData data) {
    return BillingAssignment(
      licenseId: data.dec(_f$licenseId),
      targetDid: data.dec(_f$targetDid),
      assignedAt: data.dec(_f$assignedAt),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static BillingAssignment fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<BillingAssignment>(map);
  }

  static BillingAssignment fromJson(String json) {
    return ensureInitialized().decodeJson<BillingAssignment>(json);
  }
}

mixin BillingAssignmentMappable {
  BillingAssignmentCopyWith<
    BillingAssignment,
    BillingAssignment,
    BillingAssignment
  >
  get copyWith =>
      _BillingAssignmentCopyWithImpl<BillingAssignment, BillingAssignment>(
        this as BillingAssignment,
        $identity,
        $identity,
      );
  @override
  bool operator ==(Object other) {
    return BillingAssignmentMapper.ensureInitialized().equalsValue(
      this as BillingAssignment,
      other,
    );
  }

  @override
  int get hashCode {
    return BillingAssignmentMapper.ensureInitialized().hashValue(
      this as BillingAssignment,
    );
  }
}

extension BillingAssignmentValueCopy<$R, $Out>
    on ObjectCopyWith<$R, BillingAssignment, $Out> {
  BillingAssignmentCopyWith<$R, BillingAssignment, $Out>
  get $asBillingAssignment => $base.as(
    (v, t, t2) => _BillingAssignmentCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class BillingAssignmentCopyWith<
  $R,
  $In extends BillingAssignment,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? licenseId, Did? targetDid, DateTime? assignedAt});
  BillingAssignmentCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _BillingAssignmentCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, BillingAssignment, $Out>
    implements BillingAssignmentCopyWith<$R, BillingAssignment, $Out> {
  _BillingAssignmentCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<BillingAssignment> $mapper =
      BillingAssignmentMapper.ensureInitialized();
  @override
  $R call({String? licenseId, Did? targetDid, DateTime? assignedAt}) => $apply(
    FieldCopyWithData({
      if (licenseId != null) #licenseId: licenseId,
      if (targetDid != null) #targetDid: targetDid,
      if (assignedAt != null) #assignedAt: assignedAt,
    }),
  );
  @override
  BillingAssignment $make(CopyWithData data) => BillingAssignment(
    licenseId: data.get(#licenseId, or: $value.licenseId),
    targetDid: data.get(#targetDid, or: $value.targetDid),
    assignedAt: data.get(#assignedAt, or: $value.assignedAt),
  );

  @override
  BillingAssignmentCopyWith<$R2, BillingAssignment, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _BillingAssignmentCopyWithImpl<$R2, $Out2>($value, $cast, t);
}
