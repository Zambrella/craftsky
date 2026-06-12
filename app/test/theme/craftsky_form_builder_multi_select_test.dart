import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_form_builder_select_fields.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('UT-004 free-text multi-select saves chips and enforces max', (
    tester,
  ) async {
    final formKey = GlobalKey<FormBuilderState>();

    await tester.pumpWidget(
      _Harness(
        child: FormBuilder(
          key: formKey,
          child: const CraftskyFormBuilderMultiSelectField<String>(
            name: 'materials',
            label: 'Materials',
            helperText: 'Add up to 2 materials',
            allowCustomValues: true,
            maxSelected: 2,
            customValueHintText: 'Add material',
            addCustomValueLabel: 'Add',
            maxSelectedErrorText: 'Choose no more than 2 materials.',
          ),
        ),
      ),
    );

    expect(find.text('Add material'), findsOneWidget);
    expect(find.text('Add'), findsOneWidget);
    expect(
      tester
          .widget<TextButton>(find.byKey(const Key('materials-add-custom')))
          .onPressed,
      isNull,
    );

    await tester.enterText(
      find.byKey(const Key('materials-custom-input')),
      'linen',
    );
    await tester.pump();
    expect(
      tester
          .widget<TextButton>(find.byKey(const Key('materials-add-custom')))
          .onPressed,
      isNotNull,
    );
    await tester.tap(find.byKey(const Key('materials-add-custom')));
    await tester.pump();

    await _addCustom(tester, 'materials', 'cotton');
    await _addCustom(tester, 'materials', 'wool');

    expect(formKey.currentState!.instantValue['materials'], [
      'linen',
      'cotton',
    ]);
    expect(find.text('linen'), findsOneWidget);
    expect(find.text('cotton'), findsOneWidget);
    expect(find.text('Choose no more than 2 materials.'), findsOneWidget);

    await tester.tap(find.byKey(const Key('materials-remove-linen')));
    await tester.pump();

    expect(formKey.currentState!.instantValue['materials'], ['cotton']);
    expect(find.text('linen'), findsNothing);
  });

  testWidgets('UT-004 known-option multi-select searches and saves strings', (
    tester,
  ) async {
    final formKey = GlobalKey<FormBuilderState>();

    await tester.pumpWidget(
      _Harness(
        child: FormBuilder(
          key: formKey,
          child: const CraftskyFormBuilderMultiSelectField<String>(
            name: 'colours',
            label: 'Colours',
            initialValue: ['blue'],
            maxSelected: 2,
            searchHintText: 'Search colours',
            options: [
              CraftskySelectOption(value: 'blue', label: 'Blue'),
              CraftskySelectOption(value: 'cream', label: 'Cream'),
              CraftskySelectOption(value: 'red', label: 'Red'),
            ],
          ),
        ),
      ),
    );

    expect(formKey.currentState!.instantValue['colours'], ['blue']);
    expect(find.text('Blue'), findsOneWidget);
    expect(find.byKey(const Key('colours-search-input')), findsOneWidget);
    expect(find.byKey(const Key('colours-option-red')), findsNothing);

    await tester.tap(find.byKey(const Key('colours-search-input')));
    await tester.pumpAndSettle();
    expect(find.text('Search colours'), findsOneWidget);
    expect(find.byType(Divider), findsNothing);
    expect(
      tester
          .widget<TextField>(find.byKey(const Key('colours-search-input')))
          .focusNode
          ?.hasFocus,
      isTrue,
    );
    expect(find.byKey(const Key('colours-option-cream')), findsOneWidget);
    expect(find.byKey(const Key('colours-option-red')), findsOneWidget);

    await tester.sendKeyEvent(LogicalKeyboardKey.escape);
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('colours-search-input')), findsOneWidget);
    expect(find.byKey(const Key('colours-option-cream')), findsNothing);

    await tester.tap(find.byKey(const Key('colours-search-input')));
    await tester.pumpAndSettle();

    await tester.enterText(
      find.byKey(const Key('colours-search-input')),
      'cre',
    );
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('colours-option-cream')), findsOneWidget);
    expect(find.byKey(const Key('colours-option-red')), findsNothing);
    expect(find.byType(CheckboxListTile), findsOneWidget);

    await tester.tap(find.byKey(const Key('colours-option-cream')));
    await tester.pumpAndSettle();
    expect(formKey.currentState!.instantValue['colours'], ['blue', 'cream']);
    expect(find.byKey(const Key('colours-option-cream')), findsNothing);
    final searchField = tester.widget<TextField>(
      find.byKey(const Key('colours-search-input')),
    );
    expect(searchField.controller?.text, isEmpty);
    expect(searchField.focusNode?.hasFocus, isTrue);
    expect(find.byKey(const Key('colours-option-blue')), findsNothing);

    await tester.enterText(
      find.byKey(const Key('colours-search-input')),
      'cre',
    );
    await tester.pumpAndSettle();
    expect(
      tester
          .widget<CheckboxListTile>(
            find.byKey(const Key('colours-option-cream')),
          )
          .value,
      isTrue,
    );
    final selectedTile = tester.widget<CheckboxListTile>(
      find.byKey(const Key('colours-option-cream')),
    );
    expect(selectedTile.value, isTrue);

    await tester.enterText(
      find.byKey(const Key('colours-search-input')),
      'blue',
    );
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('colours-option-blue')), findsOneWidget);

    await tester.tapAt(Offset.zero);
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('colours-remove-blue')));
    await tester.pump();
    expect(formKey.currentState!.instantValue['colours'], ['cream']);
  });

  testWidgets(
    'UT-004 known-option search is inline and opens options after typing',
    (
      tester,
    ) async {
      await tester.pumpWidget(
        const _Harness(
          child: FormBuilder(
            child: CraftskyFormBuilderMultiSelectField<String>(
              name: 'designTags',
              label: 'Design tags',
              initialValue: ['floral'],
              searchHintText: 'Search design tags',
              options: [
                CraftskySelectOption(value: 'floral', label: 'Floral'),
                CraftskySelectOption(value: 'striped', label: 'Striped'),
              ],
            ),
          ),
        ),
      );

      expect(find.byKey(const Key('designTags-search-input')), findsOneWidget);
      expect(find.text('Floral'), findsOneWidget);

      await tester.tap(find.byKey(const Key('designTags-search-input')));
      await tester.pumpAndSettle();

      expect(find.byKey(const Key('designTags-search-input')), findsOneWidget);
      expect(
        find.byKey(const Key('designTags-option-striped')),
        findsOneWidget,
      );

      await tester.enterText(
        find.byKey(const Key('designTags-search-input')),
        'str',
      );
      await tester.pumpAndSettle();

      expect(
        find.byKey(const Key('designTags-option-striped')),
        findsOneWidget,
      );
    },
  );

  testWidgets(
    'UT-004 inline search enter with empty text selects first option',
    (
      tester,
    ) async {
      final formKey = GlobalKey<FormBuilderState>();

      await tester.pumpWidget(
        _Harness(
          child: FormBuilder(
            key: formKey,
            child: const CraftskyFormBuilderMultiSelectField<String>(
              name: 'colours',
              label: 'Colours',
              searchHintText: 'Search colours',
              options: [
                CraftskySelectOption(value: 'blue', label: 'Blue'),
                CraftskySelectOption(value: 'cream', label: 'Cream'),
              ],
            ),
          ),
        ),
      );

      await tester.tap(find.byKey(const Key('colours-search-input')));
      await tester.sendKeyEvent(LogicalKeyboardKey.enter);
      await tester.pump();

      expect(formKey.currentState!.instantValue['colours'], ['blue']);
      expect(find.byKey(const Key('colours-option-blue')), findsNothing);
    },
  );

  testWidgets('UT-004 inline search shows no results for unmatched text', (
    tester,
  ) async {
    await tester.pumpWidget(
      const _Harness(
        child: FormBuilder(
          child: CraftskyFormBuilderMultiSelectField<String>(
            name: 'colours',
            label: 'Colours',
            searchHintText: 'Search colours',
            options: [
              CraftskySelectOption(value: 'blue', label: 'Blue'),
              CraftskySelectOption(value: 'cream', label: 'Cream'),
            ],
          ),
        ),
      ),
    );

    await tester.tap(find.byKey(const Key('colours-search-input')));
    await tester.enterText(
      find.byKey(const Key('colours-search-input')),
      'zz',
    );
    await tester.pumpAndSettle();

    expect(find.text('No results'), findsOneWidget);
    expect(find.byKey(const Key('colours-option-blue')), findsNothing);
  });

  testWidgets('UT-004 inline search enter selects and keeps focus', (
    tester,
  ) async {
    final formKey = GlobalKey<FormBuilderState>();

    await tester.pumpWidget(
      _Harness(
        child: FormBuilder(
          key: formKey,
          child: const CraftskyFormBuilderMultiSelectField<String>(
            name: 'colours',
            label: 'Colours',
            searchHintText: 'Search colours',
            options: [
              CraftskySelectOption(value: 'blue', label: 'Blue'),
              CraftskySelectOption(value: 'cream', label: 'Cream'),
            ],
          ),
        ),
      ),
    );

    await tester.tap(find.byKey(const Key('colours-search-input')));
    await tester.enterText(
      find.byKey(const Key('colours-search-input')),
      'cre',
    );
    await tester.sendKeyEvent(LogicalKeyboardKey.enter);
    await tester.pump();

    expect(formKey.currentState!.instantValue['colours'], ['cream']);
    expect(
      tester
          .widget<TextField>(find.byKey(const Key('colours-search-input')))
          .focusNode
          ?.hasFocus,
      isTrue,
    );
  });

  testWidgets('UT-004 inline search tabs to next field once', (tester) async {
    final firstFocusNode = FocusNode(debugLabel: 'first-field');
    final nextFocusNode = FocusNode(debugLabel: 'next-field');
    addTearDown(firstFocusNode.dispose);
    addTearDown(nextFocusNode.dispose);

    await tester.pumpWidget(
      _Harness(
        child: FormBuilder(
          child: Column(
            children: [
              TextField(
                key: const Key('first-field'),
                focusNode: firstFocusNode,
              ),
              const SizedBox(height: 24),
              const CraftskyFormBuilderMultiSelectField<String>(
                name: 'colours',
                label: 'Colours',
                searchHintText: 'Search colours',
                options: [
                  CraftskySelectOption(value: 'blue', label: 'Blue'),
                  CraftskySelectOption(value: 'cream', label: 'Cream'),
                ],
              ),
              const SizedBox(height: 24),
              TextField(
                key: const Key('next-field'),
                focusNode: nextFocusNode,
              ),
            ],
          ),
        ),
      ),
    );

    firstFocusNode.requestFocus();
    await tester.pump();
    await tester.sendKeyEvent(LogicalKeyboardKey.tab);
    await tester.pump();
    expect(nextFocusNode.hasFocus, isFalse);

    await tester.sendKeyEvent(LogicalKeyboardKey.tab);
    await tester.pumpAndSettle();

    expect(find.byKey(const Key('colours-options-panel')), findsNothing);
    expect(nextFocusNode.hasFocus, isTrue);
  });

  testWidgets('UT-004 disabling an open multi-select closes its overlay', (
    tester,
  ) async {
    var enabled = true;

    Widget buildSubject() {
      return _Harness(
        child: FormBuilder(
          child: CraftskyFormBuilderMultiSelectField<String>(
            name: 'colours',
            label: 'Colours',
            enabled: enabled,
            searchHintText: 'Search colours',
            options: const [
              CraftskySelectOption(value: 'blue', label: 'Blue'),
              CraftskySelectOption(value: 'cream', label: 'Cream'),
            ],
          ),
        ),
      );
    }

    await tester.pumpWidget(buildSubject());
    await tester.tap(find.byKey(const Key('colours-search-input')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('colours-options-panel')), findsOneWidget);

    enabled = false;
    await tester.pumpWidget(buildSubject());
    await tester.pumpAndSettle();

    expect(find.byKey(const Key('colours-options-panel')), findsNothing);

    await tester.tap(find.byKey(const Key('colours-search-input')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('colours-options-panel')), findsNothing);
  });

  testWidgets('UT-004 known-option max error clears after parent reset', (
    tester,
  ) async {
    var values = <String>['blue'];
    late StateSetter setHarnessState;

    await tester.pumpWidget(
      _Harness(
        child: StatefulBuilder(
          builder: (context, setState) {
            setHarnessState = setState;
            return CraftskySearchableMultiSelectInput<String>(
              label: 'Colours',
              values: values,
              maxSelected: 1,
              maxSelectedErrorText: 'Choose no more than 1 colour.',
              searchHintText: 'Search colours',
              options: const [
                CraftskySelectOption(value: 'blue', label: 'Blue'),
                CraftskySelectOption(value: 'cream', label: 'Cream'),
              ],
              onChanged: (nextValues) {
                setHarnessState(() => values = nextValues);
              },
            );
          },
        ),
      ),
    );

    await tester.tap(find.byKey(const Key('Colours-search-input')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('Colours-option-cream')));
    await tester.pump();
    expect(find.text('Choose no more than 1 colour.'), findsOneWidget);

    setHarnessState(() => values = const []);
    await tester.pump();

    expect(find.text('Choose no more than 1 colour.'), findsNothing);
  });

  testWidgets('UT-004 token enter adds value and keeps focus', (tester) async {
    final formKey = GlobalKey<FormBuilderState>();

    await tester.pumpWidget(
      _Harness(
        child: FormBuilder(
          key: formKey,
          child: const CraftskyFormBuilderMultiSelectField<String>(
            name: 'materials',
            label: 'Materials',
            allowCustomValues: true,
            customValueHintText: 'Add material',
          ),
        ),
      ),
    );

    await tester.tap(find.byKey(const Key('materials-custom-input')));
    await tester.enterText(
      find.byKey(const Key('materials-custom-input')),
      'linen',
    );
    await tester.testTextInput.receiveAction(TextInputAction.done);
    await tester.pump();

    expect(formKey.currentState!.instantValue['materials'], ['linen']);
    expect(
      tester
          .widget<TextField>(find.byKey(const Key('materials-custom-input')))
          .focusNode
          ?.hasFocus,
      isTrue,
    );
  });

  testWidgets('UT-004 token max error clears after parent reset', (
    tester,
  ) async {
    var values = <String>['linen'];
    late StateSetter setHarnessState;

    await tester.pumpWidget(
      _Harness(
        child: StatefulBuilder(
          builder: (context, setState) {
            setHarnessState = setState;
            return CraftskyTokenInput(
              label: 'Materials',
              values: values,
              maxSelected: 1,
              maxSelectedErrorText: 'Choose no more than 1 material.',
              inputHintText: 'Add material',
              onChanged: (nextValues) {
                setHarnessState(() => values = nextValues);
              },
            );
          },
        ),
      ),
    );

    await tester.enterText(
      find.byKey(const Key('Materials-custom-input')),
      'cotton',
    );
    await tester.pump();
    await tester.tap(find.byKey(const Key('Materials-add-custom')));
    await tester.pump();
    expect(find.text('Choose no more than 1 material.'), findsOneWidget);

    setHarnessState(() => values = const []);
    await tester.pump();

    expect(find.text('Choose no more than 1 material.'), findsNothing);
  });

  testWidgets('UT-004 custom values require string multi-selects', (
    tester,
  ) async {
    expect(
      () => CraftskyFormBuilderMultiSelectField<int>(
        name: 'numbers',
        label: 'Numbers',
        allowCustomValues: true,
      ),
      throwsAssertionError,
    );
  });

  testWidgets('UT-016 multi-select disabled copy is supplied by caller', (
    tester,
  ) async {
    await tester.pumpWidget(
      const _Harness(
        child: FormBuilder(
          child: CraftskyFormBuilderMultiSelectField<String>(
            name: 'materials',
            label: 'Materials',
            enabled: false,
            disabledText: 'Unavailable',
          ),
        ),
      ),
    );

    expect(find.text('Unavailable'), findsOneWidget);
    expect(find.text('Disabled'), findsNothing);
  });
}

Future<void> _addCustom(WidgetTester tester, String name, String value) async {
  await tester.enterText(find.byKey(Key('$name-custom-input')), value);
  await tester.pump();
  await tester.tap(find.byKey(Key('$name-add-custom')));
  await tester.pump();
}

class _Harness extends StatelessWidget {
  const _Harness({required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      theme: AppTheme.lightThemeData,
      home: Scaffold(
        body: Padding(padding: const EdgeInsets.all(24), child: child),
      ),
    );
  }
}
