import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_form_builder_text_field.dart';
import 'package:flutter/material.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('UT-001 forwards value, error, save, reset and enabled state', (
    tester,
  ) async {
    final formKey = GlobalKey<FormBuilderState>();

    await tester.pumpWidget(
      _Harness(
        child: FormBuilder(
          key: formKey,
          child: CraftskyFormBuilderTextField(
            name: 'title',
            label: 'Project title',
            hintText: 'Name this project',
            helperText: 'Optional but useful',
            initialValue: 'Old title',
            validator: (value) => value == 'bad' ? 'Choose kinder words' : null,
          ),
        ),
      ),
    );

    expect(find.text('Project title'), findsOneWidget);
    expect(find.text('Optional but useful'), findsOneWidget);
    expect(formKey.currentState!.instantValue['title'], 'Old title');

    await tester.enterText(find.byType(TextField), 'New title');
    expect(formKey.currentState!.instantValue['title'], 'New title');
    expect(formKey.currentState!.saveAndValidate(), isTrue);
    expect(formKey.currentState!.value['title'], 'New title');

    await tester.enterText(find.byType(TextField), 'bad');
    expect(formKey.currentState!.saveAndValidate(), isFalse);
    await tester.pump();
    expect(find.text('Choose kinder words'), findsOneWidget);

    formKey.currentState!.reset();
    await tester.pump();
    expect(formKey.currentState!.instantValue['title'], 'Old title');
    expect(find.text('Old title'), findsOneWidget);
  });

  testWidgets('UT-001 disabled field is not editable', (tester) async {
    final formKey = GlobalKey<FormBuilderState>();

    await tester.pumpWidget(
      _Harness(
        child: FormBuilder(
          key: formKey,
          child: const CraftskyFormBuilderTextField(
            name: 'title',
            label: 'Project title',
            initialValue: 'Locked title',
            enabled: false,
          ),
        ),
      ),
    );

    await tester.enterText(find.byType(TextField), 'Ignored edit');
    await tester.pump();

    expect(formKey.currentState!.instantValue['title'], 'Locked title');
    expect(find.text('Locked title'), findsOneWidget);
  });

  testWidgets(
    'UT-002 multiline field honours controller, focus and callbacks',
    (
      tester,
    ) async {
      final formKey = GlobalKey<FormBuilderState>();
      final controller = TextEditingController(text: 'Seed notes');
      final focusNode = FocusNode(debugLabel: 'notes');
      String? changed;
      String? submitted;

      addTearDown(controller.dispose);
      addTearDown(focusNode.dispose);

      await tester.pumpWidget(
        _Harness(
          child: FormBuilder(
            key: formKey,
            child: CraftskyFormBuilderMultilineTextField(
              name: 'notes',
              label: 'Notes',
              helperText: 'Add extra detail',
              controller: controller,
              focusNode: focusNode,
              minLines: 4,
              maxLines: 7,
              textInputAction: TextInputAction.done,
              onChanged: (value) => changed = value,
              onSubmitted: (value) => submitted = value,
            ),
          ),
        ),
      );

      final textField = tester.widget<TextField>(find.byType(TextField));
      expect(textField.minLines, 4);
      expect(textField.maxLines, 7);
      expect(formKey.currentState!.instantValue['notes'], 'Seed notes');

      await tester.tap(find.byType(TextField));
      expect(focusNode.hasFocus, isTrue);

      await tester.enterText(find.byType(TextField), 'Line one\nLine two');
      expect(controller.text, 'Line one\nLine two');
      expect(changed, 'Line one\nLine two');
      expect(formKey.currentState!.instantValue['notes'], 'Line one\nLine two');

      await tester.testTextInput.receiveAction(TextInputAction.done);
      expect(submitted, 'Line one\nLine two');
    },
  );
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
