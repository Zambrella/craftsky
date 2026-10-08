import 'package:craftsky_app/shared/observability/error_reporter.dart';

final class RecordingErrorReporter implements ErrorReporter {
  final errors = <Object>[];
  final contexts = <ReportContext>[];
  @override
  bool get enabled => true;
  @override
  Future<String?> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) async {
    errors.add(error);
    contexts.add(context);
    return null;
  }
}
