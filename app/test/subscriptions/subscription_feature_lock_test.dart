import 'dart:async';

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/subscription_build_config.dart';
import 'package:craftsky_app/subscriptions/widgets/plus_action_icon.dart';
import 'package:craftsky_app/subscriptions/widgets/plus_feature_lock.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:phosphor_icons/phosphor_icons.dart';

void main() {
  testWidgets('Plus sparkle overlays the action icon', (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(
          body: Center(child: PlusActionIcon(icon: Icons.folder)),
        ),
      ),
    );
    final icon = tester.getRect(find.byIcon(Icons.folder));
    final sparkle = tester.getRect(find.byIcon(CraftskyIcons.plusTier));
    expect(icon.overlaps(sparkle), isTrue);
    expect(CraftskyIcons.plusTier, PhosphorIconsFill.sparkle);
    final sparkleIcon = tester.widget<Icon>(
      find.byIcon(CraftskyIcons.plusTier),
    );
    expect(
      sparkleIcon.color,
      Theme.of(tester.element(find.byType(PlusActionIcon))).colorScheme.primary,
    );
    expect(
      find.descendant(
        of: find.byType(PlusActionIcon),
        matching: find.byType(DecoratedBox),
      ),
      findsNothing,
    );
  }, skip: !subscriptionsEnabled);

  testWidgets('UT-005 Plus action stays locked during access refresh', (
    tester,
  ) async {
    var refreshing = false;
    final pending = Completer<SubscriptionAccess>();
    var actions = 0;
    final provider = FutureProvider<SubscriptionAccess>(
      (ref) => refreshing
          ? pending.future
          : Future.value(
              SubscriptionAccess(
                did: Did.parse('did:plc:test'),
                effectiveTier: SubscriptionTier.plus,
                givesAccess: true,
                assignedTier: SubscriptionTier.plus,
              ),
            ),
    );
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: Center(
              child: Consumer(
                builder: (context, ref, child) => PlusFeatureLock(
                  feature: 'Pinned posts',
                  access: ref.watch(provider),
                  onUnlocked: () => actions++,
                  onRetry: () {},
                  child: const Text('Pin'),
                ),
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    final container = ProviderScope.containerOf(
      tester.element(find.byType(PlusFeatureLock)),
    );
    refreshing = true;
    container.invalidate(provider);
    await tester.pump();
    final loading = container.read(provider);
    expect(loading, isA<AsyncData<SubscriptionAccess>>());
    expect(loading.isLoading, isTrue);
    await tester.pump();
    await tester.tap(find.byType(PlusFeatureLock));
    await tester.pumpAndSettle();
    expect(actions, 0);
    expect(
      find.byIcon(CraftskyIcons.plusTier),
      subscriptionsEnabled ? findsOneWidget : findsNothing,
    );
    expect(find.byType(CraftskyDialog), findsOneWidget);
    await tester.tap(find.text('OK'));
    await tester.pumpAndSettle();
    pending.complete(
      SubscriptionAccess(
        did: Did.parse('did:plc:test'),
        effectiveTier: SubscriptionTier.plus,
        givesAccess: true,
        assignedTier: SubscriptionTier.plus,
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.byType(PlusFeatureLock));
    await tester.pump();
    expect(actions, subscriptionsEnabled ? 1 : 0);
  });

  testWidgets(
    'AT-007 keyboard can activate the labelled upgrade CTA',
    (
      tester,
    ) async {
      var learned = 0;
      final semantics = tester.ensureSemantics();
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: Center(
              child: PlusFeatureLock(
                feature: 'Saved-post folders',
                access: AsyncData(
                  SubscriptionAccess(
                    did: Did.parse('did:plc:test'),
                    effectiveTier: SubscriptionTier.free,
                    givesAccess: false,
                  ),
                ),
                onUnlocked: () => fail('Free action executed'),
                onLearnMore: () => learned++,
                child: const Text('Folders'),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.byType(PlusFeatureLock));
      await tester.pumpAndSettle();
      await tester.sendKeyEvent(LogicalKeyboardKey.tab);
      await tester.pump();
      await tester.sendKeyEvent(LogicalKeyboardKey.tab);
      await tester.pump();
      await tester.sendKeyEvent(LogicalKeyboardKey.enter);
      await tester.pumpAndSettle();
      expect(learned, 1);
      semantics.dispose();
    },
    skip: !subscriptionsEnabled,
  );
  testWidgets(
    'UT-005 uncertain Plus access offers retry without running action',
    (tester) async {
      var actions = 0;
      var retries = 0;
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: Center(
              child: PlusFeatureLock(
                feature: 'Follower growth',
                access: const AsyncError<SubscriptionAccess>(
                  'offline',
                  StackTrace.empty,
                ),
                onUnlocked: () => actions++,
                onRetry: () => retries++,
                child: const Text('Growth'),
              ),
            ),
          ),
        ),
      );
      expect(
        find.bySemanticsLabel(
          'Subscription access is unavailable. Try again before using '
          'Follower growth.',
        ),
        findsOneWidget,
      );
      await tester.tap(find.byType(PlusFeatureLock));
      await tester.pumpAndSettle();
      await tester.tap(find.text('OK'));
      await tester.pumpAndSettle();
      expect(retries, 1);
      expect(actions, 0);
    },
    skip: !subscriptionsEnabled,
  );
  testWidgets(
    'AT-008 Plus action runs without an upgrade dialog',
    (
      tester,
    ) async {
      var actions = 0;
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: Center(
              child: PlusFeatureLock(
                feature: 'Pinned posts',
                access: AsyncData(
                  SubscriptionAccess(
                    did: Did.parse('did:plc:test'),
                    effectiveTier: SubscriptionTier.plus,
                    givesAccess: true,
                    assignedTier: SubscriptionTier.plus,
                  ),
                ),
                onUnlocked: () => actions++,
                child: const Text('Pin'),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('Pin'));
      await tester.pumpAndSettle();
      expect(actions, 1);
      expect(find.byType(CraftskyDialog), findsNothing);
      expect(find.byIcon(CraftskyIcons.plusTier), findsNothing);
    },
    skip: !subscriptionsEnabled,
  );
  testWidgets(
    'AT-007 locked Plus affordance names feature and opens subscriptions',
    (tester) async {
      var actions = 0;
      var learned = 0;
      final semantics = tester.ensureSemantics();
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: Center(
              child: PlusFeatureLock(
                feature: 'Scheduled posts',
                access: AsyncData(
                  SubscriptionAccess(
                    did: Did.parse('did:plc:test'),
                    effectiveTier: SubscriptionTier.free,
                    givesAccess: false,
                  ),
                ),
                onUnlocked: () => actions++,
                onLearnMore: () => learned++,
                child: const Text('Schedule'),
              ),
            ),
          ),
        ),
      );
      expect(find.byIcon(CraftskyIcons.plusTier), findsOneWidget);
      expect(
        find.bySemanticsLabel('Scheduled posts requires Plus'),
        findsOneWidget,
      );
      await tester.tap(find.byType(InkWell).first);
      await tester.pumpAndSettle();
      expect(find.text('Scheduled posts requires Plus'), findsOneWidget);
      expect(actions, 0);
      await tester.tap(find.text('Learn about subscriptions'));
      await tester.pumpAndSettle();
      expect(learned, 1);
      semantics.dispose();
    },
    skip: !subscriptionsEnabled,
  );

  testWidgets(
    'beta shows coming soon even when access appears paid',
    (tester) async {
      var actions = 0;
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: PlusFeatureLock(
              feature: 'Pinned posts',
              access: AsyncData(
                SubscriptionAccess(
                  did: Did.parse('did:plc:test'),
                  effectiveTier: SubscriptionTier.plus,
                  givesAccess: true,
                  assignedTier: SubscriptionTier.plus,
                ),
              ),
              onUnlocked: () => actions++,
              child: const Text('Pin'),
            ),
          ),
        ),
      );
      expect(
        tester
            .widget<Opacity>(
              find.descendant(
                of: find.byType(PlusFeatureLock),
                matching: find.byType(Opacity),
              ),
            )
            .opacity,
        0.72,
      );
      await tester.tap(find.byType(PlusFeatureLock));
      await tester.pumpAndSettle();
      expect(actions, 0);
      expect(find.text('Pinned posts is coming soon'), findsOneWidget);
      expect(
        find.text(
          "Pinned posts isn't available yet. "
          "We'll let you know when it launches.",
        ),
        findsOneWidget,
      );
      expect(find.text('Learn about subscriptions'), findsNothing);
      expect(find.byIcon(CraftskyIcons.plusTier), findsNothing);
    },
    skip: subscriptionsEnabled,
  );
}
