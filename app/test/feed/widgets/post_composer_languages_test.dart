import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/recording_messenger.dart';

final _primaryProvider = NotifierProvider<_Primary, String>(_Primary.new);

final class _Primary extends Notifier<String> {
  @override
  String build() => 'en';

  String get value => state;

  set value(String value) => state = value;
}

void main() {
  testWidgets('IT-019 reads initialized preferences synchronously', (
    tester,
  ) async {
    final container = ProviderContainer.test(
      overrides: [
        activeLanguagePreferencesProvider.overrideWith(
          (ref) => const LanguagePreferences(
            primaryLanguage: 'fr',
            contentLanguages: ['fr'],
          ),
        ),
      ],
    );
    addTearDown(container.dispose);

    await tester.pumpWidget(
      _testApp(container, key: const ValueKey('ready')),
    );
    await tester.pumpAndSettle();
    expect(find.text('French'), findsOneWidget);
  });

  testWidgets(
    'UT-017 open composer keeps its selection and the next uses new Primary',
    (tester) async {
      final container = ProviderContainer.test(
        overrides: [
          activeLanguagePreferencesProvider.overrideWith(
            (ref) {
              final primary = ref.watch(_primaryProvider);
              return LanguagePreferences(
                primaryLanguage: primary,
                contentLanguages: [primary],
              );
            },
          ),
        ],
      );
      addTearDown(container.dispose);

      await tester.pumpWidget(_testApp(container, key: const ValueKey('en')));
      await tester.pumpAndSettle();
      expect(find.text('English'), findsOneWidget);

      container.read(_primaryProvider.notifier).value = 'fr';
      await tester.pumpAndSettle();
      expect(find.text('English'), findsOneWidget);
      expect(find.text('French'), findsNothing);

      await tester.pumpWidget(_testApp(container, key: const ValueKey('fr')));
      await tester.pumpAndSettle();
      expect(find.text('French'), findsOneWidget);
      expect(find.text('English'), findsNothing);
    },
  );

  testWidgets('UT-017 reply does not inherit the parent languages', (
    tester,
  ) async {
    final container = ProviderContainer.test(
      overrides: [
        activeLanguagePreferencesProvider.overrideWith(
          (ref) => const LanguagePreferences(
            primaryLanguage: 'en',
            contentLanguages: ['en'],
          ),
        ),
      ],
    );
    addTearDown(container.dispose);

    await tester.pumpWidget(
      _testApp(
        container,
        key: const ValueKey('reply'),
        replyTarget: _post(langs: const ['fr', 'cy']),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('English'), findsOneWidget);
    expect(find.text('French'), findsNothing);
    expect(find.text('Welsh'), findsNothing);
  });
}

Widget _testApp(
  ProviderContainer container, {
  required Key key,
  Post? replyTarget,
}) => UncontrolledProviderScope(
  container: container,
  child: MessengerScope(
    messenger: RecordingMessenger(),
    child: MaterialApp(
      key: key,
      theme: AppTheme.lightThemeData,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: PostComposerSheet(
        key: key,
        composerId: key.toString(),
        replyTarget: replyTarget,
      ),
    ),
  ),
);

Post _post({required List<String> langs}) => Post(
  uri: 'at://did:plc:alice/social.craftsky.feed.post/parent',
  cid: 'bafy-parent',
  rkey: 'parent',
  text: 'parent',
  langs: langs,
  tags: const [],
  likeCount: 0,
  repostCount: 0,
  replyCount: 0,
  viewerHasLiked: false,
  viewerHasReposted: false,
  viewerHasSaved: false,
  createdAt: DateTime(2026),
  indexedAt: DateTime(2026),
  author: PostAuthor(did: 'did:plc:alice', handle: 'alice.test'),
);
