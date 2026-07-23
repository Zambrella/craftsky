import 'dart:convert';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart' as registry;
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/instagram_migration/data/instagram_migration_repository.dart';
import 'package:craftsky_app/instagram_migration/data/instagram_verification_storage.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_account.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_suggestion.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_verification.dart';
import 'package:craftsky_app/instagram_migration/pages/instagram_migration_page.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_migration_repository_provider.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_json_file_picker.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/recording_messenger.dart';

void main() {
  testWidgets(
    'IT-016 import and suggestion controls stay hidden until verification',
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
        find.text('Complete verification to sync the accounts you follow.'),
        findsOneWidget,
      );
      expect(
        find.byKey(const Key('instagram-import-composer-card')),
        findsNothing,
      );
      expect(find.byKey(const Key('instagram-imports-card')), findsNothing);
      expect(find.byKey(const Key('instagram-suggestions-card')), findsNothing);
      semantics.dispose();
    },
  );

  testWidgets('FR-025 manual import directly uploads normalized entries', (
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
          initialSuggestionCount: 0,
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
          instagramJsonFilePickerProvider.overrideWithValue(
            () async => Uint8List.fromList(
              utf8.encode(
                jsonEncode({
                  'relationships_following': [
                    {
                      'string_list_data': [
                        {'value': 'BobMaker'},
                      ],
                    },
                  ],
                }),
              ),
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
      'Enter the Instagram handles of accounts you follow, one per line. '
      'CraftSky keeps them until you unlink Instagram.',
    );
    final jsonDescription = find.text(
      'Select the JSON file containing accounts you follow. CraftSky reads it '
      'only on this device and uploads usernames. Follower data is ignored, '
      'and ZIP archives are not supported.',
    );
    expect(manualDescription, findsOneWidget);
    expect(jsonDescription, findsNothing);
    expect(
      tester
          .getBottomLeft(
            find.byKey(const Key('instagram-import-kind-selector')),
          )
          .dy,
      lessThan(tester.getTopLeft(manualDescription).dy),
    );
    expect(
      tester.getBottomLeft(manualDescription).dy,
      lessThan(
        tester.getTopLeft(find.byKey(const Key('instagram-manual-handles'))).dy,
      ),
    );
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
    final notificationSettings = find.text('Notification settings');
    expect(notificationSettings, findsOneWidget);
    expect(
      find.widgetWithText(TextButton, 'Notification settings'),
      findsNothing,
    );
    final notificationSettingsLink = find.ancestor(
      of: notificationSettings,
      matching: find.byType(InkWell),
    );
    expect(notificationSettingsLink, findsOneWidget);
    expect(
      tester.widget<InkWell>(notificationSettingsLink).onTap,
      isNotNull,
    );
    final importButton = find.text('Import handles');
    await tester.drag(find.byType(ListView), const Offset(0, -300));
    await tester.pumpAndSettle();
    await tester.tap(importButton);
    await tester.pumpAndSettle();

    expect(sentRequests.single.entries, hasLength(1));
    expect(sentRequests.single.entries.single.username, 'alice');
    expect(sentRequests.single.sourceType, InstagramImportSourceType.manual);
    expect(messenger.calls, [('info', 'Instagram import created', null)]);

    await tester.ensureVisible(find.text('Choose JSON'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Choose JSON'));
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
      lessThan(tester.getTopLeft(find.text('Select Instagram JSON file')).dy),
    );
    await tester.ensureVisible(find.text('Select Instagram JSON file'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Select Instagram JSON file'));
    await tester.pumpAndSettle();

    expect(sentRequests, hasLength(2));
    expect(
      sentRequests.last.sourceType,
      InstagramImportSourceType.instagramJson,
    );
    expect(sentRequests.last.entries.single.username, 'bobmaker');
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
        'When enabled, eligible CraftSky members who imported your '
        'Instagram username may see a suggestion. This never follows '
        'anyone automatically.',
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
                'Instagram username may see a suggestion. This never follows '
                'anyone automatically.',
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
        'When enabled, eligible CraftSky members who imported your '
        'Instagram username may see a suggestion. This never follows '
        'anyone automatically.',
      ),
      findsNothing,
    );
    expect(
      find.text(
        'Your Instagram account will be linked, but it will not be suggested '
        'to people who imported your username.',
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

  testWidgets('FR-024 linked Instagram handle is bold', (tester) async {
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
      imports: InstagramImportPage(items: const [], cursor: null),
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
          widget.text.toPlainText() == 'Linked as @actual_maker',
    );
    expect(linkedFinder, findsOneWidget);
    final linkedText = tester.widget<RichText>(linkedFinder);
    final handleSpan = _textSpans(
      linkedText.text as TextSpan,
    ).singleWhere((span) => span.text == '@actual_maker');
    expect(handleSpan.style?.fontWeight, FontWeight.bold);
  });
}

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
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}
