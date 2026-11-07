import 'dart:typed_data';

import 'package:craftsky_app/feed/models/video_upload_limits.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/feed/providers/composer_video_controller.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

import '../../fakes/recording_messenger.dart';

void main() {
  testWidgets(
    'picker options show count, format and size limits as helper text',
    (tester) async {
      await _pumpComposer(
        tester,
        messenger: RecordingMessenger(),
        limits: const VideoUploadLimits(canUpload: true),
      );
      await _openMediaMenu(tester);
      expect(find.text('1 photo · up to 100 MB'), findsOneWidget);
      expect(find.text('Up to 8 photos · 100 MB each'), findsOneWidget);
      expect(find.text('MP4 or MOV · up to 10 min · 300 MB'), findsOneWidget);
      final title = tester.widget<Text>(find.text('Choose a video'));
      final helper = tester.widget<Text>(
        find.text('MP4 or MOV · up to 10 min · 300 MB'),
      );
      expect(helper.style!.fontSize, lessThan(title.style!.fontSize!));
    },
  );

  for (final project in [false, true]) {
    testWidgets('video inspection retains description (project=$project)', (
      tester,
    ) async {
      await _pumpComposer(
        tester,
        messenger: RecordingMessenger(),
        limits: const VideoUploadLimits(canUpload: true),
        project: project,
      );
      await _chooseVideo(tester);
      expect(find.byKey(const Key('composer-video-details')), findsOneWidget);
      final field = find.byKey(const Key('composer-video-alt-text'));
      await tester.ensureVisible(field);
      await tester.enterText(field, 'Hands knitting a scarf');
      await tester.pageBack();
      await tester.pumpAndSettle();
      await tester.ensureVisible(
        find.byKey(const Key('composer-inspect-video')),
      );
      await tester.tap(find.byKey(const Key('composer-inspect-video')));
      await tester.pumpAndSettle();
      expect(
        tester.widget<TextField>(field).controller?.text,
        'Hands knitting a scarf',
      );
    });
  }

  for (final project in [false, true]) {
    for (final tooLarge in [true, false]) {
      final name = project ? 'project' : 'standard';
      final rejection = tooLarge ? 'size' : 'duration';
      testWidgets(
        '$name composer explains $rejection rejection',
        (tester) async {
          final messenger = RecordingMessenger();
          await _pumpComposer(
            tester,
            messenger: messenger,
            limits: const VideoUploadLimits(canUpload: true),
            project: project,
            picker: _VideoPicker(
              byteLength: tooLarge ? 300000001 : 12,
              duration: tooLarge ? null : const Duration(seconds: 601),
            ),
          );
          await _chooseVideo(tester);
          expect(messenger.calls.single.$1, 'error');
          expect(
            messenger.calls.single.$2,
            tooLarge
                ? 'This video is too large. Videos must be 300 MB or smaller. '
                      'Choose a smaller file or export a smaller copy.'
                : 'This video is too long. '
                      'Videos must be ten minutes or shorter. '
                      'Trim it and try again.',
          );
          expect(messenger.calls.single.$2, isNot("Couldn't post."));
          expect(find.text('Selected video'), findsNothing);
        },
      );
    }
  }

  testWidgets('standard composer offers video without a feature flag', (
    tester,
  ) async {
    await _pumpComposer(
      tester,
      messenger: RecordingMessenger(),
      limits: const VideoUploadLimits(canUpload: true),
    );

    expect(find.text('Add a photo'), findsNothing);
    expect(find.text('Add photos or a video'), findsOneWidget);
    expect(find.text('Choose up to 8 photos or a video.'), findsOneWidget);
    await _openMediaMenu(tester);
    expect(find.byKey(const Key('composer-choose-video')), findsOneWidget);
  });

  testWidgets('project composer offers video without a feature flag', (
    tester,
  ) async {
    await _pumpComposer(
      tester,
      messenger: RecordingMessenger(),
      limits: const VideoUploadLimits(canUpload: true),
      project: true,
    );

    expect(find.text('Add a photo'), findsNothing);
    expect(find.text('Add photos or a video'), findsOneWidget);
    expect(find.text('Choose up to 8 photos or a video.'), findsOneWidget);
    await _openMediaMenu(tester);
    expect(find.byKey(const Key('composer-choose-video')), findsOneWidget);
  });

  testWidgets('AT-003 standard composer shows exact video quota threshold', (
    tester,
  ) async {
    final messenger = RecordingMessenger();
    await _pumpComposer(
      tester,
      messenger: messenger,
      limits: const VideoUploadLimits(
        canUpload: true,
        remainingDailyVideos: 1,
      ),
    );

    await _chooseVideo(tester);

    expect(messenger.calls, [
      ('info', '1 video remaining today', null),
    ]);
  });

  testWidgets('AT-003 standard composer omits unconstrained quota', (
    tester,
  ) async {
    final messenger = RecordingMessenger();
    await _pumpComposer(
      tester,
      messenger: messenger,
      limits: const VideoUploadLimits(
        canUpload: true,
        remainingDailyVideos: 2,
        remainingDailyBytes: 300000000,
      ),
    );

    await _chooseVideo(tester);

    expect(messenger.calls, isEmpty);
  });

  testWidgets('AT-003 project composer shows exact byte quota threshold', (
    tester,
  ) async {
    final messenger = RecordingMessenger();
    await _pumpComposer(
      tester,
      messenger: messenger,
      limits: const VideoUploadLimits(
        canUpload: true,
        remainingDailyBytes: 299999999,
      ),
      project: true,
    );

    await _chooseVideo(tester);

    expect(messenger.calls, [
      ('info', '299999999 bytes remaining today', null),
    ]);
  });

  testWidgets('AT-003 project composer omits unconstrained quota', (
    tester,
  ) async {
    final messenger = RecordingMessenger();
    await _pumpComposer(
      tester,
      messenger: messenger,
      limits: const VideoUploadLimits(
        canUpload: true,
        remainingDailyVideos: 2,
        remainingDailyBytes: 300000000,
      ),
      project: true,
    );

    await _chooseVideo(tester);

    expect(messenger.calls, isEmpty);
  });
}

Future<void> _pumpComposer(
  WidgetTester tester, {
  required RecordingMessenger messenger,
  required VideoUploadLimits limits,
  bool project = false,
  ExistingVideoPicker? picker,
}) async {
  final controller = ComposerVideoController(
    picker: picker ?? _VideoPicker(),
    checkEligibility: () async => limits,
  );
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        activeLanguagePreferencesProvider.overrideWith(
          (ref) => const LanguagePreferences(
            primaryLanguage: 'en',
            contentLanguages: ['en'],
          ),
        ),
        if (project)
          composerImagesProvider('quota-project').overrideWithValue(
            const ComposerImagesState(images: []),
          ),
      ],
      child: MessengerScope(
        messenger: messenger,
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: project
              ? ProjectComposerSheet(
                  composerId: 'quota-project',
                  videoController: controller,
                )
              : PostComposerSheet(
                  composerId: 'quota-standard',
                  videoController: controller,
                ),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

Future<void> _chooseVideo(WidgetTester tester) async {
  await _openMediaMenu(tester);
  await tester.tap(find.byKey(const Key('composer-choose-video')));
  await tester.pumpAndSettle();
}

Future<void> _openMediaMenu(WidgetTester tester) async {
  final addMedia = find.byKey(const Key('composer-add-image'));
  await tester.ensureVisible(addMedia);
  await tester.tap(addMedia);
  await tester.pumpAndSettle();
}

final class _VideoPicker implements ExistingVideoPicker {
  new({this.byteLength = 12, this.duration});
  final int byteLength;
  final Duration? duration;
  @override
  Future<LocalVideoSelection?> pickExisting() async => LocalVideoSelection(
    displayName: 'quota.mp4',
    localUri: Uri.file('/tmp/quota-video.mp4'),
    mimeType: 'video/mp4',
    byteLength: byteLength,
    duration: duration,
    headerBytes: Uint8List.fromList(const [
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
    ]),
    openRead: () => const Stream.empty(),
  );
}
