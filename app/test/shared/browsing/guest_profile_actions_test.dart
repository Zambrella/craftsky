import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/widgets/profile_actions.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  testWidgets(
    'guest profile uses visitor controls and keeps Copy link public',
    (
      tester,
    ) async {
      var copied = false;
      var prompts = 0;
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: ProfileActions(
              actions: VisitorProfileActionSet(
                isFollowing: false,
                isBusy: false,
                onFollowToggle: () => prompts++,
                onShare: () {},
                onReport: () => prompts++,
                onMuteToggle: () => prompts++,
                onBlockToggle: () => prompts++,
                onCopyLink: () => copied = true,
                onSendViaChat: () => prompts++,
              ),
            ),
          ),
        ),
      );
      expect(find.text('Follow'), findsOneWidget);
      await tester.tap(find.text('Follow'));
      expect(prompts, 1);
      expect(find.text('Settings'), findsNothing);
      await tester.tap(find.byType(ProfileMoreActionsButton));
      await tester.pumpAndSettle();
      expect(find.text('Followers'), findsNothing);
      expect(find.text('Following'), findsNothing);
      await tester.tap(find.text('Copy link'));
      await tester.pumpAndSettle();
      expect(copied, true);
    },
  );
}
