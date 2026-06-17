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
    'AT-005 blocks before details until page one required fields pass',
    (
      tester,
    ) async {
      await tester.pumpWidget(
        ProviderScope(
          child: MessengerScope(
            messenger: RecordingMessenger(),
            child: MaterialApp(
              theme: AppTheme.lightThemeData,
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: const ProjectComposerSheet(),
            ),
          ),
        ),
      );

      await tester.tap(find.widgetWithText(TextButton, 'Next'));
      await tester.pumpAndSettle();

      expect(find.text('Choose a craft type.'), findsOneWidget);
      expect(find.text('Add at least one photo.'), findsOneWidget);
      expect(find.text('Project type'), findsNothing);
    },
  );

  testWidgets('AT-005 shows sewing detail fields for sewing projects', (
    tester,
  ) async {
    await _openDetailsForCraft(tester, 'Sewing');

    expect(find.text('Project type'), findsOneWidget);
    expect(find.text('Project subtype'), findsOneWidget);
    expect(find.text('Size made'), findsOneWidget);
    expect(find.text('Fit notes'), findsOneWidget);
    expect(find.text('Yarn weight'), findsNothing);
    expect(find.text('Needle size'), findsNothing);
    expect(find.text('Hook size'), findsNothing);
  });

  testWidgets('AT-005 shows knitting detail fields for knitting projects', (
    tester,
  ) async {
    await _openDetailsForCraft(tester, 'Knitting');

    expect(find.text('Project type'), findsOneWidget);
    expect(find.text('Project subtype'), findsOneWidget);
    expect(find.text('Yarn weight'), findsOneWidget);
    expect(find.text('Needle size'), findsOneWidget);
    expect(find.text('Gauge stitches'), findsOneWidget);
    expect(find.text('Gauge rows'), findsOneWidget);
    expect(find.text('Gauge measurement'), findsOneWidget);
    expect(find.text('Gauge unit'), findsOneWidget);
    expect(find.text('Finished size'), findsOneWidget);
    expect(find.text('Fit notes'), findsNothing);
    expect(find.text('Hook size'), findsNothing);
  });

  testWidgets('AT-005 shows crochet detail fields for crochet projects', (
    tester,
  ) async {
    await _openDetailsForCraft(tester, 'Crochet');

    expect(find.text('Project type'), findsOneWidget);
    expect(find.text('Project subtype'), findsOneWidget);
    expect(find.text('Yarn weight'), findsOneWidget);
    expect(find.text('Hook size'), findsOneWidget);
    expect(find.text('Gauge stitches'), findsOneWidget);
    expect(find.text('Gauge rows'), findsOneWidget);
    expect(find.text('Gauge measurement'), findsOneWidget);
    expect(find.text('Gauge unit'), findsOneWidget);
    expect(find.text('Finished size'), findsOneWidget);
    expect(find.text('Needle size'), findsNothing);
    expect(find.text('Fit notes'), findsNothing);
  });

  testWidgets('AT-005 shows quilting detail fields for quilting projects', (
    tester,
  ) async {
    await _openDetailsForCraft(tester, 'Quilting');

    expect(find.text('Project type'), findsOneWidget);
    expect(find.text('Project subtype'), findsOneWidget);
    expect(find.text('Size'), findsOneWidget);
    expect(find.text('Piecing technique'), findsOneWidget);
    expect(find.text('Quilting method'), findsOneWidget);
    expect(find.text('Yarn weight'), findsNothing);
    expect(find.text('Gauge stitches'), findsNothing);
    expect(find.text('Fit notes'), findsNothing);
  });

  testWidgets('AT-005 shows no empty-state prompt for crafts without details', (
    tester,
  ) async {
    await _openDetailsForCraft(tester, 'Embroidery');

    expect(find.text('Select Craft Type'), findsNothing);
    expect(find.text('Project type'), findsNothing);
    expect(find.text('Yarn weight'), findsNothing);
    expect(find.text('Fit notes'), findsNothing);
  });

  testWidgets('AT-005 filters and clears project subtype selections', (
    tester,
  ) async {
    await _openDetailsForCraft(tester, 'Sewing');

    final projectSubtype = find.byKey(
      const Key('sewingProjectSubtype-select-button'),
    );

    await _searchAndSelect(
      tester,
      fieldName: 'sewingProjectType',
      query: 'gar',
      option: 'Garment',
    );

    await tester.ensureVisible(projectSubtype);
    await tester.pumpAndSettle();
    await _openOptions(
      tester,
      fieldName: 'sewingProjectSubtype',
      query: 'dre',
    );
    expect(find.text('Dress'), findsOneWidget);
    expect(find.text('Bag'), findsNothing);
    await tester.tap(find.text('Dress').last);
    await tester.pumpAndSettle();
    expect(find.text('Dress'), findsOneWidget);

    await _searchAndSelect(
      tester,
      fieldName: 'sewingProjectType',
      query: 'acc',
      option: 'Accessory',
    );

    expect(find.text('Dress'), findsNothing);
    await tester.ensureVisible(projectSubtype);
    await tester.pumpAndSettle();
    await _openOptions(
      tester,
      fieldName: 'sewingProjectSubtype',
      query: 'bag',
    );
    expect(find.text('Bag'), findsOneWidget);
    expect(find.text('Dress'), findsNothing);
  });

  testWidgets('AT-005 clears craft-specific details when craft changes', (
    tester,
  ) async {
    await _openDetailsForCraft(tester, 'Sewing');

    await _searchAndSelect(
      tester,
      fieldName: 'sewingProjectType',
      query: 'gar',
      option: 'Garment',
    );
    expect(find.text('Garment'), findsOneWidget);

    await _returnToFirstPage(tester);
    await _selectCraft(tester, 'Knitting');
    await tester.tap(find.widgetWithText(TextButton, 'Next'));
    await tester.pumpAndSettle();
    await _returnToFirstPage(tester);
    await _selectCraft(tester, 'Sewing');
    await tester.tap(find.widgetWithText(TextButton, 'Next'));
    await tester.pumpAndSettle();

    expect(find.text('Garment'), findsNothing);
  });
}

Future<void> _searchAndSelect(
  WidgetTester tester, {
  required String fieldName,
  required String query,
  required String option,
}) async {
  await _openOptions(tester, fieldName: fieldName, query: query);
  await tester.tap(find.text(option).last);
  await tester.pumpAndSettle();
}

Future<void> _openOptions(
  WidgetTester tester, {
  required String fieldName,
  required String query,
}) async {
  await tester.ensureVisible(find.byKey(Key('$fieldName-select-button')));
  await tester.pumpAndSettle();
  final searchInput = find.byKey(Key('$fieldName-search-input'));
  if (searchInput.evaluate().isNotEmpty) {
    await tester.enterText(searchInput, query);
  } else {
    await tester.tap(find.byKey(Key('$fieldName-select-button')));
  }
  await tester.pumpAndSettle();
}

Future<void> _returnToFirstPage(WidgetTester tester) async {
  await tester.tap(find.byTooltip('Back'));
  await tester.pumpAndSettle();
}

Future<void> _selectCraft(WidgetTester tester, String craft) async {
  final craftDropdown = find.byKey(const Key('craftType-select-button'));
  await tester.ensureVisible(craftDropdown);
  await tester.pumpAndSettle();
  await tester.tap(craftDropdown);
  await tester.pumpAndSettle();
  await tester.tap(find.text(craft).last);
  await tester.pumpAndSettle();
}

Future<void> _openDetailsForCraft(WidgetTester tester, String craft) async {
  final composerId = 'details-$craft';
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        composerImagesProvider(composerId).overrideWithValue(_readyImagesState),
      ],
      child: MessengerScope(
        messenger: RecordingMessenger(),
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: ProjectComposerSheet(composerId: composerId),
        ),
      ),
    ),
  );

  await _selectCraft(tester, craft);
  await tester.tap(find.widgetWithText(TextButton, 'Next'));
  await tester.pumpAndSettle();
}

const _readyImagesState = ComposerImagesState(
  images: [
    ComposerImageDraft(
      id: 'image-1',
      fileName: 'project.jpg',
      mimeType: 'image/jpeg',
      altText: 'Finished project photo',
      phase: ImageUploaded(
        UploadedDraftImage(cid: 'bafkimage', mime: 'image/jpeg', size: 123),
      ),
    ),
  ],
);
