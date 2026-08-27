import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/external_card.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/image_cache_fakes.dart';

void main() {
  testWidgets(
    'UT-015 derives bounded presentation and launches exact final URI',
    (
      tester,
    ) async {
      Uri? launched;
      const external = PostExternal(
        uri: 'https://EXAMPLE.com:8443/pattern?token=hidden#section',
        title: 'A very long pattern title that must remain bounded in the card',
        description: '',
      );
      await tester.pumpWidget(
        _app(
          SizedBox(
            width: 240,
            child: ExternalCard(
              external: external,
              launchUrl: (uri) async {
                launched = uri;
                return true;
              },
            ),
          ),
        ),
      );

      expect(find.text('example.com:8443'), findsOneWidget);
      expect(find.text('token=hidden'), findsNothing);
      expect(find.byType(ExternalCard), findsOneWidget);
      expect(tester.takeException(), isNull);
      await tester.tap(find.byType(ExternalCard));
      await tester.pumpAndSettle();
      expect(launched, isNull);
      expect(find.text('Open link?'), findsOneWidget);
      await tester.tap(find.text('Open link'));
      await tester.pumpAndSettle();
      expect(
        launched.toString(),
        'https://example.com:8443/pattern?token=hidden#section',
      );
    },
  );

  testWidgets('UT-015 compact card collapses optional description safely', (
    tester,
  ) async {
    await tester.pumpWidget(
      _app(
        const ExternalCard(
          external: PostExternal(
            uri: 'https://example.com/pattern',
            title: 'Pattern',
            description: '',
          ),
          variant: ExternalCardVariant.compact,
        ),
      ),
    );

    expect(find.text('Pattern'), findsOneWidget);
    expect(find.text('example.com'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('AT-004 thumbnail card remains bounded at large text', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          feedImageCacheManagerProvider.overrideWithValue(
            FakeBaseCacheManager(),
          ),
        ],
        child: _app(
          MediaQuery(
            data: const MediaQueryData(textScaler: TextScaler.linear(2)),
            child: SizedBox(
              width: 260,
              child: ExternalCard(
                external: PostExternal(
                  uri: 'https://example.com/pattern',
                  title: 'A long accessible pattern title',
                  description: 'A long description that remains safely bounded',
                  thumb: PostExternalThumb(
                    cid: 'bafythumb',
                    mime: 'image/png',
                    size: 3,
                    url: 'https://appview.example/v1/blobs/bafythumb',
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );

    expect(tester.takeException(), isNull);
    expect(find.text('example.com'), findsOneWidget);
    expect(
      tester.getTopLeft(find.byKey(const Key('external-card-thumbnail'))).dy,
      lessThan(
        tester.getTopLeft(find.text('A long accessible pattern title')).dy,
      ),
    );
    expect(
      tester
          .widget<DecoratedBox>(
            find.byKey(const Key('external-card-outline')),
          )
          .position,
      DecorationPosition.foreground,
    );
  });
}

Widget _app(Widget child) => ProviderScope(
  child: MaterialApp(
    theme: AppTheme.lightThemeData,
    localizationsDelegates: AppLocalizations.localizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    home: Scaffold(body: child),
  ),
);
