import 'dart:io';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'IT-015 default platform adapter emits in product AOT',
    () async {
      final directory = await Directory.systemTemp.createTemp(
        'craftsky-product-log-',
      );
      addTearDown(() => directory.delete(recursive: true));
      final executable = '${directory.path}/probe';
      final compiled = await Process.run('dart', [
        'compile',
        'exe',
        'test/fixtures/platform_log_product_probe.dart',
        '-o',
        executable,
      ]);
      expect(
        compiled.exitCode,
        0,
        reason: '${compiled.stdout}\n${compiled.stderr}',
      );
      final result = await Process.run(executable, []);
      expect(result.exitCode, 0);
      expect(result.stdout, contains('PRODUCT_PROBE_COMPLETED'));
      expect(result.stdout, contains('"severity":"WARNING"'));
      expect(result.stdout, contains('"cause":"StateError"'));
      expect(result.stdout, contains('"appViewRequestId":"product-probe"'));
    },
    timeout: const Timeout(Duration(minutes: 2)),
  );
}
