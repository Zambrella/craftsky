import 'dart:async';

import 'package:craftsky_app/feed/models/post.dart';
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
  testWidgets(
    'AT-009 closes immediately when the project composer is unchanged',
    (
      tester,
    ) async {
      await _openProjectComposer(tester);

      await tester.tap(find.byType(CloseButton));
      await tester.pumpAndSettle();

      expect(find.text('Host'), findsOneWidget);
      expect(find.text('Discard draft?'), findsNothing);
    },
  );

  testWidgets('AT-009 confirms before discarding body text edits', (
    tester,
  ) async {
    await _openProjectComposer(tester);

    await tester.ensureVisible(_bodyTextField());
    await tester.enterText(_bodyTextField(), 'A finished hoop');
    await tester.pump();

    await tester.tap(find.byType(CloseButton));
    await tester.pumpAndSettle();

    expect(find.text('Discard draft?'), findsOneWidget);
    expect(find.text("Your draft won't be saved."), findsOneWidget);

    await tester.tap(find.text('Keep editing'));
    await tester.pumpAndSettle();

    expect(find.text('Discard draft?'), findsNothing);
    expect(find.text('Project post'), findsOneWidget);

    await tester.tap(find.byType(CloseButton));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Discard'));
    await tester.pumpAndSettle();

    expect(find.text('Host'), findsOneWidget);
  });

  testWidgets('AT-009 confirms before discarding selected images', (
    tester,
  ) async {
    await _openProjectComposer(
      tester,
      composerId: 'image-draft-composer',
      overrides: [
        composerImagesProvider('image-draft-composer').overrideWithValue(
          const ComposerImagesState(
            images: [
              ComposerImageDraft(
                id: 'image-1',
                fileName: 'project.jpg',
                mimeType: 'image/jpeg',
                altText: 'Finished project photo',
                phase: ImageUploaded(
                  UploadedDraftImage(
                    cid: 'bafkimage',
                    mime: 'image/jpeg',
                    size: 123,
                  ),
                ),
              ),
            ],
          ),
        ),
      ],
    );

    await tester.tap(find.byType(CloseButton));
    await tester.pumpAndSettle();

    expect(find.text('Discard draft?'), findsOneWidget);
  });

  testWidgets('AT-009 confirms before discarding project metadata edits', (
    tester,
  ) async {
    await _openProjectComposer(tester);

    final craftDropdown = find.byType(DropdownButton<String>).first;
    await tester.ensureVisible(craftDropdown);
    await tester.pumpAndSettle();
    await tester.tap(craftDropdown);
    await tester.pumpAndSettle();
    await tester.tap(find.text('Embroidery').last);
    await tester.pumpAndSettle();

    await tester.tap(find.byType(CloseButton));
    await tester.pumpAndSettle();

    expect(find.text('Discard draft?'), findsOneWidget);
  });
}

Finder _bodyTextField() {
  return find.descendant(
    of: find.byKey(const Key('project-composer-body-editor')),
    matching: find.byType(TextField),
  );
}

Future<void> _openProjectComposer(
  WidgetTester tester, {
  List<dynamic> overrides = const [],
  String? composerId,
}) async {
  await tester.pumpWidget(
    ProviderScope(
      overrides: List.from(overrides),
      child: MessengerScope(
        messenger: RecordingMessenger(),
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (context) {
              return Scaffold(
                body: Center(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      const Text('Host'),
                      ElevatedButton(
                        onPressed: () {
                          unawaited(
                            Navigator.of(context).push<Post?>(
                              MaterialPageRoute<Post?>(
                                fullscreenDialog: true,
                                builder: (_) => ProjectComposerSheet(
                                  composerId: composerId,
                                ),
                              ),
                            ),
                          );
                        },
                        child: const Text('Open composer'),
                      ),
                    ],
                  ),
                ),
              );
            },
          ),
        ),
      ),
    ),
  );

  await tester.tap(find.text('Open composer'));
  await tester.pumpAndSettle();
}
