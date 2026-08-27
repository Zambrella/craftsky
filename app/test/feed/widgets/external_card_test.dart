import 'package:craftsky_app/feed/media/youtube_consent.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/external_card.dart';
import 'package:craftsky_app/feed/widgets/youtube_inline_player.dart';
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

  testWidgets('YouTube card asks for consent then creates a lazy player', (
    tester,
  ) async {
    final consent = _FakeYouTubeConsent();
    Uri? launched;
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          youtubeConsentPreferencesProvider.overrideWithValue(consent),
          youtubePlayerBuilderProvider.overrideWithValue(
            (context, external, onPlaybackError) => Text(
              'player:${external.videoId}:${external.startSeconds}',
              key: const Key('fake-youtube-player'),
            ),
          ),
        ],
        child: _materialApp(
          ExternalCard(
            external: const PostExternal(
              uri: 'https://youtu.be/dQw4w9WgXcQ?t=90',
              title: 'Knitting tutorial',
              description: 'A tutorial',
            ),
            launchUrl: (uri) async {
              launched = uri;
              return true;
            },
          ),
        ),
      ),
    );

    expect(find.byKey(const Key('youtube-play-indicator')), findsOneWidget);
    expect(find.byKey(const Key('fake-youtube-player')), findsNothing);

    await tester.tap(find.byType(ExternalCard));
    await tester.pumpAndSettle();
    expect(find.text('Play video from YouTube?'), findsOneWidget);
    expect(find.byKey(const Key('fake-youtube-player')), findsNothing);

    await tester.tap(find.text('Allow once'));
    await tester.pumpAndSettle();
    expect(find.text('player:dQw4w9WgXcQ:90'), findsOneWidget);
    expect(find.text('Open in YouTube'), findsOneWidget);
    expect(consent.setAlwaysAllowCalls, 0);

    await tester.tap(find.text('Open in YouTube'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Open link'));
    await tester.pumpAndSettle();
    expect(launched, Uri.parse('https://youtu.be/dQw4w9WgXcQ?t=90'));
  });

  testWidgets('remembered YouTube consent skips the disclosure', (
    tester,
  ) async {
    final consent = _FakeYouTubeConsent(alwaysAllow: true);
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          youtubeConsentPreferencesProvider.overrideWithValue(consent),
          youtubePlayerBuilderProvider.overrideWithValue(
            (context, external, onPlaybackError) =>
                const SizedBox(key: Key('fake-youtube-player')),
          ),
        ],
        child: _materialApp(
          const ExternalCard(
            external: PostExternal(
              uri: 'https://www.youtube.com/watch?v=dQw4w9WgXcQ',
              title: 'Crochet tutorial',
              description: '',
            ),
          ),
        ),
      ),
    );

    await tester.tap(find.byType(ExternalCard));
    await tester.pump();
    expect(find.text('Play video from YouTube?'), findsNothing);
    expect(find.byKey(const Key('fake-youtube-player')), findsOneWidget);
  });

  testWidgets('always allow persists YouTube consent', (tester) async {
    final consent = _FakeYouTubeConsent();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          youtubeConsentPreferencesProvider.overrideWithValue(consent),
          youtubePlayerBuilderProvider.overrideWithValue(
            (context, external, onPlaybackError) =>
                const SizedBox(key: Key('fake-youtube-player')),
          ),
        ],
        child: _materialApp(
          const ExternalCard(
            external: PostExternal(
              uri: 'https://youtube.com/shorts/dQw4w9WgXcQ',
              title: 'Short tutorial',
              description: '',
            ),
          ),
        ),
      ),
    );

    await tester.tap(find.byType(ExternalCard));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Always allow YouTube'));
    await tester.pumpAndSettle();

    expect(consent.setAlwaysAllowCalls, 1);
    expect(find.byKey(const Key('fake-youtube-player')), findsOneWidget);
  });

  testWidgets('YouTube playback errors leave a stable external fallback', (
    tester,
  ) async {
    final consent = _FakeYouTubeConsent(alwaysAllow: true);
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          youtubeConsentPreferencesProvider.overrideWithValue(consent),
          youtubePlayerBuilderProvider.overrideWithValue(
            (context, external, onPlaybackError) => TextButton(
              key: const Key('simulate-youtube-error'),
              onPressed: onPlaybackError,
              child: const Text('Simulate error'),
            ),
          ),
        ],
        child: _materialApp(
          const ExternalCard(
            external: PostExternal(
              uri: 'https://youtu.be/pQ9NBUuwDMg',
              title: '- YouTube',
              description: '',
            ),
          ),
        ),
      ),
    );

    await tester.tap(find.byType(ExternalCard));
    await tester.pump();
    await tester.tap(find.byKey(const Key('simulate-youtube-error')));
    await tester.pump();

    expect(find.text('Simulate error'), findsNothing);
    expect(
      find.text(
        'This video can’t be played here. It may be private, unavailable, '
        'or restricted from embedded playback.',
      ),
      findsOneWidget,
    );
    expect(find.text('Open in YouTube'), findsOneWidget);
  });

  testWidgets('compact YouTube cards remain ordinary external links', (
    tester,
  ) async {
    Uri? launched;
    await tester.pumpWidget(
      _app(
        ExternalCard(
          external: const PostExternal(
            uri: 'https://youtu.be/dQw4w9WgXcQ',
            title: 'Compact tutorial',
            description: '',
          ),
          variant: ExternalCardVariant.compact,
          launchUrl: (uri) async {
            launched = uri;
            return true;
          },
        ),
      ),
    );

    expect(find.byKey(const Key('youtube-play-indicator')), findsNothing);
    await tester.tap(find.byType(ExternalCard));
    await tester.pumpAndSettle();
    expect(find.text('Play video from YouTube?'), findsNothing);
    expect(find.text('Open link?'), findsOneWidget);
    await tester.tap(find.text('Open link'));
    await tester.pumpAndSettle();
    expect(launched, Uri.parse('https://youtu.be/dQw4w9WgXcQ'));
  });
}

Widget _app(Widget child) => ProviderScope(
  child: _materialApp(child),
);

Widget _materialApp(Widget child) => MaterialApp(
  theme: AppTheme.lightThemeData,
  localizationsDelegates: AppLocalizations.localizationsDelegates,
  supportedLocales: AppLocalizations.supportedLocales,
  home: Scaffold(body: child),
);

final class _FakeYouTubeConsent implements YouTubeConsentPreferences {
  _FakeYouTubeConsent({this.alwaysAllow = false});

  @override
  bool alwaysAllow;

  int setAlwaysAllowCalls = 0;

  @override
  Future<void> setAlwaysAllow() async {
    setAlwaysAllowCalls += 1;
    alwaysAllow = true;
  }
}
