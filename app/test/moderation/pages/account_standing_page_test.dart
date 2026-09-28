import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/moderation/data/moderation_repository.dart';
import 'package:craftsky_app/moderation/models/account_moderation.dart';
import 'package:craftsky_app/moderation/pages/account_standing_page.dart';
import 'package:craftsky_app/moderation/providers/moderation_providers.dart';
import 'package:craftsky_app/moderation/widgets/moderation_history_entry.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

void main() {
  final account = AccountKey('did:plc:owner');

  testWidgets('shows standing and distinct owner-safe consequence states', (
    tester,
  ) async {
    await _pumpPage(tester, account: account);

    expect(find.text('Your account has active moderation action'), findsOne);
    expect(find.textContaining('Active strikes: 1 of the 3'), findsOne);
    expect(find.text('Spam'), findsOne);
    expect(find.textContaining('Formal warning · Applied'), findsOne);
    expect(find.textContaining('Account strike · Expired'), findsOne);
    expect(
      find.textContaining('Until Thursday, September 10, 2026'),
      findsOne,
    );
    expect(find.textContaining('Content hidden · Overturned'), findsOne);
    expect(find.text('Post'), findsOne);
    expect(find.textContaining('remaining'), findsNothing);
    expect(find.textContaining('allowed'), findsNothing);
  });

  testWidgets('uses CraftSky card and state-specific paper colour', (
    tester,
  ) async {
    final theme = AppTheme.lightThemeData;
    final semanticColors = theme.extension<SemanticColorsTheme>()!;

    for (final state in [
      (
        standing: const AccountStanding(
          activeStrikeCount: 0,
          strikeThreshold: 3,
          thresholdSuspended: false,
          severeSuspended: false,
          suspended: false,
        ),
        colour: semanticColors.successSurface,
      ),
      (
        standing: const AccountStanding(
          activeStrikeCount: 1,
          strikeThreshold: 3,
          thresholdSuspended: false,
          severeSuspended: false,
          suspended: false,
        ),
        colour: semanticColors.warningSurface,
      ),
      (
        standing: const AccountStanding(
          activeStrikeCount: 3,
          strikeThreshold: 3,
          thresholdSuspended: true,
          severeSuspended: false,
          suspended: true,
        ),
        colour: semanticColors.errorSurface,
      ),
    ]) {
      await _pumpPage(
        tester,
        account: account,
        repository: _FakeModerationRepository(standing: state.standing),
      );

      expect(
        find.byKey(const Key('account-standing-summary-card')),
        findsOneWidget,
      );
      expect(find.byType(CraftskyCard), findsNWidgets(2));
      expect(
        tester
            .widget<CraftskyCard>(
              find.byKey(const Key('account-standing-summary-card')),
            )
            .clipBehavior,
        Clip.none,
      );
      final tile = tester.widget<Container>(
        find.byKey(const Key('account-standing-summary-icon-tile')),
      );
      expect((tile.decoration! as BoxDecoration).color, state.colour);
      expect(
        tester
            .widget<CraftskyCard>(
              find.byKey(const Key('moderation-history-entry-card')),
            )
            .clipBehavior,
        Clip.none,
      );
    }
  });

  testWidgets('appeal launches mailto and copy fallbacks remain available', (
    tester,
  ) async {
    Uri? launched;
    final copied = <String>[];
    await _pumpPage(
      tester,
      account: account,
      overrides: [
        moderationMailLauncherProvider.overrideWithValue((uri) async {
          launched = uri;
          return false;
        }),
        moderationTextCopierProvider.overrideWithValue((value) async {
          copied.add(value);
        }),
      ],
    );

    await tester.ensureVisible(find.text('Appeal by email'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Appeal by email'));
    await tester.pump();
    expect(launched?.scheme, 'mailto');
    expect(launched?.path, moderationAppealAddress);
    expect(
      launched?.queryParameters['subject'],
      contains('MOD-550e8400-e29b-41d4-a716-446655440000'),
    );
    expect(find.text('Copy email address'), findsOne);
    expect(find.text('Copy case reference'), findsOne);

    await tester.ensureVisible(find.text('Copy email address'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Copy email address'));
    await tester.pump();
    await tester.ensureVisible(find.text('Copy case reference'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Copy case reference'));
    await tester.pump();
    expect(copied, [
      moderationAppealAddress,
      'MOD-550e8400-e29b-41d4-a716-446655440000',
    ]);
  });

  testWidgets('post history card exposes an in-app view action', (
    tester,
  ) async {
    var opened = false;
    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: ModerationHistoryEntryCard(
            entry: _FakeModerationRepository._entry,
            onOpenSubject: () => opened = true,
            onAppeal: () {},
            onCopyAddress: () {},
            onCopyReference: () {},
          ),
        ),
      ),
    );

    await tester.tap(find.text('View post'));

    expect(opened, isTrue);
  });

  testWidgets('post history view action opens the post thread route', (
    tester,
  ) async {
    final repository = _FakeModerationRepository();
    final router = GoRouter(
      initialLocation: '/standing',
      routes: [
        GoRoute(
          path: '/standing',
          builder: (_, _) => AccountStandingPage(account: account),
        ),
        GoRoute(
          path: '/posts/:did/:rkey',
          builder: (_, state) => Scaffold(
            body: Text(
              '${state.pathParameters['did']}/${state.pathParameters['rkey']}',
            ),
          ),
        ),
      ],
    );
    addTearDown(router.dispose);
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          accountModerationRepositoryProvider(
            account,
          ).overrideWith((_) async => repository),
        ],
        child: MaterialApp.router(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          routerConfig: router,
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('View post'));
    await tester.pumpAndSettle();

    expect(find.text('did:plc:owner/abc'), findsOneWidget);
  });

  testWidgets('does not link malformed or mismatched post snapshots', (
    tester,
  ) async {
    await _pumpPage(
      tester,
      account: account,
      repository: _FakeModerationRepository(
        entries: [
          ModerationHistoryEntry(
            caseReference: 'MOD-550e8400-e29b-41d4-a716-446655440000',
            eventType: 'decision',
            effects: const [],
            safeSnapshot: const ModerationSafeSnapshot(
              type: 'post',
              did: 'did:plc:owner',
              uri: 'at://did:plc:other/social.craftsky.feed.post/abc',
            ),
            occurredAt: DateTime.utc(2026, 9, 10),
          ),
        ],
      ),
    );

    expect(find.text('View post'), findsNothing);
  });

  testWidgets('offers appeal only for an unappealed decision entry', (
    tester,
  ) async {
    await _pumpPage(
      tester,
      account: account,
      repository: _FakeModerationRepository(
        entries: [
          _historyEntry(eventType: 'decision'),
          _historyEntry(
            eventType: 'strikeExpired',
            occurredAt: DateTime.utc(2026, 9, 11),
          ),
          _historyEntry(
            eventType: 'severeRestored',
            occurredAt: DateTime.utc(2026, 9, 12),
          ),
          _historyEntry(
            eventType: 'decision',
            appealStatus: ModerationAppealState.pending,
            occurredAt: DateTime.utc(2026, 9, 13),
          ),
        ],
      ),
    );

    expect(find.text('Appeal by email'), findsOne);
  });

  testWidgets('renders immutable moderation effect actions historically', (
    tester,
  ) async {
    await _pumpPage(
      tester,
      account: account,
      repository: _FakeModerationRepository(
        entries: [
          _historyEntry(
            eventType: 'decision',
            effects: const [
              ModerationHistoryEffect(
                type: ModerationEffectType.formalWarning,
                action: ModerationEffectAction.apply,
              ),
              ModerationHistoryEffect(
                type: ModerationEffectType.strike,
                action: ModerationEffectAction.expire,
              ),
              ModerationHistoryEffect(
                type: ModerationEffectType.visibilityHide,
                action: ModerationEffectAction.negate,
              ),
              ModerationHistoryEffect(
                type: ModerationEffectType.severeSuspension,
                action: ModerationEffectAction.restore,
              ),
            ],
          ),
        ],
      ),
    );

    expect(find.text('Formal warning · Applied'), findsOne);
    expect(find.text('Account strike · Expired'), findsOne);
    expect(find.text('Content hidden · Overturned'), findsOne);
    expect(find.text('Severe suspension · Access restored'), findsOne);
  });

  testWidgets('constrains content on wide layouts', (tester) async {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(1200, 900);
    addTearDown(tester.view.resetDevicePixelRatio);
    addTearDown(tester.view.resetPhysicalSize);
    await _pumpPage(tester, account: account);

    expect(
      tester.getSize(find.byType(ModerationHistoryEntryCard)).width,
      lessThanOrEqualTo(760),
    );
  });

  testWidgets('renders repeated event types from the same case', (
    tester,
  ) async {
    final repeated = ModerationHistoryEntry(
      caseReference: _FakeModerationRepository._entry.caseReference,
      eventType: _FakeModerationRepository._entry.eventType,
      reason: ModerationReason.spam,
      userSafeDetail: 'The consequence was applied again after review.',
      effects: const [
        ModerationHistoryEffect(
          type: ModerationEffectType.formalWarning,
          action: ModerationEffectAction.apply,
        ),
      ],
      safeSnapshot: _FakeModerationRepository._entry.safeSnapshot,
      occurredAt: DateTime.utc(2026, 9, 11, 12),
    );

    await _pumpPage(
      tester,
      account: account,
      repository: _FakeModerationRepository(
        entries: [_FakeModerationRepository._entry, repeated],
      ),
    );

    expect(find.byType(ModerationHistoryEntryCard), findsNWidgets(2));
    expect(tester.takeException(), isNull);
  });
}

ModerationHistoryEntry _historyEntry({
  required String eventType,
  List<ModerationHistoryEffect> effects = const [],
  ModerationAppealState appealStatus = ModerationAppealState.none,
  DateTime? occurredAt,
}) => ModerationHistoryEntry(
  caseReference: 'MOD-550e8400-e29b-41d4-a716-446655440000',
  eventType: eventType,
  effects: effects,
  safeSnapshot: const ModerationSafeSnapshot(
    type: 'account',
    did: 'did:plc:owner',
  ),
  appealStatus: appealStatus,
  occurredAt: occurredAt ?? DateTime.utc(2026, 9, 10),
);

Future<void> _pumpPage(
  WidgetTester tester, {
  required AccountKey account,
  List<dynamic> overrides = const [],
  _FakeModerationRepository? repository,
}) async {
  final moderationRepository = repository ?? _FakeModerationRepository();
  await tester.pumpWidget(
    ProviderScope(
      key: ValueKey(moderationRepository),
      overrides: List.from([
        accountModerationRepositoryProvider(
          account,
        ).overrideWith((_) async => moderationRepository),
        ...overrides,
      ]),
      child: MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: AccountStandingPage(account: account),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

final class _FakeModerationRepository implements ModerationRepository {
  _FakeModerationRepository({
    List<ModerationHistoryEntry>? entries,
    AccountStanding? standing,
  }) : _entries = entries ?? [_entry],
       _standing =
           standing ??
           const AccountStanding(
             activeStrikeCount: 1,
             strikeThreshold: 3,
             thresholdSuspended: false,
             severeSuspended: false,
             suspended: false,
           );

  final List<ModerationHistoryEntry> _entries;
  final AccountStanding _standing;

  @override
  Future<AccountStanding> getStanding() async => _standing;

  @override
  Future<ModerationHistoryPage> getHistory({
    String? cursor,
    int? limit,
  }) async => ModerationHistoryPage(items: _entries);

  @override
  Future<ModerationHistoryPage> getHistoryEntry(
    ModerationCaseReference reference,
  ) async => ModerationHistoryPage(items: [_entry]);

  static final _entry = ModerationHistoryEntry(
    caseReference: 'MOD-550e8400-e29b-41d4-a716-446655440000',
    eventType: 'decision',
    reason: ModerationReason.spam,
    userSafeDetail: 'This post was reviewed under the spam policy.',
    effects: [
      const ModerationHistoryEffect(
        type: ModerationEffectType.formalWarning,
        action: ModerationEffectAction.apply,
      ),
      ModerationHistoryEffect(
        type: ModerationEffectType.strike,
        action: ModerationEffectAction.expire,
        dueAt: DateTime.utc(2026, 9, 10),
      ),
      const ModerationHistoryEffect(
        type: ModerationEffectType.visibilityHide,
        action: ModerationEffectAction.negate,
      ),
    ],
    safeSnapshot: const ModerationSafeSnapshot(
      type: 'post',
      did: 'did:plc:owner',
      uri: 'at://did:plc:owner/social.craftsky.feed.post/abc',
    ),
    occurredAt: DateTime.utc(2026, 9, 10, 12),
  );
}
