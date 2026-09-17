import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_field_scaffold.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import '../test_support/widget_pump.dart';

void main() {
  testWidgets('syncs focus lift when focus node changes', (tester) async {
    final firstFocusNode = FocusNode(debugLabel: 'first');
    final secondFocusNode = FocusNode(debugLabel: 'second');
    addTearDown(firstFocusNode.dispose);
    addTearDown(secondFocusNode.dispose);

    late StateSetter setHarnessState;
    var activeFocusNode = firstFocusNode;

    await pumpCraftskyWidget(
      tester,
      StatefulBuilder(
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

  testWidgets('keeps character counters close to their field', (tester) async {
    await pumpCraftskyWidget(
      tester,
      const CraftskyFieldScaffold(
        label: 'Field',
        counterText: '0/5',
        child: SizedBox(height: 48, width: 160),
      ),
    );

    final fieldBottom = tester.getBottomLeft(find.byType(CraftskyFocusLift)).dy;
    final counterTop = tester.getTopLeft(find.text('0/5')).dy;
    final spacing = AppTheme.lightThemeData.extension<SpacingTheme>()!;

    expect(counterTop - fieldBottom, spacing.sp1);
  });
}
