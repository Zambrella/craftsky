import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';

/// Preserve the same content-free failure at each submission/media boundary.
void requireComposerOwnership(bool Function() isCurrent) {
  if (!isCurrent()) {
    throw DiagnosticStateError('submission ownership changed');
  }
}
