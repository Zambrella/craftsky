import 'package:craftsky_app/service_status/models/service_status_document.dart';
import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';

export 'service_status_repository_stub.dart'
    if (dart.library.io) 'service_status_repository_native.dart'
    if (dart.library.js_interop) 'service_status_repository_web.dart';

final class StatusRequest {
  const StatusRequest(this.result, this.cancel);
  final Future<ServiceStatusDocument> result;
  final void Function() cancel;
}

/// Anonymous status retrieval, independent of the authenticated API client.
abstract class ServiceStatusRepository {
  const ServiceStatusRepository();
  StatusRequest start();
  void close() {}
}

Uri validateStatusEndpoint(String value, {required bool production}) {
  final uri = Uri.tryParse(value);
  if (uri == null ||
      uri.userInfo.isNotEmpty ||
      uri.hasQuery ||
      uri.hasFragment ||
      (production && value != 'https://status.craftsky.social/app.json') ||
      (uri.scheme != 'https' &&
          !(uri.scheme == 'http' &&
              ['localhost', '127.0.0.1', '::1'].contains(uri.host))) ||
      uri.host.isEmpty) {
    throw ArgumentError('Invalid status endpoint');
  }
  return uri;
}

final class StatusNetworkFailure implements Exception, DiagnosticFailureCause {
  const StatusNetworkFailure(this.diagnosticCause, this.diagnosticStack);
  @override
  final Object diagnosticCause;
  @override
  final StackTrace diagnosticStack;
  @override
  String toString() => 'StatusNetworkFailure';
}
