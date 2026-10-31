import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/craftsky_select_inputs.dart';
import 'package:flutter/services.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

import '../test_support/widget_pump.dart';

const List<CraftskySelectOption<String>> _options = [
  CraftskySelectOption(value: 'wool', label: 'Wool'),
  CraftskySelectOption(value: 'cotton', label: 'Cotton'),
];

void main() {
  for (final searchable in [false, true]) {
    testWidgets(
      'clear ${searchable ? 'searchable' : 'plain'} selection and reselect',
      (tester) async {
        final formKey = GlobalKey<FormBuilderState>();
        await tester.pumpWidget(
          craftskyTestWidget(
            child: FormBuilder(
              key: formKey,
              child: CraftskyFormSingleSelectField<String>(
                name: 'material',
                label: 'Material',
                initialValue: 'wool',
                options: _options,
                searchThreshold: searchable ? 0 : 5,
              ),
            ),
          ),
        );
        await tester.pumpAndSettle();
        expect(find.byTooltip('Clear Material'), findsOneWidget);
        final clearIconCenter = tester.getCenter(
          find.byIcon(CraftskyIconsBold.clear),
        );
        final selectedHeight = tester
            .getSize(find.byKey(const Key('material-select-button')))
            .height;
        if (searchable) {
          await tester.enterText(
            find.byKey(const Key('material-search-input')),
            'woo',
          );
          await tester.pumpAndSettle();
          expect(
            find.byKey(const Key('material-options-panel')),
            findsOneWidget,
          );
        }
        await tester.tap(find.byKey(const Key('material-clear')));
        await tester.pumpAndSettle();
        expect(formKey.currentState!.fields['material']!.value, isNull);
        formKey.currentState!.save();
        expect(formKey.currentState!.value['material'], isNull);
        expect(find.byKey(const Key('material-clear')), findsNothing);
        expect(
          tester.getCenter(
            find.byIcon(
              searchable ? CraftskyIcons.search : CraftskyIconsBold.expand,
            ),
          ),
          clearIconCenter,
        );
        expect(
          tester
              .getSize(find.byKey(const Key('material-select-button')))
              .height,
          selectedHeight,
        );
        expect(find.byKey(const Key('material-options-panel')), findsNothing);
        await tester.tap(
          find.byKey(
            Key(
              searchable ? 'material-search-input' : 'material-select-button',
            ),
          ),
        );
        await tester.pumpAndSettle();
        await tester.tap(find.text('Cotton'));
        await tester.pumpAndSettle();
        expect(formKey.currentState!.fields['material']!.value, 'cotton');
        expect(find.byKey(const Key('material-clear')), findsOneWidget);
      },
    );
  }

  testWidgets('clear button supports keyboard activation', (tester) async {
    String? value = 'wool';
    await tester.pumpWidget(
      craftskyTestWidget(
        child: StatefulBuilder(
          builder: (context, setState) => CraftskySingleSelectInput<String>(
            label: 'Material',
            options: _options,
            value: value,
            onChanged: (next) => setState(() => value = next),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.sendKeyEvent(LogicalKeyboardKey.tab);
    await tester.pumpAndSettle();
    await tester.sendKeyEvent(LogicalKeyboardKey.tab);
    await tester.pumpAndSettle();
    await tester.sendKeyEvent(LogicalKeyboardKey.enter);
    await tester.pumpAndSettle();
    expect(value, isNull);
    expect(find.byKey(const Key('Material-options-panel')), findsNothing);
  });

  for (final mode in [
    'required',
    'non-clearable',
    'empty',
    'disabled',
    'loading',
  ]) {
    testWidgets('$mode field cannot be cleared', (tester) async {
      var changed = false;
      await tester.pumpWidget(
        craftskyTestWidget(
          child: CraftskySingleSelectInput<String>(
            label: 'Material',
            options: _options,
            value: mode == 'empty' ? null : 'wool',
            required: mode == 'required',
            clearable: mode != 'non-clearable',
            enabled: mode != 'disabled',
            isLoading: mode == 'loading',
            onChanged: (_) => changed = true,
          ),
        ),
      );
      await tester.pump();
      final clear = find.byKey(const Key('Material-clear'));
      if (mode == 'disabled' || mode == 'loading') {
        expect(tester.widget<IconButton>(clear).onPressed, isNull);
      } else {
        expect(clear, findsNothing);
      }
      expect(changed, isFalse);
      await tester.pumpWidget(const SizedBox());
    });
  }
}
