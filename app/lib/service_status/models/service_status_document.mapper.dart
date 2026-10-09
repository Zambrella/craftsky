// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'service_status_document.dart';

class ServiceStatusModeMapper extends EnumMapper<ServiceStatusMode> {
  ServiceStatusModeMapper._();

  static ServiceStatusModeMapper? _instance;
  static ServiceStatusModeMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ServiceStatusModeMapper._());
    }
    return _instance!;
  }

  static ServiceStatusMode fromValue(dynamic value) {
    ensureInitialized();
    return MapperContainer.globals.fromValue(value);
  }

  @override
  ServiceStatusMode decode(dynamic value) {
    switch (value) {
      case r'normal':
        return ServiceStatusMode.normal;
      case r'announcement':
        return ServiceStatusMode.announcement;
      case r'maintenance':
        return ServiceStatusMode.maintenance;
      default:
        throw MapperException.unknownEnumValue(value);
    }
  }

  @override
  dynamic encode(ServiceStatusMode self) {
    switch (self) {
      case ServiceStatusMode.normal:
        return r'normal';
      case ServiceStatusMode.announcement:
        return r'announcement';
      case ServiceStatusMode.maintenance:
        return r'maintenance';
    }
  }
}

extension ServiceStatusModeMapperExtension on ServiceStatusMode {
  String toValue() {
    ServiceStatusModeMapper.ensureInitialized();
    return MapperContainer.globals.toValue<ServiceStatusMode>(this) as String;
  }
}

class ServiceStatusDocumentMapper
    extends ClassMapperBase<ServiceStatusDocument> {
  ServiceStatusDocumentMapper._();

  static ServiceStatusDocumentMapper? _instance;
  static ServiceStatusDocumentMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ServiceStatusDocumentMapper._());
      ServiceStatusModeMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'ServiceStatusDocument';

  static ServiceStatusMode _$mode(ServiceStatusDocument v) => v.mode;
  static const Field<ServiceStatusDocument, ServiceStatusMode> _f$mode = Field(
    'mode',
    _$mode,
  );
  static String _$revision(ServiceStatusDocument v) => v.revision;
  static const Field<ServiceStatusDocument, String> _f$revision = Field(
    'revision',
    _$revision,
  );
  static int _$schemaVersion(ServiceStatusDocument v) => v.schemaVersion;
  static const Field<ServiceStatusDocument, int> _f$schemaVersion = Field(
    'schemaVersion',
    _$schemaVersion,
    opt: true,
    def: 1,
  );
  static String? _$title(ServiceStatusDocument v) => v.title;
  static const Field<ServiceStatusDocument, String> _f$title = Field(
    'title',
    _$title,
    opt: true,
  );
  static String? _$message(ServiceStatusDocument v) => v.message;
  static const Field<ServiceStatusDocument, String> _f$message = Field(
    'message',
    _$message,
    opt: true,
  );
  static DateTime? _$estimatedRecoveryAt(ServiceStatusDocument v) =>
      v.estimatedRecoveryAt;
  static const Field<ServiceStatusDocument, DateTime> _f$estimatedRecoveryAt =
      Field('estimatedRecoveryAt', _$estimatedRecoveryAt, opt: true);

  @override
  final MappableFields<ServiceStatusDocument> fields = const {
    #mode: _f$mode,
    #revision: _f$revision,
    #schemaVersion: _f$schemaVersion,
    #title: _f$title,
    #message: _f$message,
    #estimatedRecoveryAt: _f$estimatedRecoveryAt,
  };
  @override
  final bool ignoreNull = true;

  static ServiceStatusDocument _instantiate(DecodingData data) {
    return ServiceStatusDocument(
      mode: data.dec(_f$mode),
      revision: data.dec(_f$revision),
      schemaVersion: data.dec(_f$schemaVersion),
      title: data.dec(_f$title),
      message: data.dec(_f$message),
      estimatedRecoveryAt: data.dec(_f$estimatedRecoveryAt),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ServiceStatusDocument fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ServiceStatusDocument>(map);
  }

  static ServiceStatusDocument fromJson(String json) {
    return ensureInitialized().decodeJson<ServiceStatusDocument>(json);
  }
}

mixin ServiceStatusDocumentMappable {
  String toJson() {
    return ServiceStatusDocumentMapper.ensureInitialized()
        .encodeJson<ServiceStatusDocument>(this as ServiceStatusDocument);
  }

  Map<String, dynamic> toMap() {
    return ServiceStatusDocumentMapper.ensureInitialized()
        .encodeMap<ServiceStatusDocument>(this as ServiceStatusDocument);
  }
}

