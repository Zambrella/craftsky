class ReportSubmission {
  const ReportSubmission({required this.reasonType, this.details});

  final String reasonType;
  final String? details;

  Map<String, dynamic> toMap() => {
    'reasonType': reasonType,
    'details': ?details,
  };
}
