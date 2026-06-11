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
          ),
        ),
      ),
    );

    await _addCustom(tester, 'materials', 'linen');
    await _addCustom(tester, 'materials', 'cotton');
    await _addCustom(tester, 'materials', 'wool');

    expect(formKey.currentState!.instantValue['materials'], [
      'linen',
      'cotton',
    ]);
    expect(find.text('linen'), findsOneWidget);
    expect(find.text('cotton'), findsOneWidget);
    expect(find.text('You can choose up to 2.'), findsOneWidget);

    await tester.tap(find.byKey(const Key('materials-remove-linen')));
    await tester.pump();

    expect(formKey.currentState!.instantValue['materials'], ['cotton']);
    expect(find.text('linen'), findsNothing);
  });

  testWidgets('UT-004 known-option multi-select saves and removes strings', (
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
            options: [
              CraftskySelectOption(value: 'blue', label: 'Blue'),
              CraftskySelectOption(value: 'cream', label: 'Cream'),
            ],
          ),
        ),
      ),
    );

    expect(formKey.currentState!.instantValue['colours'], ['blue']);
    await tester.tap(find.byKey(const Key('colours-option-cream')));
    await tester.pump();
    expect(formKey.currentState!.instantValue['colours'], ['blue', 'cream']);

    await tester.tap(find.byKey(const Key('colours-option-blue')));
    await tester.pump();
    expect(formKey.currentState!.instantValue['colours'], ['cream']);
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
