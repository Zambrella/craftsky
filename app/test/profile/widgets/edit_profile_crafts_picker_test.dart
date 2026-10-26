import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/data/crafts_catalog.dart';
import 'package:craftsky_app/profile/widgets/edit_profile_crafts_picker.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  for (final count in [1, 2]) {
    for (final kind in [PointerDeviceKind.touch, PointerDeviceKind.mouse]) {
      testWidgets(
        '$count selected: $kind reorder controls preserve selection',
        (
          tester,
        ) async {
          var selected = canonicalSelectableCrafts.take(count).toSet();
          final original = selected.toList();
          var reorderCalls = 0;
          await tester.pumpWidget(
            MaterialApp(
              theme: AppTheme.lightThemeData,
              localizationsDelegates: appLocalizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: Scaffold(
                body: StatefulBuilder(
                  builder: (context, setState) => EditProfileCraftsPicker(
                    selected: selected,
                    onToggle: (craft) => setState(() {
                      selected = {...selected};
                      selected.contains(craft)
                          ? selected.remove(craft)
                          : selected.add(craft);
                    }),
                    onReorder: (_) => reorderCalls++,
                    onRequestMore: () {},
                  ),
                ),
              ),
            ),
          );
          final row = find.byKey(ValueKey('craft-choice-${original.first.id}'));
          final handle = find.descendant(
            of: row,
            matching: find.byWidgetPredicate(
              (widget) => widget is ReorderableDragStartListener,
            ),
          );
          final menu = find.descendant(
            of: row,
            matching: find.byType(CraftskyContextMenuButton),
          );
          if (count == 1) {
            expect(handle, findsNothing);
            expect(menu, findsNothing);
            expect(find.byIcon(Icons.drag_handle), findsNothing);
          } else {
            expect(
              tester.widget<ReorderableDragStartListener>(handle).enabled,
              isTrue,
            );
            expect(
              tester.widget<CraftskyContextMenuButton>(menu).enabled,
              isTrue,
            );
            await tester.tap(handle, kind: kind);
            await tester.pumpAndSettle();
            expect(selected.toList(), original);
            expect(reorderCalls, 0);

            await tester.tap(menu, kind: kind);
            await tester.pumpAndSettle();
            expect(selected.toList(), original);
            expect(find.text('Move down'), findsOneWidget);
            await tester.tapAt(const Offset(10, 10));
            await tester.pumpAndSettle();
          }
          expect(reorderCalls, 0);

          await tester.tap(
            find.descendant(of: row, matching: find.byType(Checkbox)),
            kind: kind,
          );
          await tester.pumpAndSettle();
          expect(selected, isNot(contains(original.first)));
          expect(find.byType(CraftskyContextMenuButton), findsNothing);
          expect(find.byIcon(Icons.drag_handle), findsNothing);
        },
      );
    }
  }
}
