import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_select_inputs.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

import '../../fakes/recording_messenger.dart';

void main() {
  testWidgets('SIM-AT-02 named types offer only listed subtypes', (
    tester,
  ) async {
    await _openDetailsForCraft(tester, 'Knitting');
    await _searchAndSelect(
      tester,
      fieldName: 'knittingProjectType',
      query: 'home',
      option: 'Home goods',
    );
    await _openOptions(
      tester,
      fieldName: 'knittingProjectType',
      query: 'add your own',
    );
    expect(find.widgetWithText(ListTile, 'Add your own'), findsNothing);
    expect(find.byKey(const Key('project-custom-subtype')), findsNothing);
    expect(find.text('Needle sizes'), findsNothing);
  });

  testWidgets('AT-005 optional details do not gate the primary form', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          activeLanguagePreferencesProvider.overrideWith(
            (ref) => const LanguagePreferences(
              primaryLanguage: 'en',
              contentLanguages: ['en'],
            ),
          ),
        ],
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProjectComposerSheet(),
          ),
        ),
      ),
    );

    await tester.tap(
      find.byKey(const Key('project-composer-primary-action')),
    );
    await tester.pumpAndSettle();

    expect(find.text('Choose a craft type.'), findsOneWidget);
    expect(find.text('Add at least one photo.'), findsOneWidget);
    expect(find.text('Materials and style'), findsNothing);
    expect(find.text('Duration'), findsNothing);
    expect(find.text('Project type'), findsNothing);
  });

  const activeOptions = {
    'Sewing': ['Project type'],
    'Knitting': ['Project type', 'Knitting methods'],
    'Crochet': ['Project type', 'Crochet methods'],
    'Quilting': ['Project type'],
    'Embroidery': [
      'Project type',
      'Embroidery method',
      'Embroidery style',
    ],
    'Felting': ['Project type', 'Felting methods'],
    'Weaving': ['Project type', 'Loom type'],
  };
  for (final entry in activeOptions.entries) {
    testWidgets('SIM-AT-04 shows reviewed controls for ${entry.key}', (
      tester,
    ) async {
      await _openDetailsForCraft(tester, entry.key);
      for (final label in entry.value) {
        expect(find.text(label), findsOneWidget);
      }
      for (final retired in [
        'Project subtype',
        'Size made',
        'Fit notes',
        'Yarn weight',
        'Needle sizes',
        'Hook sizes',
        'Gauge stitches',
        'Gauge rows',
        'Gauge measurement',
        'Gauge unit',
        'Finished size',
        'Piecing technique',
        'Quilting method',
        'Techniques',
      ]) {
        expect(find.text(retired), findsNothing);
      }
    });
  }

  testWidgets('finds a subtype with its parent and can select the parent', (
    tester,
  ) async {
    await _openDetailsForCraft(tester, 'Sewing');

    await _openOptions(
      tester,
      fieldName: 'sewingProjectType',
      query: 'dre',
    );
    final parent = find.widgetWithText(ListTile, 'Garment');
    final child = find.widgetWithText(ListTile, 'Dress');
    expect(parent, findsOneWidget);
    expect(child, findsOneWidget);
    expect(
      (tester.widget<ListTile>(parent).title! as Text).style?.fontWeight,
      FontWeight.bold,
    );
    expect((tester.widget<ListTile>(child).title! as Text).style, isNull);
    expect(find.text('Bag'), findsNothing);
    expect(tester.getBottomLeft(parent).dy, tester.getTopLeft(child).dy);
    expect(tester.getTopLeft(parent).dx, lessThan(tester.getTopLeft(child).dx));
    await tester.tap(child);
    await tester.pumpAndSettle();
    expect(find.text('Garment · Dress'), findsOneWidget);

    await _openOptions(
      tester,
      fieldName: 'sewingProjectType',
      query: 'garment',
    );
    final scrollable = find.descendant(
      of: find.byKey(const Key('sewingProjectType-options-panel')),
      matching: find.byType(Scrollable),
    );
    await tester.scrollUntilVisible(child, 60, scrollable: scrollable);
    expect(child, findsOneWidget);
    tester.state<ScrollableState>(scrollable).position.jumpTo(0);
    await tester.pumpAndSettle();
    await tester.tap(parent);
    await tester.pumpAndSettle();
    expect(find.text('Garment'), findsOneWidget);
    expect(find.text('Garment · Dress'), findsNothing);

    await _searchAndSelect(
      tester,
      fieldName: 'sewingProjectType',
      query: 'acc',
      option: 'Accessory',
    );

    expect(find.text('Dress'), findsNothing);
    await _openOptions(
      tester,
      fieldName: 'sewingProjectType',
      query: 'bag',
    );
    expect(find.text('Bag'), findsOneWidget);
    expect(find.text('Dress'), findsNothing);
  });

  testWidgets('project type choices put Other last and allow a name', (
    tester,
  ) async {
    await _openDetailsForCraft(tester, 'Sewing');
    final type = tester.widget<CraftskySingleSelectInput<String>>(
      find.byWidgetPredicate(
        (widget) =>
            widget is CraftskySingleSelectInput<String> &&
            widget.label == 'Project type',
      ),
    );
    final labels = type.options
        .where((option) => option.parentValue == null)
        .map((option) => option.label)
        .toList();
    expect(type.options.last.label, 'Other');
    final other = labels.removeLast();
    expect(other, 'Other');
    expect(labels, [...labels]..sort());
    await _searchAndSelect(
      tester,
      fieldName: 'sewingProjectType',
      query: 'other',
      option: 'Other',
    );
    final custom = find.byKey(const Key('project-custom-subtype'));
    await tester.enterText(custom, 'Travel organizer');
    await tester.pumpAndSettle();
    await _returnToFirstPage(tester);
    expect(find.text('Other · Travel organizer'), findsOneWidget);
    await _openCraftDetails(tester);
    expect(find.text('Travel organizer'), findsOneWidget);
    await _openOptions(
      tester,
      fieldName: 'sewingProjectType',
      query: 'costume',
    );
    final costume = find.widgetWithText(ListTile, 'Costume');
    expect(
      (tester.widget<ListTile>(costume).title! as Text).style?.fontWeight,
      FontWeight.bold,
    );
    await tester.tap(costume);
    await tester.pumpAndSettle();
    expect(custom, findsNothing);
    expect(find.text('Travel organizer'), findsNothing);
    await _searchAndSelect(
      tester,
      fieldName: 'sewingProjectType',
      query: 'other',
      option: 'Other',
    );
    expect(find.text('Travel organizer'), findsNothing);
    await tester.enterText(custom, 'Travel organizer');
    await tester.pumpAndSettle();
    await tester.enterText(custom, '');
    await tester.pumpAndSettle();
    await _returnToFirstPage(tester);
    await _openCraftDetails(tester);
    expect(find.text('Travel organizer'), findsNothing);
    expect(find.text('Other'), findsOneWidget);
  });

  testWidgets('item options show alternate names under their main label', (
    tester,
  ) async {
    await _openDetailsForCraft(tester, 'Sewing');
    await _openOptions(
      tester,
      fieldName: 'sewingProjectType',
      query: 'home decor',
    );
    final homeGoods = find.widgetWithText(ListTile, 'Home goods');
    expect(
      find.descendant(of: homeGoods, matching: find.text('Home decor')),
      findsOneWidget,
    );
    await tester.tap(homeGoods);
    await tester.pumpAndSettle();
    await _searchAndSelect(
      tester,
      fieldName: 'sewingProjectType',
      query: 'garment',
      option: 'Garment',
    );
    await _openOptions(
      tester,
      fieldName: 'sewingProjectType',
      query: 'pants or trousers',
    );
    final trousers = find.widgetWithText(ListTile, 'Trousers');
    expect(
      find.descendant(of: trousers, matching: find.text('Pants')),
      findsOneWidget,
    );
    await tester.tap(trousers);
    await tester.pumpAndSettle();
    expect(find.text('Garment · Trousers'), findsOneWidget);
    expect(find.text('Pants'), findsNothing);

    await _searchAndSelect(
      tester,
      fieldName: 'sewingProjectType',
      query: 'alteration',
      option: 'Alteration and repair',
    );
    await _openOptions(
      tester,
      fieldName: 'sewingProjectType',
      query: 'mending',
    );
    final repair = find.widgetWithText(ListTile, 'Repair');
    expect(
      find.descendant(of: repair, matching: find.text('Mending')),
      findsOneWidget,
    );
    await tester.tap(repair);
    await tester.pumpAndSettle();
    expect(find.text('Alteration and repair · Repair'), findsOneWidget);
    expect(find.text('Mending'), findsNothing);
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
    await _openCraftDetails(tester);
    await _returnToFirstPage(tester);
    await _selectCraft(tester, 'Sewing');
    await _openCraftDetails(tester);

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

Future<void> _openCraftDetails(WidgetTester tester) async {
  final action = find.byKey(
    const Key('project-composer-craft-details-action'),
  );
  await tester.ensureVisible(action);
  await tester.pumpAndSettle();
  await tester.tap(action);
  await tester.pumpAndSettle();
}

Future<void> _selectCraft(WidgetTester tester, String craft) async {
  final craftDropdown = find.byKey(const Key('craftType-select-button'));
  await tester.ensureVisible(craftDropdown);
  await tester.pumpAndSettle();
  await tester.tap(craftDropdown);
  await tester.pumpAndSettle();
  await tester.enterText(
    find.byKey(const Key('craftType-search-input')),
    craft,
  );
  await tester.pumpAndSettle();
  await tester.tap(find.text(craft).last);
  await tester.pumpAndSettle();
}

Future<void> _openDetailsForCraft(WidgetTester tester, String craft) async {
  final composerId = 'details-$craft';
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        activeLanguagePreferencesProvider.overrideWith(
          (ref) => const LanguagePreferences(
            primaryLanguage: 'en',
            contentLanguages: ['en'],
          ),
        ),
        composerImagesProvider(composerId).overrideWithValue(_readyImagesState),
      ],
      child: MessengerScope(
        messenger: RecordingMessenger(),
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: ProjectComposerSheet(composerId: composerId),
        ),
      ),
    ),
  );

  await _selectCraft(tester, craft);
  await _openCraftDetails(tester);
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
