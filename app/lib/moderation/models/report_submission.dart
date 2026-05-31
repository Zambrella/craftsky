import 'package:dart_mappable/dart_mappable.dart';

part 'report_submission.mapper.dart';

@MappableClass(ignoreNull: true)
class ReportSubmission with ReportSubmissionMappable {
  const ReportSubmission({required this.reasonType, this.details});

  final String reasonType;
  final String? details;
}
