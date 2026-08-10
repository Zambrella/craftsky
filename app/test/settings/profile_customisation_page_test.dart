import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/settings/pages/profile_customisation_page.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/recording_messenger.dart';
import '../profile/fakes/fake_profile_repository.dart';

void main() {
  testWidgets('edits a live local preview and saves all three choices', (
    tester,
  ) async {
    ProfileCustomisation? submitted;
    final messenger = RecordingMessenger();
    final repository = FakeProfileRepository(
      onFetchMe: () async => Profile(
        did: 'did:plc:alice',
        handle: 'alice.example',
        crafts: [],
      ),
      onUpdateCustomisation: (value) async {
        submitted = value;
        return value;
      },
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          profileRepositoryProvider.overrideWithValue(repository),
        ],
        child: MessengerScope(
          messenger: messenger,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProfileCustomisationPage(),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Colour'), findsOneWidget);
    expect(find.text('Profile border'), findsOneWidget);
    expect(find.text('Profile background'), findsOneWidget);
    await tester.tap(find.text('Teal'));
    await tester.tap(find.text('Thick'));
    await tester.tap(find.text('Crosshatch'));
    await tester.pump();

    final avatar = tester.widget<Container>(
      find.byKey(const Key('profile-avatar-border')),
    );
    final border = (avatar.decoration! as BoxDecoration).border! as Border;
    expect(border.top.color, const Color(0xFF15D6B6));
    expect(border.top.width, 8);
    expect(
      find.byKey(const Key('profile-header-background-texture')),
      findsOneWidget,
    );

    await tester.scrollUntilVisible(
      find.text('Save'),
      300,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();

    expect(
      submitted,
      const ProfileCustomisation(
        colour: 'teal',
        border: 'thick',
        background: 'x2',
      ),
    );
    expect(messenger.calls.last.$2, 'Profile customisation saved');
  });

  testWidgets('retains the draft and shows exact feedback when save fails', (
    tester,
  ) async {
    final messenger = RecordingMessenger();
    final repository = FakeProfileRepository(
      onFetchMe: () async => Profile(
        did: 'did:plc:alice',
        handle: 'alice.example',
        crafts: [],
      ),
      onUpdateCustomisation: (_) async => throw StateError('save failed'),
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          profileRepositoryProvider.overrideWithValue(repository),
        ],
        child: MessengerScope(
          messenger: messenger,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProfileCustomisationPage(),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Rose'));
    await tester.scrollUntilVisible(
      find.text('Save'),
      300,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();

    expect(find.widgetWithText(ChoiceChip, 'Rose'), findsOneWidget);
    expect(
      messenger.calls.last.$2,
      "Couldn't save your profile customisation.",
    );
    expect(find.text('Save'), findsOneWidget);
  });
}
