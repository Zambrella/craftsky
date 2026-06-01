import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/pages/edit_profile_dialog.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/shared/messaging/scaffold_messenger_impl.dart';
import 'package:craftsky_app/shared/rich_text/data/facet_suggestion_repository.dart';
import 'package:craftsky_app/shared/rich_text/data/mock_facet_suggestion_repository.dart';
import 'package:craftsky_app/shared/rich_text/providers/facet_suggestion_providers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

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
  group('EditProfileDialog facets', () {
    testWidgets(
      'AT-002 sends generated descriptionFacets while preserving profile '
      'save fields',
      (tester) async {
        String? capturedDisplayName;
        String? capturedDescription;
        List<String>? capturedCrafts;
        List<Map<String, dynamic>>? capturedDescriptionFacets;

        final repo = FakeProfileRepository(
          onFetch: (_) async => _seedProfile,
          onUpdateMeWithFacets:
              ({
                displayName,
                description,
                descriptionFacets,
                crafts,
                avatar,
                clearAvatar = false,
                banner,
                clearBanner = false,
              }) async {
                capturedDisplayName = displayName;
                capturedDescription = description;
                capturedCrafts = crafts;
                capturedDescriptionFacets = descriptionFacets;
                return _seedProfile.copyWith(description: description);
              },
        );

        await _pumpEditDialog(
          tester,
          repo: repo,
          overrides: [
            accountSuggestionRepositoryProvider.overrideWithValue(
              const MockAccountSuggestionRepository(
                accounts: [
                  AccountSuggestion(
                    did: 'did:plc:alice',
                    handle: 'alice.craftsky.social',
                    displayName: 'Alice',
                    avatar: null,
                    isCraftskyProfile: true,
                    viewerIsFollowing: true,
                  ),
                ],
              ),
            ),
          ],
        );

        await tester.enterText(
          find.widgetWithText(TextField, 'Sewist in Bristol'),
          'Knitting with @alice.craftsky.social #Lace',
        );
        await tester.pump();
        await tester.tap(find.widgetWithText(TextButton, 'Save'));
        await tester.pumpAndSettle();

        expect(capturedDisplayName, 'Test User');
        expect(
          capturedDescription,
          'Knitting with @alice.craftsky.social #Lace',
        );
        expect(capturedCrafts, ['sewing', 'quilting']);

        // Flutter intentionally sends descriptionFacets even though the current
        // live AppView may reject the field until backend support lands.
        expect(capturedDescriptionFacets, isNotNull);
        final features = capturedDescriptionFacets!
            .expand((facet) => facet['features']! as List<dynamic>)
            .cast<Map<String, dynamic>>()
            .toList();
        expect(
          features,
          anyElement(
            allOf(
              containsPair(r'$type', 'app.bsky.richtext.facet#mention'),
              containsPair('did', 'did:plc:alice'),
            ),
          ),
        );
        expect(
          features,
          anyElement(
            allOf(
              containsPair(r'$type', 'app.bsky.richtext.facet#tag'),
              containsPair('tag', 'Lace'),
            ),
          ),
        );
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
          localizationsDelegates: AppLocalizations.localizationsDelegates,
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
