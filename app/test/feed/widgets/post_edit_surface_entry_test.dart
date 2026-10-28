import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_comment_section.dart';
import 'package:craftsky_app/feed/models/post_edit.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/models/profile_pin_state.dart';
import 'package:craftsky_app/feed/pages/post_quotes_page.dart';
import 'package:craftsky_app/feed/pages/post_thread_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/profile/widgets/profile_tabs/profile_comments_tab.dart';
import 'package:craftsky_app/profile/widgets/profile_tabs/profile_posts_tab.dart';
import 'package:craftsky_app/profile/widgets/profile_tabs/profile_projects_tab.dart';
import 'package:craftsky_app/profile/widgets/profile_tabs/profile_reposts_tab.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/widgets/saved_post_row.dart';
import 'package:craftsky_app/search/models/search_post_page.dart';
import 'package:craftsky_app/search/pages/search_page.dart';
import 'package:craftsky_app/search/pages/tag_search_page.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_access_provider.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

import '../../fakes/auth_session_fakes.dart';
import '../../fakes/recording_messenger.dart';
import '../../search/fakes/fake_search_repository.dart';
import '../../test_support/post_fixtures.dart';
import '../../test_support/session_registry_storage.dart';
import '../fakes/fake_post_repository.dart';

// TD-011: these instantiate each actual caller of the shared menu/metadata,
// rather than infer its target from a shared-card unit test alone.
void main() {
  for (final surface in _Surface.values) {
    for (final owner in [false, true]) {
      testWidgets('REG-005 ${surface.name} supplies the correct edit identity '
          '(owner=$owner)', (tester) async {
        tester.view
          ..devicePixelRatio = 1
          ..physicalSize = const Size(800, 1800);
        addTearDown(tester.view.reset);
        final root = postFixture(rkey: 'root', text: 'Root publication');
        final rootRef = PostRef(uri: root.uri, cid: root.cid);
        final comment = postFixture(rkey: 'comment', text: 'Published comment')
            .copyWith(
              reply: PostReply(root: rootRef, parent: rootRef),
              replyCount: 1,
            );
        final reply = postFixture(rkey: 'reply', text: 'Published reply')
            .copyWith(
              reply: PostReply(
                root: rootRef,
                parent: PostRef(uri: comment.uri, cid: comment.cid),
              ),
            );
        var target = switch (surface) {
          _Surface.threadComment || _Surface.profileComments => comment,
          _Surface.threadReply => reply,
          _ => root,
        };
        if (surface == _Surface.profileProjects) {
          target = target.copyWith(
            project: const Project(
              common: ProjectCommon(
                craftType: 'social.craftsky.feed.defs#sewing',
                title: 'A linen shirt',
              ),
            ),
          );
        }
        if (surface == _Surface.quotes) {
          target = target.copyWith(
            quote: PostRef(
              uri: 'at://did:plc:quoted/social.craftsky.feed.post/quoted',
              cid: 'bafyquoted',
            ),
          );
        }
        if (surface == _Surface.profileReposts) {
          target = target.copyWith(viewerHasReposted: true);
        }
        target = target.copyWith(
          editedAt: DateTime.utc(2026, 10, 26),
        );
        final rootPost = target.rkey.value == 'root' ? target : root;
        final commentPost = target.rkey.value == 'comment' ? target : comment;
        final replyPost = target.rkey.value == 'reply' ? target : reply;
        final page = PostPage(items: [target]);
        final requested = <(Did, RecordKey)>[];
        final loading = Completer<PostEditSeed>();
        final repository = FakePostRepository(
          onProfilePins: () async => const ProfilePinState(),
          onListByAuthor: (id, {cursor, limit}) async => page,
          onListProjectsByAuthor: (id, {cursor, limit}) async => page,
          onListCommentsByAuthor: (id, {cursor, limit}) async => page,
          onListRepostsByAuthor: (id, {cursor, limit}) async => page,
          onListQuotes: (did, key, {cursor, limit}) async => page,
          onFetch: (did, key) async => rootPost,
          onCommentSection: (did, key, {cursor, sort, focus, limit}) async =>
              PostCommentSection(
                post: rootPost,
                sort: CommentSort.oldest,
                comments: CommentPage(
                  items: [
                    CommentItem(
                      post: commentPost,
                      placement: CommentPlacement.normal,
                      replies: const ReplyPage(loaded: false, items: []),
                    ),
                  ],
                ),
              ),
          onListCommentBranchReplies: (did, key, {cursor, limit}) async =>
              ReplyPage(
                loaded: true,
                items: [ReplyItem(post: replyPost, flattened: false)],
              ),
          onFetchEdit: (did, key) {
            requested.add((did, key));
            return loading.future;
          },
        );
        final did = target.author.did;
        final sliver = switch (surface) {
          _Surface.profilePosts => ProfilePostsTab(
            did: did,
            isOwnProfile: owner,
          ),
          _Surface.profileProjects => ProfileProjectsTab(
            did: did,
            isOwnProfile: owner,
          ),
          _Surface.profileComments => ProfileCommentsTab(
            did: did,
            isOwnProfile: owner,
          ),
          _Surface.profileReposts => ProfileRepostsTab(did: did),
          _ => null,
        };
        final body = sliver != null
            ? Scaffold(body: CustomScrollView(slivers: [sliver]))
            : switch (surface) {
                _Surface.quotes => PostQuotesPage(did: did, rkey: root.rkey),
                _Surface.search => const SearchPage(q: 'linen'),
                _Surface.tagSearch => const TagSearchPage(tag: 'linen'),
                _Surface.saved => Scaffold(
                  body: SavedPostRow(
                    account: AccountKey('did:plc:alice'),
                    item: SavedPostItem(
                      post: target,
                      savedAt: DateTime.utc(2026),
                    ),
                    onOpen: () {},
                    onMove: () {},
                    onUnsave: () {},
                  ),
                ),
                _ => PostThreadPage(
                  did: did,
                  rkey: root.rkey,
                  showComments: true,
                ),
              };
        final viewer = owner ? 'did:plc:alice' : 'did:plc:bob';
        final registry = SessionRegistry.empty().upsertAndActivate(
          token: 'fixture-token',
          did: viewer,
          handle: owner ? 'alice.test' : 'bob.test',
        );
        await tester.pumpWidget(
          ProviderScope(
            retry: (_, _) => null,
            overrides: [
              secureSessionRegistryStorageProvider.overrideWithValue(
                InMemorySessionRegistryStorage(registry),
              ),
              authSessionProvider.overrideWith(
                () => SignedInAuthSession(did: viewer),
              ),
              activeLanguagePreferencesProvider.overrideWith(
                (ref) => const LanguagePreferences(
                  primaryLanguage: 'en',
                  contentLanguages: ['en'],
                ),
              ),
              subscriptionAccessProvider.overrideWith(
                (ref, lease) async => SubscriptionAccess(
                  did: lease.account.did,
                  effectiveTier: SubscriptionTier.free,
                  givesAccess: false,
                ),
              ),
              postRepositoryProvider.overrideWithValue(repository),
              searchRepositoryProvider.overrideWithValue(
                FakeSearchRepository(
                  onSearchPosts: ({required q, cursor, limit}) async =>
                      SearchPostPage(items: [target]),
                  onSearchHashtagPosts: (tag, {sort, cursor, limit}) async =>
                      SearchPostPage(items: [target]),
                ),
              ),
            ],
            child: MessengerScope(
              messenger: RecordingMessenger(),
              child: MaterialApp(
                theme: AppTheme.lightThemeData,
                localizationsDelegates: appLocalizationsDelegates,
                supportedLocales: AppLocalizations.supportedLocales,
                home: FormFactorWidget(child: body),
              ),
            ),
          ),
        );
        await tester.pumpAndSettle();
        if (surface == _Surface.threadReply) {
          await tester.tap(find.text('Show 1 reply'));
          await tester.pumpAndSettle();
        }
        if (surface == _Surface.saved) {
          expect(find.byIcon(CraftskyIcons.edit), findsOneWidget);
          await tester.tap(find.byTooltip('Saved post actions'));
          await tester.pumpAndSettle();
          expect(find.text('Edit'), findsNothing);
          return;
        }
        final card = find.byWidgetPredicate(
          (w) => w is PostCard && w.post.uri == target.uri,
        );
        expect(card, findsOneWidget);
        expect(
          find.descendant(of: card, matching: find.byIcon(CraftskyIcons.edit)),
          findsOneWidget,
        );
        await tester.ensureVisible(card);
        await tester.tap(
          find.descendant(
            of: card,
            matching: find.byIcon(CraftskyIconsBold.more),
          ),
        );
        await tester.pumpAndSettle();
        expect(find.text('Edit'), owner ? findsOneWidget : findsNothing);
        if (owner) {
          await tester.tap(find.text('Edit'));
          await tester.pump();
          await tester.pump();
          expect(requested, [(target.author.did, target.rkey)]);
        }
      });
    }
  }
}

enum _Surface {
  profilePosts,
  profileProjects,
  profileComments,
  profileReposts,
  quotes,
  search,
  tagSearch,
  threadRoot,
  threadComment,
  threadReply,
  saved,
}
