class ReportResult {
  const ReportResult({required this.reportId, required this.status});

  final String reportId;
  final String status;

  factory ReportResult.fromMap(Map<String, dynamic> map) => ReportResult(
    reportId: map['reportId'] as String,
    status: map['status'] as String,
  );
}
