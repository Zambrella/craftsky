import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/pages/edit_profile_dialog.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/auth_session_fakes.dart';
import 'fakes/fake_profile_repository.dart';

const _seedProfile = Profile(
  did: 'did:plc:test',
  handle: 'test.bsky.social',
  displayName: 'Test User',
  description: 'Sewist in Bristol',
  crafts: ['sewing', 'quilting'],
);

/// Test harness that hosts [EditProfileDialog] via the same
/// `showEditProfileDialog` helper the app uses, so pop semantics
/// (close-on-save, system-back-with-confirm) exercise the real route
/// shape. The host exposes an `Open` button that opens the dialog so
/// a subsequent pop returns to a known marker.
Future<void> _pumpEditDialog(
  WidgetTester tester, {
  required FakeProfileRepository repo,
}) async {
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        authSessionProvider.overrideWith(SignedInAuthSession.new),
        profileRepositoryProvider.overrideWithValue(repo),
      ],
      child: MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Builder(
          builder: (context) {
            return Scaffold(
              body: Center(
                child: ElevatedButton(
                  onPressed: () => showEditProfileDialog(context),
                  child: const Text('Open'),
                ),
              ),
            );
          },
        ),
      ),
    ),
  );
  await tester.tap(find.text('Open'));
  await tester.pumpAndSettle();
}

void main() {
  group('EditProfileDialog', () {
    testWidgets('seeds form with current display name and bio', (tester) async {
      final repo = FakeProfileRepository(onFetch: (_) async => _seedProfile);
      await _pumpEditDialog(tester, repo: repo);

      // Display name and bio are seeded into the controllers — assert
      // by widget value rather than text occurrence so we don't pick
      // up the page's own static labels by accident.
      expect(
        find.widgetWithText(TextField, 'Test User'),
        findsOneWidget,
      );
      expect(
        find.widgetWithText(TextField, 'Sewist in Bristol'),
        findsOneWidget,
      );
    });

    testWidgets('save is disabled until the user makes a change', (
      tester,
    ) async {
      final repo = FakeProfileRepository(onFetch: (_) async => _seedProfile);
      await _pumpEditDialog(tester, repo: repo);

      final saveButton = tester.widget<TextButton>(
        find.widgetWithText(TextButton, 'Save'),
      );
      expect(saveButton.onPressed, isNull);

      await tester.enterText(
        find.widgetWithText(TextField, 'Test User'),
        'Renamed',
      );
      await tester.pump();

      final updated = tester.widget<TextButton>(
        find.widgetWithText(TextButton, 'Save'),
      );
      expect(updated.onPressed, isNotNull);
    });

    testWidgets('saving sends the full current form state', (tester) async {
      String? capturedDisplayName;
      String? capturedDescription;
      List<String>? capturedCrafts;
      var updateCallCount = 0;

      final repo = FakeProfileRepository(
        onFetch: (_) async => _seedProfile,
        onUpdateMe: ({displayName, description, crafts}) async {
          updateCallCount++;
          capturedDisplayName = displayName;
          capturedDescription = description;
          capturedCrafts = crafts;
          return _seedProfile.copyWith(displayName: displayName);
        },
      );

      await _pumpEditDialog(tester, repo: repo);

      // Only displayName visibly changes — but the save call still
      // sends every field's current value, because atproto profile
      // records are atomic and a partial PUT would clear the absent
      // fields on the PDS.
      await tester.enterText(
        find.widgetWithText(TextField, 'Test User'),
        'Renamed',
      );
      await tester.pump();
      await tester.tap(find.widgetWithText(TextButton, 'Save'));
      await tester.pumpAndSettle();

      expect(updateCallCount, 1);
      expect(capturedDisplayName, 'Renamed');
      expect(capturedDescription, 'Sewist in Bristol');
      expect(capturedCrafts, ['sewing', 'quilting']);
    });

    testWidgets('successful save pops back to the previous route', (
      tester,
    ) async {
      final repo = FakeProfileRepository(
        onFetch: (_) async => _seedProfile,
        onUpdateMe: ({displayName, description, crafts}) async =>
            _seedProfile.copyWith(displayName: displayName ?? 'Test User'),
      );
      await _pumpEditDialog(tester, repo: repo);

      // Dialog covers the host (the route is opaque), so the host's
      // Open button is offstage and `find.text` skips it by default.
      expect(find.text('Open'), findsNothing);

      await tester.enterText(
        find.widgetWithText(TextField, 'Test User'),
        'Renamed',
      );
      await tester.pump();
      await tester.tap(find.widgetWithText(TextButton, 'Save'));
      await tester.pumpAndSettle();

      // Dialog popped — host's Open button is back on screen.
      expect(find.text('Open'), findsOneWidget);
    });

    testWidgets('successful save updates the cached profile without refetch', (
      tester,
    ) async {
      var fetchCallCount = 0;
      final repo = FakeProfileRepository(
        onFetch: (_) async {
          fetchCallCount++;
          return _seedProfile;
        },
        onUpdateMe: ({displayName, description, crafts}) async =>
            _seedProfile.copyWith(displayName: displayName),
      );
      await _pumpEditDialog(tester, repo: repo);

      // Keep a listener on the family entry so it stays alive past
      // the edit page's pop and we can observe its cached value.
      final container = ProviderScope.containerOf(
        tester.element(find.byType(EditProfileDialog)),
      );
      final sub = container.listen<AsyncValue<Profile>>(
        userProfileProvider('test.bsky.social'),
        (_, _) {},
      );
      addTearDown(sub.close);

      // Edit page's initial fetch already happened.
      expect(fetchCallCount, 1);

      await tester.enterText(
        find.widgetWithText(TextField, 'Test User'),
        'Renamed',
      );
      await tester.pump();
      await tester.tap(find.widgetWithText(TextButton, 'Save'));
      await tester.pumpAndSettle();

      // Cache reflects the saved profile — pushed via setCached, no
      // second fetch fired.
      expect(sub.read().value?.displayName, 'Renamed');
      expect(fetchCallCount, 1);
    });

    testWidgets('failed save surfaces an error snackbar and stays on page', (
      tester,
    ) async {
      final repo = FakeProfileRepository(
        onFetch: (_) async => _seedProfile,
        onUpdateMe: ({displayName, description, crafts}) async {
          throw Exception('boom');
        },
      );
      await _pumpEditDialog(tester, repo: repo);

      await tester.enterText(
        find.widgetWithText(TextField, 'Test User'),
        'Renamed',
      );
      await tester.pump();
      await tester.tap(find.widgetWithText(TextButton, 'Save'));
      await tester.pumpAndSettle();

      // Snackbar text comes from the editProfileSaveError ARB key.
      expect(find.text("Couldn't save your profile."), findsOneWidget);
      // Still on the edit page — the host's Open button hasn't returned.
      expect(find.text('Open'), findsNothing);
    });

    testWidgets(
      'clearing the display name is valid (empty is a permitted value)',
      (tester) async {
        final repo = FakeProfileRepository(onFetch: (_) async => _seedProfile);
        await _pumpEditDialog(tester, repo: repo);

        await tester.enterText(
          find.widgetWithText(TextField, 'Test User'),
          '',
        );
        await tester.pump();

        // The maxLength validator must not fire on empty values —
        // form_builder_validators 11.x's BaseValidator otherwise treats
        // null/empty as a hard failure.
        expect(
          find.text('Display name must be 64 characters or fewer'),
          findsNothing,
        );

        // Form is dirty (was 'Test User', now '') and valid → save
        // is enabled.
        final saveButton = tester.widget<TextButton>(
          find.widgetWithText(TextButton, 'Save'),
        );
        expect(saveButton.onPressed, isNotNull);
      },
    );

    testWidgets(
      'editing keeps the active field focused when validation toggles',
      (tester) async {
        final repo = FakeProfileRepository(onFetch: (_) async => _seedProfile);
        await _pumpEditDialog(tester, repo: repo);

        // Tap the field to give it focus, then drive it past the limit.
        // Use a low-level `enterText`-then-`requestFocus` flow because
        // tester.tap on a TextField doesn't always reliably hand focus
        // in widget tests.
        final fieldFinder = find.widgetWithText(TextField, 'Test User');
        await tester.tap(fieldFinder);
        await tester.pumpAndSettle();

        final focusNode = tester
            .widget<TextField>(fieldFinder)
            .focusNode!;
        expect(focusNode.hasFocus, isTrue);

        // Type something that fails validation. The
        // FormBuilderField.validate path used to focus a sibling
        // FocusNode here, stealing focus from the TextField — sharing
        // the focus node fixes that.
        await tester.enterText(fieldFinder, 'x' * 65);
        await tester.pump();

        expect(
          find.text('Display name must be 64 characters or fewer'),
          findsOneWidget,
        );
        expect(focusNode.hasFocus, isTrue);
      },
    );

    testWidgets(
      'display name longer than 64 characters surfaces a validator error '
      'and disables save',
      (tester) async {
        final repo = FakeProfileRepository(onFetch: (_) async => _seedProfile);
        await _pumpEditDialog(tester, repo: repo);

        await tester.enterText(
          find.widgetWithText(TextField, 'Test User'),
          'x' * 65,
        );
        await tester.pump();

        // The validator's errorText (sourced from the ARB) renders
        // beneath the field once autovalidate kicks in.
        expect(
          find.text('Display name must be 64 characters or fewer'),
          findsOneWidget,
        );

        // Save is disabled even though the form is dirty — invalid
        // fields fail the canSave gate.
        final saveButton = tester.widget<TextButton>(
          find.widgetWithText(TextButton, 'Save'),
        );
        expect(saveButton.onPressed, isNull);
      },
    );

    testWidgets('tapping a craft chip toggles its selection state', (
      tester,
    ) async {
      final repo = FakeProfileRepository(onFetch: (_) async => _seedProfile);
      await _pumpEditDialog(tester, repo: repo);

      // 'Knitting' starts unselected (seed has only sewing + quilting).
      // Use the Semantics' selected flag rather than colour to verify
      // toggle, since colours are theme-dependent.
      final knittingFinder = find.byWidgetPredicate(
        (w) => w is Semantics && w.properties.label == 'Knitting',
      );
      expect(knittingFinder, findsOneWidget);

      // The crafts grid is below the fold in the default 800x600 test
      // viewport — scroll it into view before tapping.
      await tester.ensureVisible(knittingFinder);
      await tester.pumpAndSettle();

      Semantics knitting() => tester.widget<Semantics>(knittingFinder);
      expect(knitting().properties.selected, isFalse);

      await tester.tap(knittingFinder);
      await tester.pump();
      expect(knitting().properties.selected, isTrue);

      await tester.tap(knittingFinder);
      await tester.pump();
      expect(knitting().properties.selected, isFalse);
    });
  });
}
