import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/errors/app_error.dart';
import 'package:craftsky_app/shared/errors/app_error_mapper.dart';
import 'package:craftsky_app/shared/errors/reportability_classifier.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('ReportabilityClassifier', () {
    test('does not report expected states by default', () {
      for (final error in [
        const ApiBadRequest('handle_required'),
        const ApiUnauthorized(),
        const ApiCanceled(),
        const ApiNetworkError('offline'),
        const AppError(AppErrorKind.contentUnavailable),
      ]) {
        final mapped = error is AppError ? error : AppErrorMapper.map(error);
        expect(
          ReportabilityClassifier.shouldReport(mapped),
          isFalse,
          reason: error.toString(),
        );
      }
    });

    test('reports app/backend defects and degraded states', () {
      for (final error in [
        const ApiServerError('http_500'),
        const FormatException('bad json'),
        StateError('secure storage failed'),
      ]) {
        final mapped = AppErrorMapper.map(
          error,
          source: error is StateError
              ? AppErrorSource.storage
              : AppErrorSource.backgroundLoad,
        );
        expect(
          ReportabilityClassifier.shouldReport(mapped),
          isTrue,
          reason: error.toString(),
        );
      }
    });
  });
}
