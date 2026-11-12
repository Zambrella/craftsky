import 'dart:async';
import 'dart:io';

import 'package:craftsky_app/profile/sharing/profile_share_card_share.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:path_provider_platform_interface/path_provider_platform_interface.dart';
import 'package:share_plus/share_plus.dart';

class _Paths extends PathProviderPlatform {
  new(this.path);
  final String path;
  @override
  Future<String> getTemporaryPath() async => path;
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  const channel = MethodChannel('dev.fluttercommunity.plus/share');
  late Directory directory;
  late PathProviderPlatform original;
  setUp(() async {
    directory = await Directory.systemTemp.createTemp('share-card-test-');
    original = PathProviderPlatform.instance;
    PathProviderPlatform.instance = _Paths(directory.path);
  });
  tearDown(() async {
    PathProviderPlatform.instance = original;
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, null);
    await directory.delete(recursive: true);
  });

  for (final dismissed in [false, true]) {
    test(
      'shares a PNG and retains it until sheet completes ($dismissed)',
      () async {
        final pending = Completer<String>();
        final invoked = Completer<File>();
        final bytes = Uint8List.fromList([137, 80, 78, 71]);
        const origin = Rect.fromLTWH(20, 30, 100, 40);
        TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
            .setMockMethodCallHandler(channel, (call) async {
              expect(call.method, 'share');
              final args = call.arguments as Map;
              expect(args['mimeTypes'], ['image/png']);
              expect(args['text'], isNull);
              expect(args['uri'], isNull);
              expect(args['originWidth'], origin.width);
              final file = File((args['paths'] as List).single as String);
              expect(file.uri.pathSegments.last, 'CraftSky-profile.png');
              expect(await file.readAsBytes(), bytes);
              invoked.complete(file);
              return await pending.future;
            });
        final operation = shareProfileCardImage(bytes, origin: origin);
        final file = await invoked.future;
        expect(file.existsSync(), isTrue);
        pending.complete(
          dismissed ? '' : 'recipient',
        );
        final result = await operation;
        expect(
          result.status,
          dismissed ? ShareResultStatus.dismissed : ShareResultStatus.success,
        );
        expect(file.existsSync(), isFalse);
        expect(await directory.list().toList(), isEmpty);
      },
    );
  }

  test('preserves platform failure and removes temporary image', () async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (_) async {
          throw PlatformException(code: 'unavailable');
        });
    await expectLater(
      shareProfileCardImage(
        Uint8List(4),
        origin: const Rect.fromLTWH(1, 1, 10, 10),
      ),
      throwsA(
        isA<PlatformException>().having((e) => e.code, 'code', 'unavailable'),
      ),
    );
    expect(await directory.list().toList(), isEmpty);
  });
}
