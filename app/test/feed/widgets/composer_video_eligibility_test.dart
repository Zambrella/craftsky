import 'dart:typed_data';

import 'package:craftsky_app/feed/models/video_upload_limits.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/feed/providers/composer_video_controller.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/recording_messenger.dart';

void main() {
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
}) async {
  final controller = ComposerVideoController(
    picker: _VideoPicker(),
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
          localizationsDelegates: AppLocalizations.localizationsDelegates,
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
  final addMedia = find.byKey(const Key('composer-add-image'));
  await tester.ensureVisible(addMedia);
  await tester.tap(addMedia);
  await tester.pumpAndSettle();
  await tester.tap(find.byKey(const Key('composer-choose-video')));
  await tester.pumpAndSettle();
}

final class _VideoPicker implements ExistingVideoPicker {
  @override
  Future<LocalVideoSelection?> pickExisting() async => LocalVideoSelection(
    displayName: 'quota.mp4',
    mimeType: 'video/mp4',
    byteLength: 12,
    duration: null,
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
