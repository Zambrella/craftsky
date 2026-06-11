import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_form_builder_select_fields.dart';
import 'package:flutter/material.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('UT-005 radio saves, validates, resets and disables', (
    tester,
  ) async {
    final formKey = GlobalKey<FormBuilderState>();
    String? changed;

    await tester.pumpWidget(
      _Harness(
        child: FormBuilder(
          key: formKey,
          child: CraftskyFormBuilderRadioField<String>(
            name: 'status',
            label: 'Status',
            helperText: 'Status defaults to finished',
            initialValue: 'finished',
            options: const [
              CraftskySelectOption(value: 'finished', label: 'Finished'),
              CraftskySelectOption(value: 'wip', label: 'Work in progress'),
            ],
            validator: (value) => value == null ? 'Choose a status' : null,
            onChanged: (value) => changed = value,
          ),
        ),
      ),
    );

    expect(formKey.currentState!.instantValue['status'], 'finished');
    await tester.tap(find.text('Work in progress'));
    await tester.pump();
    expect(changed, 'wip');
    expect(formKey.currentState!.instantValue['status'], 'wip');

    formKey.currentState!.fields['status']!.didChange(null);
    expect(formKey.currentState!.saveAndValidate(), isFalse);
    await tester.pump();
    expect(find.text('Choose a status'), findsOneWidget);

    formKey.currentState!.reset();
    await tester.pump();
    expect(formKey.currentState!.instantValue['status'], 'finished');
  });

  testWidgets('UT-005 disabled radio prevents changes', (tester) async {
    final formKey = GlobalKey<FormBuilderState>();

    await tester.pumpWidget(
      _Harness(
        child: FormBuilder(
          key: formKey,
          child: const CraftskyFormBuilderRadioField<String>(
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

    await tester.tap(find.text('Work in progress'));
    await tester.pump();
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
