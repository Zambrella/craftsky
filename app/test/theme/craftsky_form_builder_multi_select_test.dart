import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_form_builder_select_fields.dart';
import 'package:flutter/material.dart';
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
            addCustomValueLabel: 'Add material',
            maxSelectedErrorText: 'Choose no more than 2 materials.',
          ),
        ),
      ),
    );

    expect(find.text('Add material'), findsNWidgets(2));

    await _addCustom(tester, 'materials', 'linen');
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
    expect(find.text('Search colours'), findsOneWidget);
    expect(find.byKey(const Key('colours-option-red')), findsNothing);

    await tester.enterText(
      find.byKey(const Key('colours-search-input')),
      'cre',
    );
    await tester.pump();
    expect(find.byKey(const Key('colours-option-cream')), findsOneWidget);
    expect(find.byKey(const Key('colours-option-red')), findsNothing);
    expect(find.byType(ListTile), findsOneWidget);

    await tester.tap(find.byKey(const Key('colours-option-cream')));
    await tester.pump();
    expect(formKey.currentState!.instantValue['colours'], ['blue', 'cream']);
    final searchField = tester.widget<TextField>(
      find.byKey(const Key('colours-search-input')),
    );
    expect(searchField.controller?.text, isEmpty);

    await tester.enterText(
      find.byKey(const Key('colours-search-input')),
      'blue',
    );
    await tester.pump();
    expect(find.byKey(const Key('colours-option-blue')), findsNothing);

    await tester.tap(find.byKey(const Key('colours-remove-blue')));
    await tester.pump();
    expect(formKey.currentState!.instantValue['colours'], ['cream']);
  });

  testWidgets('UT-004 inline known-option search stays centred with chips', (
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

    final chipCenter = tester.getCenter(find.byType(InputChip));
    final fieldCenter = tester.getCenter(
      find.byKey(const Key('designTags-search-input')),
    );
    expect((chipCenter.dy - fieldCenter.dy).abs(), lessThanOrEqualTo(8));
    expect(
      tester.getSize(find.byKey(const Key('designTags-search-input'))).width,
      greaterThan(240),
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
