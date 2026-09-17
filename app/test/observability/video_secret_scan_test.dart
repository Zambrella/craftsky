import 'package:craftsky_app/observability/video_diagnostics.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-012 diagnostic API cannot accept authored or credential data', () {
    expect(
      VideoDiagnosticEvent.new,
      isA<
        VideoDiagnosticEvent Function({
          required VideoOperation operation,
          required VideoOperationOutcome outcome,
          int? byteCount,
          String? requestId,
        })
      >(),
    );
  });
}
