import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/data/profile_link_policy.dart';
import 'package:craftsky_app/profile/models/profile_link.dart';
import 'package:craftsky_app/profile/widgets/profile_links.dart';
import 'package:craftsky_app/profile/widgets/profile_links_editor.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';
import 'package:material_ui/material_ui.dart';

Widget _app(Widget child, {Dio? dio}) => ProviderScope(
  overrides: [if (dio != null) dioProvider.overrideWithValue(dio)],
  child: MaterialApp(
    theme: AppTheme.lightThemeData,
    localizationsDelegates: appLocalizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    home: Scaffold(body: SingleChildScrollView(child: child)),
  ),
);

void main() {
  for (final (size, keyboardHeight) in [
    (const Size(1200, 800), 440.0),
    (const Size(390, 844), 340.0),
  ]) {
    testWidgets('add link avoids keyboard and restores position at $size', (
      tester,
    ) async {
      tester.view.devicePixelRatio = 1;
      tester.view.physicalSize = size;
      final keyboardTop = size.height - keyboardHeight;
      addTearDown(tester.view.resetDevicePixelRatio);
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetViewInsets);

      await tester.pumpWidget(
        _app(ProfileLinksEditor(links: const [], onChanged: (_) {})),
      );
      await tester.tap(find.byKey(const Key('profile-link-add')));
      await tester.pumpAndSettle();

      final surface = find
          .descendant(
            of: find.byType(CraftskyDialog),
            matching: find.byType(Material),
          )
          .first;
      final originalRect = tester.getRect(surface);
      final url = find.byType(TextField).first;
      await tester.tap(url);
      await tester.enterText(url, 'https://example.com');
      tester.view.viewInsets = FakeViewPadding(bottom: keyboardHeight);
      await tester.pumpAndSettle();

      expect(tester.getRect(surface).bottom, lessThanOrEqualTo(keyboardTop));
      expect(tester.getRect(surface).top, lessThan(originalRect.top));
      expect(tester.takeException(), isNull);

      // Short viewports must allow the remaining fields and actions to scroll
      // into the visible area above the keyboard.
      final scrollable = find
          .descendant(
            of: find.byType(CraftskyDialog),
            matching: find.byType(Scrollable),
          )
          .first;
      final save = find.byKey(const Key('profile-link-save'));
      await tester.scrollUntilVisible(save, 80, scrollable: scrollable);
      await tester.pumpAndSettle();
      expect(tester.getRect(save).bottom, lessThanOrEqualTo(keyboardTop));

      tester.view.viewInsets = FakeViewPadding.zero;
      await tester.pumpAndSettle();
      expect(tester.getRect(surface), originalRect);
      expect(find.text('https://example.com'), findsOneWidget);

      await tester.tap(find.text('Cancel'));
      await tester.pumpAndSettle();
      expect(find.byType(CraftskyDialog), findsNothing);

      // Reopening after dismissal also retains the centered layout.
      await tester.tap(find.byKey(const Key('profile-link-add')));
      await tester.pumpAndSettle();
      expect(tester.getRect(surface), originalRect);
    });
  }

  test('overflow retains the first row when the second row must make room', () {
    expect(profileLinkVisibleCount([300, 300, 300], 320), 1);
    expect(profileLinkVisibleCount([250, 250, 250], 320), 2);
    expect(profileLinkVisibleCount([100, 100], 320), 2);
  });

  test(
    'matches HTTPS, duplicate normalization and support-provider policy',
    () {
      expect(
        normalizeProfileLink('example.com'),
        Uri.parse('https://example.com/'),
      );
      for (final url in [
        'http://example.com',
        'https://bit.ly/a',
        'https://user:secret@example.com',
        'javascript:alert(1)',
      ]) {
        expect(normalizeProfileLink(url), isNull);
      }
      expect(profileLinkSupportProvider('https://ko-fi.com/maker'), 'Ko-fi');
      expect(profileLinkSupportProvider('https://ko-fi.com/'), isNull);
      expect(
        profileLinkSupportProvider('https://ko-fi.com.evil.example/maker'),
        isNull,
      );
      expect(profileLinkSupportProvider('https://paypal.com/about'), isNull);
      expect(
        profileLinkSupportProvider('https://paypal.com/paypalme/maker'),
        'PayPal',
      );
    },
  );

  testWidgets('tap confirms external link; cancel does not open it', (
    tester,
  ) async {
    Uri? opened;
    const link = ProfileLink(
      url: 'https://example.com/shop?item=1#details',
      title: 'Shop',
    );
    await tester.pumpWidget(
      _app(
        ProfileLinks(
          links: const [link],
          launchExternal: (uri) async {
            opened = uri;
            return true;
          },
        ),
      ),
    );
    final ink = tester.widget<InkWell>(
      find.descendant(
        of: find.byType(ProfileLinkPill),
        matching: find.byType(InkWell),
      ),
    );
    expect(ink.onLongPress, isNull);
    await tester.tap(find.text('Shop'));
    await tester.pumpAndSettle();
    expect(opened, isNull);
    expect(find.byType(CraftskyDialog), findsOneWidget);
    expect(find.text('Open link?'), findsOneWidget);
    expect(find.text('This will open outside CraftSky.'), findsOneWidget);
    expect(find.text(link.url), findsOneWidget);
    expect(find.text('Copy link'), findsNothing);
    expect(find.text('Report link'), findsNothing);
    await tester.tap(find.text('Cancel'));
    await tester.pumpAndSettle();
    expect(opened, isNull);
    expect(find.byType(CraftskyDialog), findsNothing);
    await tester.tap(find.text('Shop'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Open link'));
    await tester.pumpAndSettle();
    expect(opened.toString(), link.url);
    expect(find.byType(CraftskyDialog), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets(
    'two-line row expands overflow and remains usable at large text scale',
    (tester) async {
      final links = List.generate(
        10,
        (i) => ProfileLink(url: 'https://example.com/$i', title: 'Website $i'),
      );
      await tester.pumpWidget(
        _app(
          SizedBox(
            width: 320,
            child: MediaQuery(
              data: const MediaQueryData(textScaler: TextScaler.linear(1.5)),
              child: ProfileLinks(links: links),
            ),
          ),
        ),
      );
      expect(find.byKey(const Key('profile-links-more')), findsOneWidget);
      await tester.tap(find.byKey(const Key('profile-links-more')));
      await tester.pump();
      expect(find.byType(ProfileLinkPill), findsNWidgets(10));
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets(
    'editor adds normalized link locally, reorders, removes and reopens',
    (tester) async {
      final dio = Dio(BaseOptions(baseUrl: 'https://appview.example'));
      DioAdapter(dio: dio).onPost(
        '/v1/profile-link-previews',
        (server) => server.reply(200, {'reachability': 'reachable'}),
        data: {'url': 'https://example.com/'},
      );
      var links = <ProfileLink>[
        const ProfileLink(
          url: 'https://other.example/',
          title: 'Other',
          rkey: '3m5abc2345672',
          createdAt: '2026-10-27T12:00:00Z',
        ),
      ];
      await tester.pumpWidget(
        _app(
          StatefulBuilder(
            builder: (context, setState) => ProfileLinksEditor(
              links: links,
              onChanged: (next) => setState(() => links = next),
            ),
          ),
          dio: dio,
        ),
      );
      await tester.tap(find.byKey(const Key('profile-link-add')));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(TextField).first, 'example.com');
      await tester.enterText(find.byType(TextField).last, 'My website');
      await tester.tap(find.byKey(const Key('profile-link-save')));
      await tester.pumpAndSettle();
      expect(links.map((link) => link.url), [
        'https://other.example/',
        'https://example.com/',
      ]);
      expect(
        links.last.recordKey,
        matches(
          RegExp(r'^[234567abcdefghij][234567abcdefghijklmnopqrstuvwxyz]{12}$'),
        ),
      );
      expect(find.text('Add up to 10 links'), findsOneWidget);
      expect(find.byType(ProfileLinkPill), findsNothing);
      await tester.tap(
        find.byKey(ValueKey('profile-link-drag-${links.last.recordKey}')),
      );
      await tester.pumpAndSettle();
      expect(find.byType(TextField), findsNothing);
      final firstPill = tester.getCenter(find.byType(ListTile).first);
      final lastPill = tester.getCenter(
        find.byKey(ValueKey('profile-link-drag-${links.last.recordKey}')),
      );
      final drag = await tester.startGesture(lastPill);
      await tester.pump(const Duration(milliseconds: 600));
      await drag.moveTo(firstPill);
      await tester.pump();
      await drag.up();
      await tester.pumpAndSettle();
      expect(links.first.title, 'My website');
      await tester.tap(find.byType(ListTile).last);
      await tester.pumpAndSettle();
      expect(find.text('Remove link'), findsNothing);
      await tester.tap(find.text('Cancel'));
      await tester.pumpAndSettle();
      await tester.tap(find.byType(CraftskyContextMenuButton).last);
      await tester.pumpAndSettle();
      await tester.tap(find.text('Remove link'));
      await tester.pumpAndSettle();
      expect(links.single.url, 'https://example.com/');
      expect(find.byIcon(Icons.drag_handle), findsNothing);
      // Reopening uses the same local draft until the parent profile is saved.
      await tester.tap(find.byType(ListTile).first);
      await tester.pumpAndSettle();
      expect(
        find.widgetWithText(TextField, 'https://example.com/'),
        findsOneWidget,
      );
      expect(find.widgetWithText(TextField, 'My website'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets('unreachable link requires explicit Save anyway', (tester) async {
    final dio = Dio(BaseOptions(baseUrl: 'https://appview.example'));
    DioAdapter(dio: dio).onPost(
      '/v1/profile-link-previews',
      (server) => server.reply(200, {'reachability': 'unreachable'}),
      data: {'url': 'https://offline.example/'},
    );
    var links = <ProfileLink>[];
    await tester.pumpWidget(
      _app(
        StatefulBuilder(
          builder: (context, setState) => ProfileLinksEditor(
            links: links,
            onChanged: (next) => setState(() => links = next),
          ),
        ),
        dio: dio,
      ),
    );
    await tester.tap(find.byKey(const Key('profile-link-add')));
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byType(TextField).first,
      'https://offline.example/',
    );
    await tester.tap(find.byKey(const Key('profile-link-save')));
    await tester.pumpAndSettle();
    expect(links, isEmpty);
    expect(find.text('Save anyway'), findsOneWidget);
    await tester.tap(find.text('Save anyway'));
    await tester.pumpAndSettle();
    expect(links.single.url, 'https://offline.example/');
  });
}
