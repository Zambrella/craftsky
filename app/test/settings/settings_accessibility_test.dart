import 'dart:ui' show Tristate;

import 'package:craftsky_app/settings/models/settings_row.dart';
import 'package:craftsky_app/settings/widgets/settings_row_tile.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('disclosure row is an enabled button with a 48px target', (
    tester,
  ) async {
    final semantics = tester.ensureSemantics();
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(
          body: SettingsRowTile(
            descriptor: SettingsRowDescriptor(
              id: SettingsRowId.account,
              kind: SettingsRowKind.disclosure,
            ),
            label: 'Account',
            leading: CraftskyIcons.accountSettings,
            onTap: _noop,
          ),
        ),
      ),
    );

    expect(find.byIcon(CraftskyIconsBold.next), findsOneWidget);
    expect(
      tester.getSize(find.byType(SettingsRowTile)).height,
      greaterThanOrEqualTo(48),
    );
    final node = tester.getSemantics(find.bySemanticsLabel('Account'));
    expect(node.getSemanticsData().flagsCollection.isButton, isTrue);
    expect(
      node.getSemanticsData().flagsCollection.isEnabled,
      Tristate.isTrue,
    );
    semantics.dispose();
  });

  testWidgets('RTL disclosure points forward and direct actions have no icon', (
    tester,
  ) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: Directionality(
          textDirection: TextDirection.rtl,
          child: Scaffold(
            body: Column(
              children: [
                SettingsRowTile(
                  descriptor: SettingsRowDescriptor(
                    id: SettingsRowId.about,
                    kind: SettingsRowKind.disclosure,
                  ),
                  label: 'About',
                  leading: CraftskyIcons.info,
                  onTap: _noop,
                ),
                SettingsRowTile(
                  descriptor: SettingsRowDescriptor(
                    id: SettingsRowId.signOut,
                    kind: SettingsRowKind.destructiveAction,
                  ),
                  label: 'Sign out',
                  leading: CraftskyIcons.logout,
                  onTap: _noop,
                ),
              ],
            ),
          ),
        ),
      ),
    );

    expect(find.byIcon(CraftskyIconsBold.previous), findsOneWidget);
    expect(find.byIcon(CraftskyIconsBold.next), findsNothing);
    expect(find.byIcon(CraftskyIconsBold.externalLink), findsNothing);
  });

  testWidgets('disabled row exposes disabled semantics', (tester) async {
    final semantics = tester.ensureSemantics();
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(
          body: SettingsRowTile(
            descriptor: SettingsRowDescriptor(
              id: SettingsRowId.switchAccount,
              kind: SettingsRowKind.disclosure,
            ),
            label: 'Switch account',
            leading: CraftskyIconsBold.switchAccount,
          ),
        ),
      ),
    );

    final node = tester.getSemantics(find.bySemanticsLabel('Switch account'));
    expect(
      node.getSemanticsData().flagsCollection.isEnabled,
      Tristate.isFalse,
    );
    semantics.dispose();
  });
}

void _noop() {}
