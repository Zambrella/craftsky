import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/business/data/business_repository.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/providers/business_repository_provider.dart';
import 'package:craftsky_app/feed/media/composer_image_media_service.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/profile/models/profile_image_failure.dart';
import 'package:craftsky_app/profile/pages/edit_profile_dialog.dart';
import 'package:craftsky_app/profile/providers/profile_image_picker_provider.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/link/external_link.dart';
import 'package:craftsky_app/shared/media/blob_api_client.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/shared/messaging/scaffold_messenger_impl.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_access_provider.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:dio/dio.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image_picker/image_picker.dart';
import 'package:material_ui/material_ui.dart';

import '../business/accessibility_test_helpers.dart';
import '../fakes/auth_session_fakes.dart';
import '../test_support/recording_error_reporter.dart';
import 'fakes/fake_profile_repository.dart';

final _seedProfile = Profile(
  did: 'did:plc:test',
  handle: 'test.bsky.social',
  displayName: 'Test User',
  pronouns: 'they/them',
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
  BusinessRepository? businessRepository,
  bool businessAccess = false,
  ProfileImageFailure? imageFailure,
  Object? imageException,
  ErrorReporter? reporter,
  ExternalLinkLauncher linkLauncher = launchExternalLink,
  ExternalLinkConfirmer confirmOpenLink = showOpenLinkDialog,
}) async {
  repo.fallbackFetchMeToFetch = true;
  final scaffoldMessengerKey = GlobalKey<ScaffoldMessengerState>();
  final messenger = ScaffoldMessengerImpl(scaffoldMessengerKey);

  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        if (reporter != null) errorReporterProvider.overrideWithValue(reporter),
        if (imageFailure != null || imageException != null)
          profileImagePickerProvider.overrideWithValue(
            _FailingProfileImagePicker(
              imageException ??
                  ProfileImagePickException(reason: imageFailure!),
            ),
          ),
        secureSessionRegistryStorageProvider.overrideWithValue(
          _EditRegistryStorage(
            SessionRegistry.empty().upsertAndActivate(
              token: 'test-token',
              did: 'did:plc:test',
              handle: 'test.bsky.social',
            ),
          ),
        ),
        subscriptionAccessProvider.overrideWith((ref, lease) async {
          final tier = businessAccess || businessRepository != null
              ? SubscriptionTier.business
              : SubscriptionTier.plus;
          return SubscriptionAccess(
            did: lease.account.did,
            effectiveTier: tier,
            givesAccess: true,
            assignedTier: tier,
          );
        }),
        authSessionProvider.overrideWith(SignedInAuthSession.new),
        profileRepositoryProvider.overrideWithValue(repo),
        pdsRecordOperationControllerProvider.overrideWithValue(
          PdsRecordOperationController(schedule: (_, _) {}),
        ),
        if (businessRepository != null)
          businessRepositoryProvider.overrideWithValue(businessRepository),
      ],
      child: MessengerScope(
        messenger: messenger,
        child: MaterialApp(
          scaffoldMessengerKey: scaffoldMessengerKey,
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (context) {
              return Scaffold(
                body: Center(
                  child: ElevatedButton(
                    onPressed: () => showEditProfileDialog(
                      context,
                      linkLauncher: linkLauncher,
                      confirmOpenLink: confirmOpenLink,
                    ),
                    child: const Text('Open'),
                  ),
                ),
              );
            },
          ),
        ),
      ),
    ),
  );
  await tester.tap(find.text('Open'));
  await tester.pumpAndSettle();
}

final class _EditRegistryStorage implements SessionRegistryStorage {
  new(this.registry);
  SessionRegistry registry;
  @override
  Future<SessionRegistry> read() async => registry;
  @override
  Future<void> write(SessionRegistry registry) async =>
      this.registry = registry;
}

void main() {
  group('EditProfileDialog', () {
    testWidgets(
      'reports an unknown photo failure once and retains recovery UI',
      (tester) async {
        final reporter = RecordingErrorReporter();
        final error = StateError('PRIVATE_IMAGE_ERROR');
        await _pumpEditDialog(
          tester,
          repo: FakeProfileRepository(onFetchMe: () async => _seedProfile),
          imageException: error,
          reporter: reporter,
        );
        await tester.tap(find.byTooltip('Change avatar'));
        await tester.pumpAndSettle();
        final choosePhotos = find.byKey(
          const Key('profile-avatar-choose-photos'),
        );
        if (choosePhotos.evaluate().isNotEmpty) {
          await tester.tap(choosePhotos);
          await tester.pumpAndSettle();
        }
        expect(find.text("Couldn't upload that photo."), findsOneWidget);
        expect(reporter.errors, [same(error)]);
        expect(
          reporter
              .contexts
              .single
              .workflow!
              .selectedFields['operationAccountDid'],
          'did:plc:test',
        );
      },
    );

    for (final failure in [
      ProfileImageFailure.tooManyPixels,
      ProfileImageFailure.fileTooLarge,
    ]) {
      testWidgets('shows actionable photo error for ${failure.name}', (
        tester,
      ) async {
        await _pumpEditDialog(
          tester,
          repo: FakeProfileRepository(onFetchMe: () async => _seedProfile),
          imageFailure: failure,
        );
        await tester.tap(find.byTooltip('Change avatar'));
        await tester.pumpAndSettle();
        final choosePhotos = find.byKey(
          const Key('profile-avatar-choose-photos'),
        );
        if (choosePhotos.evaluate().isNotEmpty) {
          await tester.tap(choosePhotos);
          await tester.pumpAndSettle();
        }
        expect(
          find.text(
            failure == ProfileImageFailure.tooManyPixels
                ? 'That photo is over 48 megapixels. Choose a smaller photo or '
                      'reduce its resolution and try again.'
                : 'That photo is over 100 MB. '
                      'Choose a smaller file and try again.',
          ),
          findsOneWidget,
        );
      });
    }

    testWidgets('avatar action offers camera and gallery sources', (
      tester,
    ) async {
      await _pumpEditDialog(
        tester,
        repo: FakeProfileRepository(onFetch: (_) async => _seedProfile),
      );

      await tester.tap(find.byTooltip('Change avatar'));
      await tester.pumpAndSettle();

      expect(
        find.byKey(const Key('profile-avatar-take-photo')),
        findsOneWidget,
      );
      expect(
        find.byKey(const Key('profile-avatar-choose-photos')),
        findsOneWidget,
      );
    });

    for (final constraint in businessAccessibilityMatrix) {
      testWidgets(
        'R12 AT-012 REG-010 business editor fits and remains reachable '
        '${businessConstraintLabel(constraint)}',
        (tester) async {
          await setBusinessAccessibilityConstraint(tester, constraint);
          final profile = _seedProfile.copyWith(
            accountType: AccountType.business,
            business: _businessDeclaration,
          );
          await _pumpEditDialog(
            tester,
            repo: FakeProfileRepository(onFetch: (_) async => profile),
            businessRepository: _RecordingBusinessRepository(),
          );

          final reachableControls = <Finder>[
            find.text('Business types'),
            find.text('Offerings'),
            find.widgetWithText(TextField, 'Thoughtful classes'),
            find.widgetWithText(TextField, 'Weekdays'),
            find.widgetWithText(TextField, 'South West'),
            find.widgetWithText(TextField, 'GB'),
            find.widgetWithText(TextField, 'Bristol'),
            find.text('Primary action'),
            find.widgetWithText(TextField, 'https://example.com/classes'),
          ];
          for (final control in reachableControls) {
            expect(control, findsWidgets);
            await tester.scrollUntilVisible(
              control.first,
              120,
              scrollable: find.byType(Scrollable).last,
            );
            expectNoAccessibilityLayoutException(tester);
          }

          await tester.enterText(
            find.widgetWithText(TextField, 'Thoughtful classes'),
            'Updated classes',
          );
          await tester.pump();
          final save = find.widgetWithText(TextButton, 'Save');
          expect(tester.widget<TextButton>(save).onPressed, isNotNull);
          expect(save.hitTestable(), findsOneWidget);

          await tester.tap(find.byType(CloseButton));
          await tester.pumpAndSettle();
          expect(find.text('Discard changes?'), findsOneWidget);
          await tester.scrollUntilVisible(
            find.text('Discard'),
            80,
            scrollable: find.descendant(
              of: find.byType(CraftskyDialog),
              matching: find.byType(Scrollable),
            ),
          );
          expect(find.text('Discard').hitTestable(), findsOneWidget);
          expectNoAccessibilityLayoutException(tester);
          await tester.tap(find.text('Discard'));
          await tester.pumpAndSettle();
          expect(find.text('Open'), findsOneWidget);
        },
      );
    }

    testWidgets(
      'R12 AT-012 business fields follow order and Save activates by keyboard',
      (tester) async {
        final profile = _seedProfile.copyWith(
          accountType: AccountType.business,
          business: _businessDeclaration,
        );
        final businessRepository = _RecordingBusinessRepository();
        await _pumpEditDialog(
          tester,
          repo: FakeProfileRepository(onFetch: (_) async => profile),
          businessRepository: businessRepository,
        );

        final tagline = find.widgetWithText(TextField, 'Thoughtful classes');
        final hours = find.widgetWithText(TextField, 'Weekdays');
        await tester.ensureVisible(tagline);
        tester.widget<TextField>(tagline).focusNode!.requestFocus();
        await tester.pump();
        expect(tester.widget<TextField>(tagline).focusNode!.hasFocus, isTrue);
        await tester.sendKeyEvent(LogicalKeyboardKey.tab);
        await tester.pump();
        expect(tester.widget<TextField>(hours).focusNode!.hasFocus, isTrue);

        await tester.enterText(tagline, 'Updated classes');
        await tester.pump();
        final save = find.widgetWithText(TextButton, 'Save');
        expect(tester.widget<TextButton>(save).onPressed, isNotNull);
        final saveLabel = find.text('Save');
        requestKeyboardFocus(tester, saveLabel);
        await tester.pump();
        expectKeyboardFocusOn(saveLabel);
        await tester.sendKeyEvent(LogicalKeyboardKey.space);
        await tester.pumpAndSettle();

        expect(businessRepository.putCalls, 1);
        expect(find.text('Open'), findsOneWidget);
      },
    );

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

    testWidgets('regular profile keeps the ordinary editor unchanged', (
      tester,
    ) async {
      final repo = FakeProfileRepository(onFetch: (_) async => _seedProfile);
      await _pumpEditDialog(tester, repo: repo);

      expect(find.text('Business details'), findsNothing);
      expect(find.text('Primary action'), findsNothing);
    });

    testWidgets('business profile shows every prefilled detail field', (
      tester,
    ) async {
      final businessProfile = _seedProfile.copyWith(
        accountType: AccountType.business,
        business: _businessDeclaration,
      );
      final repo = FakeProfileRepository(onFetch: (_) async => businessProfile);
      await _pumpEditDialog(tester, repo: repo, businessAccess: true);

      expect(find.text('Business details'), findsOneWidget);
      expect(find.text('Teacher'), findsWidgets);
      expect(find.text('Classes'), findsWidgets);
      expect(
        find.widgetWithText(TextField, 'Thoughtful classes'),
        findsOneWidget,
      );
      expect(find.widgetWithText(TextField, 'Weekdays'), findsOneWidget);
      expect(find.widgetWithText(TextField, 'South West'), findsOneWidget);
      expect(find.widgetWithText(TextField, 'GB'), findsOneWidget);
      expect(find.widgetWithText(TextField, 'Bristol'), findsOneWidget);
      expect(find.text('Book class'), findsOneWidget);
      expect(
        find.widgetWithText(TextField, 'https://example.com/classes'),
        findsOneWidget,
      );
      expect(
        tester
            .widget<TextButton>(find.widgetWithText(TextButton, 'Save'))
            .onPressed,
        isNull,
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
      String? capturedPronouns;
      String? capturedDescription;
      List<String>? capturedCrafts;
      var updateCallCount = 0;

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
              updateCallCount++;
              capturedDisplayName = displayName;
              capturedPronouns = pronouns;
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
      expect(capturedPronouns, 'they/them');
      expect(capturedDescription, 'Sewist in Bristol');
      expect(capturedCrafts, ['sewing', 'quilting']);
    });

    testWidgets('clearing pronouns sends null with the full profile snapshot', (
      tester,
    ) async {
      String? capturedPronouns = 'not-called';
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
              capturedPronouns = pronouns;
              capturedDescription = description;
              capturedCrafts = crafts;
              return _seedProfile.copyWith(pronouns: pronouns);
            },
      );
      await _pumpEditDialog(tester, repo: repo);

      await tester.enterText(
        find.widgetWithText(TextField, 'they/them'),
        '   ',
      );
      await tester.pump();
      await tester.tap(find.widgetWithText(TextButton, 'Save'));
      await tester.pumpAndSettle();

      expect(capturedPronouns, isNull);
      expect(capturedDescription, 'Sewist in Bristol');
      expect(capturedCrafts, ['sewing', 'quilting']);
    });

    testWidgets('accepts pronouns at the 20-grapheme boundary', (tester) async {
      final pronouns = List.filled(20, 'e\u0301').join();
      final profile = _seedProfile.copyWith(pronouns: pronouns);
      var updateCallCount = 0;
      final repo = FakeProfileRepository(
        onFetch: (_) async => profile,
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
              updateCallCount++;
              return profile.copyWith(
                displayName: displayName,
                pronouns: pronouns,
              );
            },
      );
      await _pumpEditDialog(tester, repo: repo);

      expect(find.text('20/20'), findsOneWidget);
      await tester.enterText(
        find.widgetWithText(TextField, 'Test User'),
        'Renamed',
      );
      await tester.pump();
      await tester.tap(find.widgetWithText(TextButton, 'Save'));
      await tester.pumpAndSettle();

      expect(updateCallCount, 1);
    });

    testWidgets('save invisibly preserves legacy and unknown crafts', (
      tester,
    ) async {
      final profile = _seedProfile.copyWith(
        crafts: ['sewing', 'weaving', 'future-craft'],
      );
      List<String>? capturedCrafts;
      final repo = FakeProfileRepository(
        onFetch: (_) async => profile,
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
              capturedCrafts = crafts;
              return profile.copyWith(
                displayName: displayName,
                pronouns: pronouns,
                crafts: crafts,
              );
            },
      );

      await _pumpEditDialog(tester, repo: repo);
      expect(find.text('Weaving'), findsNothing);
      expect(find.text('future-craft'), findsNothing);

      await tester.enterText(
        find.widgetWithText(TextField, 'Test User'),
        'Renamed',
      );
      await tester.pump();
      await tester.tap(find.widgetWithText(TextButton, 'Save'));
      await tester.pumpAndSettle();

      expect(capturedCrafts, ['sewing', 'weaving', 'future-craft']);
    });

    testWidgets(
      'Request more uses the safe support-link flow without changing state',
      (tester) async {
        var confirmationCalls = 0;
        Uri? launchedUri;
        await _pumpEditDialog(
          tester,
          repo: FakeProfileRepository(onFetch: (_) async => _seedProfile),
          confirmOpenLink: (context, uri) async {
            confirmationCalls++;
            return false;
          },
          linkLauncher: (uri) async {
            launchedUri = uri;
            return false;
          },
        );

        final requestMore = find.text('Request more');
        await tester.ensureVisible(requestMore);
        await tester.tap(requestMore);
        await tester.pumpAndSettle();

        const expected =
            'https://userinput.app/s/did:plc:lmmx63zcns6gewgxqfdt4kof/'
            '3mpr5izppvt2k?lang=en';
        expect(confirmationCalls, 0);
        expect(launchedUri.toString(), expected);
        expect(find.text("Couldn't open that link."), findsOneWidget);
        expect(find.text('Edit profile'), findsOneWidget);
        expect(
          tester
              .widget<TextButton>(find.widgetWithText(TextButton, 'Save'))
              .onPressed,
          isNull,
        );
      },
    );

    testWidgets(
      'crafts-only save preserves the profile and closes only the editor',
      (tester) async {
        final profile = _seedProfile.copyWith(
          followerCount: 7,
          avatar: 'https://example.com/cleared-avatar.jpg',
          banner: 'https://example.com/cleared-banner.jpg',
          customisation: const ProfileCustomisation(colour: 'teal'),
        );
        final repo = FakeProfileRepository(
          onFetch: (_) async => profile,
          onUpdateMe: ({
            displayName,
            pronouns,
            description,
            crafts,
            avatar,
            clearAvatar = false,
            banner,
            clearBanner = false,
          }) async => _seedProfile.copyWith(crafts: crafts),
        );
        await _pumpEditDialog(tester, repo: repo);
        final container = ProviderScope.containerOf(
          tester.element(find.byType(EditProfileDialog)),
        );
        final subscription = container.listen(
          userProfileProvider(profile.did),
          (_, _) {},
        );
        addTearDown(subscription.close);
        final knitting = find.byKey(const ValueKey('craft-choice-knitting'));
        await tester.ensureVisible(knitting);
        await tester.pumpAndSettle();
        await tester.tap(knitting);
        await tester.pump();
        await tester.tap(find.widgetWithText(TextButton, 'Save'));
        await tester.pumpAndSettle();
        expect(find.byType(EditProfileDialog), findsNothing);
        expect(find.text('Open'), findsOneWidget);
        final saved = subscription.read().requireValue;
        expect(saved.crafts, containsAll(['sewing', 'quilting', 'knitting']));
        expect(saved.customisation, profile.customisation);
        expect(saved.followerCount, 7);
        expect(saved.avatar, profile.avatar);
        expect(saved.banner, profile.banner);
        expect(saved.did, profile.did);
        await tester.tap(find.text('Open'));
        await tester.pumpAndSettle();
        expect(
          tester
              .widget<Checkbox>(
                find.descendant(
                  of: find.byKey(const ValueKey('craft-choice-knitting')),
                  matching: find.byType(Checkbox),
                ),
              )
              .value,
          isTrue,
        );
      },
    );

    testWidgets('successful save pops back to the previous route', (
      tester,
    ) async {
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
            }) async => _seedProfile.copyWith(
              displayName: displayName ?? 'Test User',
              pronouns: pronouns,
            ),
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

    testWidgets('successful save publishes the active account overlay', (
      tester,
    ) async {
      var fetchCallCount = 0;
      final repo = FakeProfileRepository(
        onFetch: (_) async {
          fetchCallCount++;
          return _seedProfile;
        },
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
            }) async => _seedProfile.copyWith(
              displayName: displayName,
              pronouns: pronouns,
            ),
      );
      await _pumpEditDialog(tester, repo: repo);

      // Keep a listener on the family entry so it stays alive past
      // the edit page's pop and we can observe its cached value.
      final container = ProviderScope.containerOf(
        tester.element(find.byType(EditProfileDialog)),
      );
      final sub = container.listen<AsyncValue<Profile>>(
        userProfileProvider(Did.parse('did:plc:test')),
        (_, _) {},
      );
      addTearDown(sub.close);

      // The active-account identity and editor may each request the profile.
      final baselineFetches = fetchCallCount;
      expect(baselineFetches, greaterThanOrEqualTo(1));

      await tester.enterText(
        find.widgetWithText(TextField, 'Test User'),
        'Renamed',
      );
      await tester.pump();
      await tester.tap(find.widgetWithText(TextButton, 'Save'));
      await tester.pumpAndSettle();

      // A leased account refreshes the profile and applies the shared overlay.
      expect(sub.read().value?.displayName, 'Renamed');
      expect(fetchCallCount, baselineFetches + 1);
    });

    testWidgets('failed save surfaces an error snackbar and stays on page', (
      tester,
    ) async {
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

    testWidgets('partial save retries only the failed ordinary record', (
      tester,
    ) async {
      var ordinaryCalls = 0;
      final profile = _seedProfile.copyWith(
        accountType: AccountType.business,
        business: _businessDeclaration,
      );
      final repo = FakeProfileRepository(
        onFetch: (_) async => profile,
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
              ordinaryCalls++;
              if (ordinaryCalls == 1) throw StateError('ordinary failed');
              return profile.copyWith(
                displayName: displayName,
                pronouns: pronouns,
              );
            },
      );
      final businessRepository = _RecordingBusinessRepository();
      await _pumpEditDialog(
        tester,
        repo: repo,
        businessRepository: businessRepository,
      );

      await tester.enterText(
        find.widgetWithText(TextField, 'Test User'),
        'Renamed',
      );
      await tester.enterText(
        find.widgetWithText(TextField, 'Thoughtful classes'),
        'Updated classes',
      );
      await tester.pump();
      await tester.tap(find.widgetWithText(TextButton, 'Save'));
      await tester.pumpAndSettle();

      expect(find.text("Couldn't save your profile."), findsOneWidget);
      expect(find.text('Open'), findsNothing);
      expect(ordinaryCalls, 1);
      expect(businessRepository.putCalls, 1);

      await tester.tap(find.widgetWithText(TextButton, 'Save'));
      await tester.pumpAndSettle();

      expect(find.text('Open'), findsOneWidget);
      expect(ordinaryCalls, 2);
      expect(businessRepository.putCalls, 1);
    });

    testWidgets(
      'business conflict refresh keeps edits and adopts untouched newer fields',
      (tester) async {
        var latest = _seedProfile.copyWith(
          accountType: AccountType.business,
          business: _businessDeclaration,
        );
        final repo = FakeProfileRepository(
          onFetch: (_) async => latest,
          onFetchMe: () async => latest,
        );
        final business = _RecordingBusinessRepository(
          firstError: const ApiBadRequest('pds_record_conflict'),
        );
        await _pumpEditDialog(tester, repo: repo, businessRepository: business);
        await tester.enterText(
          find.widgetWithText(TextField, 'Thoughtful classes'),
          'My edited tagline',
        );
        await tester.pump();
        latest = latest.copyWith(
          business: _businessDeclaration.copyWith(
            cid: Cid.parse(
              'bafyreifbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
            ),
            hoursNote: 'New hours elsewhere',
          ),
        );
        await tester.tap(find.widgetWithText(TextButton, 'Save'));
        await tester.pumpAndSettle();
        expect(
          find.widgetWithText(TextField, 'My edited tagline'),
          findsOneWidget,
        );
        expect(
          find.widgetWithText(TextField, 'New hours elsewhere'),
          findsOneWidget,
        );
        expect(
          find.text(
            'Your business details changed elsewhere. '
            'Your edits have been kept; review the updated details '
            'and save again.',
          ),
          findsOneWidget,
        );
        expect(business.putCalls, 1);
        await tester.tap(find.widgetWithText(TextButton, 'Save'));
        await tester.pumpAndSettle();
        expect(business.expectedCids, [
          _businessDeclaration.cid,
          latest.business!.cid,
        ]);
        expect(business.bodies.last['tagline'], 'My edited tagline');
        expect(business.bodies.last['hoursNote'], 'New hours elsewhere');
        expect(business.operationKeys.toSet(), hasLength(2));
        expect(find.text('Open'), findsOneWidget);
      },
    );

    testWidgets('business validation errors identify the field', (
      tester,
    ) async {
      final profile = _seedProfile.copyWith(
        accountType: AccountType.business,
        business: _businessDeclaration,
      );
      final repo = FakeProfileRepository(onFetch: (_) async => profile);
      final business = _RecordingBusinessRepository(
        firstError: const ApiBadRequest(
          'validation_failed',
          details: ApiFailureDetails(fields: {'tagline': 'is invalid'}),
        ),
      );
      await _pumpEditDialog(tester, repo: repo, businessRepository: business);
      await tester.enterText(
        find.widgetWithText(TextField, 'Thoughtful classes'),
        'Edited tagline',
      );
      await tester.pump();
      await tester.tap(find.widgetWithText(TextButton, 'Save'));
      await tester.pumpAndSettle();
      expect(find.text('Tagline: is invalid'), findsOneWidget);
      expect(find.widgetWithText(TextField, 'Edited tagline'), findsOneWidget);
      expect(find.text('Open'), findsNothing);
    });

    testWidgets('inverse partial save retries only the business record', (
      tester,
    ) async {
      var ordinaryCalls = 0;
      final profile = _seedProfile.copyWith(
        accountType: AccountType.business,
        business: _businessDeclaration,
      );
      final repo = FakeProfileRepository(
        onFetch: (_) async => profile,
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
              ordinaryCalls++;
              return profile.copyWith(
                displayName: displayName,
                pronouns: pronouns,
              );
            },
      );
      final businessRepository = _RecordingBusinessRepository(failFirst: true);
      await _pumpEditDialog(
        tester,
        repo: repo,
        businessRepository: businessRepository,
      );

      await tester.enterText(
        find.widgetWithText(TextField, 'Test User'),
        'Renamed',
      );
      await tester.enterText(
        find.widgetWithText(TextField, 'Thoughtful classes'),
        'Updated classes',
      );
      await tester.pump();
      await tester.tap(find.widgetWithText(TextButton, 'Save'));
      await tester.pumpAndSettle();

      expect(
        find.text("Couldn't save your business details."),
        findsOneWidget,
      );
      expect(find.text('Open'), findsNothing);
      expect(ordinaryCalls, 1);
      expect(businessRepository.putCalls, 1);

      await tester.tap(find.widgetWithText(TextButton, 'Save'));
      await tester.pumpAndSettle();

      expect(find.text('Open'), findsOneWidget);
      expect(ordinaryCalls, 1);
      expect(businessRepository.putCalls, 2);
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
      'correcting an existing over-limit name keeps the active field focused',
      (tester) async {
        final profile = _seedProfile.copyWith(displayName: 'x' * 65);
        final repo = FakeProfileRepository(onFetch: (_) async => profile);
        await _pumpEditDialog(tester, repo: repo);

        // Existing values are preserved rather than silently truncated. Editing
        // another field triggers validation of the over-limit display name.
        final bio = find.widgetWithText(TextField, 'Sewist in Bristol');
        final bioFocusNode = tester.widget<TextField>(bio).focusNode!;
        await tester.enterText(bio, 'Updated bio');
        await tester.pump();
        expect(
          find.text('Display name must be 64 characters or fewer'),
          findsOneWidget,
        );
        expect(bioFocusNode.hasFocus, isTrue);

        final name = find.widgetWithText(TextField, 'x' * 65);
        await tester.showKeyboard(name);
        await tester.pump();
        final focusNode = tester.widget<TextField>(name).focusNode!;
        expect(focusNode.hasFocus, isTrue);

        // Correcting the invalid value must not transfer focus to a sibling
        // FormBuilder focus node as the validation error disappears.
        await tester.enterText(name, 'x' * 64);
        await tester.pump();
        expect(
          find.text('Display name must be 64 characters or fewer'),
          findsNothing,
        );
        expect(focusNode.hasFocus, isTrue);
        expect(tester.testTextInput.isVisible, isTrue);
      },
    );

    testWidgets('display name input caps pasted text at 64 graphemes', (
      tester,
    ) async {
      final repo = FakeProfileRepository(onFetch: (_) async => _seedProfile);
      await _pumpEditDialog(tester, repo: repo);

      final name = find.widgetWithText(TextField, 'Test User');
      final controller = tester.widget<TextField>(name).controller!;
      await tester.enterText(name, '🧶' * 65);
      await tester.pump();

      expect(controller.text, '🧶' * 64);
      expect(find.text('64/64'), findsOneWidget);
      expect(
        find.text('Display name must be 64 characters or fewer'),
        findsNothing,
      );
      expect(
        tester
            .widget<TextButton>(find.widgetWithText(TextButton, 'Save'))
            .onPressed,
        isNotNull,
      );
    });

    testWidgets(
      'business text input keeps keyboard focus while another field is invalid',
      (tester) async {
        final profile = _seedProfile.copyWith(
          accountType: AccountType.business,
          business: _businessDeclaration.copyWith(
            location: const BusinessLocation(
              country: 'XX',
              locality: 'Bristol',
            ),
          ),
        );
        await _pumpEditDialog(
          tester,
          repo: FakeProfileRepository(onFetch: (_) async => profile),
          businessAccess: true,
        );

        final tagline = find.widgetWithText(TextField, 'Thoughtful classes');
        await tester.ensureVisible(tagline);
        await tester.tap(tagline);
        await tester.pumpAndSettle();
        final focusNode = tester.widget<TextField>(tagline).focusNode!;
        expect(focusNode.hasFocus, isTrue);
        final focusedField = find.byWidgetPredicate(
          (widget) => widget is TextField && widget.focusNode == focusNode,
        );

        for (final text in ['Thoughtful classes!', 'Thoughtful classes!!']) {
          await tester.enterText(focusedField, text);
          await tester.pump();
          expect(focusNode.hasFocus, isTrue);
          expect(tester.testTextInput.isVisible, isTrue);
        }
      },
    );

    testWidgets(
      'existing display name longer than 64 characters surfaces a validator '
      'error without disabling dirty save',
      (tester) async {
        final profile = _seedProfile.copyWith(displayName: 'x' * 65);
        final repo = FakeProfileRepository(onFetch: (_) async => profile);
        await _pumpEditDialog(tester, repo: repo);

        await tester.enterText(
          find.widgetWithText(TextField, 'Sewist in Bristol'),
          'Updated bio',
        );
        await tester.pump();

        // The validator's errorText (sourced from the ARB) renders
        // beneath the field once autovalidate kicks in.
        expect(
          find.text('Display name must be 64 characters or fewer'),
          findsOneWidget,
        );

        // Save remains available so pressing it can run validation and direct
        // the user to any invalid fields.
        final saveButton = tester.widget<TextButton>(
          find.widgetWithText(TextButton, 'Save'),
        );
        expect(saveButton.onPressed, isNotNull);

        await tester.tap(find.widgetWithText(TextButton, 'Save'));
        await tester.pump();
        expect(
          find.text('Display name must be 64 characters or fewer'),
          findsOneWidget,
        );
        expect(repo.updateOperationKeys, isEmpty);
      },
    );

    testWidgets('reordering alone saves and survives reopening', (
      tester,
    ) async {
      List<String>? savedCrafts;
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
              savedCrafts = crafts;
              return _seedProfile.copyWith(crafts: crafts);
            },
      );
      await _pumpEditDialog(tester, repo: repo);
      final reorder = find.byTooltip('Reorder Quilting');
      await tester.ensureVisible(reorder);
      await tester.tap(reorder);
      await tester.pumpAndSettle();
      await tester.tap(find.text('Move up'));
      await tester.pumpAndSettle();
      await tester.tap(find.widgetWithText(TextButton, 'Save'));
      await tester.pumpAndSettle();
      expect(savedCrafts, ['quilting', 'sewing']);
      await tester.tap(find.text('Open'));
      await tester.pumpAndSettle();
      expect(
        find.descendant(
          of: find.byKey(const ValueKey('craft-choice-quilting')),
          matching: find.text('Default'),
        ),
        findsOneWidget,
      );
    });

    testWidgets('tapping a craft row toggles its selection state', (
      tester,
    ) async {
      final repo = FakeProfileRepository(onFetch: (_) async => _seedProfile);
      await _pumpEditDialog(tester, repo: repo);

      // 'Knitting' starts unselected (seed has only sewing + quilting).
      final knittingFinder = find.byKey(
        const ValueKey('craft-choice-knitting'),
      );
      expect(knittingFinder, findsOneWidget);

      // The crafts grid is below the fold in the default 800x600 test
      // viewport — scroll it into view before tapping.
      await tester.ensureVisible(knittingFinder);
      await tester.pumpAndSettle();

      Checkbox knitting() => tester.widget<Checkbox>(
        find.descendant(of: knittingFinder, matching: find.byType(Checkbox)),
      );
      expect(knitting().value, isFalse);

      await tester.tap(knittingFinder);
      await tester.pump();
      expect(knitting().value, isTrue);

      await tester.tap(knittingFinder);
      await tester.pump();
      expect(knitting().value, isFalse);
    });
  });
}

final _businessDeclaration = BusinessProfile(
  cid: 'bafyreiaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
  businessTypes: const [BusinessOpenValue(value: 'teacher', known: true)],
  offerings: const [BusinessOpenValue(value: 'classes', known: true)],
  tagline: 'Thoughtful classes',
  hoursNote: 'Weekdays',
  serviceArea: 'South West',
  location: const BusinessLocation(country: 'GB', locality: 'Bristol'),
  primaryAction: const BusinessAction(
    type: 'book-class',
    destination: 'https://example.com/classes',
  ),
  products: const [
    BusinessProductView(title: 'Pattern', uri: 'https://example.com/pattern'),
  ],
);

final class _RecordingBusinessRepository extends Fake
    implements BusinessRepository {
  new({this.failFirst = false, this.firstError});

  final bool failFirst;
  final Exception? firstError;
  final expectedCids = <Cid?>[];
  final bodies = <Map<String, dynamic>>[];
  final operationKeys = <String>[];
  int putCalls = 0;

  @override
  Future<RecordMutationResult> putBusinessProfile(
    Map<String, dynamic> body, {
    required String operationKey,
    required Cid? expectedCid,
  }) async {
    putCalls++;
    expectedCids.add(expectedCid);
    bodies.add(body);
    operationKeys.add(operationKey);
    if (putCalls == 1 && firstError != null) throw firstError!;
    if (failFirst && putCalls == 1) throw Exception('business failed');
    return RecordMutationResult(
      cid: 'bafyreifbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
    );
  }
}

class _FailingProfileImagePicker extends ProfileImagePicker {
  new(this.failure)
    : super(
        picker: ImagePicker(),
        media: const ComposerImageMediaService(),
        blobApi: BlobApiClient(Dio()),
      );
  final Object failure;

  @override
  Future<ProfileImagePickResult?> pickAndUpload({
    required ImageSource source,
    required void Function(Uint8List bytes) onPreviewReady,
  }) async => Error.throwWithStackTrace(failure, StackTrace.current);
}
