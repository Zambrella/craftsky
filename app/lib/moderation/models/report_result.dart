import 'package:dart_mappable/dart_mappable.dart';

part 'report_result.mapper.dart';

@MappableClass()
class ReportResult with ReportResultMappable {
  const ReportResult({required this.reportId, required this.status});

  final String reportId;
  final String status;
}
