import 'dart:async';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart' as model;
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_account_page.dart';
import 'package:craftsky_app/profile/models/profile_account_summary.dart';
import 'package:craftsky_app/profile/models/profile_relationship.dart';
import 'package:craftsky_app/profile/providers/profile_relationship_provider.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/providers/recent_members_provider.dart';
import 'package:craftsky_app/profile/providers/toggle_block_profile_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/profile/widgets/profile_account_list_tile.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/profile/widgets/profile_presentation_page.dart';
import 'package:craftsky_app/projects/pages/projects_page.dart';
import 'package:craftsky_app/projects/providers/preferred_craft_links_provider.dart';
import 'package:craftsky_app/projects/providers/project_repository_provider.dart';
import 'package:craftsky_app/router/app_shell.dart';
import 'package:craftsky_app/router/route_locations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/settings/providers/relationship_list_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:craftsky_app/shared/widgets/craft_icon.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_access_provider.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';

import '../fakes/auth_session_fakes.dart';
import '../fakes/recording_messenger.dart';
import '../profile/fakes/fake_profile_repository.dart';
import '../projects/fakes/fake_project_repository.dart';

class _Registry extends SessionRegistry {
  void switchTo(String did) => state = AsyncData(
    state.requireValue.upsertAndActivate(
      token: 'test-token',
      did: did,
      handle: 'other.test',
    ),
  );
  @override
  Future<model.SessionRegistry> build() async =>
      model.SessionRegistry.empty().upsertAndActivate(
        token: 'test-token',
        did: 'did:plc:test',
        handle: 'test.test',
      );
}

class _Profile extends UserProfile {
  new([this.profile, this.readProfile]);
  final Future<Profile>? profile;
  final Future<Profile> Function()? readProfile;
  @override
  Future<Profile> build(Did did) =>
      readProfile?.call() ??
      profile ??
      Future.value(
        Profile(
          did: did.value,
          handle: 'test.test',
          crafts: ['knitting', 'sewing'],
        ),
      );
}

class _Zero implements NotificationNewnessRepository {
  @override
  Future<int> count() async => 0;
  @override
  Future<void> markSeen() async {}
}

Future<GoRouter> _pumpSidebar(
  WidgetTester tester, {
  String location = '/projects',
  List<String>? queries,
  Future<Profile>? profile,
  FakeProfileRepository? repository,
  Future<List<ProfileAccountSummary>>? members,
  bool realProfile = false,
  bool settle = true,
  Future<Profile> Function()? readProfile,
  Future<List<ProfileAccountSummary>> Function()? readMembers,
  Future<List<ProfileAccountSummary>> Function(ActiveAccountLease)?
  readMembersFor,
  Size size = const Size(1440, 1000),
  double textScale = 1,
  ThemeMode themeMode = ThemeMode.light,
}) async {
  tester.view.devicePixelRatio = 1;
  tester.view.physicalSize = size;
  addTearDown(tester.view.resetDevicePixelRatio);
  addTearDown(tester.view.resetPhysicalSize);
  final router = GoRouter(
    initialLocation: location,
    routes: [
      ShellRoute(
        builder: (context, state, child) => AuthenticatedShell(child: child),
        routes: [
          GoRoute(
            path: '/profiles/:did',
            pageBuilder: (context, state) => realProfile
                ? UserProfileRoute(
                    did: Did.parse(state.pathParameters['did']!),
                    $extra: state.extra as ProfilePresentationRequest?,
                  ).buildPage(context, state)
                : MaterialPage(
                    child: Scaffold(
                      body: Text('Profile ${state.pathParameters['did']}'),
                    ),
                  ),
          ),
          for (final path in [
            RouteLocations.search,
            RouteLocations.notifications,
            RouteLocations.profile,
            RouteLocations.savedPosts,
            RouteLocations.scheduledPosts,
            RouteLocations.drafts,
            RouteLocations.settings,
          ])
            GoRoute(
              path: path,
              builder: (context, state) =>
                  Scaffold(body: Text('$path content')),
            ),
          GoRoute(
            path: '/feed',
            builder: (context, state) => Scaffold(
              body: TextButton(
                onPressed: () {},
                child: const Text('Feed content'),
              ),
            ),
          ),
          GoRoute(
            path: '/projects',
            builder: (context, state) =>
                ProjectsPage(craft: state.uri.queryParameters['craft']),
          ),
        ],
      ),
    ],
  );
  addTearDown(router.dispose);
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        sessionRegistryProvider.overrideWith(_Registry.new),
        accountProfileRepositoryProvider.overrideWith(
          (ref, lease) async => FakeProfileRepository(
            onFetchRecentMembers: () =>
                readMembersFor?.call(lease) ??
                readMembers?.call() ??
                members ??
                Future.value([]),
          ),
        ),
        authSessionProvider.overrideWith(SignedInAuthSession.new),
        if (repository != null)
          profileRepositoryProvider.overrideWithValue(repository),
        if (repository != null)
          accountRelationshipRepositoryProvider.overrideWith(
            (ref, account) async => repository,
          ),
        relationshipReconciliationSchedulerProvider.overrideWithValue(
          (delay, callback) => () {},
        ),
        pdsRecordOperationControllerProvider.overrideWithValue(
          PdsRecordOperationController(schedule: (_, _) {}),
        ),
        subscriptionAccessProvider.overrideWith(
          (ref, lease) async => SubscriptionAccess(
            did: lease.account.did,
            effectiveTier: SubscriptionTier.free,
            givesAccess: false,
          ),
        ),
        if (repository == null)
          userProfileProvider(Did.parse('did:plc:test'))
              .overrideWith(() => _Profile(profile, readProfile)),
        accountNotificationNewnessRepositoryProvider.overrideWith(
          (ref, account) async => _Zero(),
        ),
        notificationNewnessRepositoryProvider.overrideWithValue(_Zero()),
        activeLanguagePreferencesProvider.overrideWith(
          (ref) => const LanguagePreferences(
            primaryLanguage: 'en',
            contentLanguages: ['en'],
          ),
        ),
        projectRepositoryProvider.overrideWithValue(
          FakeProjectRepository(
            onListProjects: ({required query, limit, cursor}) async {
              queries?.add(query.craftTypes.single);
              return const PostPage(items: []);
            },
          ),
        ),
      ],
      child: MaterialApp.router(
        routerConfig: router,
        theme: AppTheme.lightThemeData,
        darkTheme: AppTheme.darkThemeData,
        themeMode: themeMode,
        localizationsDelegates: appLocalizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        builder: (context, child) => MessengerScope(
          messenger: RecordingMessenger(),
          child: MediaQuery(
            data: MediaQuery.of(context)
                .copyWith(textScaler: TextScaler.linear(textScale)),
            child: FormFactorWidget(child: child!),
          ),
        ),
      ),
    ),
  );
  if (settle) {
    await tester.pumpAndSettle();
  } else {
    await _pumpFrames(tester);
  }
  return router;
}

Future<void> _pumpFrames(WidgetTester tester) async {
  for (var i = 0; i < 8; i++) {
    await tester.pump(const Duration(milliseconds: 50));
  }
}

String _selectedCraft(WidgetTester tester) {
  final tabs = tester.widget<TabBar>(find.byType(TabBar));
  return ((tabs.tabs[tabs.controller!.index] as Tab).child! as CraftIconLabel)
      .craft;
}

void main() {
  setUpAll(initializeMappers);
  final destinations = [
    RouteLocations.feed,
    RouteLocations.projects,
    RouteLocations.search,
    RouteLocations.notifications,
    RouteLocations.profile,
    RouteLocations.savedPosts,
    RouteLocations.scheduledPosts,
    RouteLocations.drafts,
    RouteLocations.settings,
  ];
  for (final width in [600.0, 900.0, 1200.0, 1201.0, 1440.0]) {
    testWidgets('AT-001 shared shell destinations at width $width', (
      tester,
    ) async {
      var reads = 0;
      final router = await _pumpSidebar(
        tester,
        location: '/feed',
        size: Size(width, 900),
        readMembers: () async {
          reads++;
          return [];
        },
      );
      for (final destination in destinations) {
        router.go(destination);
        await tester.pumpAndSettle();
        final sidebar = find.byKey(const Key('desktop-discovery-sidebar'));
        expect(
          sidebar,
          width > 1200 ? findsOneWidget : findsNothing,
          reason: destination,
        );
        expect(
          find.byKey(const Key('desktop-sidebar-placeholder')),
          findsNothing,
        );
        if (width > 1200) {
          final l10n = AppLocalizations.of(tester.element(sidebar));
          expect(
            tester.getTopLeft(find.text(l10n.discoveryYourCrafts)).dy,
            lessThan(tester.getTopLeft(find.text(l10n.discoveryNewMembers)).dy),
          );
          expect(find.byType(NavigationRail), findsOneWidget);
        }
        expect(router.state.matchedLocation, destination);
        expect(tester.takeException(), isNull);
      }
      expect(reads, width > 1200 ? greaterThan(0) : 0);
    });
  }
  for (final size in [const Size(1201, 500), const Size(1440, 900)]) {
    for (final scale in [1.0, 2.0]) {
      testWidgets('AT-008 scroll keyboard semantics $size text $scale', (
        tester,
      ) async {
        final semantics = tester.ensureSemantics();
        try {
          final members = List.generate(
            5,
            (i) => ProfileAccountSummary(
              did: 'did:plc:member$i',
              handle: 'averylongpublichandlefortextscaling$i.example.test',
              isCraftskyProfile: true,
              displayName: 'A long member display name for enlarged text $i',
            ),
          );
          final router = await _pumpSidebar(
            tester,
            location: '/feed',
            size: size,
            textScale: scale,
            profile: Future.value(
              Profile(
                did: 'did:plc:test',
                handle: 'test.test',
                crafts: [
                  'knitting',
                  'crochet',
                  'sewing',
                  'embroidery',
                  'quilting',
                ],
              ),
            ),
            members: Future.value(members),
          );
          final firstCraft = find.byKey(const Key('discovery-craft-knitting'));
          expect(
            tester.getSemantics(firstCraft).flagsCollection.isButton,
            isTrue,
          );
          expect(
            tester.getSemantics(firstCraft).getSemanticsData().label,
            contains('Knitting'),
          );
          final last = find.byKey(
            const Key('discovery-member-did:plc:member4'),
          );
          await tester.ensureVisible(last);
          await tester.pumpAndSettle();
          expect(last.hitTestable(), findsOneWidget);
          expect(
            tester.getSemantics(last).getSemanticsData().label,
            contains('@averylongpublichandlefortextscaling4.example.test'),
          );
          await tester.ensureVisible(firstCraft);
          await tester.pumpAndSettle();
          final reached = <String>{};
          final focusLabels = <String>[];
          for (var i = 0; i < 40 && reached.length < 10; i++) {
            await tester.sendKeyEvent(LogicalKeyboardKey.tab);
            await tester.pumpAndSettle();
            final focusContext = FocusManager.instance.primaryFocus?.context;
            focusLabels.add(
              '${FocusManager.instance.primaryFocus?.debugLabel} '
              'context=${focusContext?.widget.runtimeType}',
            );
            final craft = focusContext
                ?.findAncestorWidgetOfExactType<TextButton>();
            final tile = focusContext
                ?.findAncestorWidgetOfExactType<ProfileAccountListTile>();
            final key = tile?.key ?? craft?.key;
            if (key is! ValueKey<String> ||
                !key.value.startsWith('discovery-')) {
              continue;
            }
            reached.add(key.value);
            expect(find.byKey(key).hitTestable(), findsOneWidget);
          }
          expect(reached, hasLength(10), reason: focusLabels.toString());
          for (final action in reached) {
            final target = find.byKey(ValueKey(action));
            await tester.ensureVisible(target);
            await tester.pumpAndSettle();
            final text = find
                .descendant(of: target, matching: find.byType(Text))
                .first;
            final node = Focus.maybeOf(
              tester.element(text),
              createDependency: false,
            )!;
            expect(node.canRequestFocus, isTrue);
            node.requestFocus();
            await tester.pump();
            await tester.sendKeyEvent(LogicalKeyboardKey.enter);
            await tester.pumpAndSettle();
            if (action.startsWith('discovery-member-')) {
              final did = action.substring('discovery-member-'.length);
              expect(find.text('Profile $did'), findsOneWidget);
              router.pop();
              await tester.pumpAndSettle();
            } else {
              expect(
                _selectedCraft(tester),
                'social.craftsky.feed.defs#'
                '${action.substring('discovery-craft-'.length)}',
              );
            }
          }
          expect(tester.takeException(), isNull);
        } finally {
          semantics.dispose();
        }
      });
    }
  }
  for (final theme in [ThemeMode.light, ThemeMode.dark]) {
    testWidgets('AT-008 constrained loading and failures $theme', (
      tester,
    ) async {
      final profile = Completer<Profile>();
      final members = Completer<List<ProfileAccountSummary>>();
      addTearDown(() {
        if (!profile.isCompleted) {
          profile.complete(
            Profile(did: 'did:plc:test', handle: 'test.test', crafts: []),
          );
        }
        if (!members.isCompleted) members.complete([]);
      });
      await _pumpSidebar(
        tester,
        location: '/feed',
        size: const Size(1201, 500),
        textScale: 2,
        themeMode: theme,
        settle: false,
        profile: profile.future,
        members: members.future,
      );
      expect(find.byType(StitchProgressIndicator), findsNWidgets(2));
      final section = find.byKey(const Key('discovery-members-section'));
      await tester.ensureVisible(section);
      await _pumpFrames(tester);
      expect(tester.takeException(), isNull);
      profile.completeError(const ApiServerError('profile fixture error'));
      members.completeError(const ApiServerError('member fixture error'));
      await _pumpFrames(tester);
      expect(find.text('Retry'), findsNWidgets(2));
      for (var i = 0; i < 2; i++) {
        final retry = find.text('Retry').at(i);
        await tester.ensureVisible(retry);
        await _pumpFrames(tester);
        expect(retry.hitTestable(), findsOneWidget);
        final node = Focus.maybeOf(
          tester.element(retry),
          createDependency: false,
        )!;
        expect(node.canRequestFocus, isTrue);
      }
      expect(tester.takeException(), isNull);
    });
  }
  testWidgets('AT-007 settings unmute refreshes current eligible members', (
    tester,
  ) async {
    var hidden = true;
    var reads = 0;
    final member = ProfileAccountSummary(
      did: 'did:plc:member',
      handle: 'member.test',
      isCraftskyProfile: true,
      muted: true,
    );
    final repo = FakeProfileRepository(
      onFetchMe: () async => Profile(
        did: 'did:plc:test',
        handle: 'test.test',
        crafts: ['knitting'],
      ),
      onListMutedProfiles: ({limit, cursor}) async =>
          ProfileAccountPage(items: [member], totalCount: 1),
      onUnmute: (_) async {
        hidden = false;
        return const ProfileRelationship();
      },
    );
    await _pumpSidebar(
      tester,
      location: '/feed',
      repository: repo,
      readMembers: () async {
        reads++;
        return hidden ? [] : [member.copyWith(muted: false)];
      },
    );
    final container = ProviderScope.containerOf(
      tester.element(find.byType(AuthenticatedShell)),
    );
    final list = relationshipListProvider(RelationshipListKind.muted);
    final subscription = container.listen(list, (_, _) {});
    addTearDown(subscription.close);
    await container.read(list.future);
    await container
        .read(relationshipListProvider(RelationshipListKind.muted).notifier)
        .reverse(member);
    await _pumpFrames(tester);
    expect(find.byType(ProfileAccountListTile), findsOneWidget);
    expect(reads, greaterThan(1));
  });
  testWidgets('AT-007 accepted block hides stale rows until projected refill', (
    tester,
  ) async {
    final candidates = [
      for (var i = 0; i < 6; i++)
        ProfileAccountSummary(
          did: 'did:plc:member$i',
          handle: 'member$i.test',
          isCraftskyProfile: true,
        ),
    ];
    var projected = false;
    var reads = 0;
    final repo = FakeProfileRepository(
      onFetchMe: () async => Profile(
        did: 'did:plc:test',
        handle: 'test.test',
        crafts: ['knitting'],
      ),
      onBlock: (_, _) async => const ProfileRelationship(blocking: true),
    );
    await _pumpSidebar(
      tester,
      location: '/feed',
      repository: repo,
      readMembers: () async {
        reads++;
        return projected ? candidates.sublist(1) : candidates.sublist(0, 5);
      },
    );
    expect(find.byType(ProfileAccountListTile), findsNWidgets(5));
    final container = ProviderScope.containerOf(
      tester.element(find.byType(AuthenticatedShell)),
    );
    final lease = container
        .read(sessionRegistryProvider)
        .requireValue
        .activeLease!;
    await container
        .read(toggleBlockProfileProvider.notifier)
        .toggle(
          targetDid: candidates.first.did,
          isBlocking: false,
        );
    await _pumpFrames(tester);
    expect(
      find.byKey(const Key('discovery-member-did:plc:member0')),
      findsNothing,
    );
    expect(find.byType(ProfileAccountListTile), findsNWidgets(4));
    for (var i = 0; i < 2; i++) {
      container.invalidate(recentMembersProvider(lease));
      await _pumpFrames(tester);
      expect(
        find.byKey(const Key('discovery-member-did:plc:member0')),
        findsNothing,
      );
      expect(find.byType(ProfileAccountListTile), findsNWidgets(4));
    }
    projected = true;
    container.invalidate(recentMembersProvider(lease));
    await _pumpFrames(tester);
    expect(find.byType(ProfileAccountListTile), findsNWidgets(5));
    expect(
      find.byKey(const Key('discovery-member-did:plc:member0')),
      findsNothing,
    );
    expect(
      find.byKey(const Key('discovery-member-did:plc:member5')),
      findsOneWidget,
    );
    expect(reads, greaterThanOrEqualTo(5));
    // Absence from a bounded list is not proof that the overlay reconciled.
    projected = false;
    container.invalidate(recentMembersProvider(lease));
    await _pumpFrames(tester);
    expect(
      find.byKey(const Key('discovery-member-did:plc:member0')),
      findsNothing,
    );
    expect(tester.takeException(), isNull);
  });

  testWidgets(
    'AT-007 block suppression follows accepted state and account session',
    (
      tester,
    ) async {
      var fail = true;
      final member = ProfileAccountSummary(
        did: 'did:plc:member',
        handle: 'member.test',
        isCraftskyProfile: true,
      );
      final repo = FakeProfileRepository(
        onFetchMe: () async => Profile(
          did: 'did:plc:test',
          handle: 'test.test',
          crafts: ['knitting'],
        ),
        onBlock: (_, _) async {
          if (fail) throw StateError('block failed');
          return const ProfileRelationship(blocking: true);
        },
        onUnblock: (_, _) async {
          if (fail) throw StateError('unblock failed');
        },
      );
      await _pumpSidebar(
        tester,
        location: '/feed',
        repository: repo,
        readMembers: () async => [member],
      );
      final container = ProviderScope.containerOf(
        tester.element(find.byType(AuthenticatedShell)),
      );
      Future<void> toggle({required bool blocking}) async {
        await container
            .read(toggleBlockProfileProvider.notifier)
            .toggle(
              targetDid: member.did,
              isBlocking: blocking,
            );
        final lease = container
            .read(sessionRegistryProvider)
            .requireValue
            .activeLease!;
        container.invalidate(recentMembersProvider(lease));
        await _pumpFrames(tester);
      }

      // Failed block must not hide a member even on a subsequent read.
      await toggle(blocking: false);
      expect(find.byType(ProfileAccountListTile), findsOneWidget);
      fail = false;
      await toggle(blocking: false);
      expect(find.byType(ProfileAccountListTile), findsNothing);
      // Failed unblock preserves the last accepted block.
      fail = true;
      await toggle(blocking: true);
      expect(find.byType(ProfileAccountListTile), findsNothing);
      fail = false;
      await toggle(blocking: true);
      expect(find.byType(ProfileAccountListTile), findsOneWidget);
      await toggle(blocking: false);
      expect(find.byType(ProfileAccountListTile), findsNothing);

      final registry =
          (container.read(sessionRegistryProvider.notifier) as _Registry)
            ..switchTo('did:plc:other');
      await _pumpFrames(tester);
      expect(find.byType(ProfileAccountListTile), findsOneWidget);
      // Reauthentication of the same DID creates a different session lease.
      registry.switchTo('did:plc:test');
      await _pumpFrames(tester);
      expect(find.byType(ProfileAccountListTile), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );

  for (final action in ['block', 'mute']) {
    testWidgets(
      'AT-007 accepted local $action refreshes current member eligibility',
      (tester) async {
        var hidden = false;
        var reads = 0;
        final member = ProfileAccountSummary(
          did: 'did:plc:member',
          handle: 'member.test',
          isCraftskyProfile: true,
        );
        final repo = FakeProfileRepository(
          onFetchMe: () async => Profile(
            did: 'did:plc:test',
            handle: 'test.test',
            crafts: ['knitting'],
          ),
          onBlock: (did, key) async {
            hidden = true;
            return const ProfileRelationship(blocking: true);
          },
          onMute: (did) async {
            hidden = true;
            return const ProfileRelationship(muted: true);
          },
        );
        await _pumpSidebar(
          tester,
          location: '/feed',
          repository: repo,
          readMembers: () async {
            reads++;
            return hidden ? [] : [member];
          },
        );
        expect(find.byType(ProfileAccountListTile), findsOneWidget);
        final container = ProviderScope.containerOf(
          tester.element(find.byType(AuthenticatedShell)),
        );
        final account = container
            .read(sessionRegistryProvider)
            .requireValue
            .activeLease!
            .session
            .account;
        if (action == 'block') {
          await container
              .read(toggleBlockProfileProvider.notifier)
              .toggle(targetDid: member.did, isBlocking: false);
        } else {
          await container
              .read(profileRelationshipProvider(account, member.did).notifier)
              .mutate(ProfileRelationshipAction.mute);
        }
        await _pumpFrames(tester);
        expect(find.byType(ProfileAccountListTile), findsNothing);
        expect(reads, greaterThan(1));
        expect(tester.takeException(), isNull);
      },
    );
  }
  for (final variant in [
    'cached',
    'A-before-B',
    'A-after-B',
    'late-profile-only',
    'late-members-only',
  ]) {
    testWidgets('AT-007 account-scoped sidebar $variant', (tester) async {
      final a = Profile(
        did: 'did:plc:test',
        handle: 'test.test',
        crafts: ['knitting'],
      );
      final b = Profile(
        did: 'did:plc:other',
        handle: 'other.test',
        crafts: ['sewing'],
      );
      final memberA = ProfileAccountSummary(
        did: 'did:plc:membera',
        handle: 'membera.test',
        isCraftskyProfile: true,
      );
      final memberB = ProfileAccountSummary(
        did: 'did:plc:memberb',
        handle: 'memberb.test',
        isCraftskyProfile: true,
      );
      final oldProfile = Completer<Profile>();
      final newProfile = Completer<Profile>();
      final oldMembers = Completer<List<ProfileAccountSummary>>();
      final newMembers = Completer<List<ProfileAccountSummary>>();
      var activeDid = 'did:plc:test';
      var profilePending = false;
      var memberPending = false;
      addTearDown(() {
        if (!oldProfile.isCompleted) oldProfile.complete(a);
        if (!newProfile.isCompleted) newProfile.complete(b);
        if (!oldMembers.isCompleted) oldMembers.complete([memberA]);
        if (!newMembers.isCompleted) newMembers.complete([memberB]);
      });
      Future<Profile> read(String did) => did == b.did.value
          ? newProfile.future
          : profilePending
          ? oldProfile.future
          : Future.value(a);
      final repo = FakeProfileRepository(
        onFetchMe: () => read(activeDid),
        onFetch: read,
      );
      await _pumpSidebar(
        tester,
        location: '/feed',
        repository: repo,
        readMembersFor: (lease) => lease.session.account.did == b.did
            ? newMembers.future
            : memberPending
            ? oldMembers.future
            : Future.value([memberA]),
      );
      expect(find.byKey(const Key('discovery-craft-knitting')), findsOneWidget);
      expect(
        find.byKey(const Key('discovery-member-did:plc:membera')),
        findsOneWidget,
      );
      final container = ProviderScope.containerOf(
        tester.element(find.byType(AuthenticatedShell)),
      );
      final leaseA = container
          .read(sessionRegistryProvider)
          .requireValue
          .activeLease!;
      if (variant != 'cached') {
        profilePending = variant != 'late-members-only';
        memberPending = variant != 'late-profile-only';
        if (profilePending) container.invalidate(userProfileProvider(a.did));
        if (memberPending) container.invalidate(recentMembersProvider(leaseA));
        await _pumpFrames(tester);
      }
      activeDid = b.did.value;
      (container.read(sessionRegistryProvider.notifier) as _Registry).switchTo(
        activeDid,
      );
      await _pumpFrames(tester);
      expect(find.byKey(const Key('discovery-craft-knitting')), findsNothing);
      expect(
        find.byKey(const Key('discovery-member-did:plc:membera')),
        findsNothing,
      );
      if (variant == 'A-before-B') {
        oldProfile.complete(a);
        oldMembers.complete([memberA]);
        await _pumpFrames(tester);
        expect(
          find.byKey(const Key('discovery-member-did:plc:membera')),
          findsNothing,
        );
        expect(find.byKey(const Key('discovery-craft-knitting')), findsNothing);
      }
      newProfile.complete(b);
      newMembers.complete([memberB]);
      await _pumpFrames(tester);
      expect(find.byKey(const Key('discovery-craft-sewing')), findsOneWidget);
      expect(
        find.byKey(const Key('discovery-member-did:plc:memberb')),
        findsOneWidget,
      );
      if (!oldProfile.isCompleted) oldProfile.complete(a);
      if (!oldMembers.isCompleted) oldMembers.complete([memberA]);
      await _pumpFrames(tester);
      expect(find.byKey(const Key('discovery-craft-knitting')), findsNothing);
      expect(
        find.byKey(const Key('discovery-member-did:plc:membera')),
        findsNothing,
      );
      expect(
        find.byKey(const Key('discovery-member-did:plc:memberb')),
        findsOneWidget,
      );
      container
          .read(userProfileProvider(b.did).notifier)
          .setCached(b.copyWith(crafts: ['quilting']));
      await _pumpFrames(tester);
      expect(find.byKey(const Key('discovery-craft-sewing')), findsNothing);
      expect(find.byKey(const Key('discovery-craft-quilting')), findsOneWidget);
      expect(tester.takeException(), isNull);
    });
  }
  for (final pending in ['crafts', 'members', 'both']) {
    testWidgets('AT-006 independent loading failure retry $pending', (
      tester,
    ) async {
      final profileGate = Completer<Profile>();
      final memberGate = Completer<List<ProfileAccountSummary>>();
      final profileRetry = Completer<Profile>();
      final memberRetry = Completer<List<ProfileAccountSummary>>();
      final profile = Profile(
        did: 'did:plc:test',
        handle: 'test.test',
        crafts: ['knitting'],
      );
      final member = ProfileAccountSummary(
        did: 'did:plc:member',
        handle: 'member.test',
        isCraftskyProfile: true,
      );
      final profilePending = pending != 'members';
      final membersPending = pending != 'crafts';
      var recoverProfile = false;
      var recoverMembers = false;
      addTearDown(() {
        if (!profileGate.isCompleted) profileGate.complete(profile);
        if (!memberGate.isCompleted) memberGate.complete([member]);
        if (!profileRetry.isCompleted) profileRetry.complete(profile);
        if (!memberRetry.isCompleted) memberRetry.complete([member]);
      });
      await _pumpSidebar(
        tester,
        location: '/feed',
        settle: false,
        readProfile: () => !profilePending
            ? Future.value(profile)
            : recoverProfile
            ? profileRetry.future
            : profileGate.future,
        readMembers: () => !membersPending
            ? Future.value([member])
            : recoverMembers
            ? memberRetry.future
            : memberGate.future,
      );
      final craftsSection = find.byKey(const Key('discovery-crafts-section'));
      final membersSection = find.byKey(const Key('discovery-members-section'));
      expect(
        find.descendant(
          of: craftsSection,
          matching: find.byType(StitchProgressIndicator),
        ),
        profilePending ? findsOneWidget : findsNothing,
      );
      expect(
        find.descendant(
          of: membersSection,
          matching: find.byType(StitchProgressIndicator),
        ),
        membersPending ? findsOneWidget : findsNothing,
      );
      expect(find.text('Feed content'), findsOneWidget);
      expect(find.text('Choose your crafts'), findsNothing);
      if (profilePending) {
        profileGate.completeError(const ApiServerError('profile test failure'));
      }
      if (membersPending) {
        memberGate.completeError(const ApiServerError('member test failure'));
      }
      await _pumpFrames(tester);
      final container = ProviderScope.containerOf(
        tester.element(find.byType(AuthenticatedShell)),
      );
      final lease = container
          .read(sessionRegistryProvider)
          .requireValue
          .activeLease!;
      final cs = container.read(preferredCraftLinksProvider(lease));
      final ms = container.read(recentMembersProvider(lease));
      // State flags distinguish automatic retry from an initial pending read.
      if (profilePending) {
        expect(
          find.descendant(of: craftsSection, matching: find.text('Retry')),
          findsOneWidget,
          reason:
              '${cs.runtimeType} loading=${cs.isLoading} '
              'error=${cs.hasError} retrying=${cs.retrying}',
        );
        recoverProfile = true;
        await tester.tap(
          find.descendant(of: craftsSection, matching: find.text('Retry')),
        );
        await _pumpFrames(tester);
        expect(
          tester
              .widget<TextButton>(
                find.descendant(
                  of: craftsSection,
                  matching: find.byType(TextButton),
                ),
              )
              .onPressed,
          isNull,
        );
        expect(
          find.descendant(
            of: craftsSection,
            matching: find.byType(StitchProgressIndicator),
          ),
          findsOneWidget,
        );
        profileRetry.complete(profile);
        await _pumpFrames(tester);
        expect(
          find.byKey(const Key('discovery-craft-knitting')),
          findsOneWidget,
        );
        if (membersPending) {
          expect(
            find.descendant(of: membersSection, matching: find.text('Retry')),
            findsOneWidget,
            reason:
                '${ms.runtimeType} loading=${ms.isLoading} '
                'error=${ms.hasError} retrying=${ms.retrying}',
          );
        }
      }
      if (membersPending) {
        expect(
          find.descendant(of: membersSection, matching: find.text('Retry')),
          findsOneWidget,
          reason:
              '${ms.runtimeType} loading=${ms.isLoading} '
              'error=${ms.hasError} retrying=${ms.retrying}',
        );
        recoverMembers = true;
        await tester.tap(
          find.descendant(of: membersSection, matching: find.text('Retry')),
        );
        await _pumpFrames(tester);
        expect(
          tester
              .widget<TextButton>(
                find.descendant(
                  of: membersSection,
                  matching: find.byType(TextButton),
                ),
              )
              .onPressed,
          isNull,
        );
        expect(
          find.descendant(
            of: membersSection,
            matching: find.byType(StitchProgressIndicator),
          ),
          findsOneWidget,
        );
        memberRetry.complete([member]);
        await _pumpFrames(tester);
      }
      expect(find.byKey(const Key('discovery-craft-knitting')), findsOneWidget);
      expect(find.byType(ProfileAccountListTile), findsOneWidget);
      expect(tester.takeException(), isNull);
    });
  }
  testWidgets('AT-006 successful empty member section is distinct from error', (
    tester,
  ) async {
    await _pumpSidebar(tester, location: '/feed');
    expect(find.text('No new members to show yet.'), findsOneWidget);
    expect(find.text('Retry'), findsNothing);
    expect(find.byKey(const Key('discovery-craft-knitting')), findsOneWidget);
  });
  testWidgets(
    'AT-005 member unavailable after listing uses existing profile error',
    (tester) async {
      final member = ProfileAccountSummary(
        did: 'did:plc:member',
        handle: 'member.test',
        isCraftskyProfile: true,
      );
      await _pumpSidebar(
        tester,
        location: '/feed',
        members: Future.value([member]),
        realProfile: true,
        repository: FakeProfileRepository(
          onFetchMe: () async => Profile(
            did: 'did:plc:test',
            handle: 'test.test',
            crafts: ['knitting'],
          ),
          onFetch: (_) async => throw const ApiBadRequest('profile_not_found'),
        ),
      );
      await tester.tap(find.byType(ProfileAccountListTile));
      await tester.pumpAndSettle();
      expect(
        find.text(
          AppLocalizations.of(
            tester.element(find.byType(ProfileAccountListTile)),
          ).profileLoadErrorTitle,
        ),
        findsOneWidget,
      );
      expect(tester.takeException(), isNull);
    },
  );
  for (final count in [1, 4, 5]) {
    testWidgets('AT-005 member rows and profile navigation count $count', (
      tester,
    ) async {
      final rows = List.generate(
        count,
        (index) => ProfileAccountSummary(
          did: 'did:plc:member$index',
          handle: 'member$index.test',
          isCraftskyProfile: true,
          displayName: index == 0 ? null : 'Member $index',
        ),
      );
      final router = await _pumpSidebar(
        tester,
        location: '/feed',
        members: Future.value(rows),
      );
      expect(find.byType(ProfileAccountListTile), findsNWidgets(count));
      expect(
        tester
            .widgetList<ProfileAccountListTile>(
              find.byType(ProfileAccountListTile),
            )
            .map((tile) => tile.account.did),
        rows.map((row) => row.did),
      );
      expect(find.byType(ProfileAvatar), findsNWidgets(count));
      for (final row in rows) {
        await tester.tap(
          find.byType(ProfileAccountListTile).at(rows.indexOf(row)),
        );
        await tester.pumpAndSettle();
        expect(find.text('Profile ${row.did}'), findsOneWidget);
        expect(
          GoRouterState.of(tester.element(find.text('Profile ${row.did}')))
              .pathParameters['did'],
          row.did.value,
        );
        router.pop();
        await tester.pumpAndSettle();
      }
    });
  }
  for (final crafts in <List<String>>[
    [],
    ['unknown'],
  ]) {
    testWidgets(
      'AT-004 empty crafts opens existing editor '
      'and save updates links $crafts',
      (tester) async {
        final profile = Profile(
          did: 'did:plc:test',
          handle: 'test.test',
          crafts: crafts,
        );
        final repo = FakeProfileRepository(
          onFetchMe: () async => profile,
          onUpdateMe: ({
            displayName,
            pronouns,
            description,
            crafts,
            avatar,
            clearAvatar = false,
            banner,
            clearBanner = false,
          }) async => profile.copyWith(crafts: crafts ?? []),
        );
        await _pumpSidebar(
          tester,
          location: '/feed',
          profile: Future.value(profile),
          repository: repo,
        );
        await tester.tap(find.text('Choose your crafts'));
        await tester.pumpAndSettle();
        final craft = find.byKey(const ValueKey('craft-choice-knitting'));
        await tester.ensureVisible(craft);
        await tester.tap(craft);
        await tester.pumpAndSettle();
        await tester.tap(find.widgetWithText(TextButton, 'Save'));
        await tester.pumpAndSettle();
        expect(
          find.byKey(const Key('discovery-craft-knitting')),
          findsOneWidget,
        );
        expect(repo.updateOperationKeys, hasLength(1));
      },
    );
  }
  testWidgets('UT-002 explicit route survives delayed preferences', (
    tester,
  ) async {
    final gate = Completer<Profile>();
    final queries = <String>[];
    addTearDown(() {
      if (!gate.isCompleted) {
        gate.complete(
          Profile(did: 'did:plc:test', handle: 'test.test', crafts: []),
        );
      }
    });
    final router = await _pumpSidebar(
      tester,
      location: '/projects?craft=quilting',
      profile: gate.future,
      queries: queries,
      settle: false,
    );
    expect(_selectedCraft(tester), 'social.craftsky.feed.defs#quilting');
    gate.complete(
      Profile(
        did: 'did:plc:test',
        handle: 'test.test',
        crafts: ['sewing', 'quilting'],
      ),
    );
    await tester.pumpAndSettle();
    expect(_selectedCraft(tester), 'social.craftsky.feed.defs#quilting');
    expect(queries, contains('social.craftsky.feed.defs#quilting'));
    router.go('/projects');
    await tester.pumpAndSettle();
    expect(_selectedCraft(tester), 'social.craftsky.feed.defs#sewing');
  });
  for (final input in [null, 'unknown', 'bad#token', 'sewing']) {
    testWidgets('UT-002 craft boundary $input', (tester) async {
      await _pumpSidebar(
        tester,
        location: input == null
            ? '/projects'
            : '/projects?craft=${Uri.encodeQueryComponent(input)}',
      );
      expect(
        _selectedCraft(tester),
        input == 'sewing'
            ? 'social.craftsky.feed.defs#sewing'
            : 'social.craftsky.feed.defs#knitting',
      );
      final router = GoRouter.of(tester.element(find.byType(ProjectsPage)))
        ..go('/projects?craft=quilting');
      await tester.pumpAndSettle();
      expect(_selectedCraft(tester), 'social.craftsky.feed.defs#quilting');
      router.go('/projects?craft=unknown');
      await tester.pumpAndSettle();
      expect(_selectedCraft(tester), 'social.craftsky.feed.defs#knitting');
    });
  }
  testWidgets(
    'AT-002 distinct supported preferred craft links in profile order',
    (tester) async {
      await _pumpSidebar(
        tester,
        location: '/feed',
        profile: Future.value(
          Profile(
            did: 'did:plc:test',
            handle: 'test.test',
            crafts: ['quilting', 'knitting', 'quilting', 'unknown'],
          ),
        ),
      );
      final links = find.descendant(
        of: find.byKey(const Key('desktop-discovery-sidebar')),
        matching: find.byType(TextButton),
      );
      expect(tester.widgetList<TextButton>(links).map((button) => button.key), [
        const ValueKey('discovery-craft-quilting'),
        const ValueKey('discovery-craft-knitting'),
      ]);
      expect(find.byKey(const Key('discovery-craft-sewing')), findsNothing);
    },
  );
  for (final location in ['/projects', '/feed']) {
    final startingPage = location == '/projects'
        ? 'a mounted Projects tab'
        : 'fresh Projects';
    testWidgets(
      'AT-003 sidebar craft link selects $startingPage',
      (tester) async {
        final queries = <String>[];
        final router = await _pumpSidebar(
          tester,
          location: location,
          queries: queries,
        );
        if (location == '/projects') {
          await tester.tap(find.text('Sewing').first);
          await tester.pumpAndSettle();
          expect(_selectedCraft(tester), 'social.craftsky.feed.defs#sewing');
        }
        for (final craft in ['knitting', 'sewing', 'knitting', 'knitting']) {
          await tester.tap(find.byKey(Key('discovery-craft-$craft')));
          await tester.pumpAndSettle();
          expect(_selectedCraft(tester), 'social.craftsky.feed.defs#$craft');
          expect(
            router.routeInformationProvider.value.uri.queryParameters['craft'],
            craft,
          );
          expect(queries, contains('social.craftsky.feed.defs#$craft'));
        }
        await tester.tap(find.text('Sewing').first);
        await tester.pumpAndSettle();
        expect(
          router.routeInformationProvider.value.uri.queryParameters['craft'],
          'sewing',
        );
        await tester.tap(find.byKey(const Key('discovery-craft-knitting')));
        await tester.pumpAndSettle();
        expect(_selectedCraft(tester), 'social.craftsky.feed.defs#knitting');
        final container = ProviderScope.containerOf(
          tester.element(find.byType(ProjectsPage)),
        );
        expect(
          container
              .read(userProfileProvider(Did.parse('did:plc:test')))
              .requireValue
              .crafts,
          ['knitting', 'sewing'],
        );
        container
            .read(userProfileProvider(Did.parse('did:plc:test')).notifier)
            .setCached(
              Profile(
                did: 'did:plc:test',
                handle: 'test.test',
                crafts: ['sewing'],
              ),
            );
        await tester.pumpAndSettle();
        expect(_selectedCraft(tester), 'social.craftsky.feed.defs#knitting');
        expect(
          router.routeInformationProvider.value.uri.queryParameters['craft'],
          'knitting',
        );
        expect(tester.takeException(), isNull);
      },
    );
  }
}
