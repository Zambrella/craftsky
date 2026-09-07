import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

typedef _IdentitySurface = ({
  String source,
  String behaviorTest,
  List<String> didContracts,
});

void main() {
  const inventory = <String, _IdentitySurface>{
    'route declarations': (
      source: 'lib/router/router.dart',
      behaviorTest: 'test/router/profile_routes_test.dart',
      didContracts: [
        r"path: '${RouteLocations.profiles}/:did'",
        'final Did did;',
        'return UserProfileRoute(did: profile.did).location;',
      ],
    ),
    'own profile': (
      source: 'lib/auth/providers/active_account_identity_provider.dart',
      behaviorTest:
          'test/auth/providers/active_account_identity_provider_test.dart',
      didContracts: ['userProfileProvider(target.did).future'],
    ),
    'posts': (
      source: 'lib/feed/widgets/post_card.dart',
      behaviorTest: 'test/feed/widgets/post_card_test.dart',
      didContracts: ['did: post.author.did'],
    ),
    'notifications': (
      source: 'lib/notifications/services/notification_navigation.dart',
      behaviorTest: 'test/notifications/notifications_page_test.dart',
      didContracts: ['case ProfileDestination(:final did):', 'did: did'],
    ),
    'search': (
      source: 'lib/search/pages/search_results_tabs.dart',
      behaviorTest: 'test/search/search_page_test.dart',
      didContracts: ['did: profile.did'],
    ),
    'follow list': (
      source: 'lib/profile/widgets/profile_account_list_tile.dart',
      behaviorTest: 'test/settings/follow_list_page_test.dart',
      didContracts: ['showUserProfileCard(context, did: account.did)'],
    ),
    'relationship list': (
      source: 'lib/settings/pages/relationship_list_page.dart',
      behaviorTest: 'test/settings/relationship_list_page_test.dart',
      didContracts: ['did: account.did'],
    ),
    'Instagram suggestions': (
      source: 'lib/instagram_migration/pages/instagram_migration_page.dart',
      behaviorTest: 'test/instagram_migration/instagram_suggestions_test.dart',
      didContracts: ['did: Did.parse(suggestion.target.did)'],
    ),
    'recent search': (
      source: 'lib/search/pages/blank_search_view.dart',
      behaviorTest: 'test/search/models/recent_search_test.dart',
      didContracts: [
        'case ProfileRecentSearchPayload(:final did):',
        'onOpenProfile(did)',
      ],
    ),
    'mention': (
      source: 'lib/shared/rich_text/facet_action_handler.dart',
      behaviorTest: 'test/shared/rich_text/faceted_text_actions_test.dart',
      didContracts: ['case MentionFacetFeature(:final did):', 'Did.parse(did)'],
    ),
    'account switcher': (
      source: 'lib/auth/models/account_switcher_state.dart',
      behaviorTest: 'test/router/app_shell_account_switcher_test.dart',
      didContracts: [
        'AccountKey(session.did.value)',
        'session.did == registry.activeDid',
      ],
    ),
    'deletion': (
      source: 'lib/auth/models/pending_account_deletion.dart',
      behaviorTest: 'test/settings/account_deletion_did_acceptance_test.dart',
      didContracts: [
        'required this.confirmationDid',
        'parsedConfirmationDid != lease.session.account.did',
      ],
    ),
    'profile cache publication': (
      source: 'lib/profile/providers/profile_cache_publication.dart',
      behaviorTest: 'test/profile/providers/user_profile_provider_test.dart',
      didContracts: ['userProfileProvider(profile.did)'],
    ),
    'authored post cache publication': (
      source: 'lib/feed/providers/author_post_cache.dart',
      behaviorTest: 'test/feed/providers/user_posts_provider_test.dart',
      didContracts: [
        'Iterable<Did> authorPostCacheIds',
        '[post.author.did]',
      ],
    ),
  };

  test('UT-018 inventories every approved identity surface', () {
    expect(inventory.keys, {
      'route declarations',
      'own profile',
      'posts',
      'notifications',
      'search',
      'follow list',
      'relationship list',
      'Instagram suggestions',
      'recent search',
      'mention',
      'account switcher',
      'deletion',
      'profile cache publication',
      'authored post cache publication',
    });
  });

  for (final MapEntry(key: name, value: surface) in inventory.entries) {
    test('UT-018 $name is DID-first with focused behavior coverage', () {
      final sourceFile = File(surface.source);
      final behaviorTest = File(surface.behaviorTest);

      expect(sourceFile.existsSync(), isTrue, reason: '$name source moved');
      expect(
        behaviorTest.existsSync(),
        isTrue,
        reason: '$name has no focused provider/widget test',
      );
      final source = sourceFile.readAsStringSync();
      for (final contract in surface.didContracts) {
        expect(
          source,
          contains(contract),
          reason: '$name lost DID contract `$contract`',
        );
      }
    });
  }

  test('UT-018 provider and cache identities have no handle fallback', () {
    final profileProvider = File(
      'lib/profile/providers/user_profile_provider.dart',
    ).readAsStringSync();
    final profilePublication = File(
      'lib/profile/providers/profile_cache_publication.dart',
    ).readAsStringSync();
    final postPublication = File(
      'lib/feed/providers/author_post_cache.dart',
    ).readAsStringSync();

    expect(profileProvider, contains('Future<Profile> build(Did did)'));
    expect(profileProvider, isNot(contains('build(String handleOrDid)')));
    expect(profilePublication, isNot(contains('profile.handle')));
    expect(postPublication, isNot(contains('post.author.handle')));
    expect(
      RegExp(r'userProfileProvider\(').allMatches(profilePublication),
      hasLength(1),
      reason: 'Profile cache publication must not dual-publish by DID/handle.',
    );
  });
}
