import 'package:craftsky_app/shared/errors/app_error.dart';

final class ReportabilityClassifier {
  const ReportabilityClassifier._();

  static bool shouldReport(AppError error) => error.reportable;
}
