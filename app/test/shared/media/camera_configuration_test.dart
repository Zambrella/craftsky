import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

void main() {
  test('all iOS build configurations declare camera usage', () {
    for (final path in [
      'ios/Runner/Info.plist',
      'ios/Runner/Info-Debug.plist',
    ]) {
      final plist = File(path).readAsStringSync();
      expect(
        plist,
        contains('<key>NSCameraUsageDescription</key>'),
        reason: '$path must explain camera access before image_picker uses it',
      );
    }
  });
}
