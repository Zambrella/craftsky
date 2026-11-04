import 'dart:io';
import 'dart:ui' as ui;

import 'package:flutter/rendering.dart';
import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

const ValueKey<String> nativeReferenceKey = ValueKey(
  'native-reference-capture',
);

/// Loads real bundled fonts; widget tests otherwise render the Ahem test font.
Future<void> prepareNativeReference(
  WidgetTester tester, {
  Size size = const Size(402, 874),
}) async {
  tester.view.devicePixelRatio = 1;
  tester.view.physicalSize = size;
  addTearDown(tester.view.resetDevicePixelRatio);
  addTearDown(tester.view.resetPhysicalSize);
  for (final entry in <String, List<String>>{
    'Outfit': [
      'assets/fonts/Outfit-400.ttf',
      'assets/fonts/Outfit-600.ttf',
      'assets/fonts/Outfit-700.ttf',
    ],
    'DMSerifDisplay': ['assets/fonts/DMSerifDisplay-400.ttf'],
    'packages/phosphor_icons/PhosphorRegular': [
      'packages/phosphor_icons/lib/fonts/Phosphor.ttf',
    ],
    'packages/phosphor_icons/PhosphorBold': [
      'packages/phosphor_icons/lib/fonts/Phosphor-Bold.ttf',
    ],
    'packages/phosphor_icons/PhosphorFill': [
      'packages/phosphor_icons/lib/fonts/Phosphor-Fill.ttf',
    ],
  }.entries) {
    final loader = FontLoader(entry.key);
    for (final asset in entry.value) {
      loader.addFont(rootBundle.load(asset));
    }
    await loader.load();
  }
}

class NativeReferenceFrame extends StatelessWidget {
  const new({required this.child, super.key});
  final Widget child;

  @override
  Widget build(BuildContext context) => RepaintBoundary(
    key: nativeReferenceKey,
    child: child,
  );
}

/// Writes a review artifact, never a golden baseline or a production asset.
Future<void> saveNativeReference(
  WidgetTester tester,
  String name, {
  String directory = const String.fromEnvironment('NATIVE_REFERENCE_DIRECTORY'),
}) async {
  if (!RegExp(r'^[a-z0-9_-]+$').hasMatch(name)) {
    throw ArgumentError.value(name, 'name', 'Use a simple artifact name');
  }
  await tester.pump();
  final boundary = tester.renderObject<RenderRepaintBoundary>(
    find.byKey(nativeReferenceKey),
  );
  await tester.runAsync(() async {
    final image = await boundary.toImage(pixelRatio: 2);
    try {
      final bytes = await image.toByteData(format: ui.ImageByteFormat.png);
      if (bytes == null) throw StateError('PNG encoder returned no data');
      final output = directory.isEmpty
          ? '${Directory.systemTemp.path}/craftsky-parity/native'
          : directory;
      await Directory(output).create(recursive: true);
      await File('$output/$name.png').writeAsBytes(
        bytes.buffer.asUint8List(bytes.offsetInBytes, bytes.lengthInBytes),
      );
    } finally {
      image.dispose();
    }
  });
}
