import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/shared/messaging/app_messenger.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/recording_messenger.dart';
import '../../feed/fakes/fake_post_repository.dart';

typedef ProfilePageRequest = ({String? cursor, int? limit});

Future<void> pumpProfileTab(
  WidgetTester tester, {
  required Widget sliver,
  required FakePostRepository repository,
  AppMessenger? messenger,
  List<dynamic> overrides = const [],
}) {
  return tester.pumpWidget(
    ProviderScope(
      overrides: List.from([
        activeLanguagePreferencesProvider.overrideWith(
          (ref) => const LanguagePreferences(
            primaryLanguage: 'en',
            contentLanguages: ['en'],
          ),
        ),
        postRepositoryProvider.overrideWithValue(repository),
        ...overrides,
      ]),
      child: MessengerScope(
        messenger: messenger ?? RecordingMessenger(),
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(body: CustomScrollView(slivers: [sliver])),
        ),
      ),
    ),
  );
}

Future<void> expectProfileTabInfiniteScroll(
  WidgetTester tester, {
  required Widget sliver,
  required FakePostRepository repository,
  required List<ProfilePageRequest> requests,
  required Finder lastInitialItem,
  required Finder appendedItem,
}) async {
  await pumpProfileTab(tester, sliver: sliver, repository: repository);
  await tester.pumpAndSettle();
  await tester.scrollUntilVisible(
    lastInitialItem,
    500,
    scrollable: find.byType(Scrollable),
  );
  await tester.pumpAndSettle();

  expect(requests, [(cursor: null, limit: 10), (cursor: 'c1', limit: 10)]);
  expect(lastInitialItem, findsOneWidget);
  expect(appendedItem, findsOneWidget);
}
