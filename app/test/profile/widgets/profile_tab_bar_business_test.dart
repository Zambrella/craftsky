import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/business/data/business_repository.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/providers/business_repository_provider.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/profile/pages/profile_page.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/auth_session_fakes.dart';
import '../../feed/fakes/fake_post_repository.dart';
import '../fakes/fake_profile_repository.dart';

const _cid = 'bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq';

void main() {
  testWidgets(
    'account type changes replace the tab composition with semantics enabled',
    (tester) async {
      final semantics = tester.ensureSemantics();
      var profile = Profile(
        did: 'did:plc:maker',
        handle: 'maker.test',
        crafts: const [],
        accountType: AccountType.regular,
      );
      final container = ProviderContainer.test(
        overrides: [
          authSessionProvider.overrideWith(SignedInAuthSession.new),
          profileRepositoryProvider.overrideWithValue(
            FakeProfileRepository(onFetch: (_) async => profile),
          ),
          postRepositoryProvider.overrideWithValue(
            FakePostRepository(
              onListByAuthor: (_, {cursor, limit}) async =>
                  const PostPage(items: []),
              onListCommentsByAuthor: (_, {cursor, limit}) async =>
                  const PostPage(items: []),
            ),
          ),
          businessRepositoryProvider.overrideWithValue(
            _EmptyBusinessRepository(),
          ),
        ],
        retry: (_, _) => null,
      );
      addTearDown(container.dispose);

      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProfilePage(handle: 'maker.test'),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(_tabLabels(tester), hasLength(5));
      await tester.tap(find.text('Comments'));
      await tester.pumpAndSettle();

      profile = Profile(
        did: 'did:plc:maker',
        handle: 'maker.test',
        crafts: const [],
        accountType: AccountType.business,
        business: BusinessProfile(cid: _cid),
      );
      container.invalidate(userProfileProvider('maker.test'));
      await tester.pumpAndSettle();

      expect(tester.takeException(), isNull);
      expect(_tabLabels(tester), hasLength(7));
      expect(_selectedTab(tester), 'Comments');
      await tester.tap(find.text('Products'));
      await tester.pumpAndSettle();

      profile = profile.copyWith(
        accountType: AccountType.regular,
        business: null,
      );
      container.invalidate(userProfileProvider('maker.test'));
      await tester.pumpAndSettle();

      expect(tester.takeException(), isNull);
      expect(_tabLabels(tester), hasLength(5));
      expect(_selectedTab(tester), 'About');
      semantics.dispose();
    },
  );

  testWidgets('AT-003 profile uses exact business tabs and visitor empties', (
    tester,
  ) async {
    await _pumpProfile(
      tester,
      Profile(
        did: 'did:plc:maker',
        handle: 'maker.test',
        crafts: const [],
        accountType: AccountType.business,
        business: BusinessProfile(cid: _cid),
      ),
    );

    expect(_tabLabels(tester), [
      'Projects',
      'Posts',
      'Comments',
      'Reposts',
      'Products',
      'Upcoming Events',
      'About',
    ]);

    await tester.tap(find.text('Products'));
    await tester.pumpAndSettle();
    expect(find.text('No featured products yet.'), findsOneWidget);
    expect(find.text('Manage products'), findsNothing);

    await tester.tap(find.text('Upcoming Events'));
    await tester.pumpAndSettle();
    expect(find.text('No upcoming events yet.'), findsOneWidget);
    expect(find.text('Manage events'), findsNothing);
  });

  testWidgets('REG-004 blocked business profile stays a reduced shell', (
    tester,
  ) async {
    final repository = _EmptyBusinessRepository();
    await _pumpProfile(
      tester,
      Profile(
        did: 'did:plc:maker',
        handle: 'maker.test',
        crafts: const [],
        blocking: true,
        accountType: AccountType.business,
        business: BusinessProfile(
          cid: _cid,
          tagline: 'Must stay hidden',
          businessTypes: const [
            BusinessOpenValue(value: 'teacher', known: true),
          ],
          products: const [
            BusinessProductView(title: 'Hidden product'),
          ],
        ),
      ),
      businessRepository: repository,
    );

    expect(find.text('Blocked by you'), findsOneWidget);
    expect(find.byType(TabBar), findsNothing);
    expect(find.text('Business'), findsNothing);
    expect(find.text('Must stay hidden'), findsNothing);
    expect(find.text('Business types'), findsNothing);
    expect(find.text('Hidden product'), findsNothing);
    expect(find.text('Products'), findsNothing);
    expect(find.text('Upcoming Events'), findsNothing);
    expect(repository.profileEventListCalls, 0);
  });

  testWidgets('REG-001 regular profile keeps exactly five ordinary tabs', (
    tester,
  ) async {
    await _pumpProfile(
      tester,
      Profile(
        did: 'did:plc:maker',
        handle: 'maker.test',
        crafts: const [],
        accountType: AccountType.regular,
      ),
    );

    expect(_tabLabels(tester), [
      'Projects',
      'Posts',
      'Comments',
      'Reposts',
      'About',
    ]);
    expect(find.text('Products'), findsNothing);
    expect(find.text('Upcoming Events'), findsNothing);
  });

  testWidgets('REG-005 business tabs keep customization boundary and keys', (
    tester,
  ) async {
    await _pumpProfile(
      tester,
      Profile(
        did: 'did:plc:maker',
        handle: 'maker.test',
        crafts: const [],
        accountType: AccountType.business,
        customisation: const ProfileCustomisation(colour: 'orchid'),
        business: BusinessProfile(cid: _cid),
      ),
    );

    expect(
      Theme.of(tester.element(find.byType(TabBar))).colorScheme.primary,
      AppTheme.lightThemeData.colorScheme.primary,
    );
    for (final entry in const {
      'Projects': 'projects',
      'Posts': 'posts',
      'Comments': 'comments',
      'Reposts': 'reposts',
      'Products': 'products',
      'Upcoming Events': 'upcomingEvents',
      'About': 'about',
    }.entries) {
      await tester.ensureVisible(find.text(entry.key));
      await tester.tap(find.text(entry.key));
      await tester.pumpAndSettle();
      expect(
        find.byKey(PageStorageKey<String>('profile_tab_${entry.value}')),
        findsOneWidget,
      );
    }
  });
}

List<String?> _tabLabels(WidgetTester tester) => tester
    .widgetList<Tab>(
      find.descendant(of: find.byType(TabBar), matching: find.byType(Tab)),
    )
    .map((tab) => tab.text)
    .toList();

String? _selectedTab(WidgetTester tester) {
  final tabBar = tester.widget<TabBar>(find.byType(TabBar));
  return _tabLabels(tester)[tabBar.controller!.index];
}

Future<void> _pumpProfile(
  WidgetTester tester,
  Profile profile, {
  BusinessRepository? businessRepository,
}) async {
  final posts = FakePostRepository(
    onListByAuthor: (_, {cursor, limit}) async => const PostPage(items: []),
    onListCommentsByAuthor: (_, {cursor, limit}) async =>
        const PostPage(items: []),
  );
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        authSessionProvider.overrideWith(SignedInAuthSession.new),
        profileRepositoryProvider.overrideWithValue(
          FakeProfileRepository(onFetch: (_) async => profile),
        ),
        postRepositoryProvider.overrideWithValue(posts),
        businessRepositoryProvider.overrideWithValue(
          businessRepository ?? _EmptyBusinessRepository(),
        ),
      ],
      child: MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: const ProfilePage(handle: 'maker.test'),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

final class _EmptyBusinessRepository extends Fake
    implements BusinessRepository {
  int profileEventListCalls = 0;

  @override
  Future<BusinessEventPage> listProfileEvents(
    AtIdentifier owner, {
    String? cursor,
    int limit = 10,
  }) async {
    profileEventListCalls++;
    return const BusinessEventPage(items: []);
  }
}
