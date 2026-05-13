import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'Flutter feed API/model/provider no longer exposes stale thread usage',
    () {
      final libDir = Directory('lib/feed');
      final dartFiles = libDir
          .listSync(recursive: true)
          .whereType<File>()
          .where((file) => file.path.endsWith('.dart'));

      final offenders = <String>[];
      for (final file in dartFiles) {
        final path = file.path;
        final source = file.readAsStringSync();
        if (path.contains('post_thread_provider') ||
            path.contains('post_thread.mapper') ||
            path.endsWith('post_thread.dart') ||
            source.contains('getThread(') ||
            source.contains('postThreadProvider') ||
            source.contains('PostThreadMapper')) {
          offenders.add(path);
        }
      }

      expect(offenders, isEmpty, reason: offenders.join('\n'));
    },
  );
}
