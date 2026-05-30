import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/widgets/profile_actions.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

Future<void> _pump(WidgetTester tester, Widget child) {
  return tester.pumpWidget(
    MaterialApp(
      theme: AppTheme.lightThemeData,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: Scaffold(body: Center(child: child)),
    ),
  );
}

void main() {
  group('ProfileActions', () {
    testWidgets('visitor profile exposes report action', (tester) async {
      var reports = 0;
      await _pump(
        tester,
        ProfileActions(
          actions: VisitorProfileActionSet(
            isFollowing: false,
            isBusy: false,
            onFollowToggle: () {},
            onShare: () {},
            onReport: () => reports++,
          ),
        ),
      );

      await tester.tap(find.byTooltip('Report profile'));

      expect(reports, 1);
    });

    testWidgets('self profile does not expose report action', (tester) async {
      await _pump(
        tester,
        ProfileActions(
          actions: SelfProfileActionSet(onEdit: () {}, onSettings: () {}),
        ),
      );

      expect(find.byTooltip('Report profile'), findsNothing);
    });
  });
}
