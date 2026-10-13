import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/widgets/composer_image_attachment_section.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

ComposerImageDraft _image(String id) => ComposerImageDraft(
  id: id,
  fileName: '$id.jpg',
  mimeType: 'image/jpeg',
  altText: '',
  phase: const ImageUnavailable(width: 100, height: 100),
);

void main() {
  testWidgets(
    'selection opens details; edits persist in the thumbnail summary',
    (
      tester,
    ) async {
      final state = ValueNotifier(const ComposerImagesState(images: []));
      addTearDown(state.dispose);
      await _pump(tester, state);
      expect(find.byKey(const Key('composer-add-image')), findsOneWidget);
      expect(find.byKey(const Key('composer-edit-images')), findsNothing);
      await tester.tap(find.byKey(const Key('composer-add-image')));
      await tester.pumpAndSettle();
      expect(find.byKey(const Key('composer-photo-details')), findsOneWidget);
      final alt = find.byKey(const Key('composer-alt-1'));
      await tester.ensureVisible(alt);
      await tester.enterText(alt, 'A knitted scarf');
      await tester.pumpAndSettle();
      expect(state.value.images.single.altText, 'A knitted scarf');
      await tester.pageBack();
      await tester.pumpAndSettle();
      expect(find.byKey(const Key('composer-edit-images')), findsOneWidget);
      expect(find.byKey(const Key('composer-alt-1')), findsNothing);
      expect(find.text('1 / 1 described'), findsOneWidget);
      await tester.tap(find.byKey(const Key('composer-edit-images')));
      await tester.pumpAndSettle();
      expect(find.text('A knitted scarf'), findsOneWidget);
      await tester.pageBack();
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const Key('composer-clear-images')));
      await tester.pumpAndSettle();
      expect(state.value.images, isEmpty);
      expect(find.byKey(const Key('composer-add-image')), findsOneWidget);
    },
  );

  testWidgets(
    'restored photos open on tap and details enforce the image limit',
    (
      tester,
    ) async {
      final state = ValueNotifier(ComposerImagesState(images: [_image('1')]));
      addTearDown(state.dispose);
      await _pump(tester, state, maxImages: 2);
      expect(find.byKey(const Key('composer-photo-details')), findsNothing);
      await tester.tap(find.byKey(const Key('composer-edit-images')));
      await tester.pumpAndSettle();
      final add = find.byKey(const Key('composer-add-image'));
      await tester.ensureVisible(add);
      await tester.tap(add);
      await tester.pumpAndSettle();
      expect(state.value.images.length, 2);
      expect(add, findsNothing);
      final move = find.byKey(const Key('composer-move-up-2'));
      await tester.ensureVisible(move);
      await tester.tap(move);
      await tester.pumpAndSettle();
      expect(state.value.images.first.id, '2');
      final remove = find.byKey(const Key('composer-remove-2'));
      await tester.ensureVisible(remove);
      await tester.tap(remove);
      await tester.pumpAndSettle();
      expect(state.value.images.single.id, '1');
      expect(add, findsOneWidget);
      await tester.ensureVisible(find.byKey(const Key('composer-remove-1')));
      await tester.tap(find.byKey(const Key('composer-remove-1')));
      await tester.pumpAndSettle();
      expect(state.value.images, isEmpty);
      expect(find.byKey(const Key('composer-photo-details')), findsOneWidget);
      await tester.ensureVisible(add);
      await tester.tap(add);
      await tester.pumpAndSettle();
      expect(state.value.images.length, 1);
      await tester.pageBack();
      await tester.pumpAndSettle();
      expect(find.byKey(const Key('composer-photo-details')), findsNothing);
    },
  );

  testWidgets('cancelling selection stays in the composer', (tester) async {
    final state = ValueNotifier(const ComposerImagesState(images: []));
    addTearDown(state.dispose);
    await _pump(tester, state, cancelSelection: true);
    await tester.tap(find.byKey(const Key('composer-add-image')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('composer-photo-details')), findsNothing);
  });

  testWidgets('disabled summary disables editing and clearing', (tester) async {
    final state = ValueNotifier(ComposerImagesState(images: [_image('1')]));
    addTearDown(state.dispose);
    await _pump(tester, state, enabled: false);
    expect(
      tester
          .widget<InkWell>(
            find.byKey(
              const Key('composer-edit-images'),
            ),
          )
          .onTap,
      isNull,
    );
    expect(
      tester
          .widget<TextButton>(
            find.byKey(
              const Key('composer-clear-images'),
            ),
          )
          .onPressed,
      isNull,
    );
  });
}

Future<void> _pump(
  WidgetTester tester,
  ValueNotifier<ComposerImagesState> state, {
  int maxImages = 4,
  bool cancelSelection = false,
  bool enabled = true,
}) => tester.pumpWidget(
  MaterialApp(
    theme: AppTheme.lightThemeData,
    localizationsDelegates: appLocalizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    home: Scaffold(
      body: SingleChildScrollView(
        child: ValueListenableBuilder(
          valueListenable: state,
          builder: (context, images, _) => ComposerImageAttachmentSection(
            useDetailsRoute: true,
            imagesState: images,
            enabled: enabled,
            maxImages: maxImages,
            cameraSupported: false,
            onAddImages: () async {
              if (cancelSelection) return;
              state.value = images.copyWith(
                images: [
                  ...images.images,
                  _image('${images.images.length + 1}'),
                ],
              );
            },
            onAltTextChanged: (id, alt) {
              state.value = state.value.copyWith(
                images: [
                  for (final image in state.value.images)
                    if (image.id == id) image.copyWith(altText: alt) else image,
                ],
              );
            },
            onRemove: (id) {
              state.value = state.value.copyWith(
                images: [
                  for (final image in state.value.images)
                    if (image.id != id) image,
                ],
              );
            },
            onReplaceUnavailable: (_, _) async {},
            onReorder: (from, to) {
              final reordered = [...state.value.images];
              reordered.insert(to, reordered.removeAt(from));
              state.value = state.value.copyWith(images: reordered);
            },
          ),
        ),
      ),
    ),
  ),
);
