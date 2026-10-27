import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/widgets/project_materials_field.dart';
import 'package:craftsky_app/shared/rich_text/data/mock_facet_suggestion_repository.dart';
import 'package:craftsky_app/shared/rich_text/providers/facet_suggestion_providers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  testWidgets('replacing all material text retains its source entry identity', (
    tester,
  ) async {
    final form = await _pumpField(
      tester,
      materials: const [
        ProjectMaterial(text: 'Wool', extraFields: {'existingIndex': 0}),
      ],
    );
    await tester.tap(find.byTooltip('Edit material'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(EditableText), '');
    await tester.pump();
    await tester.enterText(find.byType(EditableText), 'Linen');
    await tester.pump();
    await tester.tap(find.byKey(const Key('materials-add-custom')));
    await tester.pumpAndSettle();
    expect(_materials(form).single.text, 'Linen');
    expect(_materials(form).single.extraFields, {'existingIndex': 0});
  });

  testWidgets(
    'editing a published material retains its source entry identity',
    (
      tester,
    ) async {
      final form = await _pumpField(
        tester,
        materials: const [
          ProjectMaterial(text: 'Wool', extraFields: {'existingIndex': 0}),
        ],
      );
      await tester.tap(find.byTooltip('Edit material'));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(EditableText), 'Merino wool');
      await tester.tap(find.byKey(const Key('materials-add-custom')));
      await tester.pumpAndSettle();
      expect(_materials(form).single.text, 'Merino wool');
      expect(_materials(form).single.extraFields, {'existingIndex': 0});

      await tester.enterText(find.byType(EditableText), 'Cotton');
      await tester.pump();
      await tester.tap(find.byKey(const Key('materials-add-custom')));
      await tester.pumpAndSettle();
      expect(_materials(form).last.text, 'Cotton');
      expect(_materials(form).last.extraFields, isEmpty);
    },
  );

  testWidgets('editing another material cannot overwrite the current edit', (
    tester,
  ) async {
    final form = await _pumpField(tester);
    await tester.tap(find.byKey(const Key('materials-edit-Wool')));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(EditableText), 'Merino wool');

    final editCotton = find.byKey(const Key('materials-edit-Cotton'));
    expect(tester.widget<IconButton>(editCotton).onPressed, isNull);
    expect(
      tester.widget<IconButton>(editCotton).tooltip,
      'Add or clear the current material first',
    );
    await tester.tap(editCotton);
    await tester.pumpAndSettle();
    expect(
      tester.widget<EditableText>(find.byType(EditableText)).controller.text,
      'Merino wool',
    );
    expect(_materials(form).map((value) => value.text), ['Cotton']);

    await tester.tap(find.byKey(const Key('materials-add-custom')));
    await tester.pumpAndSettle();
    expect(tester.widget<IconButton>(editCotton).onPressed, isNotNull);
    await tester.tap(editCotton);
    await tester.pumpAndSettle();
    expect(_materials(form).map((value) => value.text), ['Merino wool']);
    expect(
      tester.widget<EditableText>(find.byType(EditableText)).controller.text,
      'Cotton',
    );
  });

  testWidgets('new input blocks editing until cleared, including whitespace', (
    tester,
  ) async {
    final form = await _pumpField(tester);
    final editWool = find.byKey(const Key('materials-edit-Wool'));
    for (final text in ['New material', ' ']) {
      await tester.enterText(find.byType(EditableText), text);
      await tester.pump();
      expect(tester.widget<IconButton>(editWool).onPressed, isNull);
      expect(
        tester
            .widget<IconButton>(
              find.byKey(const Key('materials-remove-Wool')),
            )
            .onPressed,
        isNotNull,
      );
    }
    await tester.enterText(find.byType(EditableText), '');
    await tester.pump();
    expect(tester.widget<IconButton>(editWool).onPressed, isNotNull);
    await tester.tap(editWool);
    await tester.pumpAndSettle();
    expect(_materials(form).map((value) => value.text), ['Cotton']);
    expect(
      tester.widget<EditableText>(find.byType(EditableText)).controller.text,
      'Wool',
    );
  });

  testWidgets('edit moves a material into the input and Add saves the change', (
    tester,
  ) async {
    final form = await _pumpField(tester);
    await tester.tap(find.byTooltip('Edit material').first);
    await tester.pumpAndSettle();

    expect(_materials(form).map((value) => value.text), ['Cotton']);
    expect(find.byKey(const Key('materials-remove-Wool')), findsNothing);
    final input = tester.widget<EditableText>(find.byType(EditableText));
    expect(input.controller.text, 'Wool');
    expect(
      input.controller.selection,
      const TextSelection.collapsed(offset: 4),
    );
    expect(input.focusNode.hasFocus, isTrue);

    await tester.enterText(find.byType(EditableText), 'Merino wool #knitting');
    await tester.tap(find.byKey(const Key('materials-add-custom')));
    await tester.pumpAndSettle();

    expect(_materials(form).map((value) => value.text), [
      'Cotton',
      'Merino wool #knitting',
    ]);
    expect(_materials(form).last.facets, isNotEmpty);
    expect(input.controller.text, isEmpty);
    expect(
      find.byKey(const Key('materials-remove-Merino wool #knitting')),
      findsOneWidget,
    );
  });

  testWidgets('disabled materials cannot be edited', (tester) async {
    final form = await _pumpField(tester, enabled: false);
    for (final button in tester.widgetList<IconButton>(
      find.byWidgetPredicate(
        (widget) => widget is IconButton && widget.tooltip == 'Edit material',
      ),
    )) {
      expect(button.onPressed, isNull);
    }
    expect(find.byTooltip('Edit material'), findsNWidgets(2));
    expect(_materials(form).map((value) => value.text), ['Wool', 'Cotton']);
  });

  testWidgets('a full material list can still be edited', (tester) async {
    final form = await _pumpField(
      tester,
      materials: [
        for (var i = 0; i < 10; i++) ProjectMaterial(text: 'Yarn $i'),
      ],
    );
    await tester.tap(find.byTooltip('Edit material').first);
    await tester.pumpAndSettle();
    expect(_materials(form), hasLength(9));
    await tester.enterText(find.byType(EditableText), 'Updated yarn');
    final add = find.byKey(const Key('materials-add-custom'));
    await tester.ensureVisible(add);
    await tester.tap(add);
    await tester.pumpAndSettle();
    expect(_materials(form), hasLength(10));
    expect(_materials(form).last.text, 'Updated yarn');
  });
}

List<ProjectMaterial> _materials(GlobalKey<FormBuilderState> form) =>
    form.currentState!.fields['materials']!.value as List<ProjectMaterial>;

Future<GlobalKey<FormBuilderState>> _pumpField(
  WidgetTester tester, {
  bool enabled = true,
  List<ProjectMaterial> materials = const [
    ProjectMaterial(text: 'Wool'),
    ProjectMaterial(text: 'Cotton'),
  ],
}) async {
  final form = GlobalKey<FormBuilderState>();
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        accountSuggestionRepositoryProvider.overrideWithValue(
          const MockAccountSuggestionRepository(accounts: []),
        ),
        hashtagSuggestionRepositoryProvider.overrideWithValue(
          const MockHashtagSuggestionRepository(hashtags: []),
        ),
      ],
      child: MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: appLocalizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: SingleChildScrollView(
            child: FormBuilder(
              key: form,
              initialValue: {'materials': materials},
              child: ProjectMaterialsField(
                name: 'materials',
                label: 'Materials',
                inputHintText: 'Add a material',
                addButtonLabel: 'Add',
                disabledText: 'Disabled',
                maxSelectedErrorText: 'Too many materials',
                maxLengthErrorText: 'Material too long',
                enabled: enabled,
              ),
            ),
          ),
        ),
      ),
    ),
  );
  return form;
}
