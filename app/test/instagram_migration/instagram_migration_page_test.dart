import 'package:craftsky_app/auth/models/session_registry.dart' as registry;
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/instagram_migration/data/instagram_migration_repository.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_account.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_suggestion.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_verification.dart';
import 'package:craftsky_app/instagram_migration/pages/instagram_migration_page.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_migration_repository_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets(
    'IT-016 disabled verification keeps local import and reactivation visible',
    (tester) async {
      final initial = registry.SessionRegistry.empty().upsertAndActivate(
        token: 'token-a',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final repository = _Repository(
        imports: InstagramImportPage(
          items: [
            InstagramImportSummary(
              importId: 'import-a',
              state: InstagramImportState.membershipInactive,
              sourceType: InstagramImportSourceType.instagramJson,
              retainUnmatched: true,
              retentionExpiresAt: DateTime.utc(2026, 12),
              followingCount: 3,
              followerCount: 0,
              createdAt: DateTime.utc(2026, 7),
            ),
          ],
          cursor: null,
        ),
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
          ],
          child: const MaterialApp(
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: InstagramMigrationPage(),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Find people from Instagram'), findsOneWidget);
      expect(
        find.text('Instagram verification is unavailable right now.'),
        findsOneWidget,
      );
      expect(
        find.text('You can still import handles on this device.'),
        findsOneWidget,
      );
      await tester.scrollUntilVisible(
        find.text('Reactivate import'),
        300,
        scrollable: find.byType(Scrollable).first,
      );
      expect(find.bySemanticsLabel('Reactivate import'), findsOneWidget);
      expect(
        find.textContaining('does not extend its retention date'),
        findsOneWidget,
      );
      semantics.dispose();
    },
  );

  testWidgets('FR-025 manual preview uploads only normalized entries', (
    tester,
  ) async {
    final initial = registry.SessionRegistry.empty().upsertAndActivate(
      token: 'token-a',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    InstagramImportRequest? sentRequest;
    final repository = _Repository(
      status: const InstagramAccountStatus(
        integrationAvailable: true,
        account: null,
      ),
      imports: InstagramImportPage(items: const [], cursor: null),
      onCreateImport: (request) async {
        sentRequest = request;
        return InstagramImportCreateResult(
          import: InstagramImportSummary(
            importId: 'import-new',
            state: InstagramImportState.active,
            sourceType: request.sourceType,
            retainUnmatched: request.retainUnmatched,
            retentionExpiresAt: null,
            followingCount: request.entries.length,
            followerCount: 0,
            createdAt: DateTime.utc(2026, 7, 19),
          ),
          counts: const {},
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
          instagramMigrationRepositoryProvider.overrideWith(
            (ref, _) async => repository,
          ),
        ],
        child: const MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: InstagramMigrationPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    final directionField = find.byType(
      DropdownButtonFormField<InstagramRelationshipDirection>,
    );
    await tester.ensureVisible(directionField);
    await tester.tap(directionField);
    await tester.pumpAndSettle();
    await tester.tap(find.text('Accounts I follow').last);
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byType(TextField),
      '@Alice\nALICE\nbad name',
    );
    final previewButton = find.text('Preview normalized handles');
    await tester.ensureVisible(previewButton);
    await tester.tap(previewButton);
    await tester.pump();

    expect(find.text('1 account you follow ready'), findsOneWidget);
    expect(find.text('1 unsupported entry ignored'), findsOneWidget);
    expect(find.text('1 duplicate removed'), findsOneWidget);
    await tester.ensureVisible(find.text('Create private import'));
    await tester.tap(find.text('Create private import'));
    await tester.pumpAndSettle();

    expect(sentRequest?.entries, hasLength(1));
    expect(sentRequest?.entries.single.username, 'alice');
    expect(
      sentRequest?.entries.single.direction,
      InstagramRelationshipDirection.following,
    );
    expect(sentRequest?.sourceType, InstagramImportSourceType.manual);
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
          instagramMigrationRepositoryProvider.overrideWith(
            (ref, _) async => repository,
          ),
          instagramDmLauncherProvider.overrideWithValue((uri) async {
            opened = uri;
            return true;
          }),
        ],
        child: const MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: InstagramMigrationPage(),
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

  testWidgets('FR-024 candidate requires an explicit discovery choice', (
    tester,
  ) async {
    final initial = registry.SessionRegistry.empty().upsertAndActivate(
      token: 'token-a',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    bool? confirmedDiscoverable;
    final repository = _Repository(
      status: const InstagramAccountStatus(
        integrationAvailable: true,
        account: null,
      ),
      imports: InstagramImportPage(items: const [], cursor: null),
      onCreateVerification: () async => InstagramVerificationAttempt(
        verificationId: 'verification-a',
        state: InstagramVerificationState.pendingConfirmation,
        expiresAt: DateTime.now().toUtc().add(const Duration(minutes: 10)),
        candidateUsername: 'actual_maker',
      ),
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
          instagramMigrationRepositoryProvider.overrideWith(
            (ref, _) async => repository,
          ),
        ],
        child: const MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: InstagramMigrationPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('Create verification challenge'));
    await tester.pump();
    await tester.pump();

    expect(find.text('Instagram found @actual_maker'), findsOneWidget);
    var confirm = tester.widget<FilledButton>(
      find.widgetWithText(FilledButton, 'Confirm this account'),
    );
    expect(confirm.onPressed, isNull);
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
