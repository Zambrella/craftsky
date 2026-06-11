import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_form_builder_select_fields.dart';
import 'package:flutter/material.dart';
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
          child: CraftskyFormBuilderDropdownField<String>(
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
        ),
      ),
    );

    expect(find.text('Craft type'), findsOneWidget);
    expect(find.text('Choose the closest craft'), findsOneWidget);
    final decorator = tester.widget<InputDecorator>(
      find.byType(InputDecorator),
    );
    expect(decorator.decoration.labelText, isNull);
    expect(decorator.decoration.contentPadding, EdgeInsets.zero);
    expect(formKey.currentState!.instantValue['craftType'], 'knitting');

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
