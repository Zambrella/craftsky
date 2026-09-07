import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/search/models/recent_search.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/rich_text/widgets/faceted_text.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

import 'fakes/fake_profile_repository.dart';

void main() {
  final originalDid = Did.parse('did:plc:original');
  final currentAliasOwnerDid = Did.parse('did:plc:current-owner');

  testWidgets(
    'AT-003 historical mention text opens its facet DID after reassignment',
    (tester) async {
      final requests = <String>[];
      final router = _router(
        initialLocation: '/',
        home: FacetedText(
          text: '@old.example',
          facets: [
            _mentionFacet('@old.example', originalDid),
          ],
        ),
      );

      await _pump(
        tester,
        router: router,
        repository: _repository(requests, originalDid, currentAliasOwnerDid),
      );

      expect(find.text('@old.example'), findsOneWidget);
      await tester.tap(find.text('@old.example'));
      await tester.pumpAndSettle();

      expect(
        router.state.uri.path,
        UserProfileRoute(did: originalDid).location,
      );
      expect(find.text('did:plc:original @new.example'), findsOneWidget);
      expect(requests, ['did:plc:original']);
    },
  );

  testWidgets(
    'AT-003 explicit reassigned handle resolves its current owner '
    'then replaces URL',
    (tester) async {
      final requests = <String>[];
      final router = _router(initialLocation: '/profiles/@old.example');

      await _pump(
        tester,
        router: router,
        repository: _repository(requests, originalDid, currentAliasOwnerDid),
      );
      await tester.pumpAndSettle();

      expect(
        router.state.uri.path,
        UserProfileRoute(did: currentAliasOwnerDid).location,
      );
      expect(find.text('did:plc:current-owner @old.example'), findsOneWidget);
      expect(requests, ['old.example', 'did:plc:current-owner']);
    },
  );

  testWidgets('AT-003 canonical DID URL remains durable after handle change', (
    tester,
  ) async {
    final requests = <String>[];
    final router = _router(
      initialLocation: UserProfileRoute(did: originalDid).location,
    );

    await _pump(
      tester,
      router: router,
      repository: _repository(requests, originalDid, currentAliasOwnerDid),
    );
    await tester.pumpAndSettle();

    expect(router.state.uri.path, UserProfileRoute(did: originalDid).location);
    expect(find.text('did:plc:original @new.example'), findsOneWidget);
    expect(requests, ['did:plc:original']);
  });

  testWidgets('AT-003 recent and internal links retain the original DID', (
    tester,
  ) async {
    final requests = <String>[];
    final recent = ProfileRecentSearchPayload(
      did: originalDid,
      handle: 'old.example',
    );
    late final GoRouter router;
    router = _router(
      initialLocation: '/',
      home: Builder(
        builder: (context) => Column(
          children: [
            Text('@${recent.handle}'),
            TextButton(
              onPressed: () => UserProfileRoute(
                did: recent.did,
              ).push<void>(context),
              child: const Text('Open recent profile'),
            ),
            TextButton(
              onPressed: () => UserProfileRoute(
                did: originalDid,
              ).push<void>(context),
              child: const Text('Open internal profile'),
            ),
          ],
        ),
      ),
    );

    await _pump(
      tester,
      router: router,
      repository: _repository(requests, originalDid, currentAliasOwnerDid),
    );

    await tester.tap(find.text('Open recent profile'));
    await tester.pumpAndSettle();
    expect(router.state.uri.path, UserProfileRoute(did: originalDid).location);

    router.go('/');
    await tester.pumpAndSettle();
    expect(find.text('@old.example'), findsOneWidget);
    await tester.tap(find.text('Open internal profile'));
    await tester.pumpAndSettle();

    expect(router.state.uri.path, UserProfileRoute(did: originalDid).location);
    expect(requests, isNotEmpty);
    expect(requests, everyElement('did:plc:original'));
  });
}

FakeProfileRepository _repository(
  List<String> requests,
  Did originalDid,
  Did currentAliasOwnerDid,
) {
  return FakeProfileRepository(
    onFetch: (identifier) async {
      requests.add(identifier);
      return switch (identifier) {
        'old.example' => Profile(
          did: currentAliasOwnerDid,
          handle: 'old.example',
          crafts: const [],
        ),
        'did:plc:original' => Profile(
          did: originalDid,
          handle: 'new.example',
          crafts: const [],
        ),
        'did:plc:current-owner' => Profile(
          did: currentAliasOwnerDid,
          handle: 'old.example',
          crafts: const [],
        ),
        _ => throw StateError('Unexpected profile request: $identifier'),
      };
    },
  );
}

GoRouter _router({required String initialLocation, Widget? home}) {
  return GoRouter(
    initialLocation: initialLocation,
    routes: [
      GoRoute(
        path: '/',
        builder: (context, state) => Scaffold(body: home),
      ),
      GoRoute(
        path: '/profiles/@:handle',
        redirect: (context, state) => ProfileAliasRoute(
          handle: Handle.parse(state.pathParameters['handle']!),
        ).redirect(context, state),
      ),
      GoRoute(
        path: '/profiles/:did',
        builder: (context, state) => Scaffold(
          body: _ProfileTarget(
            did: Did.parse(state.pathParameters['did']!),
          ),
        ),
      ),
      GoRoute(
        path: '/search/tags',
        builder: (context, state) => const SizedBox.shrink(),
      ),
    ],
  );
}

Future<void> _pump(
  WidgetTester tester, {
  required GoRouter router,
  required FakeProfileRepository repository,
}) {
  return tester.pumpWidget(
    ProviderScope(
      overrides: [
        profileRepositoryProvider.overrideWithValue(repository),
      ],
      child: MaterialApp.router(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        routerConfig: router,
      ),
    ),
  );
}

class _ProfileTarget extends ConsumerWidget {
  const _ProfileTarget({required this.did});

  final Did did;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return switch (ref.watch(userProfileProvider(did))) {
      AsyncValue(:final value?) => Text('${value.did} @${value.handle}'),
      AsyncError(:final error) => Text('error:$error'),
      _ => const CircularProgressIndicator(),
    };
  }
}

Map<String, dynamic> _mentionFacet(String text, Did did) {
  return {
    'index': {'byteStart': 0, 'byteEnd': text.length},
    'features': [
      {
        r'$type': 'app.bsky.richtext.facet#mention',
        'did': did.toString(),
      },
    ],
  };
}
