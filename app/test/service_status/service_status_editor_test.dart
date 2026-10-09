import 'dart:async';

import 'package:craftsky_app/feed/composer/composer_submission_coordinator.dart';
import 'package:craftsky_app/feed/composer/submission_screen_awake.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/service_status/models/service_status_document.dart';
import 'package:craftsky_app/service_status/providers/service_status_controller.dart';
import 'package:craftsky_app/service_status/widgets/service_status_host.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/recording_messenger.dart';
import 'announcement_dismissal_test.dart' show FakeDismissalStore;
import 'service_status_controller_test.dart'
    show FakeStatusRepository, maintenance;

void main() {
  // AT-004 / FR-005, FR-009, RULE-004 / AC-008, AC-012.
  for (final project in [false, true]) {
    testWidgets(
      'cover retains ${project ? 'project' : 'post'} editor '
      'and one delayed operation',
      (tester) async {
        const composerId = 'status-editor';
        final repository = FakeStatusRepository();
        final container = ProviderContainer(
          overrides: [
            serviceStatusRepositoryProvider.overrideWithValue(repository),
            announcementDismissalStoreProvider.overrideWithValue(
              FakeDismissalStore(),
            ),
            activeLanguagePreferencesProvider.overrideWith(
              (ref) => const LanguagePreferences(
                primaryLanguage: 'en',
                contentLanguages: ['en'],
              ),
            ),
            composerImagesProvider(composerId).overrideWithValue(images),
          ],
        );
        await tester.pumpWidget(
          UncontrolledProviderScope(
            container: container,
            child: MessengerScope(
              messenger: RecordingMessenger(),
              child: MaterialApp(
                theme: AppTheme.lightThemeData,
                localizationsDelegates: AppLocalizations.localizationsDelegates,
                supportedLocales: AppLocalizations.supportedLocales,
                builder: (context, child) => ServiceStatusHost(child: child!),
                home: project
                    ? const ProjectComposerSheet(composerId: composerId)
                    : const PostComposerSheet(composerId: composerId),
              ),
            ),
          ),
        );
        await tester.pumpAndSettle();
        final field = project
            ? find.descendant(
                of: find.byKey(const Key('project-composer-body-editor')),
                matching: find.byType(TextField),
              )
            : find.byType(TextField).first;
        await tester.ensureVisible(field);
        await tester.enterText(field, 'Unsent cardigan work');
        await tester.pumpAndSettle();
        final element = tester.element(field);
        final textController = tester.widget<TextField>(field).controller!;
        final delayed = Completer<void>();
        var operations = 0;
        var cleanups = 0;
        final coordinator = ComposerSubmissionCoordinator(
          screenAwake: _Awake(),
        );
        final inFlight = coordinator.run(
          presentOverlay: () async {},
          ownershipIsCurrent: () => true,
          saveOriginSnapshot: () async {},
          operation: () async {
            operations++;
            await delayed.future;
          },
          didSucceed: () => true,
          deleteOriginAfterSuccess: () async {
            cleanups++;
          },
          onRunningChanged: ({required running}) {},
          onFailure: (_) => fail('Unexpected failure'),
        );
        await tester.pump();
        final controller = container.read(
          serviceStatusControllerProvider.notifier,
        );
        final cover = controller.refresh();
        repository.requests.last.complete(maintenance);
        await tester.pump();
        await cover;
        expect(find.text('Maintenance'), findsOneWidget);
        expect(tester.element(field), same(element));
        expect(textController.text, 'Unsent cardigan work');
        expect(
          container.read(composerImagesProvider(composerId)),
          same(images),
        );
        expect(operations, 1);
        delayed.complete();
        await tester.pump();
        await inFlight;
        expect(cleanups, 1);
        final clear = controller.refresh();
        repository.requests.last.complete(
          const ServiceStatusDocument(
            mode: ServiceStatusMode.normal,
            revision: 'B',
          ),
        );
        await tester.pump();
        await clear;
        expect(tester.element(field), same(element));
        expect(textController.text, 'Unsent cardigan work');
        expect(operations, 1);
        await tester.pumpWidget(const SizedBox.shrink());
        container.dispose();
      },
    );
  }
}

const images = ComposerImagesState(
  images: [
    ComposerImageDraft(
      id: 'image-1',
      fileName: 'cardigan.jpg',
      mimeType: 'image/jpeg',
      altText: 'Cardigan',
      phase: ImageUploaded(
        UploadedDraftImage(cid: 'bafkimage', mime: 'image/jpeg', size: 123),
      ),
    ),
  ],
);

class _Awake implements SubmissionScreenAwake {
  @override
  Future<void> enable() async {}
  @override
  Future<void> disable() async {}
}
