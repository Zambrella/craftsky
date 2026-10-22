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
      StatusMaintenanceMapper.ensureInitialized();
      StatusMinimumVersionsMapper.ensureInitialized();
      StatusMessageMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'ServiceStatusDocument';

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
    def: 2,
  );
  static StatusMaintenance? _$maintenance(ServiceStatusDocument v) =>
      v.maintenance;
  static const Field<ServiceStatusDocument, StatusMaintenance> _f$maintenance =
      Field('maintenance', _$maintenance, opt: true);
  static StatusMinimumVersions? _$minimumVersions(ServiceStatusDocument v) =>
      v.minimumVersions;
  static const Field<ServiceStatusDocument, StatusMinimumVersions>
  _f$minimumVersions = Field('minimumVersions', _$minimumVersions, opt: true);
  static StatusMessage? _$message(ServiceStatusDocument v) => v.message;
  static const Field<ServiceStatusDocument, StatusMessage> _f$message = Field(
    'message',
    _$message,
    opt: true,
  );

  @override
  final MappableFields<ServiceStatusDocument> fields = const {
    #revision: _f$revision,
    #schemaVersion: _f$schemaVersion,
    #maintenance: _f$maintenance,
    #minimumVersions: _f$minimumVersions,
    #message: _f$message,
  };
  @override
  final bool ignoreNull = true;

  static ServiceStatusDocument _instantiate(DecodingData data) {
    return ServiceStatusDocument(
      revision: data.dec(_f$revision),
      schemaVersion: data.dec(_f$schemaVersion),
      maintenance: data.dec(_f$maintenance),
      minimumVersions: data.dec(_f$minimumVersions),
      message: data.dec(_f$message),
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

class StatusMaintenanceMapper extends ClassMapperBase<StatusMaintenance> {
  StatusMaintenanceMapper._();

  static StatusMaintenanceMapper? _instance;
  static StatusMaintenanceMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = StatusMaintenanceMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'StatusMaintenance';

  static String _$title(StatusMaintenance v) => v.title;
  static const Field<StatusMaintenance, String> _f$title = Field(
    'title',
    _$title,
  );
  static String _$description(StatusMaintenance v) => v.description;
  static const Field<StatusMaintenance, String> _f$description = Field(
    'description',
    _$description,
  );
  static DateTime? _$estimatedRecoveryAt(StatusMaintenance v) =>
      v.estimatedRecoveryAt;
  static const Field<StatusMaintenance, DateTime> _f$estimatedRecoveryAt =
      Field('estimatedRecoveryAt', _$estimatedRecoveryAt, opt: true);

  @override
  final MappableFields<StatusMaintenance> fields = const {
    #title: _f$title,
    #description: _f$description,
    #estimatedRecoveryAt: _f$estimatedRecoveryAt,
  };
  @override
  final bool ignoreNull = true;

  static StatusMaintenance _instantiate(DecodingData data) {
    return StatusMaintenance(
      title: data.dec(_f$title),
      description: data.dec(_f$description),
      estimatedRecoveryAt: data.dec(_f$estimatedRecoveryAt),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static StatusMaintenance fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<StatusMaintenance>(map);
  }

  static StatusMaintenance fromJson(String json) {
    return ensureInitialized().decodeJson<StatusMaintenance>(json);
  }
}

mixin StatusMaintenanceMappable {
  String toJson() {
    return StatusMaintenanceMapper.ensureInitialized()
        .encodeJson<StatusMaintenance>(this as StatusMaintenance);
  }

  Map<String, dynamic> toMap() {
    return StatusMaintenanceMapper.ensureInitialized()
        .encodeMap<StatusMaintenance>(this as StatusMaintenance);
  }
}

class StatusMinimumVersionsMapper
    extends ClassMapperBase<StatusMinimumVersions> {
  StatusMinimumVersionsMapper._();

  static StatusMinimumVersionsMapper? _instance;
  static StatusMinimumVersionsMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = StatusMinimumVersionsMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'StatusMinimumVersions';

  static String? _$ios(StatusMinimumVersions v) => v.ios;
  static const Field<StatusMinimumVersions, String> _f$ios = Field(
    'ios',
    _$ios,
    opt: true,
  );
  static String? _$android(StatusMinimumVersions v) => v.android;
  static const Field<StatusMinimumVersions, String> _f$android = Field(
    'android',
    _$android,
    opt: true,
  );
  static String? _$web(StatusMinimumVersions v) => v.web;
  static const Field<StatusMinimumVersions, String> _f$web = Field(
    'web',
    _$web,
    opt: true,
  );

  @override
  final MappableFields<StatusMinimumVersions> fields = const {
    #ios: _f$ios,
    #android: _f$android,
    #web: _f$web,
  };
  @override
  final bool ignoreNull = true;

  static StatusMinimumVersions _instantiate(DecodingData data) {
    return StatusMinimumVersions(
      ios: data.dec(_f$ios),
      android: data.dec(_f$android),
      web: data.dec(_f$web),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static StatusMinimumVersions fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<StatusMinimumVersions>(map);
  }

  static StatusMinimumVersions fromJson(String json) {
    return ensureInitialized().decodeJson<StatusMinimumVersions>(json);
  }
}

mixin StatusMinimumVersionsMappable {
  String toJson() {
    return StatusMinimumVersionsMapper.ensureInitialized()
        .encodeJson<StatusMinimumVersions>(this as StatusMinimumVersions);
  }

  Map<String, dynamic> toMap() {
    return StatusMinimumVersionsMapper.ensureInitialized()
        .encodeMap<StatusMinimumVersions>(this as StatusMinimumVersions);
  }
}

class StatusMessageMapper extends ClassMapperBase<StatusMessage> {
  StatusMessageMapper._();

  static StatusMessageMapper? _instance;
  static StatusMessageMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = StatusMessageMapper._());
      StatusMessageCtaMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'StatusMessage';

  static String _$revision(StatusMessage v) => v.revision;
  static const Field<StatusMessage, String> _f$revision = Field(
    'revision',
    _$revision,
  );
  static String _$minVersion(StatusMessage v) => v.minVersion;
  static const Field<StatusMessage, String> _f$minVersion = Field(
    'minVersion',
    _$minVersion,
  );
  static String _$title(StatusMessage v) => v.title;
  static const Field<StatusMessage, String> _f$title = Field('title', _$title);
  static String _$description(StatusMessage v) => v.description;
  static const Field<StatusMessage, String> _f$description = Field(
    'description',
    _$description,
  );
  static String? _$maxVersion(StatusMessage v) => v.maxVersion;
  static const Field<StatusMessage, String> _f$maxVersion = Field(
    'maxVersion',
    _$maxVersion,
    opt: true,
  );
  static String? _$imageUrl(StatusMessage v) => v.imageUrl;
  static const Field<StatusMessage, String> _f$imageUrl = Field(
    'imageUrl',
    _$imageUrl,
    opt: true,
  );
  static StatusMessageCta? _$cta(StatusMessage v) => v.cta;
  static const Field<StatusMessage, StatusMessageCta> _f$cta = Field(
    'cta',
    _$cta,
    opt: true,
  );

  @override
  final MappableFields<StatusMessage> fields = const {
    #revision: _f$revision,
    #minVersion: _f$minVersion,
    #title: _f$title,
    #description: _f$description,
    #maxVersion: _f$maxVersion,
    #imageUrl: _f$imageUrl,
    #cta: _f$cta,
  };
  @override
  final bool ignoreNull = true;

  static StatusMessage _instantiate(DecodingData data) {
    return StatusMessage(
      revision: data.dec(_f$revision),
      minVersion: data.dec(_f$minVersion),
      title: data.dec(_f$title),
      description: data.dec(_f$description),
      maxVersion: data.dec(_f$maxVersion),
      imageUrl: data.dec(_f$imageUrl),
      cta: data.dec(_f$cta),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static StatusMessage fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<StatusMessage>(map);
  }

  static StatusMessage fromJson(String json) {
    return ensureInitialized().decodeJson<StatusMessage>(json);
  }
}

mixin StatusMessageMappable {
  String toJson() {
    return StatusMessageMapper.ensureInitialized().encodeJson<StatusMessage>(
      this as StatusMessage,
    );
  }

  Map<String, dynamic> toMap() {
    return StatusMessageMapper.ensureInitialized().encodeMap<StatusMessage>(
      this as StatusMessage,
    );
  }
}

class StatusMessageCtaMapper extends ClassMapperBase<StatusMessageCta> {
  StatusMessageCtaMapper._();

  static StatusMessageCtaMapper? _instance;
  static StatusMessageCtaMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = StatusMessageCtaMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'StatusMessageCta';

  static String _$text(StatusMessageCta v) => v.text;
  static const Field<StatusMessageCta, String> _f$text = Field('text', _$text);
  static String? _$url(StatusMessageCta v) => v.url;
  static const Field<StatusMessageCta, String> _f$url = Field(
    'url',
    _$url,
    opt: true,
  );
  static String? _$destination(StatusMessageCta v) => v.destination;
  static const Field<StatusMessageCta, String> _f$destination = Field(
    'destination',
    _$destination,
    opt: true,
  );

  @override
  final MappableFields<StatusMessageCta> fields = const {
    #text: _f$text,
    #url: _f$url,
    #destination: _f$destination,
  };
  @override
  final bool ignoreNull = true;

  static StatusMessageCta _instantiate(DecodingData data) {
    return StatusMessageCta(
      text: data.dec(_f$text),
      url: data.dec(_f$url),
      destination: data.dec(_f$destination),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static StatusMessageCta fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<StatusMessageCta>(map);
  }

  static StatusMessageCta fromJson(String json) {
    return ensureInitialized().decodeJson<StatusMessageCta>(json);
  }
}

mixin StatusMessageCtaMappable {
  String toJson() {
    return StatusMessageCtaMapper.ensureInitialized()
        .encodeJson<StatusMessageCta>(this as StatusMessageCta);
  }

  Map<String, dynamic> toMap() {
    return StatusMessageCtaMapper.ensureInitialized()
        .encodeMap<StatusMessageCta>(this as StatusMessageCta);
  }
}
