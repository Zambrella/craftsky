import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/recording_messenger.dart';

void main() {
  testWidgets('IT-005 shows image selection limit notices', (tester) async {
    final messenger = RecordingMessenger();

    await _pumpComposerWithNotice(
      tester,
      messenger: messenger,
      notice: const ImageSelectionLimitNotice(
        id: 1,
        maxImages: 4,
        acceptedCount: 0,
      ),
    );

    expect(
      messenger.calls,
      contains(('error', 'You can add up to 4 images', null)),
    );
  });

  testWidgets('IT-005 shows unsupported image notices', (tester) async {
    final messenger = RecordingMessenger();

    await _pumpComposerWithNotice(
      tester,
      messenger: messenger,
      notice: const UnsupportedImagesNotice(id: 2, count: 2),
    );

    expect(
      messenger.calls,
      contains(('error', '2 unsupported images', null)),
    );
  });

  testWidgets('IT-005 shows image picker failure notices', (tester) async {
    final messenger = RecordingMessenger();

    await _pumpComposerWithNotice(
      tester,
      messenger: messenger,
      notice: const ImagePickerFailedNotice(id: 3),
    );

    expect(
      messenger.calls,
      contains(('error', 'Could not open image picker', null)),
    );
  });
}

Future<void> _pumpComposerWithNotice(
  WidgetTester tester, {
  required RecordingMessenger messenger,
  required ComposerImageNotice notice,
}) async {
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        composerImagesProvider('notice-composer').overrideWithValue(
          ComposerImagesState(images: const [], notice: notice),
        ),
      ],
      child: MessengerScope(
        messenger: messenger,
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const ProjectComposerSheet(composerId: 'notice-composer'),
        ),
      ),
    ),
  );
  await tester.pump();
}
