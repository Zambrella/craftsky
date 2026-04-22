// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'login_response.dart';

class LoginResponseMapper extends ClassMapperBase<LoginResponse> {
  LoginResponseMapper._();

  static LoginResponseMapper? _instance;
  static LoginResponseMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = LoginResponseMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'LoginResponse';

  static String _$authUrl(LoginResponse v) => v.authUrl;
  static const Field<LoginResponse, String> _f$authUrl = Field(
    'authUrl',
    _$authUrl,
    key: r'auth_url',
  );

  @override
  final MappableFields<LoginResponse> fields = const {#authUrl: _f$authUrl};

  static LoginResponse _instantiate(DecodingData data) {
    return LoginResponse(authUrl: data.dec(_f$authUrl));
  }

  @override
  final Function instantiate = _instantiate;

  static LoginResponse fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<LoginResponse>(map);
  }

  static LoginResponse fromJson(String json) {
    return ensureInitialized().decodeJson<LoginResponse>(json);
  }
}

mixin LoginResponseMappable {
  String toJson() {
    return LoginResponseMapper.ensureInitialized().encodeJson<LoginResponse>(
      this as LoginResponse,
    );
  }

  Map<String, dynamic> toMap() {
    return LoginResponseMapper.ensureInitialized().encodeMap<LoginResponse>(
      this as LoginResponse,
    );
  }

  LoginResponseCopyWith<LoginResponse, LoginResponse, LoginResponse>
  get copyWith => _LoginResponseCopyWithImpl<LoginResponse, LoginResponse>(
    this as LoginResponse,
    $identity,
    $identity,
  );
  @override
  String toString() {
    return LoginResponseMapper.ensureInitialized().stringifyValue(
      this as LoginResponse,
    );
  }

  @override
  bool operator ==(Object other) {
    return LoginResponseMapper.ensureInitialized().equalsValue(
      this as LoginResponse,
      other,
    );
  }

  @override
  int get hashCode {
    return LoginResponseMapper.ensureInitialized().hashValue(
      this as LoginResponse,
    );
  }
}

extension LoginResponseValueCopy<$R, $Out>
    on ObjectCopyWith<$R, LoginResponse, $Out> {
  LoginResponseCopyWith<$R, LoginResponse, $Out> get $asLoginResponse =>
      $base.as((v, t, t2) => _LoginResponseCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class LoginResponseCopyWith<$R, $In extends LoginResponse, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? authUrl});
  LoginResponseCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _LoginResponseCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, LoginResponse, $Out>
    implements LoginResponseCopyWith<$R, LoginResponse, $Out> {
  _LoginResponseCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<LoginResponse> $mapper =
      LoginResponseMapper.ensureInitialized();
  @override
  $R call({String? authUrl}) =>
      $apply(FieldCopyWithData({if (authUrl != null) #authUrl: authUrl}));
  @override
  LoginResponse $make(CopyWithData data) =>
      LoginResponse(authUrl: data.get(#authUrl, or: $value.authUrl));

  @override
  LoginResponseCopyWith<$R2, LoginResponse, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _LoginResponseCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

