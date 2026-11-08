import 'package:craftsky_app/shared/link/external_link.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  test('canonical app links retain their encoded path, query and fragment', () {
    expect(
      craftskyLinkLocation(
        Uri.parse('https://app.craftsky.social/profiles/a%20b?tab=posts#top'),
      ),
      '/profiles/a%20b?tab=posts#top',
    );
    expect(craftskyLinkLocation(Uri.parse('https://app.craftsky.social')), '/');
    for (final value in [
      'https://app.craftsky.social.evil.example/feed',
      'https://craftsky.social/feed',
      'https://user@app.craftsky.social/feed',
      'https://app.craftsky.social:8443/feed',
      'https://app.craftsky.social//evil.example',
      'ftp://app.craftsky.social/feed',
    ]) {
      expect(craftskyLinkLocation(Uri.parse(value)), isNull, reason: value);
    }
  });

  for (final hydrated in [false, true]) {
    testWidgets('internal link pushes a route and supports back ($hydrated)', (
      tester,
    ) async {
      var confirmations = 0;
      var launches = 0;
      final router = GoRouter(
        routes: [
          GoRoute(
            path: '/',
            builder: (context, _) => TextButton(
              onPressed: () async {
                final open = hydrated
                    ? confirmAndLaunchExternalAction
                    : confirmAndLaunchExternalLink;
                await open(
                  context,
                  uri: Uri.parse(
                    'https://app.craftsky.social/posts/alice/post?focus=reply#top',
                  ),
                  confirmOpenLink: (_, _) async {
                    confirmations++;
                    return false;
                  },
                  launchUrl: (_) async {
                    launches++;
                    return true;
                  },
                );
              },
              child: const Text('Open'),
            ),
          ),
          GoRoute(
            path: '/posts/:did/:rkey',
            builder: (_, state) => Text('Destination ${state.uri}'),
          ),
        ],
      );
      addTearDown(router.dispose);
      await tester.pumpWidget(MaterialApp.router(routerConfig: router));
      await tester.tap(find.text('Open'));
      await tester.pumpAndSettle();
      expect(
        find.text('Destination /posts/alice/post?focus=reply#top'),
        findsOneWidget,
      );
      expect(confirmations, 0);
      expect(launches, 0);
      router.pop();
      await tester.pumpAndSettle();
      expect(find.text('Open'), findsOneWidget);
    });

    testWidgets('external link still requires confirmation ($hydrated)', (
      tester,
    ) async {
      var confirmations = 0;
      var launches = 0;
      await tester.pumpWidget(
        MaterialApp(
          home: Builder(
            builder: (context) {
              return TextButton(
                onPressed: () async {
                  final open = hydrated
                      ? confirmAndLaunchExternalAction
                      : confirmAndLaunchExternalLink;
                  await open(
                    context,
                    uri: Uri.parse('https://example.com/shop'),
                    confirmOpenLink: (_, _) async {
                      confirmations++;
                      return false;
                    },
                    launchUrl: (_) async {
                      launches++;
                      return true;
                    },
                  );
                },
                child: const Text('Open'),
              );
            },
          ),
        ),
      );
      await tester.tap(find.text('Open'));
      await tester.pumpAndSettle();
      expect(confirmations, 1);
      expect(launches, 0);
    });
  }
}
