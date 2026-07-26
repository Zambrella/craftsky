import 'dart:ui' show Tristate;

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/widgets/profile_actions.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
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
    testWidgets(
      'visitor profile exposes mute and moves share into compact More menu',
      (
        tester,
      ) async {
        var reports = 0;
        var mutes = 0;
        var blocks = 0;
        var shares = 0;
        await _pump(
          tester,
          ProfileActions(
            actions: VisitorProfileActionSet(
              isFollowing: false,
              isBusy: false,
              onFollowToggle: () {},
              onShare: () => shares++,
              onReport: () => reports++,
              onMuteToggle: () => mutes++,
              onBlockToggle: () => blocks++,
            ),
          ),
        );

        expect(find.byTooltip('Mute account'), findsOneWidget);
        expect(find.byTooltip('Share'), findsNothing);

        await tester.tap(find.byTooltip('Mute account'));
        await tester.pumpAndSettle();
        expect(mutes, 1);

        await tester.tap(find.byTooltip('More profile actions'));
        await tester.pumpAndSettle();
        expect(find.byType(BottomSheet), findsOneWidget);
        expect(find.text('Share'), findsOneWidget);
        expect(find.text('Mute account'), findsNothing);
        expect(find.text('Block account'), findsOneWidget);
        expect(find.text('Report profile'), findsOneWidget);

        await tester.tap(find.text('Share'));
        await tester.pumpAndSettle();
        expect(shares, 1);

        await tester.tap(find.byTooltip('More profile actions'));
        await tester.pumpAndSettle();
        await tester.tap(find.text('Block account'));
        await tester.pumpAndSettle();
        expect(blocks, 1);

        await tester.tap(find.byTooltip('More profile actions'));
        await tester.pumpAndSettle();
        await tester.tap(find.text('Report profile'));

        expect(reports, 1);
      },
    );

    testWidgets('current relationship state changes visible mute action', (
      tester,
    ) async {
      await _pump(
        tester,
        ProfileActions(
          actions: VisitorProfileActionSet(
            isFollowing: false,
            isBusy: false,
            isMuted: true,
            isBlocking: true,
            onFollowToggle: () {},
            onShare: () {},
            onReport: () {},
            onMuteToggle: () {},
            onBlockToggle: () {},
          ),
        ),
      );

      expect(find.byTooltip('Unmute account'), findsOneWidget);
      await tester.tap(find.byTooltip('More profile actions'));
      await tester.pumpAndSettle();
      expect(find.text('Unmute account'), findsNothing);
      expect(find.text('Unblock account'), findsOneWidget);
    });

    testWidgets('wide profile More menu opens an anchored popup', (
      tester,
    ) async {
      tester.view.physicalSize = const Size(1200, 800);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);

      await _pump(
        tester,
        ProfileActions(
          actions: VisitorProfileActionSet(
            isFollowing: false,
            isBusy: false,
            onFollowToggle: () {},
            onShare: () {},
            onReport: () {},
            onMuteToggle: () {},
            onBlockToggle: () {},
          ),
        ),
      );

      await tester.tap(find.byTooltip('More profile actions'));
      await tester.pumpAndSettle();

      expect(find.byType(BottomSheet), findsNothing);
      expect(
        find.byType(PopupMenuItem<CraftskyContextMenuItem>),
        findsNWidgets(3),
      );
      expect(find.text('Share'), findsOneWidget);
    });

    testWidgets('UT-013 block action exposes destructive button semantics', (
      tester,
    ) async {
      final semantics = tester.ensureSemantics();
      await _pump(
        tester,
        ProfileActions(
          actions: VisitorProfileActionSet(
            isFollowing: false,
            isBusy: false,
            onFollowToggle: () {},
            onShare: () {},
            onReport: () {},
            onMuteToggle: () {},
            onBlockToggle: () {},
          ),
        ),
      );

      await tester.tap(find.byTooltip('More profile actions'));
      await tester.pumpAndSettle();

      final node = tester.getSemantics(
        find.bySemanticsLabel('Block account'),
      );
      final data = node.getSemanticsData();
      expect(data.flagsCollection.isButton, isTrue);
      expect(data.hint, 'Destructive action');
      expect(data.flagsCollection.isEnabled, Tristate.isTrue);
      semantics.dispose();
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
