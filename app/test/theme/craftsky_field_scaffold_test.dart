import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_field_scaffold.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('syncs focus lift when focus node changes', (tester) async {
    final firstFocusNode = FocusNode(debugLabel: 'first');
    final secondFocusNode = FocusNode(debugLabel: 'second');
    addTearDown(firstFocusNode.dispose);
    addTearDown(secondFocusNode.dispose);

    late StateSetter setHarnessState;
    var activeFocusNode = firstFocusNode;

    await tester.pumpWidget(
      _Harness(
        child: StatefulBuilder(
          builder: (context, setState) {
            setHarnessState = setState;
            return CraftskyFieldScaffold(
              label: 'Field',
              focusNode: activeFocusNode,
              child: Focus(
                focusNode: activeFocusNode,
                child: const SizedBox(height: 48, width: 160),
              ),
            );
          },
        ),
      ),
    );

    final shadowOffset = AppTheme.lightThemeData
        .extension<BrandShadowTheme>()!
        .dropSm
        .first
        .offset;
    expect(
      tester.widget<CraftskyFocusLift>(find.byType(CraftskyFocusLift)).lift,
      shadowOffset,
    );

    firstFocusNode.requestFocus();
    await tester.pump();
    expect(
      tester.widget<CraftskyFocusLift>(find.byType(CraftskyFocusLift)).lift,
      Offset.zero,
    );

    setHarnessState(() => activeFocusNode = secondFocusNode);
    await tester.pump();
    expect(
      tester.widget<CraftskyFocusLift>(find.byType(CraftskyFocusLift)).lift,
      shadowOffset,
    );
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
