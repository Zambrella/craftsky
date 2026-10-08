import 'dart:io';

import 'package:craftsky_app/shared/observability/platform_log.dart';

void main() {
  writePlatformDiagnostic(
    '{"severity":"WARNING","operation":"synthetic.read",'
    '"cause":"StateError","appViewRequestId":"product-probe"}',
  );
  stdout.writeln('PRODUCT_PROBE_COMPLETED');
}
