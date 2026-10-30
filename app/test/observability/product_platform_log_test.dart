import 'dart:convert';
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
      final record = jsonDecode(
        (result.stdout as String).split('PRODUCT_PROBE_COMPLETED').first,
      ) as Map<String, dynamic>;
      expect(record['severity'], 'WARNING');
      expect(record['cause'], 'StateError');
      expect(record['appViewRequestId'], 'product-probe');
      expect(record['stack'], hasLength(240));
      expect((record['stack'] as List<dynamic>).last, {'function': 'frame239'});
      expect(
        const LineSplitter()
            .convert(result.stdout as String)
            .every(
              (line) => utf8.encode(line).length <= 800,
            ),
        isTrue,
      );
    },
    timeout: const Timeout(Duration(minutes: 2)),
  );
}
