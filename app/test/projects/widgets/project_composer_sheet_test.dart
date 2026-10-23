import 'dart:async';
import 'dart:convert';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/drafts/composer/draft_composer_hydrator.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/composer/project_draft_snapshot_adapter.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/projects/providers/preferred_project_crafts_provider.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/shared/widgets/craft_icon.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_form_builder_select_fields.dart';
import 'package:craftsky_app/theme/publishing_progress_indicator.dart';
import 'package:flutter/services.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';
import 'package:rive/rive.dart' as rive;

import '../../fakes/recording_messenger.dart';

void main() {
  for (final status in ProjectOptionCatalogs.statuses) {
    testWidgets('restored drafts retain saved status (${status.value})', (
      tester,
    ) async {
      const adapter = ProjectDraftSnapshotAdapter();
      final savedFields = jsonDecode(
        jsonEncode(adapter.encodeKnownFields({'status': status.value})),
      ) as Map<String, dynamic>;
      final seed = LocalPostDraftSeed(
        draft: LocalPostDraft(
          id: 'status-draft',
          owner: AccountKey('did:plc:test'),
          kind: LocalPostDraftKind.project,
          createdAt: DateTime.utc(2026, 10, 23),
          updatedAt: DateTime.utc(2026, 10, 23),
          content: ProjectDraftContent(
            body: '',
            languages: const ['en'],
            knownProjectFieldValues: savedFields,
          ),
          schedule: const DraftScheduleIntent.now(),
          media: const [],
        ),
        media: const [],
      );
      await _pumpComposer(tester, 'restored-status-composer', draftSeed: seed);

      final statusButton = find.byKey(const Key('status-select-button'));
      expect(
        find.descendant(of: statusButton, matching: find.text(status.label)),
        findsOneWidget,
      );
      final form = tester.state<FormBuilderState>(find.byType(FormBuilder))
        ..save();
      expect(form.value['status'], status.value);
    });
  }

  testWidgets('new projects default to complete', (tester) async {
    await _pumpComposer(tester, 'default-status-composer');
    expect(
      tester
          .state<FormBuilderState>(find.byType(FormBuilder))
          .fields['status']!
          .value,
      ProjectOptionCatalogs.finishedStatusToken,
    );
  });

  for (final savedCraft in [ProjectOptionCatalogs.sewingCraftToken, null]) {
    testWidgets('restored drafts retain their saved selection ($savedCraft)', (
      tester,
    ) async {
      final seed = LocalPostDraftSeed(
        draft: LocalPostDraft(
          id: 'craft-draft',
          owner: AccountKey('did:plc:test'),
          kind: LocalPostDraftKind.project,
          createdAt: DateTime.utc(2026, 10, 22),
          updatedAt: DateTime.utc(2026, 10, 22),
          content: ProjectDraftContent(
            body: '',
            languages: const ['en'],
            knownProjectFieldValues: {'craftType': savedCraft},
          ),
          schedule: const DraftScheduleIntent.now(),
          media: const [],
        ),
        media: const [],
      );
      await _pumpComposer(
        tester,
        'restored-craft-composer',
        defaultCraft: ProjectOptionCatalogs.quiltingCraftToken,
        draftSeed: seed,
      );
      expect(
        tester
            .state<FormBuilderState>(find.byType(FormBuilder))
            .fields['craftType']!
            .value,
        savedCraft,
      );
    });
  }

  testWidgets('new projects preselect the default craft', (tester) async {
    await _pumpComposer(
      tester,
      'default-craft-composer',
      defaultCraft: ProjectOptionCatalogs.quiltingCraftToken,
    );
    expect(
      tester
          .state<FormBuilderState>(find.byType(FormBuilder))
          .fields['craftType']!
          .value,
      ProjectOptionCatalogs.quiltingCraftToken,
    );
    expect(
      find.byKey(const Key('project-composer-craft-details-action')),
      findsOneWidget,
    );
  });

  testWidgets('late preferences apply once and preserve a manual selection', (
    tester,
  ) async {
    final preference = ValueNotifier<String?>(null);
    addTearDown(preference.dispose);
    await _pumpComposer(tester, 'late-craft-composer', preference: preference);
    final container = ProviderScope.containerOf(
      tester.element(find.byType(ProjectComposerSheet)),
    );
    preference.value = ProjectOptionCatalogs.quiltingCraftToken;
    container.invalidate(defaultProjectCraftProvider);
    await tester.pumpAndSettle();
    FormBuilderState form() =>
        tester.state<FormBuilderState>(find.byType(FormBuilder));
    expect(
      form().fields['craftType']!.value,
      ProjectOptionCatalogs.quiltingCraftToken,
    );
    await _selectCraft(tester, 'Sewing');
    preference.value = ProjectOptionCatalogs.knittingCraftToken;
    container.invalidate(defaultProjectCraftProvider);
    await tester.pumpAndSettle();
    expect(
      form().fields['craftType']!.value,
      ProjectOptionCatalogs.sewingCraftToken,
    );
  });

  testWidgets('a manual selection before preferences load is retained', (
    tester,
  ) async {
    final preference = ValueNotifier<String?>(null);
    addTearDown(preference.dispose);
    await _pumpComposer(
      tester,
      'manual-craft-composer',
      preference: preference,
    );
    await _selectCraft(tester, 'Sewing');
    final container = ProviderScope.containerOf(
      tester.element(find.byType(ProjectComposerSheet)),
    );
    preference.value = ProjectOptionCatalogs.knittingCraftToken;
    container.invalidate(defaultProjectCraftProvider);
    await tester.pumpAndSettle();
    expect(
      tester
          .state<FormBuilderState>(find.byType(FormBuilder))
          .fields['craftType']!
          .value,
      ProjectOptionCatalogs.sewingCraftToken,
    );
  });

  testWidgets('sewing fit notes stop at the lexicon character boundary', (
    tester,
  ) async {
    await _pumpComposer(tester, 'fit-notes-composer');
    await _selectCraft(tester, 'Sewing');
    final action = find.byKey(
      const Key('project-composer-craft-details-action'),
    );
    await tester.ensureVisible(action);
    await tester.tap(action);
    await tester.pumpAndSettle();
    final field = find.widgetWithText(TextField, 'Add fit notes');
    await tester.ensureVisible(field);
    await tester.enterText(field, 'é' * 601);
    await tester.pump();
    expect(tester.widget<TextField>(field).controller!.text, 'é' * 600);
    expect(find.text('600/600'), findsOneWidget);
  });

  testWidgets('AT-003 project composer primary fields render on one page', (
    tester,
  ) async {
    await _pumpComposer(tester, 'sheet-composer');

    expect(find.text('Project post'), findsOneWidget);
    expect(
      tester
          .widget<CraftskyFormBuilderDropdownField<String>>(
            find.byWidgetPredicate(
              (widget) =>
                  widget is CraftskyFormBuilderDropdownField<String> &&
                  widget.name == 'craftType',
            ),
          )
          .label,
      'Primary craft',
    );
    expect(
      tester
          .widget<CraftskyFormBuilderMultiSelectField<String>>(
            find.byWidgetPredicate(
              (widget) =>
                  widget is CraftskyFormBuilderMultiSelectField<String> &&
                  widget.name == 'additionalCrafts',
            ),
          )
          .label,
      'Additional crafts',
    );
    expect(find.text('Project title'), findsOneWidget);
    expect(find.text('Pattern tag or name'), findsOneWidget);
    expect(find.text('Duration'), findsOneWidget);
    expect(find.text('e.g. 2 days or 4 weeks'), findsOneWidget);
    final duration = find.byKey(const Key('project-duration-input'));
    expect(
      tester.getTopLeft(duration).dy,
      greaterThan(
        tester
            .getBottomLeft(
              find.byKey(const Key('project-composer-pattern-name-field')),
            )
            .dy,
      ),
    );
    expect(
      tester.getBottomLeft(duration).dy,
      lessThan(
        tester
            .getTopLeft(find.byKey(const Key('project-composer-body-editor')))
            .dy,
      ),
    );
    expect(
      find.byKey(const Key('project-composer-body-editor')),
      findsOneWidget,
    );
    expect(
      find.byKey(const Key('project-composer-common-details-action')),
      findsOneWidget,
    );
    expect(find.widgetWithText(ChunkyButton, 'Post'), findsOneWidget);
    expect(find.text('Next'), findsNothing);
  });

  testWidgets('Post button clears the three-button navigation inset', (
    tester,
  ) async {
    const bottomInset = 24.0;
    const buttonGap = 16.0;
    await _pumpComposer(
      tester,
      'nav-inset-composer',
      bottomInset: bottomInset,
    );

    final buttonBottom = tester
        .getBottomLeft(
          find.byKey(const Key('project-composer-primary-action')),
        )
        .dy;
    final screenBottom =
        tester.view.physicalSize.height / tester.view.devicePixelRatio;
    expect(
      buttonBottom,
      lessThanOrEqualTo(screenBottom - bottomInset - buttonGap),
    );
  });

  testWidgets('AT-003 tapping scaffold space clears focused field', (
    tester,
  ) async {
    await _pumpComposer(tester, 'focus-composer');

    final bodyField = _bodyTextField();
    await tester.ensureVisible(bodyField);
    await tester.pumpAndSettle();
    await tester.tap(bodyField);
    await tester.pump();
    expect(tester.widget<TextField>(bodyField).focusNode?.hasFocus, isTrue);

    final bodyRect = tester.getRect(find.byType(SafeArea).first);
    await tester.tapAt(Offset(bodyRect.left + 8, bodyRect.top + 8));
    await tester.pump();

    expect(tester.widget<TextField>(bodyField).focusNode?.hasFocus, isFalse);
  });

  testWidgets('craft selector shows branded icons in options and selection', (
    tester,
  ) async {
    await _pumpComposer(
      tester,
      'craft-icon-composer',
      crafts: ['quilting', 'sewing'],
    );

    final craft = find.byKey(const Key('craftType-select-button'));
    await tester.ensureVisible(craft);
    await tester.pumpAndSettle();
    await tester.tap(craft);
    await tester.pumpAndSettle();

    expect(find.byType(CraftIcon), findsNWidgets(5));
    final labels = tester
        .widgetList<CraftIcon>(find.byType(CraftIcon))
        .map((widget) => widget.craft.split('#').last)
        .toList();
    expect(labels, ['quilting', 'sewing', 'knitting', 'crochet', 'embroidery']);

    await tester.tap(find.text('Knitting').last);
    await tester.pumpAndSettle();

    expect(find.byType(CraftIcon), findsOneWidget);
  });

  testWidgets('AT-003 primary fields use normal widget-order tab traversal', (
    tester,
  ) async {
    await _pumpComposer(tester, 'tab-composer');
    await _selectCraft(tester, 'Embroidery');

    final bodyField = _bodyTextField();
    final languageAction = find.byKey(
      const Key('composer-language-control'),
    );
    await tester.ensureVisible(bodyField);
    await tester.pumpAndSettle();
    await tester.tap(bodyField);
    await tester.pump();

    await tester.sendKeyEvent(LogicalKeyboardKey.tab);
    await tester.pump();

    expect(tester.widget<TextField>(bodyField).focusNode?.hasFocus, isFalse);
    final focusedWidget = FocusManager.instance.primaryFocus?.context?.widget;
    expect(focusedWidget, isNotNull);
    expect(
      find.ancestor(
        of: find.byWidget(focusedWidget!),
        matching: languageAction,
      ),
      findsOneWidget,
    );
  });
}

Future<void> _pumpComposer(
  WidgetTester tester,
  String composerId, {
  double bottomInset = 0,
  String? defaultCraft,
  LocalPostDraftSeed? draftSeed,
  ValueNotifier<String?>? preference,
  List<String> crafts = const [],
}) async {
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        defaultProjectCraftProvider.overrideWith(
          (ref) => preference?.value ?? defaultCraft,
        ),
        publishingAnimationFileProvider.overrideWith(
          (ref) => Completer<rive.File>().future,
        ),
        preferredProjectCraftsProvider.overrideWithValue(
          ProjectOptionCatalogs.craftTypesForPreferences(crafts),
        ),
        activeLanguagePreferencesProvider.overrideWith(
          (ref) => const LanguagePreferences(
            primaryLanguage: 'en',
            contentLanguages: ['en'],
          ),
        ),
        if (draftSeed == null)
          composerImagesProvider(composerId)
              .overrideWithValue(_readyImagesState),
      ],
      child: MessengerScope(
        messenger: RecordingMessenger(),
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (context) => MediaQuery(
              data: MediaQuery.of(context).copyWith(
                padding: EdgeInsets.only(bottom: bottomInset),
                viewPadding: EdgeInsets.only(bottom: bottomInset),
              ),
              child: ProjectComposerSheet(
                composerId: composerId,
                draftSeed: draftSeed,
              ),
            ),
          ),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

Future<void> _selectCraft(WidgetTester tester, String craftLabel) async {
  final craft = find.byKey(const Key('craftType-select-button'));
  await tester.ensureVisible(craft);
  await tester.pumpAndSettle();
  await tester.tap(craft);
  await tester.pumpAndSettle();
  await tester.tap(find.text(craftLabel).last);
  await tester.pumpAndSettle();
}

Finder _bodyTextField() => find.descendant(
  of: find.byKey(const Key('project-composer-body-editor')),
  matching: find.byType(TextField),
);

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
