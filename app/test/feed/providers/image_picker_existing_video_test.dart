import 'package:craftsky_app/feed/providers/image_picker_existing_video.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image_picker/image_picker.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  group('iOS native video preparation', () {
    const channel = MethodChannel('social.craftsky/local_video');
    final messenger =
        TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;
    late XFile file;

    setUp(() {
      debugDefaultTargetPlatformOverride = TargetPlatform.iOS;
      file = XFile.fromData(Uint8List(12));
    });

    tearDown(() {
      debugDefaultTargetPlatformOverride = null;
      messenger.setMockMethodCallHandler(channel, null);
    });

    test('uses native portrait metadata and poster without a player', () async {
      messenger.setMockMethodCallHandler(channel, (call) async {
        expect(call.method, 'probe');
        expect(call.arguments, {'path': file.path});
        return {
          'width': 1080,
          'height': 1920,
          'durationMs': 5025,
          'posterBytes': Uint8List.fromList([1, 2, 3]),
        };
      });

      final selection = await ImagePickerExistingVideo(pick: () async => file)
          .pickExisting();
      expect(selection?.width, 1080);
      expect(selection?.height, 1920);
      expect(selection?.duration, const Duration(milliseconds: 5025));
      expect(selection?.posterBytes, [1, 2, 3]);
    });

    test('rejects an empty poster instead of falling back to MPV', () async {
      messenger.setMockMethodCallHandler(
        channel,
        (_) async => {
          'width': 1080,
          'height': 1920,
          'posterBytes': Uint8List(0),
        },
      );
      await expectLater(
        ImagePickerExistingVideo(pick: () async => file).pickExisting(),
        throwsStateError,
      );
    });

    test('preserves platform failures without invoking a player', () async {
      messenger.setMockMethodCallHandler(channel, (_) async {
        throw PlatformException(code: 'video_probe_failed');
      });
      await expectLater(
        ImagePickerExistingVideo(pick: () async => file).pickExisting(),
        throwsA(
          isA<PlatformException>().having(
            (e) => e.code,
            'code',
            'video_probe_failed',
          ),
        ),
      );
    });
  });

  test('UT-001 unknown production duration remains eligible', () async {
    final duration = await resolveVideoDuration(
      current: Duration.zero,
      changes: const Stream<Duration>.empty(),
      timeout: Duration.zero,
    );

    expect(duration, isNull);
  });

  test(
    'AT-002 picker prepares duration, dimensions, and a poster locally',
    () async {
      final bytes = Uint8List.fromList(const [
        0,
        0,
        0,
        24,
        0x66,
        0x74,
        0x79,
        0x70,
        0x69,
        0x73,
        0x6f,
        0x6d,
      ]);
      final picker = ImagePickerExistingVideo(
        pick: () async => XFile.fromData(
          bytes,
          name: 'local.mp4',
          mimeType: 'video/mp4',
        ),
        probe: (_) async => VideoProbeResult(
          duration: const Duration(seconds: 7),
          width: 1080,
          height: 1920,
          posterBytes: Uint8List.fromList([1, 2, 3]),
        ),
      );

      final selection = await picker.pickExisting();

      expect(selection?.duration, const Duration(seconds: 7));
      expect(selection?.width, 1080);
      expect(selection?.height, 1920);
      expect(selection?.posterBytes, [1, 2, 3]);
    },
  );

  test(
    'AT-002 picker keeps the source stream local and reads a bounded header',
    () async {
      final bytes = Uint8List.fromList(const [
        0,
        0,
        0,
        24,
        0x66,
        0x74,
        0x79,
        0x70,
        0x69,
        0x73,
        0x6f,
        0x6d,
        1,
        2,
      ]);
      final picker = ImagePickerExistingVideo(
        pick: () async => XFile.fromData(
          bytes,
          name: 'local.mp4',
          mimeType: 'video/mp4',
        ),
        probe: (_) async => VideoProbeResult(
          duration: const Duration(seconds: 1),
          width: 16,
          height: 9,
          posterBytes: Uint8List.fromList([1]),
        ),
      );

      final selection = await picker.pickExisting();

      expect(selection?.displayName, 'video.mp4');
      expect(selection?.byteLength, bytes.length);
      expect(selection?.headerBytes, bytes.sublist(0, 12));
      expect(
        await selection?.openRead().expand((chunk) => chunk).toList(),
        bytes,
      );
    },
  );
}
