import 'dart:async';

import 'package:craftsky_app/chat/models/chat_conversation.dart';
import 'package:craftsky_app/chat/widgets/chat_peer_profile.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar_hero.dart';
import 'package:craftsky_app/profile/widgets/profile_sliver_app_bar.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/widgets/craft_icon.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  testWidgets(
    'shared chat profile shows crafts and flies to and from profile',
    (
      tester,
    ) async {
      final did = Did.parse('did:plc:peer');
      final pending = Completer<Profile>();
      final router = GoRouter(
        routes: [
          GoRoute(
            path: '/',
            builder: (_, _) => Scaffold(
              body: ChatPeerProfile(
                peer: ChatActor.fromMap({
                  'did': did.value,
                  'handle': 'peer.test',
                }),
                builder: (context, avatar, identity, openProfile) => ListTile(
                  leading: avatar,
                  title: identity,
                  onTap: openProfile,
                ),
              ),
            ),
          ),
          GoRoute(
            path: '/profiles/:did',
            builder: (context, state) => ProfileAvatarHeroDestination(
              transition: state.extra! as ProfileAvatarTransition,
              child: const Scaffold(
                body: CustomScrollView(
                  slivers: [
                    ProfileSliverAppBar(
                      handle: 'peer.test',
                      displayName: 'Peer',
                      crafts: ['knitting'],
                    ),
                  ],
                ),
              ),
            ),
          ),
        ],
      );
      addTearDown(router.dispose);
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            userProfileProvider(did)
                .overrideWith(() => _Profile(pending.future)),
          ],
          child: MaterialApp.router(
            routerConfig: router,
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
          ),
        ),
      );
      await tester.pump();
      expect(find.text('peer.test'), findsOneWidget);
      pending.complete(
        Profile(
          did: did.value,
          handle: 'peer.test',
          displayName: 'Peer',
          crafts: const ['knitting'],
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byType(CraftIcon), findsOneWidget);
      for (final forward in [true, false]) {
        if (forward) {
          await tester.tap(find.text('Peer'));
        } else {
          router.pop();
        }
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 100));
        expect(find.byKey(const Key('profile-avatar-flight')), findsOneWidget);
        expect(find.byKey(const Key('profile-name-flight')), findsOneWidget);
        expect(find.byKey(const Key('profile-handle-flight')), findsOneWidget);
        expect(
          find.byKey(const ValueKey('profile-craft-flight-knitting')),
          findsOneWidget,
        );
        await tester.pumpAndSettle();
        expect(tester.takeException(), isNull);
      }
      expect(find.text('Peer'), findsOneWidget);
      expect(find.byType(CraftIcon), findsOneWidget);
    },
  );
}

class _Profile extends UserProfile {
  new(this.pending);
  final Future<Profile> pending;
  @override
  Future<Profile> build(Did did) => pending;
}
