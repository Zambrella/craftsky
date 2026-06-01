// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'report_submission.dart';

class ReportSubmissionMapper extends ClassMapperBase<ReportSubmission> {
  ReportSubmissionMapper._();

  static ReportSubmissionMapper? _instance;
  static ReportSubmissionMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ReportSubmissionMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'ReportSubmission';

  static String _$reasonType(ReportSubmission v) => v.reasonType;
  static const Field<ReportSubmission, String> _f$reasonType = Field(
    'reasonType',
    _$reasonType,
  );
  static String? _$details(ReportSubmission v) => v.details;
  static const Field<ReportSubmission, String> _f$details = Field(
    'details',
    _$details,
    opt: true,
  );

  @override
  final MappableFields<ReportSubmission> fields = const {
    #reasonType: _f$reasonType,
    #details: _f$details,
  };
  @override
  final bool ignoreNull = true;

  static ReportSubmission _instantiate(DecodingData data) {
    return ReportSubmission(
      reasonType: data.dec(_f$reasonType),
      details: data.dec(_f$details),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ReportSubmission fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ReportSubmission>(map);
  }

  static ReportSubmission fromJson(String json) {
    return ensureInitialized().decodeJson<ReportSubmission>(json);
  }
}

mixin ReportSubmissionMappable {
  String toJson() {
    return ReportSubmissionMapper.ensureInitialized()
        .encodeJson<ReportSubmission>(this as ReportSubmission);
  }

  Map<String, dynamic> toMap() {
    return ReportSubmissionMapper.ensureInitialized()
        .encodeMap<ReportSubmission>(this as ReportSubmission);
  }

  ReportSubmissionCopyWith<ReportSubmission, ReportSubmission, ReportSubmission>
  get copyWith =>
      _ReportSubmissionCopyWithImpl<ReportSubmission, ReportSubmission>(
        this as ReportSubmission,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return ReportSubmissionMapper.ensureInitialized().stringifyValue(
      this as ReportSubmission,
    );
  }

  @override
  bool operator ==(Object other) {
    return ReportSubmissionMapper.ensureInitialized().equalsValue(
      this as ReportSubmission,
      other,
    );
  }

  @override
  int get hashCode {
    return ReportSubmissionMapper.ensureInitialized().hashValue(
      this as ReportSubmission,
    );
  }
}

extension ReportSubmissionValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ReportSubmission, $Out> {
  ReportSubmissionCopyWith<$R, ReportSubmission, $Out>
  get $asReportSubmission =>
      $base.as((v, t, t2) => _ReportSubmissionCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class ReportSubmissionCopyWith<$R, $In extends ReportSubmission, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? reasonType, String? details});
  ReportSubmissionCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ReportSubmissionCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ReportSubmission, $Out>
    implements ReportSubmissionCopyWith<$R, ReportSubmission, $Out> {
  _ReportSubmissionCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ReportSubmission> $mapper =
      ReportSubmissionMapper.ensureInitialized();
  @override
  $R call({String? reasonType, Object? details = $none}) => $apply(
    FieldCopyWithData({
      if (reasonType != null) #reasonType: reasonType,
      if (details != $none) #details: details,
    }),
  );
  @override
  ReportSubmission $make(CopyWithData data) => ReportSubmission(
    reasonType: data.get(#reasonType, or: $value.reasonType),
    details: data.get(#details, or: $value.details),
  );

  @override
  ReportSubmissionCopyWith<$R2, ReportSubmission, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _ReportSubmissionCopyWithImpl<$R2, $Out2>($value, $cast, t);
}
