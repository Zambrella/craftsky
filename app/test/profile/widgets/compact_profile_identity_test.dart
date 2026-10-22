import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile_account_summary.dart';
import 'package:craftsky_app/profile/widgets/compact_profile_identity.dart';
import 'package:craftsky_app/profile/widgets/profile_account_list_tile.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/shared/widgets/craft_icon.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

Widget _wrap(Widget child, {double width = 400, double scale = 1}) =>
    ProviderScope(
      child: MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: Center(
            child: MediaQuery(
              data: MediaQueryData(textScaler: TextScaler.linear(scale)),
              child: SizedBox(width: width, child: child),
            ),
          ),
        ),
      ),
    );

void main() {
  const crafts = ['knitting', 'crochet', 'sewing', 'quilting', 'embroidery'];

  testWidgets('account picture, name and crafts precede the muted handle', (
    tester,
  ) async {
    var profileTaps = 0;
    var actionTaps = 0;
    await tester.pumpWidget(
      _wrap(
        ProfileAccountListTile(
          account: ProfileAccountSummary(
            did: 'did:plc:alice',
            handle: 'alice.test',
            displayName: 'Alice',
            isCraftskyProfile: true,
            crafts: const ['knitting'],
          ),
          onTap: () => profileTaps++,
          trailing: TextButton(
            onPressed: () => actionTaps++,
            child: const Text('Follow'),
          ),
        ),
      ),
    );
    final name = tester.getRect(find.text('Alice'));
    final handle = tester.getRect(find.text('@alice.test'));
    final avatar = tester.getRect(find.byType(ProfileAvatar));
    final icon = tester.getRect(find.byType(CraftIcon));
    expect(avatar.right, lessThan(name.left));
    expect(icon.left, greaterThan(name.right));
    expect(icon.center.dy, closeTo(name.center.dy, 1));
    expect(handle.top, greaterThanOrEqualTo(name.bottom));
    expect(
      tester.widget<Text>(find.text('@alice.test')).style?.color,
      AppTheme.lightThemeData.colorScheme.outline,
    );
    await tester.tap(find.text('Alice'));
    await tester.tap(find.text('Follow'));
    expect(profileTaps, 1);
    expect(actionTaps, 1);
  });

  testWidgets('shows every craft when the name leaves enough room', (
    tester,
  ) async {
    await tester.pumpWidget(
      _wrap(
        const CompactProfileIdentity(
          handle: 'craftsky.social',
          displayName: 'CraftSky',
          crafts: crafts,
        ),
        width: 250,
      ),
    );
    expect(find.byType(CraftIcon), findsNWidgets(5));
    expect(find.textContaining('+'), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets('collapses crafts before truncating a scaled display name', (
    tester,
  ) async {
    await tester.pumpWidget(
      _wrap(
        const CompactProfileIdentity(
          handle: 'hadewijchknits.bsky.social',
          displayName: 'Hadewijch',
          crafts: crafts,
        ),
        width: 240,
        scale: 1.5,
      ),
    );
    final name = tester.renderObject<RenderParagraph>(find.text('Hadewijch'));
    expect(name.didExceedMaxLines, isFalse);
    expect(find.textContaining('+'), findsOneWidget);
    expect(
      tester.widgetList<CraftIcon>(find.byType(CraftIcon)).length,
      lessThan(crafts.length),
    );
    expect(tester.takeException(), isNull);
  });

  for (final width in [100.0, 140.0, 220.0, 400.0]) {
    for (final scale in [1.0, 2.0, 3.0]) {
      testWidgets('craft overflow at width $width and text scale $scale', (
        tester,
      ) async {
        await tester.pumpWidget(
          _wrap(
            const CompactProfileIdentity(
              handle: 'a.very.long.handle.example.test',
              displayName: 'A very long display name that must truncate',
              crafts: crafts,
            ),
            width: width,
            scale: scale,
          ),
        );
        expect(tester.takeException(), isNull);
        final visible = tester
            .widgetList<CraftIcon>(find.byType(CraftIcon))
            .length;
        final hidden = crafts.length - visible;
        expect(
          find.textContaining('+'),
          hidden > 0 ? findsOneWidget : findsNothing,
        );
        if (hidden > 0) expect(find.text('+$hidden'), findsOneWidget);
        if (width == 100) expect(hidden, greaterThan(0));
        // The full list is available even when some icons are hidden.
        final tooltip = tester.widget<Tooltip>(find.byType(Tooltip));
        expect(tooltip.message, contains('Knitting'));
        expect(tooltip.message, contains('Embroidery'));
      });
    }
  }

  testWidgets('missing name, unknown crafts and duplicates remain readable', (
    tester,
  ) async {
    await tester.pumpWidget(
      _wrap(
        const CompactProfileIdentity(
          handle: 'alice.test',
          crafts: [
            'knitting',
            'social.craftsky.actor.defs#knitting',
            'pottery',
          ],
        ),
      ),
    );
    expect(find.text('alice.test'), findsOneWidget);
    expect(find.text('@alice.test'), findsOneWidget);
    expect(find.byType(CraftIcon), findsOneWidget);
    expect(find.text('+1'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });
}
