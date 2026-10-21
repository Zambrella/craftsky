import 'dart:ui' show Tristate;

import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/widgets/profile_actions.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/chunky_icon_button.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

Future<void> _pump(WidgetTester tester, Widget child) {
  return tester.pumpWidget(
    MaterialApp(
      theme: AppTheme.lightThemeData,
      localizationsDelegates: appLocalizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: Scaffold(body: Center(child: child)),
    ),
  );
}

void main() {
  group('ProfileActions', () {
    testWidgets('self profile copy menu fits a narrow screen with large text', (
      tester,
    ) async {
      tester.view.physicalSize = const Size(320, 568);
      tester.view.devicePixelRatio = 1;
      tester.platformDispatcher.textScaleFactorTestValue = 2;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      addTearDown(tester.platformDispatcher.clearTextScaleFactorTestValue);
      await _pump(
        tester,
        ProfileActions(
          actions: SelfProfileActionSet(
            onEdit: () {},
            onSettings: () {},
            onCopyLink: () {},
          ),
        ),
      );
      expect(tester.takeException(), isNull);
      await tester.tap(find.byType(CraftskyContextMenuButton));
      await tester.pumpAndSettle();
      expect(find.text('Copy link'), findsOneWidget);
      expect(tester.takeException(), isNull);
    });

    for (final own in [false, true]) {
      testWidgets(
        '${own ? 'self' : 'visitor'} profile menu exposes copy link',
        (tester) async {
          var copied = false;
          final actions = own
              ? SelfProfileActionSet(
                  onEdit: () {},
                  onSettings: () {},
                  onCopyLink: () => copied = true,
                )
              : VisitorProfileActionSet(
                  isFollowing: false,
                  isBusy: false,
                  onFollowToggle: () {},
                  onShare: () {},
                  onReport: () {},
                  onMuteToggle: () {},
                  onBlockToggle: () {},
                  onCopyLink: () => copied = true,
                );
          await _pump(tester, ProfileActions(actions: actions));
          await tester.tap(find.byType(CraftskyContextMenuButton));
          await tester.pumpAndSettle();
          await tester.tap(find.text('Copy link'));
          await tester.pumpAndSettle();
          expect(copied, isTrue);
        },
      );
    }

    testWidgets('following spins only Follow and disables competing actions', (
      tester,
    ) async {
      await _pump(
        tester,
        ProfileActions(
          actions: VisitorProfileActionSet(
            isFollowing: false,
            isBusy: true,
            isFollowingBusy: true,
            onFollowToggle: () {},
            onShare: () {},
            onReport: () {},
            onMuteToggle: () {},
            onBlockToggle: () {},
          ),
        ),
      );
      expect(
        find.descendant(
          of: find.byType(ChunkyButton).first,
          matching: find.byType(StitchProgressIndicator),
        ),
        findsOneWidget,
      );
      expect(
        find.descendant(
          of: find.byType(ChunkyIconButton),
          matching: find.byType(StitchProgressIndicator),
        ),
        findsNothing,
      );
      expect(find.byType(ChunkyIconButton), findsNothing);
      expect(
        tester
            .widget<CraftskyContextMenuButton>(
              find.byType(CraftskyContextMenuButton),
            )
            .enabled,
        isFalse,
      );
    });
    testWidgets('optimistic mute disables competing profile actions', (
      tester,
    ) async {
      await _pump(
        tester,
        ProfileActions(
          actions: VisitorProfileActionSet(
            isFollowing: false,
            isBusy: true,
            isMuted: true,
            onFollowToggle: () {},
            onShare: () {},
            onReport: () {},
            onMuteToggle: () {},
            onBlockToggle: () {},
          ),
        ),
      );
      expect(find.byType(StitchProgressIndicator), findsNothing);
      expect(find.byTooltip('Unmute account'), findsNothing);
      expect(
        tester.widget<ChunkyButton>(find.byType(ChunkyButton).first).onPressed,
        isNull,
      );
    });

    testWidgets(
      'visitor profile moves mute and share into compact More menu',
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

        expect(find.byTooltip('Mute account'), findsNothing);
        expect(find.byTooltip('Share'), findsNothing);

        await tester.tap(find.byTooltip('More profile actions'));
        await tester.pumpAndSettle();
        expect(find.byType(BottomSheet), findsOneWidget);
        expect(find.text('Share'), findsOneWidget);
        expect(find.text('Mute account'), findsOneWidget);
        expect(find.text('Block account'), findsOneWidget);
        expect(find.text('Report profile'), findsOneWidget);

        await tester.tap(find.text('Mute account'));
        await tester.pumpAndSettle();
        expect(mutes, 1);
        await tester.tap(find.byTooltip('More profile actions'));
        await tester.pumpAndSettle();
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
      var unmuted = 0;
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
            onMuteToggle: () => unmuted++,
            onBlockToggle: () {},
          ),
        ),
      );

      expect(find.byTooltip('Unmute account'), findsNothing);
      await tester.tap(find.byTooltip('More profile actions'));
      await tester.pumpAndSettle();
      expect(find.text('Unmute account'), findsOneWidget);
      expect(find.text('Unblock account'), findsOneWidget);
      await tester.tap(find.text('Unmute account'));
      await tester.pumpAndSettle();
      expect(unmuted, 1);
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
        findsNWidgets(4),
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
