// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'report_result.dart';

class ReportResultMapper extends ClassMapperBase<ReportResult> {
  ReportResultMapper._();

  static ReportResultMapper? _instance;
  static ReportResultMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ReportResultMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'ReportResult';

  static String _$reportId(ReportResult v) => v.reportId;
  static const Field<ReportResult, String> _f$reportId = Field(
    'reportId',
    _$reportId,
  );
  static String _$status(ReportResult v) => v.status;
  static const Field<ReportResult, String> _f$status = Field(
    'status',
    _$status,
  );

  @override
  final MappableFields<ReportResult> fields = const {
    #reportId: _f$reportId,
    #status: _f$status,
  };

  static ReportResult _instantiate(DecodingData data) {
    return ReportResult(
      reportId: data.dec(_f$reportId),
      status: data.dec(_f$status),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ReportResult fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ReportResult>(map);
  }

  static ReportResult fromJson(String json) {
    return ensureInitialized().decodeJson<ReportResult>(json);
  }
}

mixin ReportResultMappable {
  String toJson() {
    return ReportResultMapper.ensureInitialized().encodeJson<ReportResult>(
      this as ReportResult,
    );
  }

  Map<String, dynamic> toMap() {
    return ReportResultMapper.ensureInitialized().encodeMap<ReportResult>(
      this as ReportResult,
    );
  }

  ReportResultCopyWith<ReportResult, ReportResult, ReportResult> get copyWith =>
      _ReportResultCopyWithImpl<ReportResult, ReportResult>(
        this as ReportResult,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return ReportResultMapper.ensureInitialized().stringifyValue(
      this as ReportResult,
    );
  }

  @override
  bool operator ==(Object other) {
    return ReportResultMapper.ensureInitialized().equalsValue(
      this as ReportResult,
      other,
    );
  }

  @override
  int get hashCode {
    return ReportResultMapper.ensureInitialized().hashValue(
      this as ReportResult,
    );
  }
}

extension ReportResultValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ReportResult, $Out> {
  ReportResultCopyWith<$R, ReportResult, $Out> get $asReportResult =>
      $base.as((v, t, t2) => _ReportResultCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class ReportResultCopyWith<$R, $In extends ReportResult, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? reportId, String? status});
  ReportResultCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _ReportResultCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ReportResult, $Out>
    implements ReportResultCopyWith<$R, ReportResult, $Out> {
  _ReportResultCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ReportResult> $mapper =
      ReportResultMapper.ensureInitialized();
  @override
  $R call({String? reportId, String? status}) => $apply(
    FieldCopyWithData({
      if (reportId != null) #reportId: reportId,
      if (status != null) #status: status,
    }),
  );
  @override
  ReportResult $make(CopyWithData data) => ReportResult(
    reportId: data.get(#reportId, or: $value.reportId),
    status: data.get(#status, or: $value.status),
  );

  @override
  ReportResultCopyWith<$R2, ReportResult, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _ReportResultCopyWithImpl<$R2, $Out2>($value, $cast, t);
}
