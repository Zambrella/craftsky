// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'app_dependencies.dart';

class CraftskyDeviceInfoMapper extends ClassMapperBase<CraftskyDeviceInfo> {
  CraftskyDeviceInfoMapper._();

  static CraftskyDeviceInfoMapper? _instance;
  static CraftskyDeviceInfoMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = CraftskyDeviceInfoMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'CraftskyDeviceInfo';

  static String _$platform(CraftskyDeviceInfo v) => v.platform;
  static const Field<CraftskyDeviceInfo, String> _f$platform = Field(
    'platform',
    _$platform,
  );
  static String _$deviceId(CraftskyDeviceInfo v) => v.deviceId;
  static const Field<CraftskyDeviceInfo, String> _f$deviceId = Field(
    'deviceId',
    _$deviceId,
  );
  static String _$model(CraftskyDeviceInfo v) => v.model;
  static const Field<CraftskyDeviceInfo, String> _f$model = Field(
    'model',
    _$model,
  );
  static String _$brand(CraftskyDeviceInfo v) => v.brand;
  static const Field<CraftskyDeviceInfo, String> _f$brand = Field(
    'brand',
    _$brand,
  );
  static String _$osVersion(CraftskyDeviceInfo v) => v.osVersion;
  static const Field<CraftskyDeviceInfo, String> _f$osVersion = Field(
    'osVersion',
    _$osVersion,
  );

  @override
  final MappableFields<CraftskyDeviceInfo> fields = const {
    #platform: _f$platform,
    #deviceId: _f$deviceId,
    #model: _f$model,
    #brand: _f$brand,
    #osVersion: _f$osVersion,
  };

  static CraftskyDeviceInfo _instantiate(DecodingData data) {
    return CraftskyDeviceInfo(
      platform: data.dec(_f$platform),
      deviceId: data.dec(_f$deviceId),
      model: data.dec(_f$model),
      brand: data.dec(_f$brand),
      osVersion: data.dec(_f$osVersion),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static CraftskyDeviceInfo fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<CraftskyDeviceInfo>(map);
  }

  static CraftskyDeviceInfo fromJson(String json) {
    return ensureInitialized().decodeJson<CraftskyDeviceInfo>(json);
  }
}

mixin CraftskyDeviceInfoMappable {
  String toJson() {
    return CraftskyDeviceInfoMapper.ensureInitialized()
        .encodeJson<CraftskyDeviceInfo>(this as CraftskyDeviceInfo);
  }

  Map<String, dynamic> toMap() {
    return CraftskyDeviceInfoMapper.ensureInitialized()
        .encodeMap<CraftskyDeviceInfo>(this as CraftskyDeviceInfo);
  }

  CraftskyDeviceInfoCopyWith<
    CraftskyDeviceInfo,
    CraftskyDeviceInfo,
    CraftskyDeviceInfo
  >
  get copyWith =>
      _CraftskyDeviceInfoCopyWithImpl<CraftskyDeviceInfo, CraftskyDeviceInfo>(
        this as CraftskyDeviceInfo,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return CraftskyDeviceInfoMapper.ensureInitialized().stringifyValue(
      this as CraftskyDeviceInfo,
    );
  }

  @override
  bool operator ==(Object other) {
    return CraftskyDeviceInfoMapper.ensureInitialized().equalsValue(
      this as CraftskyDeviceInfo,
      other,
    );
  }

  @override
  int get hashCode {
    return CraftskyDeviceInfoMapper.ensureInitialized().hashValue(
      this as CraftskyDeviceInfo,
    );
  }
}

extension CraftskyDeviceInfoValueCopy<$R, $Out>
    on ObjectCopyWith<$R, CraftskyDeviceInfo, $Out> {
  CraftskyDeviceInfoCopyWith<$R, CraftskyDeviceInfo, $Out>
  get $asCraftskyDeviceInfo => $base.as(
    (v, t, t2) => _CraftskyDeviceInfoCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class CraftskyDeviceInfoCopyWith<
  $R,
  $In extends CraftskyDeviceInfo,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({
    String? platform,
    String? deviceId,
    String? model,
    String? brand,
    String? osVersion,
  });
  CraftskyDeviceInfoCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _CraftskyDeviceInfoCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, CraftskyDeviceInfo, $Out>
    implements CraftskyDeviceInfoCopyWith<$R, CraftskyDeviceInfo, $Out> {
  _CraftskyDeviceInfoCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<CraftskyDeviceInfo> $mapper =
      CraftskyDeviceInfoMapper.ensureInitialized();
  @override
  $R call({
    String? platform,
    String? deviceId,
    String? model,
    String? brand,
    String? osVersion,
  }) => $apply(
    FieldCopyWithData({
      if (platform != null) #platform: platform,
      if (deviceId != null) #deviceId: deviceId,
      if (model != null) #model: model,
      if (brand != null) #brand: brand,
      if (osVersion != null) #osVersion: osVersion,
    }),
  );
  @override
  CraftskyDeviceInfo $make(CopyWithData data) => CraftskyDeviceInfo(
    platform: data.get(#platform, or: $value.platform),
    deviceId: data.get(#deviceId, or: $value.deviceId),
    model: data.get(#model, or: $value.model),
    brand: data.get(#brand, or: $value.brand),
    osVersion: data.get(#osVersion, or: $value.osVersion),
  );

  @override
  CraftskyDeviceInfoCopyWith<$R2, CraftskyDeviceInfo, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _CraftskyDeviceInfoCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class AppDependenciesMapper extends ClassMapperBase<AppDependencies> {
  AppDependenciesMapper._();

  static AppDependenciesMapper? _instance;
  static AppDependenciesMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = AppDependenciesMapper._());
      CraftskyDeviceInfoMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'AppDependencies';

  static PackageInfo _$packageInfo(AppDependencies v) => v.packageInfo;
  static const Field<AppDependencies, PackageInfo> _f$packageInfo = Field(
    'packageInfo',
    _$packageInfo,
  );
  static CraftskyDeviceInfo _$deviceInfo(AppDependencies v) => v.deviceInfo;
  static const Field<AppDependencies, CraftskyDeviceInfo> _f$deviceInfo = Field(
    'deviceInfo',
    _$deviceInfo,
  );
  static SharedPreferences _$sharedPreferences(AppDependencies v) =>
      v.sharedPreferences;
  static const Field<AppDependencies, SharedPreferences> _f$sharedPreferences =
      Field('sharedPreferences', _$sharedPreferences);
  static Version _$appVersion(AppDependencies v) => v.appVersion;
  static const Field<AppDependencies, Version> _f$appVersion = Field(
    'appVersion',
    _$appVersion,
  );

  @override
  final MappableFields<AppDependencies> fields = const {
    #packageInfo: _f$packageInfo,
    #deviceInfo: _f$deviceInfo,
    #sharedPreferences: _f$sharedPreferences,
    #appVersion: _f$appVersion,
  };

  static AppDependencies _instantiate(DecodingData data) {
    return AppDependencies(
      packageInfo: data.dec(_f$packageInfo),
      deviceInfo: data.dec(_f$deviceInfo),
      sharedPreferences: data.dec(_f$sharedPreferences),
      appVersion: data.dec(_f$appVersion),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static AppDependencies fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<AppDependencies>(map);
  }

  static AppDependencies fromJson(String json) {
    return ensureInitialized().decodeJson<AppDependencies>(json);
  }
}

mixin AppDependenciesMappable {
  String toJson() {
    return AppDependenciesMapper.ensureInitialized()
        .encodeJson<AppDependencies>(this as AppDependencies);
  }

  Map<String, dynamic> toMap() {
    return AppDependenciesMapper.ensureInitialized().encodeMap<AppDependencies>(
      this as AppDependencies,
    );
  }

  AppDependenciesCopyWith<AppDependencies, AppDependencies, AppDependencies>
  get copyWith =>
      _AppDependenciesCopyWithImpl<AppDependencies, AppDependencies>(
        this as AppDependencies,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return AppDependenciesMapper.ensureInitialized().stringifyValue(
      this as AppDependencies,
    );
  }

  @override
  bool operator ==(Object other) {
    return AppDependenciesMapper.ensureInitialized().equalsValue(
      this as AppDependencies,
      other,
    );
  }

  @override
  int get hashCode {
    return AppDependenciesMapper.ensureInitialized().hashValue(
      this as AppDependencies,
    );
  }
}

extension AppDependenciesValueCopy<$R, $Out>
    on ObjectCopyWith<$R, AppDependencies, $Out> {
  AppDependenciesCopyWith<$R, AppDependencies, $Out> get $asAppDependencies =>
      $base.as((v, t, t2) => _AppDependenciesCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class AppDependenciesCopyWith<$R, $In extends AppDependencies, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  CraftskyDeviceInfoCopyWith<$R, CraftskyDeviceInfo, CraftskyDeviceInfo>
  get deviceInfo;
  $R call({
    PackageInfo? packageInfo,
    CraftskyDeviceInfo? deviceInfo,
    SharedPreferences? sharedPreferences,
    Version? appVersion,
  });
  AppDependenciesCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _AppDependenciesCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, AppDependencies, $Out>
    implements AppDependenciesCopyWith<$R, AppDependencies, $Out> {
  _AppDependenciesCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<AppDependencies> $mapper =
      AppDependenciesMapper.ensureInitialized();
  @override
  CraftskyDeviceInfoCopyWith<$R, CraftskyDeviceInfo, CraftskyDeviceInfo>
  get deviceInfo =>
      $value.deviceInfo.copyWith.$chain((v) => call(deviceInfo: v));
  @override
  $R call({
    PackageInfo? packageInfo,
    CraftskyDeviceInfo? deviceInfo,
    SharedPreferences? sharedPreferences,
    Version? appVersion,
  }) => $apply(
    FieldCopyWithData({
      if (packageInfo != null) #packageInfo: packageInfo,
      if (deviceInfo != null) #deviceInfo: deviceInfo,
      if (sharedPreferences != null) #sharedPreferences: sharedPreferences,
      if (appVersion != null) #appVersion: appVersion,
    }),
  );
  @override
  AppDependencies $make(CopyWithData data) => AppDependencies(
    packageInfo: data.get(#packageInfo, or: $value.packageInfo),
    deviceInfo: data.get(#deviceInfo, or: $value.deviceInfo),
    sharedPreferences: data.get(
      #sharedPreferences,
      or: $value.sharedPreferences,
    ),
    appVersion: data.get(#appVersion, or: $value.appVersion),
  );

  @override
  AppDependenciesCopyWith<$R2, AppDependencies, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _AppDependenciesCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

