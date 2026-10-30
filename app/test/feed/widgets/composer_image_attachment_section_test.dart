import 'dart:typed_data';

import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/widgets/composer_image_attachment_section.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/image/craftsky_image_attachment_preview.dart';
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
    'image editing AT-009: pending edit retains preview and '
    'hides conflicting edit actions',
    (tester) async {
      final state = ValueNotifier(
        ComposerImagesState(
          hasImageEditsInFlight: true,
          images: [
            _image('1').copyWith(
              phase: ImageReady(
                bytes: Uint8List(0),
                mimeType: 'image/png',
                width: 10,
                height: 10,
                sha256: 'one',
              ),
            ),
          ],
        ),
      );
      addTearDown(state.dispose);
      await _pump(tester, state, onEditImage: (_) async {});
      expect(find.byType(CraftskyImageAttachmentPreview), findsOneWidget);
      expect(find.byKey(const Key('composer-edit-image-1')), findsNothing);
      expect(
        tester
            .widget<TextButton>(find.byKey(const Key('composer-clear-images')))
            .onPressed,
        isNull,
      );
    },
  );

  testWidgets(
    'image editing UT-001: reference and unavailable photos cannot edit',
    (tester) async {
      final state = ValueNotifier(
        ComposerImagesState(
          images: [
            _image('missing'),
            _image('remote').copyWith(
              phase: const PublishedImageReady(
                existingIndex: 0,
                url: 'https://cdn.example/image.jpg',
              ),
            ),
          ],
        ),
      );
      addTearDown(state.dispose);
      await _pump(tester, state, onEditImage: (_) async {});
      expect(
        find.byKey(const Key('composer-edit-image-missing')),
        findsNothing,
      );
      expect(find.byKey(const Key('composer-edit-image-remote')), findsNothing);
    },
  );

  testWidgets(
    'image editing AT-001: overview has no edit overlays; '
    'details Edit selects the stable image',
    (tester) async {
      final state = ValueNotifier(
        ComposerImagesState(
          images: [
            _image('1').copyWith(
              phase: ImageReady(
                bytes: Uint8List(0),
                mimeType: 'image/png',
                width: 10,
                height: 10,
                sha256: 'one',
              ),
            ),
            _image('2').copyWith(
              phase: ImageReady(
                bytes: Uint8List(0),
                mimeType: 'image/png',
                width: 10,
                height: 10,
                sha256: 'two',
              ),
            ),
          ],
        ),
      );
      addTearDown(state.dispose);
      String? selected;
      await _pump(
        tester,
        state,
        onEditImage: (id) async {
          selected = id;
        },
      );
      expect(selected, isNull);
      expect(find.byKey(const Key('composer-edit-image-1')), findsNothing);
      expect(find.byKey(const Key('composer-edit-image-2')), findsNothing);
      await tester.tap(find.byKey(const Key('composer-edit-images')));
      await tester.pumpAndSettle();
      expect(find.byKey(const Key('composer-photo-details')), findsOneWidget);
      await tester.ensureVisible(
        find.byKey(const Key('composer-edit-image-2')),
      );
      await tester.tap(find.byKey(const Key('composer-edit-image-2')));
      await tester.pumpAndSettle();
      expect(selected, '2');
      expect(find.byKey(const Key('composer-photo-details')), findsOneWidget);
    },
  );

  testWidgets(
    'UT-001 retained published images show their current URL in summary and '
    'details',
    (tester) async {
      final state = ValueNotifier(
        ComposerImagesState(
          images: [
            _image('retained').copyWith(
              phase: const PublishedImageReady(
                existingIndex: 0,
                url: 'https://cdn.example/published.jpg',
              ),
            ),
          ],
        ),
      );
      addTearDown(state.dispose);
      await _pump(tester, state);
      expect(
        tester
            .widget<CraftskyImageAttachmentPreview>(
              find.byType(CraftskyImageAttachmentPreview).first,
            )
            .imageUrl,
        'https://cdn.example/published.jpg',
      );
      await tester.tap(find.byKey(const Key('composer-edit-images')));
      await tester.pumpAndSettle();
      expect(
        tester
            .widget<CraftskyImageAttachmentPreview>(
              find.byType(CraftskyImageAttachmentPreview).last,
            )
            .imageUrl,
        'https://cdn.example/published.jpg',
      );
    },
  );
  for (final useDetailsRoute in [true, false]) {
    for (final composing in [true, false]) {
      testWidgets('rapid mid-sentence edits preserve editing value '
          '(details: $useDetailsRoute, composing: $composing)', (tester) async {
        const original = 'The corner increase. The perspective is askew.';
        final state = ValueNotifier(
          ComposerImagesState(
            images: [
              _image('1').copyWith(altText: original),
              _image('2'),
              _image('3'),
            ],
          ),
        );
        addTearDown(state.dispose);
        await _pump(tester, state, useDetailsRoute: useDetailsRoute);
        if (useDetailsRoute) {
          await tester.tap(find.byKey(const Key('composer-edit-images')));
        }
        await tester.pumpAndSettle();
        final alt = find.byKey(const Key('composer-alt-1'));
        await tester.ensureVisible(alt);
        await tester.showKeyboard(alt);
        final editable = find.descendant(
          of: alt,
          matching: find.byType(EditableText),
        );
        TextEditingValue currentValue() =>
            tester.widget<EditableText>(editable).controller.value;
        const insertionOffset = 19;
        var text = original;
        const insertion = ', not folded in half';
        for (var i = 0; i < insertion.length; i++) {
          final offset = insertionOffset + i;
          text = text.replaceRange(offset, offset, insertion[i]);
          final edit = TextEditingValue(
            text: text,
            selection: TextSelection.collapsed(offset: offset + 1),
            composing: composing
                ? TextRange(start: insertionOffset, end: offset + 1)
                : TextRange.empty,
          );
          tester.testTextInput.updateEditingValue(edit);
          // Render a frame without waiting for the list's async diff.
          await tester.pump();
          expect(currentValue(), edit, reason: 'After keystroke $i');
        }
        final expected = currentValue();
        await tester.pumpAndSettle();
        expect(currentValue(), expected);
        expect(state.value.images.first.altText, text);
      });
    }
  }

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
  Future<void> Function(String)? onEditImage,
  bool enabled = true,
  bool useDetailsRoute = true,
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
            useDetailsRoute: useDetailsRoute,
            imagesState: images,
            onEditImage: onEditImage,
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
