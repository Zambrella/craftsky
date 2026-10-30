import 'dart:convert';
import 'dart:io';

import 'package:craftsky_app/shared/observability/platform_log.dart';

void main() {
  writePlatformDiagnostic(
    jsonEncode({
      'severity': 'WARNING',
      'operation': 'synthetic.read',
      'cause': 'StateError',
      'appViewRequestId': 'product-probe',
      'stack': [
        for (var i = 0; i < 240; i++) {'function': 'frame$i'},
      ],
    }),
  );
  stdout.writeln('PRODUCT_PROBE_COMPLETED');
}
