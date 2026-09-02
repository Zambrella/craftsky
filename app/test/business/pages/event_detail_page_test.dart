import 'dart:async';
import 'dart:convert';

import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/business/data/business_repository.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/pages/event_detail_page.dart';
import 'package:craftsky_app/business/providers/business_event_detail_provider.dart';
import 'package:craftsky_app/business/providers/business_repository_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/moderation/models/report_result.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../accessibility_test_helpers.dart';

void main() {
  for (final constraint in businessAccessibilityMatrix) {
    testWidgets(
      'AT-012 REG-010 event detail and report fit '
      '${businessConstraintLabel(constraint)}',
      (tester) async {
        await setBusinessAccessibilityConstraint(tester, constraint);
        final semantics = tester.ensureSemantics();
        await _pump(
          tester,
          repository: _Repository(
            _event(
              eventUri: 'https://events.example/fair',
              registrationUri: 'https://tickets.example/fair',
            ),
          ),
          page: EventDetailPage(
            account: AccountKey('did:plc:viewer'),
            owner: Did.parse('did:plc:business'),
            rkey: RecordKey.parse('3m4event'),
          ),
        );
        await tester.pump();

        expect(
          tester.getSemantics(find.byType(CachedNetworkImage)).label,
          contains('Summer fibre fair poster'),
        );
        await _openReportMenu(tester);

        expect(find.text('Report event'), findsOneWidget);
        expect(
          tester.getSemantics(find.text('Spam')).label,
          contains('Spam'),
        );
        final submit = tester.widget<TextButton>(
          find.widgetWithText(TextButton, 'Submit'),
        );
        expect(submit.onPressed, isNull);
        await expectKeyboardFocus(tester);
        expectNoAccessibilityLayoutException(tester);
        semantics.dispose();
      },
    );
  }

  testWidgets('IT-013 event detail renders an accepted local preview', (
    tester,
  ) async {
    await _pump(
      tester,
      repository: _Repository(_event(localPreview: true)),
      page: EventDetailPage(
        account: AccountKey('did:plc:viewer'),
        owner: Did.parse('did:plc:business'),
        rkey: RecordKey.parse('3m4event'),
      ),
    );
    await tester.pump();

    expect(find.byType(Image), findsOneWidget);
    expect(find.byType(CachedNetworkImage), findsNothing);
    expect(tester.widget<Image>(find.byType(Image)).image, isA<MemoryImage>());
  });

  testWidgets('event app bar changes contrast when collapsed', (tester) async {
    await tester.binding.setSurfaceSize(const Size(390, 350));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    await _pump(
      tester,
      repository: _Repository(_event()),
      page: EventDetailPage(
        account: AccountKey('did:plc:viewer'),
        owner: Did.parse('did:plc:business'),
        rkey: RecordKey.parse('3m4event'),
      ),
    );
    await tester.pump();

    final expanded = tester.widget<SliverAppBar>(
      find.byKey(const Key('event-detail-app-bar')),
    );
    expect(expanded.foregroundColor, Colors.white);
    expect(expanded.systemOverlayStyle, SystemUiOverlayStyle.light);

    final scrollable = tester.state<ScrollableState>(
      find.byType(Scrollable).first,
    );
    scrollable.position.jumpTo(scrollable.position.maxScrollExtent);
    await tester.pump();

    final collapsed = tester.widget<SliverAppBar>(
      find.byKey(const Key('event-detail-app-bar')),
    );
    final theme = Theme.of(tester.element(find.byType(EventDetailPage)));
    expect(collapsed.foregroundColor, theme.colorScheme.onSurface);
    expect(collapsed.systemOverlayStyle, SystemUiOverlayStyle.dark);
  });

  testWidgets('AT-010 visitor reports a visible event with existing reasons', (
    tester,
  ) async {
    final repository = _Repository(_event());
    await _pump(
      tester,
      repository: repository,
      page: EventDetailPage(
        account: AccountKey('did:plc:viewer'),
        owner: Did.parse('did:plc:business'),
        rkey: RecordKey.parse('3m4event'),
      ),
    );
    await tester.pump();

    await _openReportMenu(tester);

    expect(find.text('Report event'), findsOneWidget);
    expect(find.text('Spam'), findsOneWidget);
    await tester.tap(find.text('Spam'));
    await tester.pump();
    await tester.tap(find.widgetWithText(TextButton, 'Submit'));
    await tester.pumpAndSettle();

    expect(repository.reportRequests, [
      (
        owner: Did.parse('did:plc:business'),
        rkey: RecordKey.parse('3m4event'),
        submission: const ReportSubmission(reasonType: 'spam'),
      ),
    ]);
    expect(find.text('Report event'), findsNothing);
    expect(find.text('Summer fibre fair'), findsOneWidget);
  });

  testWidgets(
    'IT-012 report event_not_found removes stale detail and actions',
    (
      tester,
    ) async {
      final repository = _Repository(
        _event(
          eventUri: 'https://events.example/fair',
          registrationUri: 'https://tickets.example/register',
          publicSuppressionReasons: const ['record-moderated'],
        ),
        reportResult: const ApiBadRequest(
          'event_not_found',
          details: ApiFailureDetails(statusCode: 404),
        ),
      );
      await _pump(
        tester,
        repository: repository,
        page: EventDetailPage(
          account: AccountKey('did:plc:viewer'),
          owner: Did.parse('did:plc:business'),
          rkey: RecordKey.parse('3m4event'),
        ),
      );
      await tester.pump();

      await _openReportMenu(tester);
      await tester.tap(find.text('Spam'));
      await tester.pump();
      await tester.tap(find.widgetWithText(TextButton, 'Submit'));
      await tester.pumpAndSettle();

      expect(find.text('Event unavailable'), findsOneWidget);
      expect(find.text('Summer fibre fair'), findsNothing);
      expect(find.text('Event website'), findsNothing);
      expect(find.text('Register'), findsNothing);
      expect(find.text('Report'), findsNothing);
      expect(find.text('record-moderated'), findsNothing);
      expect(find.text('Record moderated'), findsNothing);
      expect(
        find.text("Couldn't submit report. Please try again."),
        findsNothing,
      );
    },
  );

  testWidgets('AT-010 owner detail does not offer self-reporting', (
    tester,
  ) async {
    await _pump(
      tester,
      repository: _Repository(_event()),
      page: EventDetailPage(
        account: AccountKey('did:plc:business'),
        owner: Did.parse('did:plc:business'),
        rkey: RecordKey.parse('3m4event'),
      ),
    );
    await tester.pump();

    expect(find.text('Summer fibre fair'), findsOneWidget);
    expect(find.byTooltip('Report event'), findsNothing);
    expect(find.textContaining('Published'), findsOneWidget);
  });

  testWidgets('IT-009 event report keeps established pending and failure UX', (
    tester,
  ) async {
    final report = Completer<ReportResult>();
    final repository = _Repository(_event(), reportResult: report.future);
    await _pump(
      tester,
      repository: repository,
      page: EventDetailPage(
        account: AccountKey('did:plc:viewer'),
        owner: Did.parse('did:plc:business'),
        rkey: RecordKey.parse('3m4event'),
      ),
    );
    await tester.pump();

    await _openReportMenu(tester);
    await tester.tap(find.text('Spam'));
    await tester.pump();
    await tester.tap(find.widgetWithText(TextButton, 'Submit'));
    await tester.pump();

    expect(find.widgetWithText(TextButton, 'Submit'), findsNothing);
    report.completeError(
      const ApiBadRequest(
        'invalid_report_request',
        details: ApiFailureDetails(statusCode: 422),
      ),
    );
    await tester.pumpAndSettle();

    expect(
      find.text("Couldn't submit report. Please try again."),
      findsOneWidget,
    );
    expect(find.text('Report event'), findsOneWidget);
    expect(find.text('Summer fibre fair'), findsNothing);
  });

  testWidgets('IT-012 refresh event_not_found replaces stale visible detail', (
    tester,
  ) async {
    final repository = _SequenceRepository([
      _event(eventUri: 'https://events.example/fair'),
      const ApiBadRequest(
        'event_not_found',
        details: ApiFailureDetails(statusCode: 404),
      ),
    ]);
    final account = AccountKey('did:plc:viewer');
    final owner = Did.parse('did:plc:business');
    final rkey = RecordKey.parse('3m4event');
    await _pump(
      tester,
      repository: repository,
      page: EventDetailPage(account: account, owner: owner, rkey: rkey),
    );
    await tester.pump();
    expect(find.text('Summer fibre fair'), findsOneWidget);

    final container = ProviderScope.containerOf(
      tester.element(find.byType(EventDetailPage)),
    );
    container
        .read(
          businessEventDetailProvider(
            BusinessEventDetailTarget(
              account: account,
              owner: owner,
              rkey: rkey,
            ),
          ).notifier,
        )
        .retry();
    await tester.pumpAndSettle();

    expect(find.text('Event unavailable'), findsOneWidget);
    expect(find.text('Summer fibre fair'), findsNothing);
    expect(find.text('Event website'), findsNothing);
    expect(find.text('Report'), findsNothing);
  });

  testWidgets('AT-009 renders complete detail and launches exact actions', (
    tester,
  ) async {
    const eventUri = 'https://events.example/fair?source=craftsky#details';
    const registrationUri =
        'https://tickets.example/register?event=fibre%20fair';
    final repository = _Repository(
      _event(eventUri: eventUri, registrationUri: registrationUri),
    );
    final launched = <Uri>[];

    await _pump(
      tester,
      repository: repository,
      page: EventDetailPage(
        account: AccountKey('did:plc:viewer'),
        owner: Did.parse('did:plc:business'),
        rkey: RecordKey.parse('3m4event'),
        launchExternal: (uri) async {
          launched.add(uri);
          return true;
        },
        confirmExternal: (_, _) async => true,
      ),
    );
    await tester.pump();

    expect(find.text('Summer fibre fair'), findsOneWidget);
    expect(find.text('A celebration of local fibre and yarn.'), findsOneWidget);
    expect(find.text('Town Hall'), findsOneWidget);
    expect(find.text('Vendor'), findsNothing);
    expect(find.textContaining('Published'), findsNothing);
    expect(find.text('In person'), findsOneWidget);
    expect(find.text('Scheduled'), findsOneWidget);
    expect(find.text('Europe/London'), findsOneWidget);
    expect(find.text('Upcoming'), findsOneWidget);
    expect(find.byType(CraftskyCard), findsWidgets);
    expect(
      tester
          .widgetList<CraftskyCard>(find.byType(CraftskyCard))
          .last
          .clipBehavior,
      Clip.none,
    );
    expect(find.byIcon(Icons.calendar_today_outlined), findsOneWidget);
    expect(find.byIcon(Icons.schedule_outlined), findsOneWidget);
    expect(
      tester
          .widget<CachedNetworkImage>(find.byType(CachedNetworkImage))
          .imageUrl,
      'https://cdn.example/event-full.jpg',
    );
    final appBar = tester.widget<SliverAppBar>(
      find.byKey(const Key('event-detail-app-bar')),
    );
    final detailWidth = tester.getSize(find.byType(EventDetailPage)).width;
    expect(appBar.pinned, isTrue);
    expect(appBar.expandedHeight, closeTo(detailWidth * 9 / 16, 0.01));
    expect(appBar.foregroundColor, Colors.white);
    expect(appBar.systemOverlayStyle, SystemUiOverlayStyle.light);
    expect(
      find.descendant(
        of: find.byType(FlexibleSpaceBar),
        matching: find.byType(CachedNetworkImage),
      ),
      findsOneWidget,
    );

    final eventAction = find.widgetWithText(OutlinedButton, 'Event website');
    await tester.drag(find.byType(CustomScrollView), const Offset(0, -800));
    await tester.pump();
    await tester.tap(eventAction);
    await tester.pump();
    final registrationAction = find.widgetWithText(OutlinedButton, 'Register');
    await tester.tap(registrationAction);
    await tester.pump();
    expect(launched, [Uri.parse(eventUri), Uri.parse(registrationUri)]);
    expect(repository.requests, ['did:plc:business/3m4event']);
  });

  testWidgets('AT-009 omits absent optional actions without empty controls', (
    tester,
  ) async {
    await _pump(
      tester,
      repository: _Repository(_event()),
      page: EventDetailPage(
        account: AccountKey('did:plc:viewer'),
        owner: Did.parse('did:plc:business'),
        rkey: RecordKey.parse('3m4event'),
      ),
    );
    await tester.pump();

    expect(find.text('Event website'), findsNothing);
    expect(find.text('Register'), findsNothing);
    expect(find.byTooltip('Report event'), findsOneWidget);
  });

  testWidgets('IT-009 event_not_found renders no stale event or actions', (
    tester,
  ) async {
    await _pump(
      tester,
      repository: _Repository(
        const ApiBadRequest(
          'event_not_found',
          details: ApiFailureDetails(statusCode: 404),
        ),
      ),
      page: EventDetailPage(
        account: AccountKey('did:plc:viewer'),
        owner: Did.parse('did:plc:business'),
        rkey: RecordKey.parse('3m4event'),
      ),
    );
    await tester.pump();

    expect(find.text('Event unavailable'), findsOneWidget);
    expect(find.text('Summer fibre fair'), findsNothing);
    expect(find.byType(OutlinedButton), findsNothing);
  });
}

Future<void> _openReportMenu(WidgetTester tester) async {
  final menuButton = find.byTooltip('Report event');
  expect(menuButton, findsOneWidget);
  expect(find.byType(CraftskyContextMenuButton), findsOneWidget);
  await tester.tap(menuButton);
  await tester.pumpAndSettle();
  await tester.tap(find.text('Report event'));
  await tester.pumpAndSettle();
}

BusinessEvent _event({
  String? eventUri,
  String? registrationUri,
  List<String> publicSuppressionReasons = const [],
  bool localPreview = false,
}) => BusinessEvent(
  did: 'did:plc:business',
  rkey: '3m4event',
  uri: 'at://did:plc:business/social.craftsky.business.event/3m4event',
  cid: 'bafy-event',
  name: 'Summer fibre fair',
  startsAt: DateTime.utc(2026, 9, 5, 9),
  endsAt: DateTime.utc(2026, 9, 5, 17),
  roles: const [BusinessOpenValue(value: 'vendor', known: true)],
  mode: const BusinessOpenValue(value: 'in-person', known: true),
  status: const BusinessOpenValue(value: 'scheduled', known: true),
  timeZone: 'Europe/London',
  isAllDay: false,
  summary: 'A celebration of local fibre and yarn.',
  venueName: 'Town Hall',
  eventUri: eventUri,
  registrationUri: registrationUri,
  image: localPreview
      ? BusinessImageView.localPreview(
          cid: 'bafy-image',
          mime: 'image/png',
          size: _transparentPng.length,
          alt: 'Summer fibre fair poster',
          previewBytes: _transparentPng,
        )
      : BusinessImageView(
          cid: 'bafy-image',
          mime: 'image/jpeg',
          size: 2400,
          alt: 'Summer fibre fair poster',
          thumb: 'https://cdn.example/event-thumb.jpg',
          fullsize: 'https://cdn.example/event-full.jpg',
        ),
  createdAt: DateTime.utc(2026, 8, 30, 12),
  past: false,
  publicSuppressionReasons: publicSuppressionReasons,
  upcomingExclusionReasons: const [],
);

final Uint8List _transparentPng = base64Decode(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVQIHWP4z8DwHwAF'
  'gAI/ScL5WQAAAABJRU5ErkJggg==',
);

Future<void> _pump(
  WidgetTester tester, {
  required BusinessRepository repository,
  required Widget page,
}) => tester.pumpWidget(
  ProviderScope(
    overrides: [businessRepositoryProvider.overrideWithValue(repository)],
    child: MaterialApp(
      theme: AppTheme.lightThemeData,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: page,
    ),
  ),
);

final class _Repository extends Fake implements BusinessRepository {
  _Repository(this.result, {this.reportResult});

  final Object result;
  final Object? reportResult;
  final List<String> requests = [];
  final List<({Did owner, RecordKey rkey, ReportSubmission submission})>
  reportRequests = [];

  @override
  Future<BusinessEvent> getEvent(Did owner, RecordKey rkey) async {
    requests.add('$owner/$rkey');
    if (result is BusinessEvent) return result as BusinessEvent;
    return switch (result) {
      final Exception error => throw error,
      final Error error => throw error,
      _ => throw StateError('Unexpected fake result'),
    };
  }

  @override
  Future<ReportResult> reportEvent(
    Did owner,
    RecordKey rkey,
    ReportSubmission submission,
  ) async {
    reportRequests.add((owner: owner, rkey: rkey, submission: submission));
    if (reportResult case final Exception error) throw error;
    if (reportResult case final Error error) throw error;
    if (reportResult case final ReportResult result) return result;
    if (reportResult case final Future<ReportResult> result) return result;
    return const ReportResult(reportId: 'report-event-1', status: 'accepted');
  }
}

final class _SequenceRepository extends Fake implements BusinessRepository {
  _SequenceRepository(this.results);

  final List<Object> results;

  @override
  Future<BusinessEvent> getEvent(Did owner, RecordKey rkey) async {
    final result = results.removeAt(0);
    if (result case final BusinessEvent event) return event;
    if (result case final Exception error) throw error;
    if (result case final Error error) throw error;
    throw StateError('Unexpected fake result');
  }
}
