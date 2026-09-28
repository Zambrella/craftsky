import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/widgets/subscription_tier_badge.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('free tier has no account switcher label', (tester) async {
    await tester.pumpWidget(
      _app(
        SubscriptionTierBadge(state: AsyncData(_access(SubscriptionTier.free))),
      ),
    );

    expect(find.text('Free'), findsNothing);
  });

  testWidgets('AT-015 badge isolates tier, loading, and unavailable states', (
    tester,
  ) async {
    for (final entry in <(AsyncValue<SubscriptionAccess>, String)>[
      (AsyncData(_access(SubscriptionTier.plus)), 'Plus'),
      (AsyncData(_access(SubscriptionTier.business)), 'Business'),
      (const AsyncLoading(), 'Loading tier'),
      (AsyncError(StateError('private'), StackTrace.empty), 'Tier unavailable'),
    ]) {
      await tester.pumpWidget(_app(SubscriptionTierBadge(state: entry.$1)));
      expect(find.text(entry.$2), findsOneWidget);
      expect(find.textContaining('private'), findsNothing);
    }
  });
}

SubscriptionAccess _access(SubscriptionTier tier) => SubscriptionAccess(
  did: Did.parse('did:plc:alice'),
  effectiveTier: tier,
  givesAccess: true,
);

Widget _app(Widget home) => MaterialApp(
  localizationsDelegates: AppLocalizations.localizationsDelegates,
  supportedLocales: AppLocalizations.supportedLocales,
  home: Scaffold(body: home),
);
