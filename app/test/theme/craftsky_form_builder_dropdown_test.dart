import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_form_builder_select_fields.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('UT-003 dropdown saves, validates, resets and disables', (
    tester,
  ) async {
    final formKey = GlobalKey<FormBuilderState>();
    String? changed;

    await tester.pumpWidget(
      _Harness(
        child: FormBuilder(
          key: formKey,
          child: Column(
            children: [
              const TextField(),
              const SizedBox(height: 24),
              CraftskyFormBuilderDropdownField<String>(
                name: 'craftType',
                label: 'Craft type',
                helperText: 'Choose the closest craft',
                initialValue: 'knitting',
                options: const [
                  CraftskySelectOption(value: 'knitting', label: 'Knitting'),
                  CraftskySelectOption(value: 'crochet', label: 'Crochet'),
                ],
                validator: (value) => value == null ? 'Choose a craft' : null,
                onChanged: (value) => changed = value,
              ),
            ],
          ),
        ),
      ),
    );

    expect(find.text('Craft type'), findsOneWidget);
    expect(find.text('Choose the closest craft'), findsOneWidget);
    expect(find.byKey(const Key('craftType-search-input')), findsNothing);
    expect(formKey.currentState!.instantValue['craftType'], 'knitting');

    await tester.tap(find.text('Knitting'));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('craftType-search-input')), findsNothing);

    await tester.sendKeyEvent(LogicalKeyboardKey.escape);
    await tester.pumpAndSettle();
    expect(find.text('Crochet'), findsNothing);

    await tester.tap(find.text('Knitting'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Crochet').last);
    await tester.pumpAndSettle();

    expect(changed, 'crochet');
    expect(formKey.currentState!.instantValue['craftType'], 'crochet');
    expect(formKey.currentState!.saveAndValidate(), isTrue);

    formKey.currentState!.fields['craftType']!.didChange(null);
    expect(formKey.currentState!.saveAndValidate(), isFalse);
    await tester.pump();
    expect(find.text('Choose a craft'), findsOneWidget);

    formKey.currentState!.reset();
    await tester.pump();
    expect(formKey.currentState!.instantValue['craftType'], 'knitting');
  });

  testWidgets('UT-003 disabled dropdown prevents changes', (tester) async {
    final formKey = GlobalKey<FormBuilderState>();

    await tester.pumpWidget(
      _Harness(
        child: FormBuilder(
          key: formKey,
          child: const CraftskyFormBuilderDropdownField<String>(
            name: 'status',
            label: 'Status',
            initialValue: 'finished',
            enabled: false,
            options: [
              CraftskySelectOption(value: 'finished', label: 'Finished'),
              CraftskySelectOption(value: 'wip', label: 'Work in progress'),
            ],
          ),
        ),
      ),
    );

    await tester.tap(find.text('Finished'));
    await tester.pumpAndSettle();

    expect(find.text('Work in progress'), findsNothing);
    expect(formKey.currentState!.instantValue['status'], 'finished');
  });

  testWidgets('UT-003 disabling an open dropdown closes its overlay', (
    tester,
  ) async {
    var enabled = true;

    Widget buildSubject() {
      return _Harness(
        child: FormBuilder(
          child: CraftskyFormBuilderDropdownField<String>(
            name: 'status',
            label: 'Status',
            initialValue: 'finished',
            enabled: enabled,
            options: const [
              CraftskySelectOption(value: 'finished', label: 'Finished'),
              CraftskySelectOption(value: 'wip', label: 'Work in progress'),
            ],
          ),
        ),
      );
    }

    await tester.pumpWidget(buildSubject());
    await tester.tap(find.byKey(const Key('status-select-button')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('status-options-panel')), findsOneWidget);

    enabled = false;
    await tester.pumpWidget(buildSubject());
    await tester.pumpAndSettle();

    expect(find.byKey(const Key('status-options-panel')), findsNothing);

    await tester.tap(find.byKey(const Key('status-select-button')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('status-options-panel')), findsNothing);
  });

  testWidgets('UT-003 disabled dropdown is skipped by tab traversal', (
    tester,
  ) async {
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
              const CraftskyFormBuilderDropdownField<String>(
                name: 'projectSubtype',
                label: 'Project subtype',
                enabled: false,
                options: [],
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

    expect(nextFocusNode.hasFocus, isTrue);
    expect(find.byKey(const Key('projectSubtype-options-panel')), findsNothing);
  });

  testWidgets('UT-003 empty dropdowns show select placeholder text', (
    tester,
  ) async {
    await tester.pumpWidget(
      const _Harness(
        child: FormBuilder(
          child: Column(
            children: [
              CraftskyFormBuilderDropdownField<String>(
                name: 'status',
                label: 'Status',
                options: [
                  CraftskySelectOption(value: 'wip', label: 'Work in progress'),
                  CraftskySelectOption(value: 'finished', label: 'Finished'),
                ],
              ),
              SizedBox(height: 24),
              CraftskyFormBuilderDropdownField<String>(
                name: 'needleSize',
                label: 'Needle size',
                options: [
                  CraftskySelectOption(value: '2.0mm', label: '2.0mm'),
                  CraftskySelectOption(value: '3.0mm', label: '3.0mm'),
                  CraftskySelectOption(value: '4.0mm', label: '4.0mm'),
                  CraftskySelectOption(value: '5.0mm', label: '5.0mm'),
                  CraftskySelectOption(value: '6.0mm', label: '6.0mm'),
                  CraftskySelectOption(value: '7.0mm', label: '7.0mm'),
                ],
              ),
            ],
          ),
        ),
      ),
    );

    expect(find.text('Select Status'), findsOneWidget);
    expect(
      tester
          .widget<TextField>(find.byKey(const Key('needleSize-search-input')))
          .decoration
          ?.hintText,
      'Select Needle size',
    );
  });

  testWidgets('UT-003 focused dropdowns use focused input decoration', (
    tester,
  ) async {
    await tester.pumpWidget(
      const _Harness(
        child: FormBuilder(
          child: Column(
            children: [
              CraftskyFormBuilderDropdownField<String>(
                name: 'status',
                label: 'Status',
                options: [
                  CraftskySelectOption(value: 'wip', label: 'Work in progress'),
                  CraftskySelectOption(value: 'finished', label: 'Finished'),
                ],
              ),
              SizedBox(height: 24),
              CraftskyFormBuilderDropdownField<String>(
                name: 'needleSize',
                label: 'Needle size',
                options: [
                  CraftskySelectOption(value: '2.0mm', label: '2.0mm'),
                  CraftskySelectOption(value: '3.0mm', label: '3.0mm'),
                  CraftskySelectOption(value: '4.0mm', label: '4.0mm'),
                  CraftskySelectOption(value: '5.0mm', label: '5.0mm'),
                  CraftskySelectOption(value: '6.0mm', label: '6.0mm'),
                  CraftskySelectOption(value: '7.0mm', label: '7.0mm'),
                ],
              ),
            ],
          ),
        ),
      ),
    );

    await tester.tap(find.byKey(const Key('status-select-button')));
    await tester.pumpAndSettle();
    final statusDecorator = tester.widget<InputDecorator>(
      find.descendant(
        of: find.byKey(const Key('status-select-button')),
        matching: find.byType(InputDecorator),
      ),
    );
    expect(statusDecorator.isFocused, isTrue);

    await tester.sendKeyEvent(LogicalKeyboardKey.escape);
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('needleSize-search-input')));
    await tester.pumpAndSettle();
    final needleDecorator = tester.widget<InputDecorator>(
      find.byKey(const Key('needleSize-select-button')),
    );
    expect(needleDecorator.isFocused, isTrue);
  });

  testWidgets('UT-003 dropdown uses inline search above five options', (
    tester,
  ) async {
    final formKey = GlobalKey<FormBuilderState>();

    await tester.pumpWidget(
      _Harness(
        child: FormBuilder(
          key: formKey,
          child: const CraftskyFormBuilderDropdownField<String>(
            name: 'needleSize',
            label: 'Needle size',
            initialValue: '4.0mm',
            options: [
              CraftskySelectOption(value: '2.0mm', label: '2.0mm'),
              CraftskySelectOption(value: '3.0mm', label: '3.0mm'),
              CraftskySelectOption(value: '4.0mm', label: '4.0mm'),
              CraftskySelectOption(value: '5.0mm', label: '5.0mm'),
              CraftskySelectOption(value: '6.0mm', label: '6.0mm'),
              CraftskySelectOption(value: '7.0mm', label: '7.0mm'),
            ],
          ),
        ),
      ),
    );

    expect(find.byKey(const Key('needleSize-search-input')), findsOneWidget);
    expect(find.byIcon(Icons.search), findsOneWidget);
    expect(find.byKey(const Key('needleSize-options-panel')), findsNothing);

    await tester.tap(find.byKey(const Key('needleSize-search-input')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('needleSize-search-input')), findsOneWidget);
    expect(find.byKey(const Key('needleSize-options-panel')), findsOneWidget);
    expect(find.byKey(const Key('needleSize-option-2.0mm')), findsOneWidget);
    final firstTile = tester.widget<ListTile>(
      find.byKey(const Key('needleSize-option-2.0mm')),
    );
    expect(firstTile.selected, isTrue);

    await tester.sendKeyEvent(LogicalKeyboardKey.enter);
    await tester.pumpAndSettle();
    expect(formKey.currentState!.instantValue['needleSize'], '2.0mm');
    expect(find.byKey(const Key('needleSize-options-panel')), findsNothing);

    await tester.enterText(
      find.byKey(const Key('needleSize-search-input')),
      '7',
    );
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('needleSize-options-panel')), findsOneWidget);
    expect(find.byType(Divider), findsNothing);
    expect(
      tester
          .widget<TextField>(find.byKey(const Key('needleSize-search-input')))
          .focusNode
          ?.hasFocus,
      isTrue,
    );
    final highlightedTile = tester.widget<ListTile>(
      find.byKey(const Key('needleSize-option-7.0mm')),
    );
    expect(highlightedTile.selected, isTrue);

    await tester.sendKeyEvent(LogicalKeyboardKey.escape);
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('needleSize-search-input')), findsOneWidget);
    expect(find.byKey(const Key('needleSize-options-panel')), findsNothing);

    await tester.enterText(
      find.byKey(const Key('needleSize-search-input')),
      '7',
    );
    await tester.pumpAndSettle();
    await tester.sendKeyEvent(LogicalKeyboardKey.enter);
    await tester.pumpAndSettle();

    expect(formKey.currentState!.instantValue['needleSize'], '7.0mm');
  });

  testWidgets('UT-003 dropdown overlay opens below the anchor', (
    tester,
  ) async {
    await tester.pumpWidget(
      const _Harness(
        child: FormBuilder(
          child: CraftskyFormBuilderDropdownField<String>(
            name: 'needleSize',
            label: 'Needle size',
            initialValue: '4.0mm',
            options: [
              CraftskySelectOption(value: '2.0mm', label: '2.0mm'),
              CraftskySelectOption(value: '3.0mm', label: '3.0mm'),
              CraftskySelectOption(value: '4.0mm', label: '4.0mm'),
              CraftskySelectOption(value: '5.0mm', label: '5.0mm'),
              CraftskySelectOption(value: '6.0mm', label: '6.0mm'),
              CraftskySelectOption(value: '7.0mm', label: '7.0mm'),
            ],
          ),
        ),
      ),
    );

    await tester.enterText(
      find.byKey(const Key('needleSize-search-input')),
      '7',
    );
    await tester.pumpAndSettle();

    final panelRect = tester.getRect(
      find.byKey(const Key('needleSize-options-panel')),
    );
    final anchorRect = tester.getRect(
      find.byKey(const Key('needleSize-select-button')),
    );
    expect(panelRect.top, closeTo(anchorRect.bottom + 4, 1));
  });

  testWidgets('UT-003 short dropdown also opens below the anchor', (
    tester,
  ) async {
    await tester.pumpWidget(
      const _Harness(
        child: FormBuilder(
          child: CraftskyFormBuilderDropdownField<String>(
            name: 'status',
            label: 'Status',
            initialValue: 'finished',
            options: [
              CraftskySelectOption(value: 'finished', label: 'Finished'),
              CraftskySelectOption(value: 'wip', label: 'Work in progress'),
            ],
          ),
        ),
      ),
    );

    await tester.tap(find.byKey(const Key('status-select-button')));
    await tester.pumpAndSettle();

    final panelRect = tester.getRect(
      find.byKey(const Key('status-options-panel')),
    );
    final anchorRect = tester.getRect(
      find.byKey(const Key('status-select-button')),
    );
    expect(panelRect.top, greaterThan(anchorRect.bottom));
    expect(panelRect.top - anchorRect.bottom, lessThanOrEqualTo(12));
  });

  testWidgets('UT-003 dropdown overlay closes when tabbing away', (
    tester,
  ) async {
    final nextFocusNode = FocusNode(debugLabel: 'next-field');
    addTearDown(nextFocusNode.dispose);

    await tester.pumpWidget(
      _Harness(
        child: FormBuilder(
          child: Column(
            children: [
              const CraftskyFormBuilderDropdownField<String>(
                name: 'needleSize',
                label: 'Needle size',
                initialValue: '4.0mm',
                options: [
                  CraftskySelectOption(value: '2.0mm', label: '2.0mm'),
                  CraftskySelectOption(value: '3.0mm', label: '3.0mm'),
                  CraftskySelectOption(value: '4.0mm', label: '4.0mm'),
                  CraftskySelectOption(value: '5.0mm', label: '5.0mm'),
                  CraftskySelectOption(value: '6.0mm', label: '6.0mm'),
                  CraftskySelectOption(value: '7.0mm', label: '7.0mm'),
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

    await tester.enterText(
      find.byKey(const Key('needleSize-search-input')),
      '7',
    );
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('needleSize-options-panel')), findsOneWidget);

    await tester.sendKeyEvent(LogicalKeyboardKey.tab);
    await tester.pumpAndSettle();

    expect(find.byKey(const Key('needleSize-options-panel')), findsNothing);
    expect(find.byKey(const Key('needleSize-search-input')), findsOneWidget);
    expect(nextFocusNode.hasFocus, isTrue);
  });

  testWidgets('UT-003 closed dropdown tabs to the next field once', (
    tester,
  ) async {
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
              const CraftskyFormBuilderDropdownField<String>(
                name: 'craftType',
                label: 'Craft type',
                initialValue: 'knitting',
                options: [
                  CraftskySelectOption(value: 'knitting', label: 'Knitting'),
                  CraftskySelectOption(value: 'crochet', label: 'Crochet'),
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
    await tester.pump();

    expect(nextFocusNode.hasFocus, isTrue);
  });

  testWidgets('UT-003 searchable dropdown selects first match with enter', (
    tester,
  ) async {
    final firstFocusNode = FocusNode(debugLabel: 'first-field');
    final formKey = GlobalKey<FormBuilderState>();
    addTearDown(firstFocusNode.dispose);

    await tester.pumpWidget(
      _Harness(
        child: FormBuilder(
          key: formKey,
          child: Column(
            children: [
              TextField(
                key: const Key('first-field'),
                focusNode: firstFocusNode,
              ),
              const SizedBox(height: 24),
              const CraftskyFormBuilderDropdownField<String>(
                name: 'needleSize',
                label: 'Needle size',
                initialValue: '4.0mm',
                options: [
                  CraftskySelectOption(value: '2.0mm', label: '2.0mm'),
                  CraftskySelectOption(value: '3.0mm', label: '3.0mm'),
                  CraftskySelectOption(value: '4.0mm', label: '4.0mm'),
                  CraftskySelectOption(value: '5.0mm', label: '5.0mm'),
                  CraftskySelectOption(value: '6.0mm', label: '6.0mm'),
                  CraftskySelectOption(value: '7.0mm', label: '7.0mm'),
                ],
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

    expect(find.byKey(const Key('needleSize-search-input')), findsOneWidget);
    expect(
      tester
          .widget<TextField>(find.byKey(const Key('needleSize-search-input')))
          .focusNode
          ?.hasFocus,
      isTrue,
    );

    await tester.enterText(
      find.byKey(const Key('needleSize-search-input')),
      '7',
    );
    await tester.pumpAndSettle();
    await tester.sendKeyEvent(LogicalKeyboardKey.enter);
    await tester.pumpAndSettle();

    expect(formKey.currentState!.instantValue['needleSize'], '7.0mm');
  });

  testWidgets('UT-003 searchable overlay follows anchor after focus scroll', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        home: const Scaffold(
          body: SingleChildScrollView(
            child: Padding(
              padding: EdgeInsets.all(24),
              child: Column(
                children: [
                  SizedBox(height: 420),
                  FormBuilder(
                    child: CraftskyFormBuilderDropdownField<String>(
                      name: 'needleSize',
                      label: 'Needle size',
                      initialValue: '4.0mm',
                      options: [
                        CraftskySelectOption(value: '2.0mm', label: '2.0mm'),
                        CraftskySelectOption(value: '3.0mm', label: '3.0mm'),
                        CraftskySelectOption(value: '4.0mm', label: '4.0mm'),
                        CraftskySelectOption(value: '5.0mm', label: '5.0mm'),
                        CraftskySelectOption(value: '6.0mm', label: '6.0mm'),
                        CraftskySelectOption(value: '7.0mm', label: '7.0mm'),
                      ],
                    ),
                  ),
                  SizedBox(height: 500),
                ],
              ),
            ),
          ),
        ),
      ),
    );

    await tester.tap(find.byKey(const Key('needleSize-search-input')));
    await tester.enterText(
      find.byKey(const Key('needleSize-search-input')),
      '7',
    );
    await tester.pumpAndSettle();

    final panelRect = tester.getRect(
      find.byKey(const Key('needleSize-options-panel')),
    );
    final anchorRect = tester.getRect(
      find.byKey(const Key('needleSize-select-button')),
    );
    expect(panelRect.top, closeTo(anchorRect.bottom + 4, 1));
  });

  testWidgets('UT-003 keyboard highlight fills the option row', (
    tester,
  ) async {
    await tester.pumpWidget(
      const _Harness(
        child: FormBuilder(
          child: CraftskyFormBuilderDropdownField<String>(
            name: 'craftType',
            label: 'Craft type',
            initialValue: 'knitting',
            options: [
              CraftskySelectOption(value: 'knitting', label: 'Knitting'),
              CraftskySelectOption(value: 'crochet', label: 'Crochet'),
            ],
          ),
        ),
      ),
    );

    await tester.tap(find.byKey(const Key('craftType-select-button')));
    await tester.pumpAndSettle();
    await tester.sendKeyEvent(LogicalKeyboardKey.arrowDown);
    await tester.pump();

    final highlightedTile = tester.widget<ListTile>(
      find.byKey(const Key('craftType-option-crochet')),
    );
    expect(highlightedTile.selected, isTrue);
  });

  testWidgets('UT-003 keyboard highlight scrolls into view', (tester) async {
    await tester.pumpWidget(
      _Harness(
        child: FormBuilder(
          child: CraftskyFormBuilderDropdownField<String>(
            name: 'longList',
            label: 'Long list',
            searchThreshold: 100,
            options: [
              for (var index = 1; index <= 20; index++)
                CraftskySelectOption(
                  value: 'item-$index',
                  label: 'Item $index',
                ),
            ],
          ),
        ),
      ),
    );

    await tester.tap(find.byKey(const Key('longList-select-button')));
    await tester.pumpAndSettle();

    for (var index = 1; index < 20; index++) {
      await tester.sendKeyEvent(LogicalKeyboardKey.arrowDown);
      await tester.pump(const Duration(milliseconds: 120));
    }
    await tester.pumpAndSettle();

    var panelRect = tester.getRect(
      find.byKey(const Key('longList-options-panel')),
    );
    expect(
      tester
          .widget<ListTile>(find.byKey(const Key('longList-option-item-20')))
          .selected,
      isTrue,
    );
    var highlightedRect = tester.getRect(
      find.byKey(const Key('longList-option-item-20')),
    );
    expect(highlightedRect.bottom, lessThanOrEqualTo(panelRect.bottom));
    expect(highlightedRect.top, greaterThanOrEqualTo(panelRect.top));

    for (var index = 1; index < 20; index++) {
      await tester.sendKeyEvent(LogicalKeyboardKey.arrowUp);
      await tester.pump(const Duration(milliseconds: 120));
    }
    await tester.pumpAndSettle();

    panelRect = tester.getRect(find.byKey(const Key('longList-options-panel')));
    highlightedRect = tester.getRect(
      find.byKey(const Key('longList-option-item-1')),
    );
    expect(highlightedRect.bottom, lessThanOrEqualTo(panelRect.bottom));
    expect(highlightedRect.top, greaterThanOrEqualTo(panelRect.top));
  });
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
