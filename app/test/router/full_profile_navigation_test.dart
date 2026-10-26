import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_account_summary.dart';
import 'package:craftsky_app/profile/pages/profile_page.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/widgets/profile_account_list_tile.dart';
import 'package:craftsky_app/profile/widgets/profile_tab_bar.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';

import '../fakes/auth_session_fakes.dart';
import '../feed/fakes/fake_post_repository.dart';
import '../profile/fakes/fake_profile_repository.dart';

void main() {
  for (final directEntry in [false, true]) {
    testWidgets(
      directEntry
          ? 'direct profile URL shows the full page'
          : 'account tap shows the full page and Back returns to the origin',
      (tester) async {
        final did = Did.parse('did:plc:alice');
        final router = GoRouter(
          initialLocation: directEntry
              ? UserProfileRoute(did: did).location
              : '/',
          routes: [
            GoRoute(
              path: '/',
              builder: (_, _) => Scaffold(
                body: ProfileAccountListTile(
                  account: ProfileAccountSummary(
                    did: did.toString(),
                    handle: 'alice.craftsky.social',
                    isCraftskyProfile: true,
                    displayName: 'Alice',
                  ),
                ),
              ),
            ),
            GoRoute(
              path: '/profiles/:did',
              builder: (context, state) => UserProfileRoute(
                did: Did.parse(state.pathParameters['did']!),
              ).build(context, state),
            ),
          ],
        );
        addTearDown(router.dispose);
        await tester.pumpWidget(
          ProviderScope(
            overrides: [
              authSessionProvider.overrideWith(SignedInAuthSession.new),
              profileRepositoryProvider.overrideWithValue(
                FakeProfileRepository(
                  onFetch: (_) async => Profile(
                    did: did.toString(),
                    handle: 'alice.craftsky.social',
                    displayName: 'Alice',
                    crafts: const [],
                  ),
                ),
              ),
              postRepositoryProvider.overrideWithValue(
                FakePostRepository(
                  onListByAuthor: (_, {cursor, limit}) async =>
                      const PostPage(items: []),
                  onListProjectsByAuthor: (_, {cursor, limit}) async =>
                      const PostPage(items: []),
                  onListCommentsByAuthor: (_, {cursor, limit}) async =>
                      const PostPage(items: []),
                ),
              ),
            ],
            child: MaterialApp.router(
              theme: AppTheme.lightThemeData,
              localizationsDelegates: appLocalizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              routerConfig: router,
            ),
          ),
        );
        await tester.pumpAndSettle();
        if (!directEntry) {
          await tester.tap(find.text('Alice'));
          await tester.pumpAndSettle();
        }
        expect(router.state.uri.path, UserProfileRoute(did: did).location);
        expect(find.byType(ProfilePage), findsOneWidget);
        expect(find.byType(ProfileTabBar), findsOneWidget);
        expect(
          ModalRoute.of(tester.element(find.byType(ProfilePage)))!.opaque,
          isTrue,
        );
        if (!directEntry) {
          await tester.pageBack();
          await tester.pumpAndSettle();
          expect(router.state.uri.path, '/');
          expect(find.byType(ProfileAccountListTile), findsOneWidget);
          expect(find.byType(ProfilePage), findsNothing);
        }
        expect(tester.takeException(), isNull);
      },
    );
  }
}
