import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/pages/projects_page.dart';
import 'package:craftsky_app/projects/providers/project_repository_provider.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/auth_session_fakes.dart';
import '../fakes/fake_project_repository.dart';

Post _post(int index) => PostMapper.fromMap({
  'uri': 'at://did:plc:alice/social.craftsky.feed.post/project-$index',
  'cid': 'bafy_project_$index',
  'rkey': 'project-$index',
  'text': 'Project $index',
  'tags': <String>[],
  'likeCount': 0,
  'repostCount': 0,
  'replyCount': 0,
  'viewerHasLiked': false,
  'viewerHasReposted': false,
  'viewerHasSaved': false,
  'createdAt': '2026-05-04T18:23:45.000Z',
  'indexedAt': '2026-05-04T18:23:47.000Z',
  'author': {'did': 'did:plc:alice', 'handle': 'alice.craftsky.social'},
});

void main() {
  setUpAll(initializeMappers);

  testWidgets('applying a filter returns the project list to the top', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(800, 600);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final posts = List.generate(20, _post);
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authSessionProvider.overrideWith(SignedOutAuthSession.new),
          activeLanguagePreferencesProvider.overrideWith(
            (ref) => const LanguagePreferences(
              primaryLanguage: 'en',
              contentLanguages: ['en'],
            ),
          ),
          projectRepositoryProvider.overrideWithValue(
            FakeProjectRepository(
              onListProjects: ({required query, limit, cursor}) async =>
                  PostPage(items: posts),
            ),
          ),
        ],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const ProjectsPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.drag(
      find.byType(CustomScrollView).first,
      const Offset(0, -1200),
    );
    await tester.pumpAndSettle();
    expect(find.text('Project 0'), findsNothing);

    await tester.tap(find.text('Filters'));
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byKey(const Key('Material-custom-input')),
      'alpaca',
    );
    await tester.testTextInput.receiveAction(TextInputAction.done);
    await tester.pump();
    tester
        .widget<FilledButton>(
          find.widgetWithText(FilledButton, 'Apply filters'),
        )
        .onPressed!();
    await tester.pumpAndSettle();

    expect(find.text('Project 0'), findsOneWidget);

    await tester.drag(
      find.byType(CustomScrollView).first,
      const Offset(0, -1200),
    );
    await tester.pumpAndSettle();
    expect(find.text('Project 0'), findsNothing);

    await tester.tap(find.text('Filters'));
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byKey(const Key('Material-custom-input')),
      'cotton',
    );
    await tester.testTextInput.receiveAction(TextInputAction.done);
    await tester.pump();
    tester
        .widget<FilledButton>(
          find.widgetWithText(FilledButton, 'Apply filters'),
        )
        .onPressed!();
    await tester.pumpAndSettle();

    expect(find.text('Project 0'), findsOneWidget);
  });

  testWidgets('swiping craft tabs updates the filter sheet craft', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authSessionProvider.overrideWith(SignedOutAuthSession.new),
          activeLanguagePreferencesProvider.overrideWith(
            (ref) => const LanguagePreferences(
              primaryLanguage: 'en',
              contentLanguages: ['en'],
            ),
          ),
          projectRepositoryProvider.overrideWithValue(
            FakeProjectRepository(
              onListProjects: ({required query, limit, cursor}) async =>
                  const PostPage(items: []),
            ),
          ),
        ],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const ProjectsPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.drag(find.byType(TabBarView), const Offset(-500, 0));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Filters'));
    await tester.pumpAndSettle();

    expect(find.textContaining('Crochet', findRichText: true), findsOneWidget);
  });
}
