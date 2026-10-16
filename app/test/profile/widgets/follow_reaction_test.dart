import 'dart:async';

import 'package:craftsky_app/auth/models/session_registry.dart' as model;
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:craftsky_app/notifications/widgets/notification_row.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/pages/profile_page.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/widgets/follow_reaction.dart';
import 'package:craftsky_app/profile/widgets/profile_route_presentation.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';
import 'package:rive/rive.dart' as rive;

import '../../fakes/auth_session_fakes.dart';
import '../../feed/fakes/fake_post_repository.dart';
import '../fakes/fake_profile_repository.dart';

class _Registry extends SessionRegistry {
  @override
  Future<model.SessionRegistry> build() async => model.SessionRegistry.empty();

  void switchAccount() {
    state = AsyncData(
      state.requireValue.upsertAndActivate(
        token: 'test-token',
        did: 'did:plc:other',
        handle: 'other.craftsky.social',
      ),
    );
  }
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  late rive.File file;
  setUpAll(() async {
    await rive.RiveNative.init();
    file = (await rive.File.asset(
      'assets/animations/follow.riv',
      riveFactory: rive.Factory.flutter,
    ))!;
  });
  tearDownAll(() => file.dispose());

  Future<ProviderContainer> setup({
    bool broken = false,
    FakeProfileRepository? repository,
  }) async {
    final container = ProviderContainer(
      retry: (_, _) => null,
      overrides: [
        sessionRegistryProvider.overrideWith(_Registry.new),
        authSessionProvider.overrideWith(SignedInAuthSession.new),
        if (repository != null)
          profileRepositoryProvider.overrideWithValue(repository),
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
        followAnimationFileProvider.overrideWith((ref) async {
          if (broken) throw const FormatException('Test unavailable asset');
          return file;
        }),
      ],
    );
    addTearDown(container.dispose);
    await container.read(sessionRegistryProvider.future);
    await container
        .read(followAnimationFileProvider.future)
        .catchError(
          (Object _) => file,
        );
    return container;
  }

  Widget scene(
    ProviderContainer container, {
    Future<bool> Function()? action,
    bool reduced = false,
    String identity = 'alice',
    VoidCallback? avatarTap,
  }) => UncontrolledProviderScope(
    container: container,
    child: MaterialApp(
      home: MediaQuery(
        data: MediaQueryData(disableAnimations: reduced),
        child: Scaffold(
          body: Row(
            children: [
              for (var i = 0; i < 2; i++)
                FollowReactionScope(
                  identity: '$identity-$i',
                  builder: (context) => Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      FollowReactionAvatar(
                        child: GestureDetector(
                          onTap: avatarTap,
                          behavior: HitTestBehavior.opaque,
                          child: SizedBox.square(
                            key: ValueKey('avatar-$i'),
                            dimension: 96,
                          ),
                        ),
                      ),
                      TextButton(
                        onPressed: () => unawaited(
                          FollowReactionScope.run(
                            context,
                            action ?? () async => true,
                          ),
                        ),
                        child: Text('Follow $i'),
                      ),
                    ],
                  ),
                ),
            ],
          ),
        ),
      ),
    ),
  );

  testWidgets(
    'confirmed follow plays once on the initiating avatar and fades',
    (
      tester,
    ) async {
      final container = await setup();
      final pending = Completer<bool>();
      var taps = 0;
      await tester.pumpWidget(
        scene(container, action: () => pending.future, avatarTap: () => taps++),
      );
      final size = tester.getSize(find.byType(FollowReactionAvatar).first);
      await tester.tap(find.text('Follow 0'));
      await tester.pump();
      expect(find.byKey(const Key('follow-reaction')), findsNothing);
      pending.complete(true);
      await tester.pump();
      final animation = find.byType(rive.RiveWidget);
      expect(animation, findsOneWidget);
      expect(
        tester.widget<rive.RiveWidget>(animation).controller.artboard.name,
        'Follow',
      );
      expect(
        tester.getCenter(animation),
        tester.getCenter(find.byKey(const ValueKey('avatar-0'))),
      );
      expect(tester.getSize(find.byType(FollowReactionAvatar).first), size);
      await tester.tap(
        find
            .ancestor(
              of: find.byKey(const ValueKey('avatar-0')),
              matching: find.byType(GestureDetector),
            )
            .first,
      );
      expect(taps, 1);
      await tester.pump(const Duration(milliseconds: 1300));
      final opacity = tester
          .widget<Opacity>(
            find.ancestor(of: animation, matching: find.byType(Opacity)).first,
          )
          .opacity;
      expect(opacity, lessThan(1));
      await tester.pump(const Duration(milliseconds: 150));
      expect(find.byKey(const Key('follow-reaction')), findsNothing);
      await tester.pumpWidget(const SizedBox());
      container.dispose();
      await tester.pump();
    },
  );

  testWidgets('failed follow or unfollow does not celebrate', (tester) async {
    final container = await setup();
    await tester.pumpWidget(scene(container, action: () async => false));
    await tester.tap(find.text('Follow 0'));
    await tester.pump();
    expect(find.byKey(const Key('follow-reaction')), findsNothing);
    await tester.pumpWidget(const SizedBox());
    container.dispose();
    await tester.pump();
  });

  testWidgets('reduced motion holds the finished knot briefly', (tester) async {
    final container = await setup();
    await tester.pumpWidget(scene(container, reduced: true));
    await tester.tap(find.text('Follow 0'));
    await tester.pump();
    final animation = find.byType(rive.RiveWidget);
    expect(
      tester.widget<rive.RiveWidget>(animation).controller.active,
      isFalse,
    );
    expect(TickerMode.valuesOf(tester.element(animation)).enabled, isFalse);
    await tester.pump(const Duration(milliseconds: 450));
    expect(find.byKey(const Key('follow-reaction')), findsNothing);
    await tester.pumpWidget(const SizedBox());
    container.dispose();
    await tester.pump();
  });

  testWidgets('account switches and recycled identities discard feedback', (
    tester,
  ) async {
    final container = await setup();
    final pending = Completer<bool>();
    await tester.pumpWidget(scene(container, action: () => pending.future));
    await tester.tap(find.text('Follow 0'));
    await tester.pump();
    (container.read(sessionRegistryProvider.notifier) as _Registry)
        .switchAccount();
    await tester.pump();
    pending.complete(true);
    await tester.pump();
    expect(find.byKey(const Key('follow-reaction')), findsNothing);
    await tester.pumpWidget(scene(container));
    await tester.tap(find.text('Follow 0'));
    await tester.pump();
    expect(find.byType(rive.RiveWidget), findsOneWidget);
    await tester.pumpWidget(scene(container, identity: 'bob'));
    expect(find.byKey(const Key('follow-reaction')), findsNothing);
    await tester.pump(const Duration(seconds: 2));
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox());
    container.dispose();
    await tester.pump();
  });

  testWidgets('unavailable asset never blocks the follow', (tester) async {
    final container = await setup(broken: true);
    var follows = 0;
    await tester.pumpWidget(
      scene(
        container,
        action: () async {
          follows++;
          return true;
        },
      ),
    );
    await tester.tap(find.text('Follow 0'));
    await tester.pump();
    expect(follows, 1);
    expect(find.byKey(const Key('follow-reaction')), findsNothing);
    expect(find.byType(CircularProgressIndicator), findsNothing);
    await tester.pumpWidget(const SizedBox());
    container.dispose();
    await tester.pump();
  });
  for (final surface in ['profile', 'compact', 'notification']) {
    testWidgets(
      '$surface follows through the repository and celebrates on avatar',
      (tester) async {
        var profile = Profile(
          did: 'did:plc:alice',
          handle: 'alice.craftsky.social',
          crafts: const [],
        );
        final pending = Completer<Profile>();
        var calls = 0;
        final container = await setup(
          repository: FakeProfileRepository(
            onFetch: (_) async => profile,
            onFollow: (_) {
              calls++;
              return pending.future;
            },
          ),
        );
        final child = switch (surface) {
          'profile' => ProfilePage(did: profile.did),
          'compact' => ProfileRoutePresentation(
            did: profile.did,
            startsCompact: true,
          ),
          _ => NotificationRow(
            notification: CraftskyNotification.fromMap({
              'uri': 'at://did:plc:alice/app.bsky.graph.follow/example',
              'cid': 'bafyexample',
              'rkey': 'example',
              'type': 'follow',
              'actor': {
                'did': 'did:plc:alice',
                'handle': 'alice.craftsky.social',
                'viewerIsFollowing': false,
              },
              'createdAt': '2026-05-28T13:00:00Z',
              'indexedAt': '2026-05-28T13:00:01Z',
            }),
          ),
        };
        await tester.pumpWidget(
          UncontrolledProviderScope(
            container: container,
            child: MaterialApp(
              theme: AppTheme.lightThemeData,
              localizationsDelegates: appLocalizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: Scaffold(body: child),
            ),
          ),
        );
        await tester.pumpAndSettle();
        await tester.tap(find.text('Follow'));
        await tester.pump();
        expect(calls, 1);
        expect(find.byKey(const Key('follow-reaction')), findsNothing);
        profile = profile.copyWith(viewerIsFollowing: true);
        pending.complete(profile);
        for (var i = 0; i < 8; i++) {
          await tester.pump(const Duration(milliseconds: 20));
        }
        expect(find.byType(rive.RiveWidget), findsOneWidget);
        expect(
          tester.getCenter(find.byType(rive.RiveWidget)),
          tester.getCenter(find.byType(FollowReactionAvatar).first),
        );
        await tester.pump(const Duration(milliseconds: 1500));
        expect(find.byKey(const Key('follow-reaction')), findsNothing);
        await tester.pumpWidget(const SizedBox());
        container.dispose();
        await tester.pump();
      },
    );
  }
}
