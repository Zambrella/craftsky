import 'package:craftsky_app/auth/models/account_switcher_state.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/widgets/account_switcher_content.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/widgets/profile_card.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets(
    'AT-006 sentinel stays on the wire and stale handle is never presented',
    (tester) async {
      final profile = ProfileMapper.fromMap({
        'did': 'did:plc:alice',
        'handle': 'handle.invalid',
        'displayName': 'Alice',
        'crafts': <String>[],
      });
      final staleRegistry = SessionRegistry.empty().upsertAndActivate(
        token: 'token',
        did: 'did:plc:alice',
        handle: 'old.example',
        cachedDisplayName: 'Alice',
      );
      final currentRegistry = staleRegistry.updateHandle(
        staleRegistry.activeLease!.session,
        Handle.parse('handle.invalid'),
      );

      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: ListView(
              children: [
                SizedBox(
                  height: 640,
                  child: ProfileCard(
                    profile: profile,
                    isOwnProfile: true,
                    onClose: () {},
                    onVisitProfile: () {},
                    onPrimaryAction: () {},
                  ),
                ),
                SizedBox(
                  height: 160,
                  child: AccountSwitcherContent(
                    state: AccountSwitcherState.fromRegistry(currentRegistry),
                    onSelect: (_) {},
                    onAddAccount: () {},
                    showAddAccount: false,
                  ),
                ),
              ],
            ),
          ),
        ),
      );

      expect(profile.handle.value, 'handle.invalid');
      expect(find.text('Alice'), findsWidgets);
      expect(find.text('Handle unavailable'), findsWidgets);
      expect(find.textContaining('handle.invalid'), findsNothing);
      expect(find.textContaining('old.example'), findsNothing);
    },
  );
}
