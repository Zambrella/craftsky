import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/account_switcher_state.dart';
import 'package:craftsky_app/auth/widgets/account_switcher_content.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/shared/widgets/craft_icon.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/widgets/subscription_tier_badge.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  for (final tier in SubscriptionTier.values) {
    testWidgets('switcher identity uses the space left by the $tier badge', (
      tester,
    ) async {
      final account = AccountKey('did:plc:doug');
      await tester.pumpWidget(
        ProviderScope(
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: Scaffold(
              body: Center(
                child: SizedBox(
                  width: 400,
                  child: AccountSwitcherRowTile(
                    row: AccountSwitcherRow(
                      lease: AccountSessionLease(
                        account: account,
                        sessionGeneration: 1,
                      ),
                      handle: 'dougtodd.dev',
                      displayName: 'Doug Todd',
                      avatarUrl: null,
                      customisation: ProfileCustomisation.defaults,
                      isCurrent: true,
                      crafts: const ['sewing', 'knitting', 'crochet'],
                    ),
                    selected: true,
                    enabled: false,
                    badge: SubscriptionTierBadge(
                      state: AsyncData(
                        SubscriptionAccess(
                          did: account.did,
                          effectiveTier: tier,
                          givesAccess: tier != SubscriptionTier.free,
                        ),
                      ),
                    ),
                    trailing: const Icon(CraftskyIcons.check),
                    onTap: () {},
                  ),
                ),
              ),
            ),
          ),
        ),
      );
      if (tier == SubscriptionTier.free) {
        expect(find.byType(CraftIcon), findsNWidgets(3));
        expect(find.textContaining('+'), findsNothing);
        expect(
          tester
              .renderObject<RenderParagraph>(find.text('Doug Todd'))
              .didExceedMaxLines,
          isFalse,
        );
      } else {
        expect(
          find.text(tier == SubscriptionTier.plus ? 'Plus' : 'Business'),
          findsOneWidget,
        );
      }
      expect(
        tester.getTopLeft(find.text('@dougtodd.dev')).dx,
        tester.getTopLeft(find.text('Doug Todd')).dx,
      );
      expect(tester.takeException(), isNull);
    });
  }
}
