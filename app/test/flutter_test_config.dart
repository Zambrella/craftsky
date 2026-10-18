import 'dart:async';

import 'package:flutter_image_compress/flutter_image_compress.dart';

import 'support/image_compression_test_platform.dart';

Future<void> testExecutable(FutureOr<void> Function() testMain) async {
  FlutterImageCompressPlatform.instance = TestImageCompressionPlatform();
  await testMain();
}
