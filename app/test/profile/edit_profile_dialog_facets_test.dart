import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_link.dart';
import 'package:craftsky_app/profile/pages/edit_profile_dialog.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/shared/messaging/scaffold_messenger_impl.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

import '../fakes/auth_session_fakes.dart';
import 'fakes/fake_profile_repository.dart';

final _seedProfile = Profile(
  did: 'did:plc:test',
  handle: 'test.bsky.social',
  displayName: 'Test User',
  description: 'Sewist in Bristol',
  crafts: ['sewing', 'quilting'],
);

void main() {
  testWidgets(
    'profile save and reopen retains edited links and explicit removals',
    (tester) async {
      const first = ProfileLink(
        url: 'https://example.com/',
        title: 'Website',
        uri: 'at://did:plc:test/app.bsky.actor.link/3m5abc2345672',
        cid: 'first-cid',
        createdAt: '2026-10-27T12:00:00Z',
      );
      const second = ProfileLink(
        url: 'https://other.example/',
        title: 'Other',
        uri: 'at://did:plc:test/app.bsky.actor.link/3m5abc2345673',
        cid: 'second-cid',
        createdAt: '2026-10-27T12:00:00Z',
      );
      var stored = _seedProfile.copyWith(
        links: const [first, second],
        editableLinks: const [first, second],
      );
      late final FakeProfileRepository repo;
      repo = FakeProfileRepository(
        onFetch: (_) async => stored,
        onUpdateMe:
            ({
              displayName,
              pronouns,
              description,
              crafts,
              avatar,
              clearAvatar = false,
              banner,
              clearBanner = false,
            }) async {
              return stored = stored.copyWith(
                links: repo.lastUpdatedLinks,
                editableLinks: repo.lastUpdatedLinks,
              );
            },
      );
      await _pumpEditDialog(tester, repo: repo);
      await tester.ensureVisible(find.text('Website'));
      await tester.tap(find.text('Website'));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(TextField).last, 'My website');
      await tester.tap(find.byKey(const Key('profile-link-save')));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Other'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Remove link'));
      await tester.pumpAndSettle();
      await tester.tap(find.widgetWithText(TextButton, 'Save'));
      await tester.pumpAndSettle();
      expect(repo.lastUpdatedLinks!.single.title, 'My website');
      expect(repo.lastRemovedLinks!.single.uri, second.uri);
      await tester.tap(find.text('Open'));
      await tester.pumpAndSettle();
      await tester.ensureVisible(find.text('My website'));
      expect(find.text('My website'), findsOneWidget);
      expect(find.text('Other'), findsNothing);
      expect(tester.takeException(), isNull);
    },
  );
  group('EditProfileDialog plain bio', () {
    testWidgets(
      'AT-004 saves description as plain text and shows no autocomplete',
      (tester) async {
        String? capturedDisplayName;
        String? capturedDescription;
        List<String>? capturedCrafts;

        final repo = FakeProfileRepository(
          onFetch: (_) async => _seedProfile,
          onUpdateMe:
              ({
                displayName,
                pronouns,
                description,
                crafts,
                avatar,
                clearAvatar = false,
                banner,
                clearBanner = false,
              }) async {
                capturedDisplayName = displayName;
                capturedDescription = description;
                capturedCrafts = crafts;
                return _seedProfile.copyWith(
                  pronouns: pronouns,
                  description: description,
                );
              },
        );

        await _pumpEditDialog(tester, repo: repo);

        await tester.enterText(
          find.widgetWithText(TextField, 'Sewist in Bristol'),
          'Knitting with @alice.craftsky.social #Lace',
        );
        await tester.pump();
        expect(find.text('Alice'), findsNothing);
        await tester.tap(find.widgetWithText(TextButton, 'Save'));
        await tester.pumpAndSettle();

        expect(capturedDisplayName, 'Test User');
        expect(
          capturedDescription,
          'Knitting with @alice.craftsky.social #Lace',
        );
        expect(capturedCrafts, ['sewing', 'quilting']);
      },
    );
  });
}

Future<void> _pumpEditDialog(
  WidgetTester tester, {
  required FakeProfileRepository repo,
  List<dynamic> overrides = const [],
}) async {
  final scaffoldMessengerKey = GlobalKey<ScaffoldMessengerState>();
  final messenger = ScaffoldMessengerImpl(scaffoldMessengerKey);

  await tester.pumpWidget(
    ProviderScope(
      overrides: List.from([
        authSessionProvider.overrideWith(SignedInAuthSession.new),
        profileRepositoryProvider.overrideWithValue(repo),
        ...overrides,
      ]),
      child: MessengerScope(
        messenger: messenger,
        child: MaterialApp(
          scaffoldMessengerKey: scaffoldMessengerKey,
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (context) => Scaffold(
              body: Center(
                child: ElevatedButton(
                  onPressed: () => showEditProfileDialog(context),
                  child: const Text('Open'),
                ),
              ),
            ),
          ),
        ),
      ),
    ),
  );
  await tester.tap(find.text('Open'));
  await tester.pumpAndSettle();
}
