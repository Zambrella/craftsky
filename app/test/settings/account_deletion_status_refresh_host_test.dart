import 'package:craftsky_app/auth/data/account_deletion_repository.dart';
import 'package:craftsky_app/auth/models/account_deletion.dart';
import 'package:craftsky_app/auth/models/account_switcher_state.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/account_deletion_repository_provider.dart';
import 'package:craftsky_app/auth/providers/deletion_status_registry_provider.dart'
    show deletionStatusRegistryProvider, deletionStatusRegistryStorageProvider;
import 'package:craftsky_app/auth/providers/deletion_status_registry_storage.dart';
import 'package:craftsky_app/auth/widgets/account_switcher_content.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/settings/widgets/account_deletion_status_refresh_host.dart';
import 'package:craftsky_app/shared/device/device_id_provider.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

final class _RefreshStorage implements DeletionStatusRegistryStorage {
  _RefreshStorage(this.value);

  DeletionStatusRegistry value;

  @override
  Future<DeletionStatusRegistry> read() async => value;

  @override
  Future<void> write(DeletionStatusRegistry registry) async => value = registry;
}

void main() {
  const jobId = '10000000-0000-0000-0000-000000000061';

  testWidgets(
    'visible switcher deletion row refreshes away at terminal success',
    (
      tester,
    ) async {
      final harness = _RefreshHarness(
        jobId: jobId,
        responses: const [
          AccountDeletionStatus.active,
          AccountDeletionStatus.deleted,
        ],
      );
      await tester.pumpWidget(
        harness.widget(
          refreshImmediately: false,
          pollBackoff: const [Duration(milliseconds: 10)],
        ),
      );
      await tester.pump();

      expect(find.byIcon(Icons.hourglass_top), findsOneWidget);
      for (var attempt = 0; attempt < 6 && harness.calls < 2; attempt++) {
        await tester.pump(const Duration(milliseconds: 10));
        await tester.pump();
      }
      await tester.pumpAndSettle();

      expect(harness.calls, 2);
      expect(find.byIcon(Icons.hourglass_top), findsNothing);
      expect(find.text('Bob'), findsOneWidget);
    },
  );

  testWidgets('app resume refreshes an observed deletion without status page', (
    tester,
  ) async {
    final harness = _RefreshHarness(
      jobId: jobId,
      responses: const [
        AccountDeletionStatus.active,
        AccountDeletionStatus.active,
        AccountDeletionStatus.active,
      ],
    );
    await tester.pumpWidget(
      harness.widget(
        refreshImmediately: false,
        pollBackoff: const [Duration(hours: 1)],
      ),
    );
    await tester.pumpAndSettle();
    final callsBeforeResume = harness.calls;
    expect(find.byIcon(Icons.hourglass_top), findsOneWidget);

    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
    await tester.pumpAndSettle();

    expect(harness.calls, callsBeforeResume + 1);
    expect(find.byIcon(Icons.hourglass_top), findsOneWidget);
  });

  testWidgets('attention response stops automatic polling', (tester) async {
    final harness = _RefreshHarness(
      jobId: jobId,
      responses: const [AccountDeletionStatus.needsAttention],
    );
    await tester.pumpWidget(
      harness.widget(
        refreshImmediately: true,
        pollBackoff: const [Duration(milliseconds: 10)],
      ),
    );
    await tester.pump(const Duration(milliseconds: 100));
    await tester.pumpAndSettle();

    expect(harness.calls, 1);
    expect(find.textContaining('Deletion needs attention'), findsOneWidget);
  });
}

final class _RefreshHarness {
  _RefreshHarness({required this.jobId, required this.responses}) {
    dio = Dio(BaseOptions(baseUrl: 'https://appview.invalid'));
    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          final status = responses[calls.clamp(0, responses.length - 1)];
          calls++;
          handler.resolve(
            Response<Map<String, dynamic>>(
              requestOptions: options,
              statusCode: 200,
              data: {
                'jobId': jobId,
                'status': status.name,
                'phase': status == AccountDeletionStatus.deleted
                    ? ''
                    : 'removingCraftskyRecords',
                'retryAllowed': status == AccountDeletionStatus.needsAttention,
                'needsReauthentication': false,
              },
            ),
          );
        },
      ),
    );
  }

  final String jobId;
  final List<AccountDeletionStatus> responses;
  late final Dio dio;
  int calls = 0;

  Widget widget({
    required bool refreshImmediately,
    List<Duration> pollBackoff = const [Duration(seconds: 2)],
  }) {
    final deletion =
        DeletionStatusEntry.pending(
          jobId: jobId,
          did: 'did:plc:alice',
          handle: 'alice.test',
          statusToken: 'status-token',
          displayName: 'Alice',
        ).withStatus(
          status: AccountDeletionStatus.active,
          phase: AccountDeletionPhase.removingPrivateData,
        );
    final storage = _RefreshStorage(
      DeletionStatusRegistry.empty().upsert(deletion),
    );
    final sessions = SessionRegistry.empty().upsertAndActivate(
      token: 'bob-token',
      did: 'did:plc:bob',
      handle: 'bob.test',
      cachedDisplayName: 'Bob',
    );
    return ProviderScope(
      overrides: [
        deletionStatusRegistryStorageProvider.overrideWithValue(storage),
        deviceIdProvider.overrideWith((ref) async => 'test-device'),
        deletionStatusApiClientProvider.overrideWith(
          (ref, key) => DeletionStatusApiClient(dio),
        ),
      ],
      child: MaterialApp(
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: AccountDeletionStatusRefreshHost(
          refreshImmediately: refreshImmediately,
          pollBackoff: pollBackoff,
          child: Consumer(
            builder: (context, ref, _) {
              final deletions =
                  ref.watch(deletionStatusRegistryProvider).value ??
                  DeletionStatusRegistry.empty();
              return Scaffold(
                body: AccountSwitcherContent(
                  state: AccountSwitcherState.fromRegistries(
                    sessions: sessions,
                    deletions: deletions,
                  ),
                  onSelect: (_) {},
                  onAddAccount: () {},
                  onOpenDeletionStatus: (_) {},
                ),
              );
            },
          ),
        ),
      ),
    );
  }
}
