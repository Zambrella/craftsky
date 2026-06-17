import 'package:craftsky_app/shared/rich_text/providers/facet_action_providers.dart';
import 'package:craftsky_app/shared/rich_text/widgets/faceted_text.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

void main() {
  group('FacetedText actions', () {
    testWidgets('AT-006 tapping a mention navigates by visible handle', (
      tester,
    ) async {
      final router = _router(
        FacetedText(
          text: '@alice.craftsky.social',
          facets: [
            _facet(0, 22, {
              r'$type': 'app.bsky.richtext.facet#mention',
              'did': 'did:plc:alice',
            }),
          ],
        ),
      );

      await _pump(tester, router: router);
      await tester.tap(find.text('@alice.craftsky.social'));
      await tester.pumpAndSettle();

      expect(find.text('profile:alice.craftsky.social'), findsOneWidget);
    });

    testWidgets('AT-006 tapping a link confirms before launching', (
      tester,
    ) async {
      Uri? launched;
      final router = _router(
        FacetedText(
          text: 'https://craftsky.social',
          facets: [
            _facet(0, 23, {
              r'$type': 'app.bsky.richtext.facet#link',
              'uri': 'https://craftsky.social',
            }),
          ],
        ),
      );

      await _pump(
        tester,
        router: router,
        overrides: [
          facetUrlLauncherProvider.overrideWithValue((uri) async {
            launched = uri;
            return true;
          }),
        ],
      );

      expect(find.text('craftsky.social'), findsOneWidget);
      expect(find.text('https://craftsky.social'), findsNothing);

      await tester.tap(find.text('craftsky.social'));
      await tester.pumpAndSettle();

      expect(find.text('Open link?'), findsOneWidget);
      expect(find.text('This will open outside Craftsky.'), findsOneWidget);
      expect(find.text('https://craftsky.social'), findsOneWidget);
      expect(launched, isNull);

      await tester.tap(find.text('Open link'));
      await tester.pumpAndSettle();

      expect(launched, Uri.parse('https://craftsky.social'));
    });

    testWidgets('AT-006 tapping a hashtag navigates with tag context', (
      tester,
    ) async {
      final router = _router(
        FacetedText(
          text: '#SockKAL',
          facets: [
            _facet(0, 8, {
              r'$type': 'app.bsky.richtext.facet#tag',
              'tag': 'SockKAL',
            }),
          ],
        ),
      );

      await _pump(tester, router: router);
      await tester.tap(find.text('#SockKAL'));
      await tester.pumpAndSettle();

      expect(find.text('search:SockKAL'), findsOneWidget);
    });

    testWidgets('AT-006 destination failures do not crash', (tester) async {
      final router = _router(
        FacetedText(
          text: 'https://craftsky.social',
          facets: [
            _facet(0, 23, {
              r'$type': 'app.bsky.richtext.facet#link',
              'uri': 'https://craftsky.social',
            }),
          ],
        ),
      );

      await _pump(
        tester,
        router: router,
        overrides: [
          facetUrlLauncherProvider.overrideWithValue((_) async {
            throw Exception('launcher failed');
          }),
        ],
      );
      await tester.tap(find.text('craftsky.social'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Open link'));
      await tester.pumpAndSettle();

      expect(tester.takeException(), isNull);
    });
  });
}

Future<void> _pump(
  WidgetTester tester, {
  required GoRouter router,
  List<dynamic> overrides = const [],
}) {
  return tester.pumpWidget(
    ProviderScope(
      overrides: List.from(overrides),
      child: MaterialApp.router(
        theme: AppTheme.lightThemeData,
        routerConfig: router,
      ),
    ),
  );
}

GoRouter _router(Widget home) {
  return GoRouter(
    routes: [
      GoRoute(
        path: '/',
        builder: (context, state) => Scaffold(body: home),
      ),
      GoRoute(
        path: '/profile/:handle',
        builder: (context, state) => Scaffold(
          body: Text('profile:${state.pathParameters['handle']}'),
        ),
      ),
      GoRoute(
        path: '/search',
        builder: (context, state) => Scaffold(
          body: Text('search:${state.uri.queryParameters['tag']}'),
        ),
      ),
    ],
  );
}

Map<String, dynamic> _facet(
  int byteStart,
  int byteEnd,
  Map<String, dynamic> feature,
) {
  return {
    'index': {'byteStart': byteStart, 'byteEnd': byteEnd},
    'features': [feature],
  };
}
