import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/projects/pages/projects_page.dart';
import 'package:craftsky_app/projects/providers/project_repository_provider.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_floating_action_button.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:phosphor_icons/phosphor_icons.dart';

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

Future<void> _pumpProjects(
  WidgetTester tester,
  FakeProjectRepository repository,
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
        projectRepositoryProvider.overrideWithValue(repository),
      ],
      child: MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: const ProjectsPage(),
      ),
    ),
  );
}

void main() {
  setUpAll(initializeMappers);

  testWidgets('shows Filters as an extended floating action', (tester) async {
    await _pumpProjects(
      tester,
      FakeProjectRepository(
        onListProjects: ({required query, limit, cursor}) async =>
            const PostPage(items: []),
      ),
    );
    await tester.pumpAndSettle();

    expect(
      find.widgetWithText(CraftskyFloatingActionButton, 'Filters'),
      findsOneWidget,
    );
    expect(find.widgetWithText(OutlinedButton, 'Filters'), findsNothing);
    expect(CraftskyIconsBold.filter, PhosphorIconsBold.funnelSimple);
    expect(find.byIcon(CraftskyIconsBold.filter), findsOneWidget);
  });

  testWidgets('each active project list refreshes, including empty data', (
    tester,
  ) async {
    final calls = <String, int>{};
    await _pumpProjects(
      tester,
      FakeProjectRepository(
        onListProjects: ({required query, limit, cursor}) async {
          final craft = query.craftTypes.single;
          calls.update(craft, (count) => count + 1, ifAbsent: () => 1);
          return PostPage(
            items: craft == ProjectOptionCatalogs.knittingCraftToken
                ? [_post(calls[craft]!)]
                : const [],
          );
        },
      ),
    );
    await tester.pumpAndSettle();

    final knittingCalls = calls[ProjectOptionCatalogs.knittingCraftToken]!;
    await tester.drag(
      find.byType(CustomScrollView).first,
      const Offset(0, 400),
    );
    await tester.pumpAndSettle();
    expect(
      calls[ProjectOptionCatalogs.knittingCraftToken],
      knittingCalls + 1,
    );

    await tester.drag(find.byType(TabBarView), const Offset(-500, 0));
    await tester.pumpAndSettle();
    expect(find.text('No projects found.'), findsOneWidget);
    final crochetCalls = calls[ProjectOptionCatalogs.crochetCraftToken]!;

    await tester.drag(
      find.byType(CustomScrollView).first,
      const Offset(0, 400),
    );
    await tester.pumpAndSettle();
    expect(calls[ProjectOptionCatalogs.crochetCraftToken], crochetCalls + 1);
  });

  testWidgets('project back-to-top follows active nonempty data', (
    tester,
  ) async {
    await _pumpProjects(
      tester,
      FakeProjectRepository(
        onListProjects: ({required query, limit, cursor}) async => PostPage(
          items:
              query.craftTypes.single ==
                  ProjectOptionCatalogs.knittingCraftToken
              ? List.generate(20, _post)
              : const [],
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.byTooltip('Back to top'), findsNothing);

    await tester.drag(
      find.byType(CustomScrollView).first,
      const Offset(0, -300),
    );
    await tester.pumpAndSettle();
    expect(find.byTooltip('Back to top'), findsOneWidget);

    await tester.drag(find.byType(TabBarView), const Offset(-500, 0));
    await tester.pumpAndSettle();
    expect(find.text('No projects found.'), findsOneWidget);
    expect(find.byTooltip('Back to top'), findsNothing);

    await tester.drag(find.byType(TabBarView), const Offset(500, 0));
    await tester.pumpAndSettle();
    expect(find.byTooltip('Back to top'), findsOneWidget);
  });

  testWidgets(
    'project back-to-top counts nested scrolling and resets header and list',
    (tester) async {
      tester.view.physicalSize = const Size(800, 600);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      await _pumpProjects(
        tester,
        FakeProjectRepository(
          onListProjects: ({required query, limit, cursor}) async =>
              PostPage(items: List.generate(20, _post)),
        ),
      );
      await tester.pumpAndSettle();
      final nested = tester.state<NestedScrollViewState>(
        find.byType(NestedScrollView),
      );

      await tester.drag(
        find.byType(CustomScrollView).first,
        const Offset(0, -250),
      );
      await tester.pumpAndSettle();

      expect(
        nested.outerController.offset + nested.innerController.offset,
        greaterThanOrEqualTo(200),
      );
      expect(find.byTooltip('Back to top'), findsOneWidget);

      await tester.tap(find.byTooltip('Back to top'));
      await tester.pumpAndSettle();

      expect(nested.outerController.offset, 0);
      expect(nested.innerController.offset, 0);
      expect(find.byTooltip('Back to top'), findsNothing);
    },
  );

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
