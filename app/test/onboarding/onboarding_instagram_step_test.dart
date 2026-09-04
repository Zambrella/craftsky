import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/instagram_migration/models/instagram_account.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_suggestion.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_account_provider.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_imports_provider.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_suggestions_provider.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_export_file_picker.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/onboarding/widgets/onboarding_instagram_step.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/recording_messenger.dart';

final _lease = ActiveAccountLease(
  session: AccountSessionLease(
    account: AccountKey('did:plc:alice'),
    sessionGeneration: 1,
  ),
  activationGeneration: 1,
);

final class _Storage implements SessionRegistryStorage {
  _Storage(this.value);
  SessionRegistry value;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

final class _Account extends InstagramAccount {
  _Account(this.value);
  final InstagramAccountStatus value;
  int reactivateCalls = 0;

  @override
  Future<InstagramAccountStatus> build(ActiveAccountLease lease) async => value;

  @override
  Future<bool> reactivate() async {
    reactivateCalls++;
    return true;
  }
}

final class _Imports extends InstagramImports {
  @override
  Future<InstagramImportPage> build(ActiveAccountLease lease) async =>
      InstagramImportPage(items: const [], cursor: null);
}

final class _ReadinessImports extends InstagramImports {
  int refreshCalls = 0;
  final requests = <InstagramImportRequest>[];

  @override
  Future<InstagramImportPage> build(ActiveAccountLease lease) =>
      Completer<InstagramImportPage>().future;

  void fail() {
    state = AsyncError(StateError('offline'), StackTrace.current);
  }

  @override
  Future<void> refresh() async {
    refreshCalls++;
    state = AsyncData(InstagramImportPage(items: const [], cursor: null));
  }

  @override
  Future<InstagramImportCreateResult?> create(
    InstagramImportRequest request,
  ) async {
    requests.add(request);
    return InstagramImportCreateResult(
      import: InstagramImportSummary(
        importId: 'import-1',
        state: InstagramImportState.active,
        sourceType: request.sourceType,
        followingCount: request.entries.length,
        createdAt: DateTime.utc(2026, 8, 31),
      ),
      followingCount: request.entries.length,
    );
  }
}

final class _Suggestions extends InstagramSuggestions {
  @override
  Future<InstagramSuggestionReviewState> build(
    ActiveAccountLease lease,
  ) async => InstagramSuggestionReviewState(items: const [], cursor: null);
}

final class _InteractiveSuggestions extends InstagramSuggestions {
  final followed = <String>[];
  final dismissed = <String>[];
  int loadMoreCalls = 0;

  static InstagramSuggestion suggestion(String id, String handle) =>
      InstagramSuggestion(
        suggestionId: id,
        target: InstagramSuggestionTarget(
          did: 'did:plc:$id',
          handle: handle,
          displayName: '$handle maker',
        ),
        createdAt: DateTime.utc(2026, 8, 31),
      );

  @override
  Future<InstagramSuggestionReviewState> build(
    ActiveAccountLease lease,
  ) async => InstagramSuggestionReviewState(
    items: [suggestion('one', 'alice'), suggestion('two', 'bob')],
    cursor: 'next',
  );

  @override
  Future<bool> accept(String suggestionId) async {
    followed.add(suggestionId);
    _remove(suggestionId);
    return true;
  }

  @override
  Future<bool> dismiss(String suggestionId) async {
    dismissed.add(suggestionId);
    _remove(suggestionId);
    return true;
  }

  @override
  Future<bool> loadMore() async {
    loadMoreCalls++;
    final current = state.requireValue;
    state = AsyncData(
      InstagramSuggestionReviewState(
        items: [...current.items, suggestion('three', 'carol')],
        cursor: null,
      ),
    );
    return true;
  }

  void _remove(String id) {
    final current = state.requireValue;
    state = AsyncData(
      InstagramSuggestionReviewState(
        items: current.items
            .where((suggestion) => suggestion.suggestionId != id)
            .toList(),
        cursor: current.cursor,
      ),
    );
  }
}

void main() {
  testWidgets('AT-009 import creation waits for readiness and retries errors', (
    tester,
  ) async {
    final status = InstagramAccountStatus(
      integrationAvailable: true,
      account: InstagramAccountLink(
        state: InstagramAccountLinkState.active,
        username: 'alice_instagram',
        discoverable: true,
        conflictPending: false,
        reactivationRequired: false,
        verifiedAt: DateTime.utc(2026, 8, 31),
      ),
    );
    final imports = _ReadinessImports();
    final registry = SessionRegistry.empty().upsertAndActivate(
      token: 'token',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final lease = registry.activeLease!;

    final container = ProviderContainer.test(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(registry),
        ),
        instagramAccountProvider.overrideWith2((_) => _Account(status)),
        instagramImportsProvider.overrideWith2((_) => imports),
        instagramSuggestionsProvider.overrideWith2((_) => _Suggestions()),
        instagramExportFilePickerProvider.overrideWithValue(
          () async => const InstagramImportParseResult(
            entries: [InstagramImportEntry(username: 'bobmaker')],
          ),
        ),
      ],
    );
    addTearDown(container.dispose);
    await container.read(sessionRegistryProvider.future);

    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: Scaffold(
              body: SingleChildScrollView(
                child: OnboardingInstagramStep(lease: lease),
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    FilledButton selectExport() => tester.widget(
      find.widgetWithText(FilledButton, 'Select Instagram export'),
    );
    expect(selectExport().onPressed, isNull);

    imports.fail();
    await tester.pump();
    expect(find.text("Your Instagram imports didn't load."), findsOneWidget);
    expect(selectExport().onPressed, isNull);

    final retry = find.byKey(const Key('instagram-import-readiness-retry'));
    await tester.ensureVisible(retry);
    await tester.tap(retry);
    await tester.pump();
    expect(imports.refreshCalls, 1);
    expect(selectExport().onPressed, isNotNull);

    final enterHandles = find.text('Enter handles');
    await tester.ensureVisible(enterHandles);
    await tester.tap(enterHandles);
    await tester.pump();
    await tester.enterText(
      find.byKey(const Key('instagram-manual-handles')),
      '@Alice\nALICE',
    );
    final importAction = find.widgetWithText(FilledButton, 'Import handles');
    await tester.ensureVisible(importAction);
    await tester.tap(importAction);
    await tester.pumpAndSettle();
    expect(imports.requests, hasLength(1));
    expect(
      imports.requests.single.sourceType,
      InstagramImportSourceType.manual,
    );
    expect(imports.requests.single.entries.single.username, 'alice');

    await tester.ensureVisible(find.text('Instagram export'));
    await tester.tap(find.text('Instagram export'));
    await tester.pumpAndSettle();
    final exportAction = find.widgetWithText(
      FilledButton,
      'Select Instagram export',
    );
    await tester.ensureVisible(exportAction);
    await tester.tap(exportAction);
    await tester.pump();
    expect(imports.requests, hasLength(2));
    expect(
      imports.requests.last.sourceType,
      InstagramImportSourceType.instagramJson,
    );
    expect(imports.requests.last.entries.single.username, 'bobmaker');
  });

  testWidgets('AT-009 to AT-011 uses scoped shared Instagram sections', (
    tester,
  ) async {
    final status = InstagramAccountStatus(
      integrationAvailable: true,
      account: InstagramAccountLink(
        state: InstagramAccountLinkState.active,
        username: 'alice_instagram',
        discoverable: true,
        conflictPending: false,
        reactivationRequired: false,
        verifiedAt: DateTime.utc(2026, 8, 31),
      ),
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          instagramAccountProvider.overrideWith2((_) => _Account(status)),
          instagramImportsProvider.overrideWith2((_) => _Imports()),
          instagramSuggestionsProvider.overrideWith2((_) => _Suggestions()),
        ],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: SingleChildScrollView(
              child: OnboardingInstagramStep(lease: _lease),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(
      find.textContaining('Connecting Instagram is optional'),
      findsOneWidget,
    );
    expect(find.byKey(const Key('instagram-account-card')), findsOneWidget);
    expect(
      find.byKey(const Key('instagram-import-composer-card')),
      findsOneWidget,
    );
    expect(find.byKey(const Key('instagram-suggestions-card')), findsOneWidget);
    expect(find.byKey(const Key('instagram-imports-card')), findsNothing);
    expect(find.text('Revoke Instagram verification'), findsNothing);
  });

  testWidgets('AT-011 suggestions follow dismiss and load more inline', (
    tester,
  ) async {
    final status = InstagramAccountStatus(
      integrationAvailable: true,
      account: InstagramAccountLink(
        state: InstagramAccountLinkState.active,
        username: 'alice_instagram',
        discoverable: true,
        conflictPending: false,
        reactivationRequired: false,
        verifiedAt: DateTime.utc(2026, 8, 31),
      ),
    );
    final suggestions = _InteractiveSuggestions();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          instagramAccountProvider.overrideWith2((_) => _Account(status)),
          instagramImportsProvider.overrideWith2((_) => _Imports()),
          instagramSuggestionsProvider.overrideWith2((_) => suggestions),
        ],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: SingleChildScrollView(
              child: OnboardingInstagramStep(lease: _lease),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    Finder action(String id, String label) => find.descendant(
      of: find.byKey(ValueKey('instagram-suggestion-$id')),
      matching: find.text(label),
    );
    final aliceRow = find.byKey(
      const ValueKey('instagram-suggestion-one'),
    );
    final aliceTapTargets = tester.widgetList<InkWell>(
      find.descendant(of: aliceRow, matching: find.byType(InkWell)),
    );
    expect(aliceTapTargets.any((target) => target.onTap == null), isTrue);

    await tester.ensureVisible(action('one', 'Follow'));
    await tester.tap(action('one', 'Follow'));
    await tester.pump();
    expect(suggestions.followed, ['one']);
    expect(aliceRow, findsNothing);

    await tester.ensureVisible(action('two', 'Dismiss'));
    await tester.tap(action('two', 'Dismiss'));
    await tester.pump();
    expect(suggestions.dismissed, ['two']);

    await tester.ensureVisible(find.text('Load more'));
    await tester.tap(find.text('Load more'));
    await tester.pump();
    expect(suggestions.loadMoreCalls, 1);
    expect(find.text('carol maker'), findsOneWidget);
  });

  testWidgets('AT-010 linked and inactive account controls are reused', (
    tester,
  ) async {
    Future<void> pump(_Account account) => tester.pumpWidget(
      ProviderScope(
        overrides: [
          instagramAccountProvider.overrideWith2((_) => account),
          instagramImportsProvider.overrideWith2((_) => _Imports()),
          instagramSuggestionsProvider.overrideWith2((_) => _Suggestions()),
        ],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: SingleChildScrollView(
              child: OnboardingInstagramStep(lease: _lease),
            ),
          ),
        ),
      ),
    );

    final active = _Account(
      InstagramAccountStatus(
        integrationAvailable: true,
        account: InstagramAccountLink(
          state: InstagramAccountLinkState.active,
          username: 'alice_instagram',
          discoverable: true,
          conflictPending: false,
          reactivationRequired: false,
          verifiedAt: DateTime.utc(2026, 8, 31),
        ),
      ),
    );
    await pump(active);
    await tester.pumpAndSettle();
    expect(find.textContaining('@alice_instagram'), findsOneWidget);
    expect(
      find.byKey(const Key('instagram-discoverable-switch')),
      findsOneWidget,
    );
    expect(find.text('Reactivate Instagram account'), findsNothing);

    await tester.pumpWidget(const SizedBox.shrink());
    final inactive = _Account(
      InstagramAccountStatus(
        integrationAvailable: true,
        account: InstagramAccountLink(
          state: InstagramAccountLinkState.membershipInactive,
          username: 'alice_instagram',
          discoverable: true,
          conflictPending: false,
          reactivationRequired: true,
          verifiedAt: DateTime.utc(2026, 8, 31),
        ),
      ),
    );
    await pump(inactive);
    await tester.pumpAndSettle();
    expect(find.textContaining('@alice_instagram'), findsOneWidget);
    expect(
      find.byKey(const Key('instagram-discoverable-switch')),
      findsNothing,
    );
    await tester.tap(find.text('Reactivate Instagram account'));
    await tester.pump();
    expect(inactive.reactivateCalls, 1);
  });
}
