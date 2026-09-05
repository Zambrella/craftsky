import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart' as registry;
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/instagram_migration/data/instagram_migration_repository.dart';
import 'package:craftsky_app/instagram_migration/data/instagram_verification_storage.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_account.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_suggestion.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_verification.dart';
import 'package:craftsky_app/instagram_migration/pages/instagram_migration_page.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_migration_repository_provider.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_export_file_picker.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_import_parser.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/recording_messenger.dart';

void main() {
  testWidgets(
    'IT-016 import controls stay hidden until verification',
    (tester) async {
      final initial = registry.SessionRegistry.empty().upsertAndActivate(
        token: 'token-a',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final repository = _Repository(
        imports: InstagramImportPage(items: const [], cursor: null),
      );
      final semantics = tester.ensureSemantics();

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _RegistryStorage(initial),
            ),
            instagramMigrationRepositoryProvider.overrideWith(
              (ref, _) async => repository,
            ),
            instagramVerificationStorageProvider.overrideWithValue(
              _EmptyVerificationStorage(),
            ),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const InstagramMigrationPage(),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Find people from Instagram'), findsOneWidget);
      expect(find.byType(Card), findsNothing);
      expect(find.byType(CraftskyCard), findsWidgets);
      expect(
        find.text('Instagram verification is unavailable right now.'),
        findsOneWidget,
      );
      expect(
        find.text(
          'Imports become available after Instagram verification is '
          'configured and your account is verified.',
        ),
        findsOneWidget,
      );
      expect(
        find.text('Complete verification to import the accounts you follow.'),
        findsOneWidget,
      );
      expect(
        find.byKey(const Key('instagram-import-composer-card')),
        findsNothing,
      );
      expect(find.byKey(const Key('instagram-imports-card')), findsNothing);
      expect(find.byKey(const Key('instagram-suggestions-card')), findsNothing);
      expect(
        tester.widget<ListView>(find.byType(ListView)).physics,
        isA<AlwaysScrollableScrollPhysics>(),
      );
      semantics.dispose();
    },
  );

  testWidgets('IT-023 manual and Instagram export imports stay normalized', (
    tester,
  ) async {
    final initial = registry.SessionRegistry.empty().upsertAndActivate(
      token: 'token-a',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final sentRequests = <InstagramImportRequest>[];
    final messenger = RecordingMessenger();
    final repository = _Repository(
      status: InstagramAccountStatus(
        integrationAvailable: true,
        account: InstagramAccountLink(
          state: InstagramAccountLinkState.active,
          username: 'alice_instagram',
          discoverable: true,
          conflictPending: false,
          reactivationRequired: false,
          verifiedAt: DateTime.utc(2026, 7, 19),
        ),
      ),
      imports: InstagramImportPage(items: const [], cursor: null),
      onCreateImport: (request) async {
        sentRequests.add(request);
        return InstagramImportCreateResult(
          import: InstagramImportSummary(
            importId: 'import-new',
            state: InstagramImportState.active,
            sourceType: request.sourceType,
            followingCount: request.entries.length,
            createdAt: DateTime.utc(2026, 7, 19),
          ),
          followingCount: request.entries.length,
        );
      },
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(initial),
          ),
          instagramVerificationStorageProvider.overrideWithValue(
            _EmptyVerificationStorage(),
          ),
          instagramMigrationRepositoryProvider.overrideWith(
            (ref, _) async => repository,
          ),
          instagramExportFilePickerProvider.overrideWithValue(
            () async => const InstagramImportParseResult(
              entries: [InstagramImportEntry(username: 'bobmaker')],
            ),
          ),
        ],
        child: MessengerScope(
          messenger: messenger,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const InstagramMigrationPage(),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    final cards = tester.widgetList<CraftskyCard>(
      find.byType(CraftskyCard),
    );
    expect(cards, isNotEmpty);
    expect(cards.every((card) => card.clipBehavior == Clip.none), isTrue);
    expect(find.text('Accounts that follow me'), findsNothing);
    final manualDescription = find.text(
      'Enter the Instagram handles of accounts you follow, one per line.',
    );
    final jsonDescription = find.text(
      'Choose an Instagram export containing Accounts you follow. CraftSky '
      'processes it on this device and uploads only those usernames. If you '
      'select an all-information ZIP, everything else stays on your device.',
    );
    expect(manualDescription, findsNothing);
    expect(jsonDescription, findsOneWidget);
    expect(
      tester.getCenter(find.text('Instagram export')).dx,
      lessThan(tester.getCenter(find.text('Enter handles')).dx),
    );
    final importSelector = tester.widget<SegmentedButton<dynamic>>(
      find.byKey(const Key('instagram-import-kind-selector')),
    );
    final theme = Theme.of(
      tester.element(find.byType(InstagramMigrationPage)),
    );
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final segmentedStyle = theme.segmentedButtonTheme.style!;
    expect(importSelector.style, isNull);
    expect(
      segmentedStyle.backgroundColor?.resolve({
        WidgetState.selected,
      }),
      swatches.moss,
    );
    expect(
      segmentedStyle.foregroundColor?.resolve({
        WidgetState.selected,
      }),
      swatches.onMoss,
    );
    expect(
      segmentedStyle.foregroundColor?.resolve({}),
      theme.colorScheme.onSurface,
    );
    expect(
      segmentedStyle.overlayColor?.resolve({
        WidgetState.pressed,
      }),
      swatches.moss.withValues(alpha: 0.12),
    );
    expect(
      segmentedStyle.overlayColor?.resolve({
        WidgetState.selected,
        WidgetState.pressed,
      }),
      swatches.onMoss.withValues(alpha: 0.12),
    );
    expect(
      tester
          .getBottomLeft(
            find.byKey(const Key('instagram-import-kind-selector')),
          )
          .dy,
      lessThan(tester.getTopLeft(jsonDescription).dy),
    );
    expect(
      tester.getBottomLeft(jsonDescription).dy,
      lessThan(
        tester.getTopLeft(find.text('Select Instagram export')).dy,
      ),
    );
    expect(
      find.widgetWithText(FilledButton, 'Select Instagram export'),
      findsOneWidget,
    );
    await tester.tap(find.text('Enter handles'));
    await tester.pumpAndSettle();
    expect(manualDescription, findsOneWidget);
    expect(jsonDescription, findsNothing);
    expect(
      find.widgetWithText(FilledButton, 'Import handles'),
      findsOneWidget,
    );
    expect(
      find.widgetWithText(OutlinedButton, 'Import handles'),
      findsNothing,
    );
    await tester.enterText(
      find.byType(TextField),
      '@Alice\nALICE\nbad name',
    );
    expect(find.text('Preview normalized handles'), findsNothing);
    await tester.drag(find.byType(ListView), const Offset(0, -300));
    await tester.pumpAndSettle();
    expect(
      find.text(
        'Importing creates private suggestions only. You choose whether to '
        'follow each account.',
      ),
      findsOneWidget,
    );
    final importButton = find.text('Import handles');
    await tester.ensureVisible(importButton);
    await tester.pumpAndSettle();
    await tester.tap(importButton);
    await tester.pumpAndSettle();

    expect(sentRequests.single.entries, hasLength(1));
    expect(sentRequests.single.entries.single.username, 'alice');
    expect(sentRequests.single.sourceType, InstagramImportSourceType.manual);
    expect(messenger.calls, [('info', 'Instagram import created', null)]);

    await tester.ensureVisible(find.text('Instagram export'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Instagram export'));
    await tester.pumpAndSettle();
    expect(manualDescription, findsNothing);
    expect(jsonDescription, findsOneWidget);
    expect(
      tester
          .getBottomLeft(
            find.byKey(const Key('instagram-import-kind-selector')),
          )
          .dy,
      lessThan(tester.getTopLeft(jsonDescription).dy),
    );
    expect(
      tester.getBottomLeft(jsonDescription).dy,
      lessThan(tester.getTopLeft(find.text('Select Instagram export')).dy),
    );
    await tester.ensureVisible(find.text('Select Instagram export'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Select Instagram export'));
    await tester.pumpAndSettle();

    expect(sentRequests, hasLength(2));
    expect(
      sentRequests.last.sourceType,
      InstagramImportSourceType.instagramJson,
    );
    expect(sentRequests.last.entries.single.username, 'bobmaker');
  });

  testWidgets(
    'IT-023 late export result is discarded after switching away and back',
    (tester) async {
      final initial = registry.SessionRegistry.empty()
          .upsertAndActivate(
            token: 'token-b',
            did: 'did:plc:bob',
            handle: 'bob.test',
          )
          .upsertAndActivate(
            token: 'token-a',
            did: 'did:plc:alice',
            handle: 'alice.test',
          );
      final picker = Completer<InstagramImportParseResult?>();
      final sentRequests = <InstagramImportRequest>[];
      final repository = _Repository(
        status: InstagramAccountStatus(
          integrationAvailable: true,
          account: InstagramAccountLink(
            state: InstagramAccountLinkState.active,
            username: 'synthetic_instagram',
            discoverable: true,
            conflictPending: false,
            reactivationRequired: false,
            verifiedAt: DateTime.utc(2026, 7, 23),
          ),
        ),
        imports: InstagramImportPage(items: const [], cursor: null),
        onCreateImport: (request) async {
          sentRequests.add(request);
          return InstagramImportCreateResult(
            import: InstagramImportSummary(
              importId: 'unexpected-import',
              state: InstagramImportState.active,
              sourceType: request.sourceType,
              followingCount: request.entries.length,
              createdAt: DateTime.utc(2026, 7, 23),
            ),
            followingCount: request.entries.length,
          );
        },
      );

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _RegistryStorage(initial),
            ),
            instagramVerificationStorageProvider.overrideWithValue(
              _EmptyVerificationStorage(),
            ),
            instagramMigrationRepositoryProvider.overrideWith(
              (ref, _) async => repository,
            ),
            instagramExportFilePickerProvider.overrideWithValue(
              () => picker.future,
            ),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const InstagramMigrationPage(),
          ),
        ),
      );
      await tester.pumpAndSettle();
      await tester.ensureVisible(find.text('Instagram export'));
      await tester.tap(find.text('Instagram export'));
      await tester.pumpAndSettle();
      await tester.ensureVisible(find.text('Select Instagram export'));
      await tester.tap(find.text('Select Instagram export'));
      await tester.pump();

      final container = ProviderScope.containerOf(
        tester.element(find.byType(InstagramMigrationPage)),
      );
      final bobLease = container
          .read(sessionRegistryProvider)
          .requireValue
          .leaseFor(AccountKey('did:plc:bob'))!;
      await container.read(sessionRegistryProvider.notifier).activate(bobLease);
      await tester.pump();
      final aliceLease = container
          .read(sessionRegistryProvider)
          .requireValue
          .leaseFor(AccountKey('did:plc:alice'))!;
      await container
          .read(sessionRegistryProvider.notifier)
          .activate(aliceLease);
      await tester.pump();
      picker.complete(
        const InstagramImportParseResult(
          entries: [InstagramImportEntry(username: 'private_alice_only')],
        ),
      );
      await tester.pumpAndSettle();

      expect(sentRequests, isEmpty);
    },
  );

  testWidgets('IT-023 cancellation and safe export errors stay local', (
    tester,
  ) async {
    final initial = registry.SessionRegistry.empty().upsertAndActivate(
      token: 'token-a',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    var picker = () async => null as InstagramImportParseResult?;
    final sentRequests = <InstagramImportRequest>[];
    final repository = _verifiedImportRepository(sentRequests);

    await _pumpVerifiedExportPage(
      tester,
      initial: initial,
      repository: repository,
      picker: () => picker(),
    );

    final selectExport = find.widgetWithText(
      FilledButton,
      'Select Instagram export',
    );
    await tester.tap(selectExport);
    await tester.pumpAndSettle();

    expect(sentRequests, isEmpty);
    expect(tester.widget<FilledButton>(selectExport).onPressed, isNotNull);

    const cases = <(InstagramImportParseErrorCode, String)>[
      (
        InstagramImportParseErrorCode.invalidJson,
        'This file is not valid JSON.',
      ),
      (
        InstagramImportParseErrorCode.unsupportedShape,
        'This is not a supported Instagram accounts-followed export. Choose '
            'an export containing Accounts you follow.',
      ),
      (
        InstagramImportParseErrorCode.unsupportedFormat,
        "This Instagram export uses a format CraftSky can't read.",
      ),
      (
        InstagramImportParseErrorCode.invalidArchive,
        'This Instagram ZIP is incomplete or damaged. Download a new export '
            'and try again.',
      ),
      (
        InstagramImportParseErrorCode.archiveTooLarge,
        'This Instagram ZIP contains too many files to process safely.',
      ),
      (
        InstagramImportParseErrorCode.fileTooLarge,
        'The accounts-followed data is larger than 20 MiB.',
      ),
      (
        InstagramImportParseErrorCode.tooManyEntries,
        'This import contains more than 10,000 unique handles.',
      ),
    ];
    for (final (code, message) in cases) {
      picker = () async => throw InstagramImportParseException(code);
      await tester.tap(selectExport);
      await tester.pumpAndSettle();
      expect(find.text(message), findsOneWidget);
    }

    picker = () async => throw StateError('synthetic picker failure');
    await tester.tap(selectExport);
    await tester.pumpAndSettle();
    expect(
      find.text("The Instagram export couldn't be opened on this device."),
      findsOneWidget,
    );
    expect(sentRequests, isEmpty);
  });

  testWidgets('IT-023 late export error is discarded after switch away/back', (
    tester,
  ) async {
    final initial = registry.SessionRegistry.empty()
        .upsertAndActivate(
          token: 'token-b',
          did: 'did:plc:bob',
          handle: 'bob.test',
        )
        .upsertAndActivate(
          token: 'token-a',
          did: 'did:plc:alice',
          handle: 'alice.test',
        );
    final picker = Completer<InstagramImportParseResult?>();
    final sentRequests = <InstagramImportRequest>[];

    await _pumpVerifiedExportPage(
      tester,
      initial: initial,
      repository: _verifiedImportRepository(sentRequests),
      picker: () => picker.future,
    );
    await tester.tap(
      find.widgetWithText(FilledButton, 'Select Instagram export'),
    );
    await tester.pump();

    final container = ProviderScope.containerOf(
      tester.element(find.byType(InstagramMigrationPage)),
    );
    final registryNotifier = container.read(sessionRegistryProvider.notifier);
    final bobLease = container
        .read(sessionRegistryProvider)
        .requireValue
        .leaseFor(AccountKey('did:plc:bob'))!;
    await registryNotifier.activate(bobLease);
    await tester.pump();
    final aliceLease = container
        .read(sessionRegistryProvider)
        .requireValue
        .leaseFor(AccountKey('did:plc:alice'))!;
    await registryNotifier.activate(aliceLease);
    await tester.pump();

    picker.completeError(
      const InstagramImportParseException(
        InstagramImportParseErrorCode.invalidArchive,
      ),
    );
    await tester.pumpAndSettle();

    expect(
      find.text(
        'This Instagram ZIP is incomplete or damaged. Download a new export '
        'and try again.',
      ),
      findsNothing,
    );
    expect(sentRequests, isEmpty);
  });

  testWidgets('IT-023 page disposal discards a late export result', (
    tester,
  ) async {
    final initial = registry.SessionRegistry.empty().upsertAndActivate(
      token: 'token-a',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final picker = Completer<InstagramImportParseResult?>();
    final sentRequests = <InstagramImportRequest>[];

    await _pumpVerifiedExportPage(
      tester,
      initial: initial,
      repository: _verifiedImportRepository(sentRequests),
      picker: () => picker.future,
    );
    await tester.tap(
      find.widgetWithText(FilledButton, 'Select Instagram export'),
    );
    await tester.pump();
    await tester.pumpWidget(const SizedBox.shrink());

    picker.complete(
      const InstagramImportParseResult(
        entries: [InstagramImportEntry(username: 'private_disposed_result')],
      ),
    );
    await tester.pumpAndSettle();

    expect(sentRequests, isEmpty);
  });

  testWidgets('FR-024 challenge can be copied opened and cancelled', (
    tester,
  ) async {
    String? clipboardText;
    tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
      SystemChannels.platform,
      (call) async {
        if (call.method == 'Clipboard.setData') {
          clipboardText =
              (call.arguments as Map<Object?, Object?>)['text'] as String?;
        }
        return null;
      },
    );
    final initial = registry.SessionRegistry.empty().upsertAndActivate(
      token: 'token-a',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    var cancelCalls = 0;
    Uri? opened;
    final messenger = RecordingMessenger();
    final repository = _Repository(
      status: const InstagramAccountStatus(
        integrationAvailable: true,
        account: null,
      ),
      imports: InstagramImportPage(items: const [], cursor: null),
      onCreateVerification: () async => InstagramVerificationAttempt(
        verificationId: 'verification-a',
        state: InstagramVerificationState.pendingDm,
        expiresAt: DateTime.now().toUtc().add(const Duration(minutes: 10)),
        challenge: 'CRAFT-TEST-123',
        dmUrl: Uri.parse('https://instagram.example/dm'),
      ),
      onCancelVerification: (_) async => cancelCalls++,
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(initial),
          ),
          instagramVerificationStorageProvider.overrideWithValue(
            _EmptyVerificationStorage(),
          ),
          instagramMigrationRepositoryProvider.overrideWith(
            (ref, _) async => repository,
          ),
          instagramDmLauncherProvider.overrideWithValue((uri) async {
            opened = uri;
            return true;
          }),
        ],
        child: MessengerScope(
          messenger: messenger,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const InstagramMigrationPage(),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Create verification challenge'));
    await tester.pump();
    await tester.pump();
    expect(find.text('CRAFT-TEST-123'), findsOneWidget);

    final copyButton = find.widgetWithText(
      OutlinedButton,
      'Copy challenge',
    );
    final openButton = find.widgetWithText(
      FilledButton,
      'Open Instagram DM',
    );
    final cancelButton = find.widgetWithText(
      TextButton,
      'Cancel verification',
    );
    final copyRect = tester.getRect(copyButton);
    final openRect = tester.getRect(openButton);
    final cancelRect = tester.getRect(cancelButton);
    expect(openRect.width, copyRect.width);
    expect(cancelRect.width, copyRect.width);
    expect(copyRect.top, lessThan(openRect.top));
    expect(openRect.top, lessThan(cancelRect.top));

    await tester.tap(find.text('Copy challenge'));
    await tester.pump();
    expect(clipboardText, 'CRAFT-TEST-123');
    expect(messenger.calls, [('info', 'Challenge copied', null)]);
    await tester.tap(find.text('Open Instagram DM'));
    await tester.pump();
    expect(opened, Uri.parse('https://instagram.example/dm'));

    await tester.tap(find.text('Cancel verification'));
    await tester.pump();
    await tester.pump();
    expect(cancelCalls, 1);
    expect(
      find.text('This verification challenge is no longer active.'),
      findsOneWidget,
    );
    tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
      SystemChannels.platform,
      null,
    );
  });

  testWidgets('FR-024 candidate defaults to discovery and explains choices', (
    tester,
  ) async {
    final initial = registry.SessionRegistry.empty().upsertAndActivate(
      token: 'token-a',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    bool? confirmedDiscoverable;
    var cancelCalls = 0;
    var createCalls = 0;
    final repository = _Repository(
      status: const InstagramAccountStatus(
        integrationAvailable: true,
        account: null,
      ),
      imports: InstagramImportPage(items: const [], cursor: null),
      onCreateVerification: () async {
        createCalls++;
        return InstagramVerificationAttempt(
          verificationId: 'verification-$createCalls',
          state: InstagramVerificationState.pendingConfirmation,
          expiresAt: DateTime.now().toUtc().add(const Duration(minutes: 10)),
          candidateUsername: 'actual_maker',
        );
      },
      onCancelVerification: (_) async => cancelCalls++,
      onConfirmVerification: (_, {required discoverable}) async {
        confirmedDiscoverable = discoverable;
        return InstagramVerificationConfirmation(
          state: InstagramVerificationState.confirmed,
          account: InstagramAccountLink(
            state: InstagramAccountLinkState.active,
            username: 'actual_maker',
            discoverable: discoverable,
            conflictPending: false,
            reactivationRequired: false,
            verifiedAt: DateTime.utc(2026, 7, 19),
          ),
        );
      },
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(initial),
          ),
          instagramVerificationStorageProvider.overrideWithValue(
            _EmptyVerificationStorage(),
          ),
          instagramMigrationRepositoryProvider.overrideWith(
            (ref, _) async => repository,
          ),
        ],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const InstagramMigrationPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('Create verification challenge'));
    await tester.pump();
    await tester.pump();

    final accountFinder = find.byWidgetPredicate(
      (widget) =>
          widget is RichText &&
          widget.text.toPlainText() == 'Account: @actual_maker',
    );
    expect(accountFinder, findsOneWidget);
    final accountText = tester.widget<RichText>(accountFinder);
    final accountSpan = accountText.text as TextSpan;
    final handleSpan = _textSpans(
      accountSpan,
    ).singleWhere((span) => span.text == '@actual_maker');
    expect(handleSpan.text, '@actual_maker');
    expect(handleSpan.style?.fontWeight, FontWeight.bold);
    expect(find.text('Allow discovery'), findsOneWidget);
    var selector = tester.widget<SegmentedButton<bool>>(
      find.byType(SegmentedButton<bool>),
    );
    expect(selector.selected, {true});
    expect(selector.emptySelectionAllowed, isFalse);
    expect(
      find.text(
        'When enabled, eligible CraftSky members who imported your Instagram '
        'username may see a private suggestion to follow you.',
      ),
      findsOneWidget,
    );
    expect(
      tester.getTopLeft(accountFinder).dy,
      lessThan(tester.getTopLeft(find.text('Allow discovery')).dy),
    );
    expect(
      tester.getTopLeft(find.text('Allow discovery')).dy,
      lessThan(
        tester
            .getTopLeft(
              find.text(
                'When enabled, eligible CraftSky members who imported your '
                'Instagram username may see a private suggestion to follow '
                'you.',
              ),
            )
            .dy,
      ),
    );
    var confirm = tester.widget<FilledButton>(
      find.widgetWithText(FilledButton, 'Confirm this account'),
    );
    expect(confirm.onPressed, isNotNull);

    await tester.tap(find.text('Keep private'));
    await tester.pump();
    selector = tester.widget<SegmentedButton<bool>>(
      find.byType(SegmentedButton<bool>),
    );
    expect(selector.selected, {false});
    expect(
      find.text(
        'When enabled, eligible CraftSky members who imported your Instagram '
        'username may see a private suggestion to follow you.',
      ),
      findsNothing,
    );
    expect(
      find.text(
        'Your Instagram account remains verified, but CraftSky will not match '
        'it with people who imported your username.',
      ),
      findsOneWidget,
    );
    expect(find.text('Cancel verification'), findsOneWidget);

    await tester.tap(find.text('Cancel verification'));
    await tester.pump();
    expect(cancelCalls, 1);
    expect(
      find.text('This verification challenge is no longer active.'),
      findsOneWidget,
    );

    await tester.tap(find.text('Try again'));
    await tester.pump();
    await tester.pump();
    confirm = tester.widget<FilledButton>(
      find.widgetWithText(FilledButton, 'Confirm this account'),
    );
    expect(confirm.onPressed, isNotNull);
    await tester.tap(find.text('Keep private'));
    await tester.pump();
    confirm = tester.widget<FilledButton>(
      find.widgetWithText(FilledButton, 'Confirm this account'),
    );
    expect(confirm.onPressed, isNotNull);
    await tester.tap(find.text('Confirm this account'));
    await tester.pump();
    await tester.pump();

    expect(confirmedDiscoverable, isFalse);
    expect(find.text('Instagram account confirmed.'), findsOneWidget);
  });

  testWidgets(
    'IT-022 current server attempt is shown without creating a replacement',
    (tester) async {
      final initial = registry.SessionRegistry.empty().upsertAndActivate(
        token: 'token-a',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final repository = _Repository(
        status: const InstagramAccountStatus(
          integrationAvailable: true,
          account: null,
        ),
        imports: InstagramImportPage(items: const [], cursor: null),
        currentVerification: InstagramVerificationAttempt(
          verificationId: 'verification-current',
          state: InstagramVerificationState.processing,
          expiresAt: DateTime.now().toUtc().add(
            const Duration(minutes: 10),
          ),
        ),
        onCancelVerification: (_) async {},
      );

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _RegistryStorage(initial),
            ),
            instagramMigrationRepositoryProvider.overrideWith(
              (ref, _) async => repository,
            ),
            instagramVerificationStorageProvider.overrideWithValue(
              _EmptyVerificationStorage(),
            ),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const InstagramMigrationPage(),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Create verification challenge'), findsNothing);
      expect(find.text('Checking your message…'), findsOneWidget);
      expect(find.text('Cancel verification'), findsOneWidget);
      expect(find.text('Copy challenge'), findsOneWidget);
      final copy = tester.widget<OutlinedButton>(
        find.widgetWithText(OutlinedButton, 'Copy challenge'),
      );
      expect(copy.onPressed, isNull);
    },
  );

  testWidgets(
    'IT-016 verified account controls use shared switch and error styling',
    (tester) async {
      final initial = registry.SessionRegistry.empty().upsertAndActivate(
        token: 'token-a',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final repository = _Repository(
        status: InstagramAccountStatus(
          integrationAvailable: true,
          account: InstagramAccountLink(
            state: InstagramAccountLinkState.active,
            username: 'actual_maker',
            discoverable: true,
            conflictPending: false,
            reactivationRequired: false,
            verifiedAt: DateTime.utc(2026, 7, 22),
          ),
        ),
        imports: InstagramImportPage(
          items: [
            InstagramImportSummary(
              importId: 'import-a',
              state: InstagramImportState.active,
              sourceType: InstagramImportSourceType.instagramJson,
              followingCount: 1,
              createdAt: DateTime.utc(2026, 7, 22),
            ),
          ],
          cursor: null,
        ),
      );

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _RegistryStorage(initial),
            ),
            instagramMigrationRepositoryProvider.overrideWith(
              (ref, _) async => repository,
            ),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const InstagramMigrationPage(),
          ),
        ),
      );
      await tester.pumpAndSettle();

      final linkedFinder = find.byWidgetPredicate(
        (widget) =>
            widget is RichText &&
            widget.text.toPlainText() == 'Verified as @actual_maker',
      );
      expect(linkedFinder, findsOneWidget);
      final linkedText = tester.widget<RichText>(linkedFinder);
      final handleSpan = _textSpans(
        linkedText.text as TextSpan,
      ).singleWhere((span) => span.text == '@actual_maker');
      expect(handleSpan.style?.fontWeight, FontWeight.bold);

      final theme = Theme.of(
        tester.element(find.byType(InstagramMigrationPage)),
      );
      final errorColor = theme.colorScheme.error;
      expect(find.byType(SwitchListTile), findsNothing);
      final discoverySwitch = tester.widget<Switch>(
        find.byKey(const Key('instagram-discoverable-switch')),
      );
      expect(discoverySwitch.activeThumbColor, isNull);
      expect(discoverySwitch.activeTrackColor, isNull);
      final revokeButton = tester.widget<TextButton>(
        find.widgetWithText(TextButton, 'Revoke Instagram verification'),
      );
      expect(
        revokeButton.style?.foregroundColor?.resolve({}),
        errorColor,
      );
      expect(
        revokeButton.style?.iconColor?.resolve({}),
        errorColor,
      );
      expect(
        tester
            .getTopLeft(
              find.byKey(const Key('instagram-revoke-verification')),
            )
            .dy,
        greaterThan(
          tester
              .getBottomLeft(
                find.byKey(const Key('instagram-imports-card')),
              )
              .dy,
        ),
      );

      final deleteButtonFinder = find.widgetWithIcon(
        IconButton,
        CraftskyIconsBold.delete,
      );
      await tester.ensureVisible(deleteButtonFinder);
      await tester.pumpAndSettle();
      final deleteButton = tester.widget<IconButton>(deleteButtonFinder);
      expect(deleteButton.tooltip, 'Delete import');
      expect(
        deleteButton.style?.foregroundColor?.resolve({}),
        errorColor,
      );
      expect(find.widgetWithText(TextButton, 'Delete import'), findsNothing);
      final importRow = tester.widget<Row>(
        find.ancestor(
          of: deleteButtonFinder,
          matching: find.byType(Row),
        ),
      );
      expect(importRow.children.last, isA<IconButton>());
    },
  );

  testWidgets(
    'IT-016 revoking Instagram verification requires confirmation',
    (tester) async {
      final initial = registry.SessionRegistry.empty().upsertAndActivate(
        token: 'token-a',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      var revokeCalls = 0;
      final repository = _Repository(
        status: InstagramAccountStatus(
          integrationAvailable: true,
          account: InstagramAccountLink(
            state: InstagramAccountLinkState.active,
            username: 'actual_maker',
            discoverable: true,
            conflictPending: false,
            reactivationRequired: false,
            verifiedAt: DateTime.utc(2026, 7, 22),
          ),
        ),
        imports: InstagramImportPage(items: const [], cursor: null),
        onRevokeAccount: () async => revokeCalls += 1,
      );

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _RegistryStorage(initial),
            ),
            instagramMigrationRepositoryProvider.overrideWith(
              (ref, _) async => repository,
            ),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const InstagramMigrationPage(),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.ensureVisible(
        find.text('Revoke Instagram verification'),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.text('Revoke Instagram verification'));
      await tester.pumpAndSettle();

      expect(find.text('Revoke Instagram verification?'), findsOneWidget);
      expect(
        find.text(
          'This removes your Instagram verification and deletes your imported '
          'handles. Existing CraftSky follows will not be affected.',
        ),
        findsOneWidget,
      );
      expect(revokeCalls, 0);

      await tester.tap(find.text('Cancel'));
      await tester.pumpAndSettle();

      expect(revokeCalls, 0);
      expect(find.text('Verified as @actual_maker'), findsOneWidget);

      await tester.ensureVisible(
        find.text('Revoke Instagram verification'),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.text('Revoke Instagram verification'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Revoke Instagram verification').last);
      await tester.pumpAndSettle();

      expect(revokeCalls, 1);
      expect(find.text('Verified as @actual_maker'), findsNothing);
    },
  );
}

Future<void> _pumpVerifiedExportPage(
  WidgetTester tester, {
  required registry.SessionRegistry initial,
  required _Repository repository,
  required InstagramExportFilePicker picker,
}) async {
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _RegistryStorage(initial),
        ),
        instagramVerificationStorageProvider.overrideWithValue(
          _EmptyVerificationStorage(),
        ),
        instagramMigrationRepositoryProvider.overrideWith(
          (ref, _) async => repository,
        ),
        instagramExportFilePickerProvider.overrideWithValue(picker),
      ],
      child: MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: const InstagramMigrationPage(),
      ),
    ),
  );
  await tester.pumpAndSettle();
  await tester.ensureVisible(find.text('Instagram export'));
  await tester.tap(find.text('Instagram export'));
  await tester.pumpAndSettle();
  await tester.ensureVisible(find.text('Select Instagram export'));
  await tester.pumpAndSettle();
}

_Repository _verifiedImportRepository(
  List<InstagramImportRequest> sentRequests,
) => _Repository(
  status: InstagramAccountStatus(
    integrationAvailable: true,
    account: InstagramAccountLink(
      state: InstagramAccountLinkState.active,
      username: 'synthetic_instagram',
      discoverable: true,
      conflictPending: false,
      reactivationRequired: false,
      verifiedAt: DateTime.utc(2026, 7, 23),
    ),
  ),
  imports: InstagramImportPage(items: const [], cursor: null),
  onCreateImport: (request) async {
    sentRequests.add(request);
    return InstagramImportCreateResult(
      import: InstagramImportSummary(
        importId: 'unexpected-import',
        state: InstagramImportState.active,
        sourceType: request.sourceType,
        followingCount: request.entries.length,
        createdAt: DateTime.utc(2026, 7, 23),
      ),
      followingCount: request.entries.length,
    );
  },
);

Iterable<TextSpan> _textSpans(TextSpan span) sync* {
  yield span;
  for (final child in span.children ?? const <InlineSpan>[]) {
    if (child case final TextSpan textSpan) yield* _textSpans(textSpan);
  }
}

final class _EmptyVerificationStorage implements InstagramVerificationStorage {
  @override
  Future<void> delete(
    AccountKey account, {
    String? verificationId,
  }) async {}

  @override
  Future<InstagramVerificationSnapshot?> read(AccountKey account) async => null;

  @override
  Future<void> write(
    AccountKey account,
    InstagramVerificationSnapshot snapshot,
  ) async {}
}

final class _RegistryStorage implements SessionRegistryStorage {
  _RegistryStorage(this.value);

  registry.SessionRegistry value;

  @override
  Future<registry.SessionRegistry> read() async => value;

  @override
  Future<void> write(registry.SessionRegistry registry) async =>
      value = registry;
}

final class _Repository implements InstagramMigrationRepository {
  const _Repository({
    required this.imports,
    this.status = const InstagramAccountStatus(
      integrationAvailable: false,
      account: null,
    ),
    this.onCreateImport,
    this.onCreateVerification,
    this.onCancelVerification,
    this.onConfirmVerification,
    this.onRevokeAccount,
    this.currentVerification,
  });

  final InstagramImportPage imports;
  final InstagramAccountStatus status;
  final Future<InstagramImportCreateResult> Function(
    InstagramImportRequest request,
  )?
  onCreateImport;
  final Future<InstagramVerificationAttempt> Function()? onCreateVerification;
  final Future<void> Function(String verificationId)? onCancelVerification;
  final Future<InstagramVerificationConfirmation> Function(
    String verificationId, {
    required bool discoverable,
  })?
  onConfirmVerification;
  final Future<void> Function()? onRevokeAccount;
  final InstagramVerificationAttempt? currentVerification;

  @override
  Future<InstagramAccountStatus> getAccount() async => status;

  @override
  Future<InstagramImportCreateResult> createImport(
    InstagramImportRequest request,
  ) => onCreateImport!.call(request);

  @override
  Future<InstagramVerificationAttempt> createVerification() =>
      onCreateVerification!.call();

  @override
  Future<InstagramVerificationAttempt?> getCurrentVerification() async =>
      currentVerification;

  @override
  Future<void> cancelVerification(String verificationId) =>
      onCancelVerification!.call(verificationId);

  @override
  Future<InstagramVerificationConfirmation> confirmVerification(
    String verificationId, {
    required bool discoverable,
  }) => onConfirmVerification!.call(
    verificationId,
    discoverable: discoverable,
  );

  @override
  Future<InstagramImportPage> listImports({int? limit, String? cursor}) async =>
      imports;

  @override
  Future<InstagramSuggestionPage> listSuggestions({
    int? limit,
    String? cursor,
  }) async => InstagramSuggestionPage(items: const [], cursor: null);

  @override
  Future<void> revokeAccount() => onRevokeAccount!.call();

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}
