import 'dart:ui' show Tristate;

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/active_account_identity_provider.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/business/data/business_repository.dart';
import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/pages/event_editor_dialog.dart';
import 'package:craftsky_app/business/pages/events_settings_page.dart';
import 'package:craftsky_app/business/providers/business_repository_provider.dart';
import 'package:craftsky_app/business/services/business_time_zone_service.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/craftsky_floating_action_button.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../accessibility_test_helpers.dart';

void main() {
  setUpAll(initializeMappers);

  testWidgets('Events manager uses the hard-shadow CraftSky FAB', (
    tester,
  ) async {
    await tester.pumpWidget(
      _app(
        _Repository(
          pages: const {
            OwnerEventFilter.upcoming: [BusinessEventPage(items: [])],
            OwnerEventFilter.history: [BusinessEventPage(items: [])],
          },
        ),
      ),
    );
    await tester.pumpAndSettle();

    final fab = find.byType(CraftskyFloatingActionButton);
    expect(fab, findsOneWidget);
    expect(
      find.descendant(of: fab, matching: find.byType(ChunkyButton)),
      findsOneWidget,
    );
  });

  for (final constraint in businessAccessibilityMatrix) {
    testWidgets(
      'AT-012 REG-010 Events manager views retry and delete fit '
      '${businessConstraintLabel(constraint)}',
      (tester) async {
        await setBusinessAccessibilityConstraint(tester, constraint);
        final semantics = tester.ensureSemantics();
        final repository = _Repository(
          pages: {
            OwnerEventFilter.upcoming: [
              BusinessEventPage(
                items: [_event('upcoming')],
                cursor: 'next-page',
              ),
              StateError('offline'),
            ],
            OwnerEventFilter.history: [
              BusinessEventPage(
                items: [_event('history', status: 'cancelled')],
              ),
            ],
          },
        );
        await tester.pumpWidget(_app(repository));
        await tester.pumpAndSettle();

        expect(find.text('Upcoming'), findsOneWidget);
        expect(find.text('History'), findsOneWidget);
        expect(
          tester.getSemantics(find.text('Upcoming')).flagsCollection.isSelected,
          Tristate.isTrue,
        );
        final upcomingList = find.byKey(
          const PageStorageKey(OwnerEventFilter.upcoming),
        );
        await tester.drag(upcomingList, const Offset(0, -300));
        await tester.pump();
        final loadMore = find.text('Load more').first;
        await tester.tap(loadMore);
        await tester.pumpAndSettle();
        expect(find.text('Couldn’t load more events.'), findsOneWidget);
        expect(
          tester.getSemantics(find.widgetWithText(TextButton, 'Retry')).label,
          contains('Retry'),
        );

        await tester.tap(find.text('History'));
        await tester.pumpAndSettle();
        final manageHistory = find.byTooltip('Manage Event history');
        await tester.ensureVisible(manageHistory);
        await tester.pump();
        await tester.tap(manageHistory);
        await tester.pumpAndSettle();
        expect(find.byType(CraftskyContextMenuButton), findsWidgets);
        await tester.tap(find.text('Delete event'));
        await tester.pumpAndSettle();
        expect(find.text('Delete this event?'), findsOneWidget);
        expect(find.byType(CraftskyDialog), findsOneWidget);
        expect(
          tester
              .getSemantics(find.widgetWithText(ChunkyButton, 'Delete'))
              .label,
          contains('Delete'),
        );
        await expectKeyboardFocus(tester);
        expectNoAccessibilityLayoutException(tester);
        semantics.dispose();
      },
    );
  }

  testWidgets('AT-006 uses independent views and shows bounded diagnostics', (
    tester,
  ) async {
    final repository = _Repository(
      pages: {
        OwnerEventFilter.upcoming: [
          BusinessEventPage(
            items: [
              _event(
                'suppressed',
                publicReasons: const [
                  'record-moderated',
                  'record-moderated',
                  '',
                ],
              ),
            ],
          ),
        ],
        OwnerEventFilter.history: [
          BusinessEventPage(
            items: [
              _event(
                'cancelled',
                status: 'cancelled',
                exclusionReasons: const ['cancelled'],
              ),
              _event('unknown', status: 'independent-status'),
            ],
          ),
        ],
      },
    );
    await tester.pumpWidget(_app(repository));
    await tester.pumpAndSettle();

    expect(find.text('Event suppressed'), findsOneWidget);
    expect(find.text('This event is hidden by moderation.'), findsOneWidget);
    expect(repository.listCalls.first.filter, OwnerEventFilter.upcoming);

    await tester.tap(find.text('History'));
    await tester.pumpAndSettle();
    expect(find.text('Event cancelled'), findsOneWidget);
    expect(find.text('Event unknown'), findsOneWidget);
    expect(find.text('This event is cancelled.'), findsOneWidget);
    expect(
      repository.listCalls.map((call) => call.filter),
      containsAll([
        OwnerEventFilter.upcoming,
        OwnerEventFilter.history,
      ]),
    );
  });

  testWidgets('R12 AT-012 Events tabs follow order and activate', (
    tester,
  ) async {
    final repository = _Repository(
      pages: {
        OwnerEventFilter.upcoming: [
          BusinessEventPage(items: [_event('upcoming')]),
        ],
        OwnerEventFilter.history: [
          BusinessEventPage(
            items: [_event('history', status: 'cancelled')],
          ),
        ],
      },
    );
    await tester.pumpWidget(_app(repository));
    await tester.pumpAndSettle();

    final upcoming = find.text('Upcoming');
    final history = find.text('History');
    requestKeyboardFocus(tester, upcoming);
    await tester.pump();
    expectKeyboardFocusOn(upcoming);
    await tester.sendKeyEvent(LogicalKeyboardKey.arrowRight);
    await tester.pump();
    expectKeyboardFocusOn(history);
    await tester.sendKeyEvent(LogicalKeyboardKey.enter);
    await tester.pumpAndSettle();

    expect(
      tester.getSemantics(history).flagsCollection.isSelected,
      Tristate.isTrue,
    );
    expect(find.text('Event history'), findsOneWidget);
  });

  testWidgets('AT-006 paginates and retries without replacing confirmed rows', (
    tester,
  ) async {
    final repository = _Repository(
      pages: {
        OwnerEventFilter.upcoming: [
          BusinessEventPage(items: [_event('first')], cursor: 'opaque +/%'),
          StateError('offline'),
          BusinessEventPage(items: [_event('second')]),
        ],
        OwnerEventFilter.history: [const BusinessEventPage(items: [])],
      },
    );
    await tester.pumpWidget(_app(repository));
    await tester.pumpAndSettle();

    await tester.tap(find.text('Load more'));
    await tester.pumpAndSettle();
    expect(find.text('Event first'), findsOneWidget);
    expect(find.text('Couldn’t load more events.'), findsOneWidget);

    await tester.tap(find.text('Retry'));
    await tester.pumpAndSettle();
    expect(find.text('Event first'), findsOneWidget);
    expect(find.text('Event second'), findsOneWidget);
    expect(repository.listCalls.last.cursor, 'opaque +/%');
  });

  testWidgets('AT-006 retains History while opening and closing the editor', (
    tester,
  ) async {
    final repository = _Repository(
      pages: {
        OwnerEventFilter.upcoming: [const BusinessEventPage(items: [])],
        OwnerEventFilter.history: [
          BusinessEventPage(items: [_event('history', status: 'postponed')]),
        ],
      },
    );
    await tester.pumpWidget(_app(repository));
    await tester.pumpAndSettle();
    await tester.tap(find.text('History'));
    await tester.pumpAndSettle();

    await tester.tap(find.text('Event history'));
    await tester.pumpAndSettle();
    expect(find.byType(EventEditorDialog), findsOneWidget);
    final route = ModalRoute.of(
      tester.element(find.byType(EventEditorDialog)),
    );
    expect(route, isA<MaterialPageRoute<void>>());
    expect((route! as MaterialPageRoute<void>).fullscreenDialog, isTrue);
    expect(find.byKey(const ValueKey('event-submit')), findsOneWidget);
    expect(
      find.byKey(const Key('event-editor-bottom-safe-space')),
      findsOneWidget,
    );
    await tester.tap(find.byType(CloseButton));
    await tester.pumpAndSettle();

    expect(find.text('Event history'), findsOneWidget);
    final controller = tester.widget<TabBar>(find.byType(TabBar)).controller!;
    expect(controller.index, 1);
  });

  testWidgets(
    'AT-008 lifecycle edits use PUT and delete requires confirmation',
    (
      tester,
    ) async {
      final repository = _Repository(
        pages: {
          OwnerEventFilter.upcoming: [
            BusinessEventPage(items: [_event('managed')]),
            BusinessEventPage(items: [_event('managed')]),
            const BusinessEventPage(items: []),
          ],
          OwnerEventFilter.history: [
            const BusinessEventPage(items: []),
            const BusinessEventPage(items: []),
            const BusinessEventPage(items: []),
          ],
        },
      );
      await tester.pumpWidget(_app(repository));
      await tester.pumpAndSettle();

      await tester.tap(find.byTooltip('Manage Event managed'));
      await tester.pumpAndSettle();
      expect(find.byIcon(CraftskyIconsBold.edit), findsOneWidget);
      expect(find.byIcon(CraftskyIconsBold.delete), findsOneWidget);
      await tester.tap(find.text('Cancel event'));
      await tester.pumpAndSettle();
      expect(repository.updates.single.draft.status, 'cancelled');
      expect(repository.deletes, isEmpty);

      await tester.tap(find.text('History'));
      await tester.pumpAndSettle();
      await tester.tap(find.byTooltip('Manage Event managed'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Delete event'));
      await tester.pumpAndSettle();
      expect(find.text('Delete this event?'), findsOneWidget);
      expect(repository.deletes, isEmpty);
      await tester.tap(find.widgetWithText(ChunkyButton, 'Delete'));
      await tester.pumpAndSettle();
      expect(repository.deletes, hasLength(1));
      expect(repository.deletes.single.toString(), 'bafy-updated');
    },
  );

  testWidgets('AT-006 regular account guard exposes no management controls', (
    tester,
  ) async {
    final repository = _Repository(pages: const {});
    await tester.pumpWidget(_app(repository, accountType: AccountType.regular));
    await tester.pumpAndSettle();

    expect(
      find.text('Event management is available to business accounts.'),
      findsOneWidget,
    );
    expect(find.text('Create event'), findsNothing);
    expect(repository.listCalls, isEmpty);
  });

  testWidgets(
    'AT-008 conflict Reload keeps editor open with current CID for retry',
    (tester) async {
      final repository =
          _Repository(
              pages: {
                OwnerEventFilter.upcoming: [
                  BusinessEventPage(items: [_event('managed')]),
                ],
                OwnerEventFilter.history: [const BusinessEventPage(items: [])],
              },
            )
            ..updateErrors.add(
              const ApiBadRequest(
                'pds_record_conflict',
                details: ApiFailureDetails(statusCode: 409),
              ),
            )
            ..currentEvent = _event(
              'managed',
              cid: 'bafy-authoritative',
              name: 'Authoritative event',
            );
      await tester.pumpWidget(_app(repository));
      await tester.pumpAndSettle();

      await tester.tap(find.text('Event managed'));
      await tester.pumpAndSettle();
      await tester.enterText(
        find.byKey(const ValueKey('event-name')),
        'Stale correction',
      );
      await tester.tap(find.byKey(const ValueKey('event-submit')));
      await tester.pumpAndSettle();
      expect(
        find.text(
          'This event changed elsewhere. Reload the current event before '
          'trying again.',
        ),
        findsWidgets,
      );

      await tester.tap(
        find.descendant(
          of: find.byType(EventEditorDialog),
          matching: find.text('Reload current event'),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.byType(EventEditorDialog), findsOneWidget);
      expect(find.text('Authoritative event'), findsOneWidget);
      await tester.enterText(
        find.byKey(const ValueKey('event-name')),
        'Current correction',
      );
      await tester.tap(find.byKey(const ValueKey('event-submit')));
      await tester.pumpAndSettle();

      expect(repository.gets, 1);
      expect(repository.updates, hasLength(2));
      expect(
        repository.updates.last.expectedCid.toString(),
        'bafy-authoritative',
      );
      expect(repository.updates.last.draft.name, 'Current correction');
      expect(find.byType(EventEditorDialog), findsNothing);
    },
  );
}

Widget _app(
  _Repository repository, {
  AccountType accountType = AccountType.business,
}) => ProviderScope(
  overrides: [
    activeAccountIdentityProvider.overrideWith(
      (_) async => ActiveAccountIdentity(
        lease: AccountSessionLease(
          account: AccountKey('did:plc:owner'),
          sessionGeneration: 1,
        ),
        profile: Profile(
          did: 'did:plc:owner',
          handle: 'owner.test',
          crafts: const [],
          accountType: accountType,
        ),
      ),
    ),
    businessRepositoryProvider.overrideWithValue(repository),
    businessTimeZoneServiceProvider.overrideWithValue(
      BusinessTimeZoneService.initialized(),
    ),
  ],
  child: MaterialApp(
    theme: AppTheme.lightThemeData,
    localizationsDelegates: AppLocalizations.localizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    home: const EventsSettingsPage(),
  ),
);

BusinessEvent _event(
  String rkey, {
  String? cid,
  String? name,
  String status = 'scheduled',
  List<String> publicReasons = const [],
  List<String> exclusionReasons = const [],
}) => BusinessEvent(
  did: 'did:plc:owner',
  rkey: rkey,
  uri: 'at://did:plc:owner/social.craftsky.business.event/$rkey',
  cid: cid ?? 'bafy-$rkey',
  name: name ?? 'Event $rkey',
  startsAt: DateTime.utc(2026, 9, 5, 9),
  endsAt: DateTime.utc(2026, 9, 5, 17),
  roles: const [BusinessOpenValue(value: 'vendor', known: true)],
  mode: const BusinessOpenValue(value: 'in-person', known: true),
  status: BusinessOpenValue(
    value: status,
    known: status != 'independent-status',
  ),
  timeZone: 'UTC',
  isAllDay: false,
  createdAt: DateTime.utc(2026, 8, 30),
  past: status != 'scheduled',
  publicSuppressionReasons: publicReasons,
  upcomingExclusionReasons: exclusionReasons,
);

final class _Update {
  const _Update(this.expectedCid, this.draft);
  final Cid expectedCid;
  final BusinessEventDraft draft;
}

final class _Repository extends Fake implements BusinessRepository {
  _Repository({required this.pages});

  final Map<OwnerEventFilter, List<Object>> pages;
  final indices = <OwnerEventFilter, int>{};
  final listCalls = <({OwnerEventFilter filter, String? cursor})>[];
  final updates = <_Update>[];
  final deletes = <Cid>[];
  final updateErrors = <Exception>[];
  BusinessEvent? currentEvent;
  int gets = 0;

  @override
  Future<BusinessEventPage> listOwnerEvents(
    OwnerEventFilter filter, {
    String? cursor,
    int limit = 20,
  }) async {
    listCalls.add((filter: filter, cursor: cursor));
    final index = indices.update(
      filter,
      (value) => value + 1,
      ifAbsent: () => 0,
    );
    final result = pages[filter]![index];
    if (result is BusinessEventPage) return result;
    if (result is Exception) throw result;
    throw StateError('Unexpected result');
  }

  @override
  Future<RecordMutationResult> createEvent(BusinessEventDraft draft) async =>
      RecordMutationResult(cid: 'bafy-created');

  @override
  Future<RecordMutationResult> updateEvent(
    Did owner,
    RecordKey rkey,
    Cid expectedCid,
    BusinessEventDraft draft,
  ) async {
    updates.add(_Update(expectedCid, draft));
    if (updateErrors.isNotEmpty) throw updateErrors.removeAt(0);
    return RecordMutationResult(cid: 'bafy-updated');
  }

  @override
  Future<BusinessEvent> getEvent(Did owner, RecordKey rkey) async {
    gets++;
    return currentEvent!;
  }

  @override
  Future<RecordMutationResult> deleteEvent(
    Did owner,
    RecordKey rkey,
    Cid expectedCid,
  ) async {
    deletes.add(expectedCid);
    return RecordMutationResult(cid: expectedCid.toString());
  }
}
