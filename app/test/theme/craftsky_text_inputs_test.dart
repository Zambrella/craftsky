import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_text_inputs.dart';
import 'package:flutter/material.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('text input exposes label, hint, value, and error semantics', (
    tester,
  ) async {
    final semantics = tester.ensureSemantics();

    await tester.pumpWidget(
      const _Harness(
        child: CraftskyTextInput(
          label: 'Project name',
          initialValue: 'Summer cardigan',
          hintText: 'Name this project',
          errorText: 'Choose a unique name.',
        ),
      ),
    );

    final node = tester.getSemantics(find.byType(CraftskyTextInput));
    expect(node.label, 'Project name');
    expect(node.hint, 'Name this project. Choose a unique name.');
    expect(node.value, 'Summer cardigan');
    semantics.dispose();
  });

  testWidgets('number form field stores typed numeric values', (tester) async {
    final formKey = GlobalKey<FormBuilderState>();
    num? changed;

    await tester.pumpWidget(
      _Harness(
        child: FormBuilder(
          key: formKey,
          child: CraftskyFormNumberField(
            name: 'gaugeStitches',
            label: 'Gauge stitches',
            initialValue: 22,
            suffixText: 'sts',
            mode: CraftskyNumberInputMode.integer,
            validator: (value) => value == null ? 'Enter a number.' : null,
            onChanged: (value) => changed = value,
          ),
        ),
      ),
    );

    expect(formKey.currentState!.instantValue['gaugeStitches'], 22);

    await tester.enterText(find.byType(TextField), '24');
    expect(changed, 24);
    expect(formKey.currentState!.instantValue['gaugeStitches'], 24);
    expect(formKey.currentState!.saveAndValidate(), isTrue);
    expect(formKey.currentState!.value['gaugeStitches'], 24);

    await tester.enterText(find.byType(TextField), '');
    expect(formKey.currentState!.saveAndValidate(), isFalse);
    await tester.pump();
    expect(find.text('Enter a number.'), findsOneWidget);
  });

  testWidgets('decimal number input parses decimals and supports suffix text', (
    tester,
  ) async {
    num? changed;

    await tester.pumpWidget(
      _Harness(
        child: CraftskyNumberInput(
          label: 'Gauge measurement',
          suffixText: 'cm',
          onChanged: (value) => changed = value,
        ),
      ),
    );

    await tester.enterText(find.byType(TextField), '10.5');
    expect(changed, 10.5);
    expect(find.text('cm'), findsOneWidget);
  });

  testWidgets('number form field supports external controller values', (
    tester,
  ) async {
    final formKey = GlobalKey<FormBuilderState>();
    final controller = TextEditingController(text: '12');
    addTearDown(controller.dispose);

    await tester.pumpWidget(
      _Harness(
        child: FormBuilder(
          key: formKey,
          child: CraftskyFormNumberField(
            name: 'rowGauge',
            label: 'Row gauge',
            controller: controller,
            mode: CraftskyNumberInputMode.integer,
            validator: (value) => value == null ? 'Enter a number.' : null,
          ),
        ),
      ),
    );

    expect(formKey.currentState!.instantValue['rowGauge'], 12);

    await tester.enterText(find.byType(TextField), '14');
    expect(controller.text, '14');
    expect(formKey.currentState!.instantValue['rowGauge'], 14);
    expect(formKey.currentState!.saveAndValidate(), isTrue);
    expect(formKey.currentState!.value['rowGauge'], 14);

    await tester.enterText(find.byType(TextField), '');
    expect(formKey.currentState!.saveAndValidate(), isFalse);
    await tester.pump();
    expect(find.text('Enter a number.'), findsOneWidget);

    formKey.currentState!.reset();
    await tester.pump();
    expect(controller.text, '12');
    expect(formKey.currentState!.instantValue['rowGauge'], 12);
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
