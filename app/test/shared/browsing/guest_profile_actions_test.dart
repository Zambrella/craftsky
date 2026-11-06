import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/widgets/profile_actions.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  testWidgets('AT-007 guest profile offers Copy link without account actions', (
    tester,
  ) async {
    var copied = false;
    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: appLocalizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: ProfileActions(
            actions: GuestProfileActionSet(onCopyLink: () => copied = true),
          ),
        ),
      ),
    );
    expect(find.text('Follow'), findsNothing);
    expect(find.text('Settings'), findsNothing);
    expect(find.text('Send via chat'), findsNothing);
    await tester.tap(find.text('Copy link'));
    expect(copied, true);
  });
}
