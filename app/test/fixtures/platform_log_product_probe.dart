import 'dart:convert';
import 'dart:io';

import 'package:craftsky_app/shared/observability/platform_log.dart';

void main() {
  writePlatformDiagnostic(
    jsonEncode({
      'severity': 'WARNING',
      'operation': 'synthetic.read',
      'cause': 'StateError',
      'appViewRequestId': '80000000-0000-4000-8000-000000000008',
      'stack': [
        for (var i = 0; i < 240; i++) {'function': 'frame$i'},
      ],
    }),
  );
  stdout.writeln('PRODUCT_PROBE_COMPLETED');
}
