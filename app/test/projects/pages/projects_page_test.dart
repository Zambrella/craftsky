import 'dart:async';

import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/projects/models/project_browse_filters.dart';
import 'package:craftsky_app/projects/options/project_option.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/projects/pages/projects_page.dart';
import 'package:craftsky_app/projects/providers/preferred_project_crafts_provider.dart';
import 'package:craftsky_app/projects/providers/project_feed_provider.dart';
import 'package:craftsky_app/projects/providers/project_repository_provider.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/shared/widgets/craft_icon.dart';
import 'package:craftsky_app/shared/widgets/craftsky_skeleton.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_floating_action_button.dart';
import 'package:craftsky_app/theme/craftsky_form_builder_select_fields.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';
import 'package:phosphor_icons/phosphor_icons.dart';

import '../../fakes/auth_session_fakes.dart';
import '../../fakes/recording_messenger.dart';
import '../../feed/fakes/fake_post_repository.dart';
import '../../test_support/deterministic_pump.dart';
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
  'sponsored': false,
  'createdAt': '2026-05-04T18:23:45.000Z',
  'indexedAt': '2026-05-04T18:23:47.000Z',
  'author': {'did': 'did:plc:alice', 'handle': 'alice.craftsky.social'},
});

Future<void> _pumpProjects(
  WidgetTester tester,
  FakeProjectRepository repository, {
  List<ProjectOption>? craftTypes,
  Future<Profile>? profile,
}) async {
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        authSessionProvider.overrideWith(
          profile == null ? SignedOutAuthSession.new : SignedInAuthSession.new,
        ),
        if (profile != null)
          userProfileProvider(Did.parse('did:plc:test')).overrideWith(
            () => _TestUserProfile(profile),
          ),
        activeLanguagePreferencesProvider.overrideWith(
          (ref) => const LanguagePreferences(
            primaryLanguage: 'en',
            contentLanguages: ['en'],
          ),
        ),
        projectRepositoryProvider.overrideWithValue(repository),
        if (craftTypes != null)
          preferredProjectCraftsProvider.overrideWithValue(craftTypes),
      ],
      child: MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: appLocalizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: const ProjectsPage(),
      ),
    ),
  );
}

void main() {
  setUpAll(initializeMappers);

  testWidgets('preferred tabs drive the project query and filter craft', (
    tester,
  ) async {
    final gate = Completer<Profile>();
    final requestedCrafts = <String>[];
    final repository = FakeProjectRepository(
      onListProjects: ({required query, limit, cursor}) async {
        requestedCrafts.add(query.craftTypes.single);
        return const PostPage(items: []);
      },
    );
    await _pumpProjects(tester, repository, profile: gate.future);
    await tester.pumpAndSettle();
    gate.complete(
      Profile(
        did: 'did:plc:test',
        handle: 'test.bsky.social',
        crafts: ['quilting', 'sewing'],
      ),
    );
    await tester.pumpAndSettle();
    final tabs = tester.widget<TabBar>(find.byType(TabBar));
    expect(
      tabs.tabs.map((tab) => ((tab as Tab).child! as CraftIconLabel).label),
      ['Quilting', 'Sewing', 'Knitting', 'Crochet', 'Embroidery'],
    );
    expect(tabs.controller!.index, 0);
    expect(requestedCrafts, contains(ProjectOptionCatalogs.quiltingCraftToken));
    await tester.tap(find.text('Sewing'));
    await tester.pumpAndSettle();
    expect(tabs.controller!.index, 1);
    final container = ProviderScope.containerOf(
      tester.element(find.byType(ProjectsPage)),
    );
    container
        .read(userProfileProvider(Did.parse('did:plc:test')).notifier)
        .setCached(
          Profile(
            did: 'did:plc:test',
            handle: 'test.bsky.social',
            crafts: ['quilting'],
          ),
        );
    await tester.pumpAndSettle();
    expect(
      tabs.controller!.index,
      ProjectOptionCatalogs.craftTypesForPreferences(['quilting']).indexWhere(
        (craft) => craft.value == ProjectOptionCatalogs.sewingCraftToken,
      ),
    );
    expect(requestedCrafts, contains(ProjectOptionCatalogs.sewingCraftToken));
    await tester.tap(find.text('Quilting'));
    await tester.pumpAndSettle();
    await tester.tap(
      find.widgetWithText(CraftskyFloatingActionButton, 'Filters'),
    );
    await tester.pumpAndSettle();
    expect(find.text('Filter Quilting projects'), findsOneWidget);
  });

  testWidgets('shows three project post skeletons while initially loading', (
    tester,
  ) async {
    final gate = Completer<PostPage>();
    await _pumpProjects(
      tester,
      FakeProjectRepository(
        onListProjects: ({required query, limit, cursor}) => gate.future,
      ),
    );

    expect(
      tester
          .widget<CraftskySkeletonSliverList>(
            find.byType(CraftskySkeletonSliverList),
          )
          .itemCount,
      3,
    );
    expect(
      tester
          .widget<PostCardSkeleton>(find.byType(PostCardSkeleton).first)
          .showMedia,
      isTrue,
    );

    gate.complete(const PostPage(items: []));
    await pumpUntilAbsent(
      tester,
      find.byType(CraftskySkeletonSliverList),
      description: 'the project loading skeleton to disappear',
    );
  });

  testWidgets('shows Filters as an extended floating action', (tester) async {
    await _pumpProjects(
      tester,
      FakeProjectRepository(
        onListProjects: ({required query, limit, cursor}) async =>
            const PostPage(items: []),
      ),
    );
    await pumpUntilFound(
      tester,
      find.widgetWithText(CraftskyFloatingActionButton, 'Filters'),
      description: 'the project filters action',
    );

    expect(
      find.widgetWithText(CraftskyFloatingActionButton, 'Filters'),
      findsOneWidget,
    );
    expect(find.widgetWithText(OutlinedButton, 'Filters'), findsNothing);
    expect(CraftskyIconsBold.filter, PhosphorIconsBold.funnelSimple);
    expect(find.byIcon(CraftskyIconsBold.filter), findsOneWidget);
  });

  testWidgets('project posts offer owner delete and visitor report actions', (
    tester,
  ) async {
    final deleted = <String>[];
    final own = _post(0);
    final other = _post(1).copyWith(
      author: _post(1).author.copyWith(
        did: 'did:plc:bob',
        handle: 'bob.craftsky.social',
      ),
    );
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authSessionProvider.overrideWith(
            () => SignedInAuthSession(did: 'did:plc:alice'),
          ),
          activeLanguagePreferencesProvider.overrideWith(
            (ref) => const LanguagePreferences(
              primaryLanguage: 'en',
              contentLanguages: ['en'],
            ),
          ),
          projectRepositoryProvider.overrideWithValue(
            FakeProjectRepository(
              onListProjects: ({required query, limit, cursor}) async =>
                  PostPage(items: [own, other]),
            ),
          ),
          postRepositoryProvider.overrideWithValue(
            FakePostRepository(
              onDelete: (did, rkey) async => deleted.add('$did/$rkey'),
            ),
          ),
        ],
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProjectsPage(),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.byType(PostCard), findsNWidgets(2));

    await tester.tap(find.byTooltip('More actions').last);
    await tester.pumpAndSettle();
    expect(find.text('Report post'), findsOneWidget);
    expect(find.text('Delete post'), findsNothing);
    await tester.tapAt(const Offset(10, 10));
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('More actions').first);
    await tester.pumpAndSettle();
    expect(find.text('Delete post'), findsOneWidget);
    expect(find.text('Report post'), findsNothing);
    await tester.tap(find.text('Delete post'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Delete'));
    await tester.pumpAndSettle();
    expect(deleted, ['did:plc:alice/project-0']);
    expect(find.text('Project 0'), findsNothing);
    expect(find.text('Project 1'), findsOneWidget);
  });

  testWidgets('Embroidery filters omit project type and subtype fields', (
    tester,
  ) async {
    await _pumpProjects(
      tester,
      FakeProjectRepository(
        onListProjects: ({required query, limit, cursor}) async =>
            const PostPage(items: []),
      ),
      craftTypes: [
        ProjectOptionCatalogs.craftTypes.singleWhere(
          (craft) => craft.value == ProjectOptionCatalogs.embroideryCraftToken,
        ),
      ],
    );
    await pumpUntilFound(
      tester,
      find.text('Filters'),
      description: 'the project filters action',
    );

    await tester.tap(find.text('Filters'));
    await tester.pumpAndSettle();

    expect(find.text('Filter Embroidery projects'), findsOneWidget);
    expect(find.text('Project type'), findsNothing);
    expect(find.text('Project subtype'), findsNothing);
    expect(find.text('Pattern difficulty'), findsOneWidget);
    expect(find.text('Color'), findsOneWidget);
  });

  testWidgets('filter sheet orders and enables dependent controls', (
    tester,
  ) async {
    await _pumpProjects(
      tester,
      FakeProjectRepository(
        onListProjects: ({required query, limit, cursor}) async =>
            const PostPage(items: []),
      ),
    );
    await pumpUntilFound(
      tester,
      find.text('Filters'),
      description: 'the project filters action',
    );

    await tester.tap(find.text('Filters'));
    await tester.pumpAndSettle();

    CraftskySearchableMultiSelectInput<String> subtypeInput() => tester.widget(
      find.byWidgetPredicate(
        (widget) =>
            widget is CraftskySearchableMultiSelectInput<String> &&
            widget.label == 'Project subtype',
      ),
    );
    CraftskySearchableMultiSelectInput<String> inputWithLabel(String label) =>
        tester.widget(
          find.byWidgetPredicate(
            (widget) =>
                widget is CraftskySearchableMultiSelectInput<String> &&
                widget.label == label,
          ),
        );
    void expectAlphabetical(String label) {
      final labels = [
        for (final option in inputWithLabel(label).options) option.label,
      ];
      expect(labels, [...labels]..sort());
    }

    expect(find.text('Project subtype'), findsOneWidget);
    expect(subtypeInput().enabled, isFalse);
    expectAlphabetical('Project type');

    await tester.tap(find.byKey(const Key('Project type-select-button')));
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(CheckboxListTile, 'Garment'));
    await tester.pumpAndSettle();

    expect(subtypeInput().enabled, isTrue);
    expectAlphabetical('Project subtype');

    FocusManager.instance.primaryFocus?.unfocus();
    await tester.pumpAndSettle();
    await tester.drag(find.byType(ListView).last, const Offset(0, -1000));
    await tester.pumpAndSettle();

    expect(find.text('Design'), findsOneWidget);
    expect(find.text('Design tag'), findsNothing);
    expectAlphabetical('Color');
    expectAlphabetical('Design');
    expect(
      tester.getTopLeft(find.text('Status')).dy,
      lessThan(tester.getTopLeft(find.text('Self drafted')).dy),
    );
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
    expect(calls[ProjectOptionCatalogs.knittingCraftToken], knittingCalls + 1);

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

  testWidgets('craft tabs retain independent offsets and fixed headers', (
    tester,
  ) async {
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

    ScrollPosition position(String craft) => tester
        .state<ScrollableState>(
          find.descendant(
            of: find.byKey(PageStorageKey<String>('projects_tab_$craft')),
            matching: find.byWidgetPredicate(
              (widget) =>
                  widget is Scrollable &&
                  widget.axisDirection == AxisDirection.down,
            ),
          ),
        )
        .position;
    const knitting = ProjectOptionCatalogs.knittingCraftToken;
    const crochet = ProjectOptionCatalogs.crochetCraftToken;
    final headerPosition = tester.getTopLeft(find.byType(TabBar));
    expect(position(knitting).pixels, 0);
    await tester.drag(
      find.byType(CustomScrollView).first,
      const Offset(0, -600),
    );
    await tester.pumpAndSettle();
    final knittingOffset = position(knitting).pixels;
    expect(knittingOffset, greaterThan(200));
    expect(tester.getTopLeft(find.byType(TabBar)), headerPosition);

    await tester.drag(find.byType(TabBarView), const Offset(-800, 0));
    await tester.pumpAndSettle();
    expect(position(crochet).pixels, 0);
    expect(
      tester.getTopLeft(find.byType(PostCard).first).dy,
      greaterThanOrEqualTo(tester.getBottomLeft(find.byType(TabBar)).dy),
    );
    await tester.drag(
      find.byType(CustomScrollView).first,
      const Offset(0, -350),
    );
    await tester.pumpAndSettle();
    final crochetOffset = position(crochet).pixels;
    expect(crochetOffset, greaterThan(200));

    // Tab taps and swipes must both restore the previous craft position.
    await tester.tap(find.text('Knitting'));
    await tester.pumpAndSettle();
    expect(position(knitting).pixels, closeTo(knittingOffset, 0.01));
    await tester.tap(find.byTooltip('Back to top'));
    await tester.pumpAndSettle();
    expect(position(knitting).pixels, 0);
    expect(find.byTooltip('Back to top'), findsNothing);
    expect(tester.getTopLeft(find.byType(TabBar)), headerPosition);

    await tester.tap(find.text('Crochet'));
    await tester.pumpAndSettle();
    expect(position(crochet).pixels, closeTo(crochetOffset, 0.01));
    expect(find.byTooltip('Back to top'), findsOneWidget);

    // Visit a distant tab and verify earlier crafts still retain their offsets.
    final tabs = tester.widget<TabBar>(find.byType(TabBar));
    tabs.controller!.animateTo(tabs.tabs.length - 1);
    await tester.pumpAndSettle();
    await tester.tap(find.text('Knitting'));
    await tester.pumpAndSettle();
    expect(position(knitting).pixels, 0);
    await tester.tap(find.text('Crochet'));
    await tester.pumpAndSettle();
    expect(position(crochet).pixels, closeTo(crochetOffset, 0.01));
  });

  testWidgets('returning to a craft keeps its loaded feed and deep offset', (
    tester,
  ) async {
    final requests = <String, int>{};
    final reload = Completer<PostPage>();
    addTearDown(() {
      if (!reload.isCompleted) reload.complete(const PostPage(items: []));
    });
    await _pumpProjects(
      tester,
      FakeProjectRepository(
        onListProjects: ({required query, limit, cursor}) {
          final craft = query.craftTypes.single;
          final count = requests.update(craft, (n) => n + 1, ifAbsent: () => 1);
          // A repeat request stays pending to expose loading-state scroll loss.
          return count == 1
              ? Future.value(PostPage(items: List.generate(30, _post)))
              : reload.future;
        },
      ),
    );
    await tester.pumpAndSettle();
    const knitting = ProjectOptionCatalogs.knittingCraftToken;
    const listKey = PageStorageKey<String>('projects_tab_$knitting');
    ScrollPosition position() => tester
        .state<ScrollableState>(
          find.descendant(
            of: find.byKey(listKey),
            matching: find.byWidgetPredicate(
              (widget) =>
                  widget is Scrollable &&
                  widget.axisDirection == AxisDirection.down,
            ),
          ),
        )
        .position;
    await tester.drag(find.byKey(listKey), const Offset(0, -1800));
    await tester.pumpAndSettle();
    final offset = position().pixels;
    expect(offset, greaterThan(1000));

    await tester.drag(find.byType(TabBarView), const Offset(-800, 0));
    await tester.pumpAndSettle();
    await tester.drag(find.byType(TabBarView), const Offset(800, 0));
    await tester.pump(const Duration(milliseconds: 350));
    await tester.pump();
    expect(requests[knitting], 1);
    expect(position().pixels, closeTo(offset, 0.01));
    expect(find.byType(CraftskySkeletonSliverList), findsNothing);

    final tabs = tester.widget<TabBar>(find.byType(TabBar));
    tabs.controller!.animateTo(tabs.tabs.length - 1);
    await tester.pumpAndSettle();
    tabs.controller!.animateTo(0);
    await tester.pumpAndSettle();
    expect(requests[knitting], 1);
    expect(position().pixels, closeTo(offset, 0.01));

    // Retention ends when the Projects page is removed.
    final context = tester.element(find.byType(ProjectsPage));
    final container = ProviderScope.containerOf(context);
    final feed = projectFeedProvider(
      const ProjectBrowseQuery(craftTypes: [knitting]),
    );
    expect(container.exists(feed), isTrue);
    Navigator.of(context).pushReplacement<void, void>(
      MaterialPageRoute<void>(builder: (_) => const SizedBox()),
    );
    await tester.pumpAndSettle();
    await tester.pump();
    expect(container.exists(feed), isFalse);
  });

  testWidgets('back-to-top jumps immediately with reduced motion', (
    tester,
  ) async {
    tester.platformDispatcher.accessibilityFeaturesTestValue =
        const FakeAccessibilityFeatures(disableAnimations: true);
    addTearDown(tester.platformDispatcher.clearAccessibilityFeaturesTestValue);
    await _pumpProjects(
      tester,
      FakeProjectRepository(
        onListProjects: ({required query, limit, cursor}) async =>
            PostPage(items: List.generate(20, _post)),
      ),
    );
    await tester.pumpAndSettle();
    await tester.drag(
      find.byType(CustomScrollView).first,
      const Offset(0, -500),
    );
    await tester.pumpAndSettle();
    final scrollable = tester.state<ScrollableState>(
      find.byWidgetPredicate(
        (widget) =>
            widget is Scrollable && widget.axisDirection == AxisDirection.down,
      ),
    );
    expect(scrollable.position.pixels, greaterThan(200));
    await tester.tap(find.byTooltip('Back to top'));
    expect(scrollable.position.pixels, 0);
    await tester.pumpAndSettle();
    expect(find.byTooltip('Back to top'), findsNothing);
  });

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
          localizationsDelegates: appLocalizationsDelegates,
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
    await tester.drag(find.byType(ListView).last, const Offset(0, -1000));
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(CheckboxListTile, 'Self drafted'));
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
    await tester.drag(find.byType(ListView).last, const Offset(0, -1000));
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(CheckboxListTile, 'Self drafted'));
    await tester.pump();
    tester
        .widget<FilledButton>(
          find.widgetWithText(FilledButton, 'Apply filters'),
        )
        .onPressed!();
    await tester.pumpAndSettle();

    expect(find.text('Project 0'), findsOneWidget);
  });

  testWidgets('each craft retains its own filters and clears independently', (
    tester,
  ) async {
    final queries = <ProjectBrowseQuery>[];
    await _pumpProjects(
      tester,
      FakeProjectRepository(
        onListProjects: ({required query, limit, cursor}) async {
          queries.add(query);
          return PostPage(items: List.generate(20, _post));
        },
      ),
    );
    await tester.pumpAndSettle();

    Future<void> openFilters({required bool selected}) async {
      await tester.tap(find.text('Filters'));
      await tester.pumpAndSettle();
      await tester.drag(find.byType(ListView).last, const Offset(0, -1000));
      await tester.pumpAndSettle();
      final checkbox = find.widgetWithText(CheckboxListTile, 'Self drafted');
      await tester.ensureVisible(checkbox);
      await tester.pumpAndSettle();
      expect(tester.widget<CheckboxListTile>(checkbox).value, selected);
    }

    Future<void> applyFilters() async {
      await tester.tap(find.widgetWithText(FilledButton, 'Apply filters'));
      await tester.pumpAndSettle();
    }

    Future<void> toggleSelfDrafted() async {
      await tester.tap(find.widgetWithText(CheckboxListTile, 'Self drafted'));
      await tester.pump();
      await applyFilters();
    }

    Future<void> switchTo(String label) async {
      await tester.tap(find.text(label));
      await tester.pumpAndSettle();
    }

    await openFilters(selected: false);
    await toggleSelfDrafted();
    expect(find.widgetWithText(InputChip, 'Self drafted'), findsOneWidget);
    await switchTo('Crochet');
    await openFilters(selected: false);
    await toggleSelfDrafted();
    final knittingRequests = queries
        .where(
          (query) =>
              query.craftTypes.single ==
              ProjectOptionCatalogs.knittingCraftToken,
        )
        .length;

    await switchTo('Knitting');
    await openFilters(selected: true);
    await applyFilters();
    expect(
      queries
          .where(
            (query) =>
                query.craftTypes.single ==
                ProjectOptionCatalogs.knittingCraftToken,
          )
          .length,
      knittingRequests,
    );

    // Removing a chip only changes the current craft.
    tester
        .widget<InputChip>(find.widgetWithText(InputChip, 'Self drafted'))
        .onDeleted!();
    await tester.pumpAndSettle();
    expect(find.byType(InputChip), findsNothing);
    await switchTo('Crochet');
    await openFilters(selected: true);
    await applyFilters();
    await switchTo('Knitting');
    await openFilters(selected: false);
    await toggleSelfDrafted();

    // Clear-all is scoped to the current craft too.
    await switchTo('Crochet');
    await tester.tap(find.widgetWithText(TextButton, 'Clear all'));
    await tester.pumpAndSettle();
    expect(find.byType(InputChip), findsNothing);
    await switchTo('Knitting');
    await openFilters(selected: true);
    await applyFilters();
    await switchTo('Crochet');
    await openFilters(selected: false);
    await applyFilters();
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
          localizationsDelegates: appLocalizationsDelegates,
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

class _TestUserProfile extends UserProfile {
  new(this.profile);
  final Future<Profile> profile;
  @override
  Future<Profile> build(Did did) => profile;
}
