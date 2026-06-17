import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_select_inputs.dart';
import 'package:flutter/material.dart';
import 'package:flutter/widget_previews.dart';

const List<CraftskySelectOption<String>> _craftOptions = [
  CraftskySelectOption(value: 'knitting', label: 'Knitting'),
  CraftskySelectOption(value: 'crochet', label: 'Crochet'),
  CraftskySelectOption(value: 'sewing', label: 'Sewing'),
  CraftskySelectOption(value: 'embroidery', label: 'Embroidery'),
  CraftskySelectOption(value: 'quilting', label: 'Quilting'),
];

const List<CraftskySelectOption<String>> _colourOptions = [
  CraftskySelectOption(value: 'black', label: 'Black'),
  CraftskySelectOption(value: 'white', label: 'White'),
  CraftskySelectOption(value: 'gray', label: 'Grey'),
  CraftskySelectOption(value: 'brown', label: 'Brown'),
  CraftskySelectOption(value: 'beige', label: 'Beige'),
  CraftskySelectOption(value: 'red', label: 'Red'),
  CraftskySelectOption(value: 'orange', label: 'Orange'),
  CraftskySelectOption(value: 'yellow', label: 'Yellow'),
  CraftskySelectOption(value: 'green', label: 'Green'),
  CraftskySelectOption(value: 'blue', label: 'Blue'),
  CraftskySelectOption(value: 'purple', label: 'Purple'),
  CraftskySelectOption(value: 'pink', label: 'Pink'),
];

@Preview(name: 'Single select', group: 'Craftsky inputs', size: Size(420, 360))
Widget craftskySingleSelectPreview() {
  return const _SelectPreviewFrame(
    child: CraftskySingleSelectInput<String>(
      label: 'Craft type',
      value: 'knitting',
      options: _craftOptions,
      helperText: 'Five options uses a simple menu without search.',
    ),
  );
}

@Preview(
  name: 'Single select with search',
  group: 'Craftsky inputs',
  size: Size(420, 460),
)
Widget craftskySingleSelectSearchPreview() {
  return const _SelectPreviewFrame(
    child: CraftskySingleSelectInput<String>(
      label: 'Main colour',
      value: 'blue',
      options: _colourOptions,
      searchHintText: 'Search colours',
      helperText: 'More than five options adds search.',
    ),
  );
}

@Preview(name: 'Multi select', group: 'Craftsky inputs', size: Size(420, 520))
Widget craftskyMultiSelectPreview() {
  return const _SelectPreviewFrame(
    child: CraftskySearchableMultiSelectInput<String>(
      label: 'Colours',
      values: ['blue', 'cream'],
      options: _colourOptions,
      searchHintText: 'Search colours',
      helperText: 'Selected colours stay visible as chips.',
    ),
  );
}

@Preview(name: 'Token input', group: 'Craftsky inputs', size: Size(420, 420))
Widget craftskyTokenInputPreview() {
  return const _SelectPreviewFrame(
    child: CraftskyTokenInput(
      label: 'Materials',
      values: ['linen', 'cotton'],
      inputHintText: 'Add material',
      addButtonLabel: 'Add material',
      helperText: 'Free-text entries are stored as tokens.',
    ),
  );
}

class _SelectPreviewFrame extends StatelessWidget {
  const _SelectPreviewFrame({required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      theme: AppTheme.lightThemeData,
      home: Scaffold(
        body: SafeArea(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(24),
            child: child,
          ),
        ),
      ),
    );
  }
}
