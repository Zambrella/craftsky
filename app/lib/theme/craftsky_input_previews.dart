import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_text_inputs.dart';
import 'package:flutter/material.dart';
import 'package:flutter/widget_previews.dart';

@Preview(name: 'Text inputs', group: 'CraftSky inputs', size: Size(420, 520))
Widget craftskyTextInputsPreview() {
  return const _InputPreviewFrame(
    child: Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        CraftskyTextInput(
          label: 'Project name',
          hintText: 'Name this project',
          helperText: 'Short names are easiest to scan.',
        ),
        SizedBox(height: 24),
        CraftskyTextInput(
          label: 'Pattern URL',
          initialValue: 'https://example.com/pattern',
          helperText: 'Optional',
        ),
        SizedBox(height: 24),
        CraftskyTextInput(
          label: 'Project name',
          initialValue: 'My summer cardigan',
          errorText: 'Choose a unique project name.',
        ),
      ],
    ),
  );
}

@Preview(
  name: 'Multiline input',
  group: 'CraftSky inputs',
  size: Size(420, 420),
)
Widget craftskyMultilineInputPreview() {
  return const _InputPreviewFrame(
    child: CraftskyMultilineTextInput(
      label: 'Notes',
      initialValue:
          'Sleeves need another fitting pass.\n'
          'Try blocking before adding buttons.',
      helperText: 'Private drafting notes stay on this device for now.',
      minLines: 5,
      maxLines: 8,
    ),
  );
}

@Preview(name: 'Number inputs', group: 'CraftSky inputs', size: Size(420, 420))
Widget craftskyNumberInputsPreview() {
  return const _InputPreviewFrame(
    child: Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        CraftskyNumberInput(
          label: 'Gauge stitches',
          initialValue: 22,
          suffixText: 'sts',
          helperText: 'Stored as a typed number.',
          mode: CraftskyNumberInputMode.integer,
        ),
        SizedBox(height: 24),
        CraftskyNumberInput(
          label: 'Gauge measurement',
          initialValue: 10,
          suffixText: 'cm',
        ),
        SizedBox(height: 24),
        CraftskyNumberInput(
          label: 'Ease',
          prefixText: '+',
          suffixText: 'cm',
          errorText: 'Enter a number between 0 and 30.',
        ),
      ],
    ),
  );
}

@Preview(
  name: 'Text inputs large type',
  group: 'CraftSky inputs',
  size: Size(360, 560),
  textScaleFactor: 1.4,
)
Widget craftskyTextInputsLargeTypePreview() {
  return const _InputPreviewFrame(
    child: CraftskyTextInput(
      label: 'A deliberately long project name label',
      hintText: 'Try a short, memorable name',
      helperText:
          'Labels and helper text should remain readable at larger text sizes.',
    ),
  );
}

class _InputPreviewFrame extends StatelessWidget {
  const _InputPreviewFrame({required this.child});

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
