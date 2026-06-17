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
    'AT-003/IT-005 add photo action uses provider and exposes alt text',
    (tester) async {
      const composerId = 'real-image-composer';
      final imagesNotifier = _FakeComposerImages();
      final container = ProviderContainer.test(
        overrides: [
          composerImagesProvider(
            composerId,
          ).overrideWith(() => imagesNotifier),
        ],
      );
      addTearDown(container.dispose);

      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MessengerScope(
            messenger: RecordingMessenger(),
            child: MaterialApp(
              theme: AppTheme.lightThemeData,
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: const ProjectComposerSheet(composerId: composerId),
            ),
          ),
        ),
      );

      final addPhoto = find.byKey(const Key('composer-add-image'));
      expect(addPhoto, findsOneWidget);
      expect(tester.widget<InkWell>(addPhoto).onTap, isNotNull);

      await tester.tap(addPhoto);
      await tester.pump();
      final uploaded = await _waitForImageState(
        tester,
        container,
        composerId,
        (state) => state.images.singleOrNull?.phase is ImageUploaded,
      );
      await tester.pump(const Duration(milliseconds: 300));

      expect(imagesNotifier.addImagesCalls, 1);
      final image = uploaded.images.single;
      final altTextField = find.byKey(Key('composer-alt-${image.id}'));
      expect(altTextField, findsOneWidget);
      expect(find.text('Help screen readers'), findsOneWidget);

      await tester.enterText(altTextField, 'Finished project photo');
      await tester.pump();

      expect(
        container
            .read(composerImagesProvider(composerId))
            .images
            .single
            .altText,
        'Finished project photo',
      );
    },
  );

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

Future<ComposerImagesState> _waitForImageState(
  WidgetTester tester,
  ProviderContainer container,
  String composerId,
  bool Function(ComposerImagesState state) predicate,
) async {
  final provider = composerImagesProvider(composerId);
  for (var i = 0; i < 100; i += 1) {
    await tester.pump(const Duration(milliseconds: 20));
    final state = container.read(provider);
    if (predicate(state)) return state;
  }
  fail(
    'Timed out waiting for composer image state: ${container.read(provider)}',
  );
}

class _FakeComposerImages extends ComposerImages {
  int addImagesCalls = 0;

  @override
  ComposerImagesState build(String composerId) {
    return const ComposerImagesState(images: []);
  }

  @override
  Future<void> addImages() async {
    addImagesCalls += 1;
    state = const ComposerImagesState(
      images: [
        ComposerImageDraft(
          id: 'project-image-1',
          fileName: 'project.png',
          mimeType: 'image/png',
          altText: '',
          phase: ImageUploaded(
            UploadedDraftImage(
              cid: 'bafkreiprojectimage',
              mime: 'image/png',
              size: 123,
            ),
          ),
        ),
      ],
    );
  }

  @override
  void setAltText(String imageId, String value) {
    state = state.copyWith(
      images: [
        for (final image in state.images)
          image.id == imageId ? image.copyWith(altText: value) : image,
      ],
    );
  }

  @override
  void remove(String imageId) {
    state = state.copyWith(
      images: state.images.where((image) => image.id != imageId).toList(),
    );
  }

  @override
  void reorder({required int fromIndex, required int toIndex}) {
    final images = [...state.images];
    final image = images.removeAt(fromIndex);
    images.insert(toIndex, image);
    state = state.copyWith(images: images);
  }
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
