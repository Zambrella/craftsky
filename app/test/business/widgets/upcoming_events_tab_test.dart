import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/business/data/business_repository.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/providers/business_repository_provider.dart';
import 'package:craftsky_app/business/providers/profile_business_events_provider.dart';
import 'package:craftsky_app/business/widgets/event_card.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/widgets/profile_tabs/profile_events_tab.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('IT-013 event list renders an accepted local preview', (
    tester,
  ) async {
    await _pump(
      tester,
      repository: _Repository([
        BusinessEventPage(
          items: [_event('accepted', name: 'Accepted', localPreview: true)],
        ),
      ]),
      child: ProfileEventsTab(
        target: _target(),
        isOwnProfile: false,
        onOpen: (_, _) {},
      ),
    );
    await tester.pump();

    expect(find.byType(Image), findsOneWidget);
    expect(find.byType(CachedNetworkImage), findsNothing);
    expect(tester.widget<Image>(find.byType(Image)).image, isA<MemoryImage>());
  });

  testWidgets('AT-009 keeps the tab visible during initial loading', (
    tester,
  ) async {
    final pending = Completer<BusinessEventPage>();
    await _pump(
      tester,
      repository: _Repository([pending]),
      child: ProfileEventsTab(
        target: _target(),
        isOwnProfile: false,
        onOpen: (_, _) {},
      ),
    );

    expect(find.byType(StitchProgressIndicator), findsOneWidget);
    expect(find.text('No upcoming events yet.'), findsNothing);
  });

  testWidgets('AT-009 initial failure remains retryable without false empty', (
    tester,
  ) async {
    final repository = _Repository([
      StateError('initial failed'),
      const BusinessEventPage(items: []),
    ]);

    await _pump(
      tester,
      repository: repository,
      child: ProfileEventsTab(
        target: _target(),
        isOwnProfile: false,
        onOpen: (_, _) {},
      ),
    );
    await tester.pump();

    expect(find.text('Upcoming events could not be loaded.'), findsOneWidget);
    expect(find.text('No upcoming events yet.'), findsNothing);
    await tester.tap(find.widgetWithText(TextButton, 'Retry'));
    await tester.pump();
    expect(find.text('No upcoming events yet.'), findsOneWidget);
  });

  testWidgets('AT-009 owner and visitor empty states expose exact controls', (
    tester,
  ) async {
    var manageCalls = 0;
    await _pump(
      tester,
      repository: _Repository([const BusinessEventPage(items: [])]),
      child: ProfileEventsTab(
        target: _target(),
        isOwnProfile: true,
        onOpen: (_, _) {},
        onManage: () => manageCalls++,
      ),
    );
    await tester.pump();
    expect(
      find.text('Add an event appearance to share what’s coming up.'),
      findsOneWidget,
    );
    await tester.tap(find.widgetWithText(TextButton, 'Manage events'));
    expect(manageCalls, 1);

    await _pump(
      tester,
      repository: _Repository([const BusinessEventPage(items: [])]),
      child: ProfileEventsTab(
        target: _target(),
        isOwnProfile: false,
        onOpen: (_, _) {},
      ),
    );
    await tester.pump();
    expect(find.text('No upcoming events yet.'), findsOneWidget);
    expect(find.text('Manage events'), findsNothing);
  });

  testWidgets('AT-009 renders cards in server order and reaches the end', (
    tester,
  ) async {
    final first = _event('first', name: 'Fibre Fair');
    final second = _event('second', name: 'Yarn Market');
    final repository = _Repository([
      BusinessEventPage(items: [first], cursor: 'opaque:next'),
      BusinessEventPage(items: [second]),
    ]);
    final opened = <String>[];

    await _pump(
      tester,
      repository: repository,
      child: ProfileEventsTab(
        target: _target(),
        isOwnProfile: false,
        onOpen: (did, rkey) => opened.add('$did/$rkey'),
      ),
    );
    await tester.pump();

    expect(
      tester
          .widgetList<EventCard>(find.byType(EventCard))
          .map((card) => card.event.name),
      ['Fibre Fair'],
    );
    expect(find.text('Sep 5, 2026'), findsOneWidget);
    final cardContext = tester.element(find.byType(EventCard));
    final expectedStart = MaterialLocalizations.of(cardContext).formatTimeOfDay(
      TimeOfDay.fromDateTime(first.startsAt.toLocal()),
    );
    expect(find.textContaining(expectedStart), findsOneWidget);
    expect(find.text('Vendor'), findsOneWidget);
    expect(find.text('In person'), findsOneWidget);
    expect(find.text('Town Hall'), findsOneWidget);
    expect(
      tester
          .widget<CachedNetworkImage>(find.byType(CachedNetworkImage))
          .imageUrl,
      'https://cdn.example/first-thumb.jpg',
    );
    expect(
      tester
          .widget<AspectRatio>(
            find.byKey(const Key('event-card-image')).first,
          )
          .aspectRatio,
      16 / 9,
    );
    expect(
      tester.getTopLeft(find.byKey(const Key('event-card-image')).first).dy,
      lessThan(tester.getTopLeft(find.text('Fibre Fair')).dy),
    );
    expect(find.byType(CraftskyCard), findsWidgets);
    expect(find.byIcon(Icons.calendar_today_outlined), findsWidgets);

    await tester.tap(find.byType(EventCard));
    expect(opened, ['did:plc:business/first']);

    final loadMore = find.widgetWithText(TextButton, 'Load more');
    await tester.scrollUntilVisible(
      loadMore,
      400,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.tap(loadMore);
    await tester.pump();
    expect(
      tester
          .widgetList<EventCard>(find.byType(EventCard))
          .map((card) => card.event.name),
      ['Fibre Fair', 'Yarn Market'],
    );
    await tester.drag(
      find.byType(CustomScrollView),
      const Offset(0, -800),
    );
    await tester.pump();
    expect(find.text('You’ve reached the end.'), findsOneWidget);
    expect(repository.cursors, [null, 'opaque:next']);
  });

  testWidgets('AT-009 incremental failure keeps confirmed cards and retry', (
    tester,
  ) async {
    final repository = _Repository([
      BusinessEventPage(
        items: [_event('confirmed', name: 'Confirmed event')],
        cursor: 'opaque:next',
      ),
      StateError('more failed'),
    ]);
    await _pump(
      tester,
      repository: repository,
      child: ProfileEventsTab(
        target: _target(),
        isOwnProfile: false,
        onOpen: (_, _) {},
      ),
    );
    await tester.pump();
    final loadMore = find.widgetWithText(TextButton, 'Load more');
    await tester.scrollUntilVisible(
      loadMore,
      400,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.tap(loadMore);
    await tester.pump();

    expect(find.text('Confirmed event'), findsOneWidget);
    expect(find.text('Couldn’t load more events.'), findsOneWidget);
    expect(find.widgetWithText(TextButton, 'Retry'), findsOneWidget);
  });

  testWidgets('AT-009 refresh failure keeps confirmed cards and feedback', (
    tester,
  ) async {
    final repository = _Repository([
      BusinessEventPage(items: [_event('confirmed', name: 'Confirmed event')]),
      StateError('refresh failed'),
    ]);
    await _pump(
      tester,
      repository: repository,
      child: ProfileEventsTab(
        target: _target(),
        isOwnProfile: false,
        onOpen: (_, _) {},
      ),
    );
    await tester.pump();
    final refresh = find.widgetWithText(TextButton, 'Refresh');
    await tester.scrollUntilVisible(
      refresh,
      400,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.tap(refresh);
    await tester.pump();

    expect(find.text('Confirmed event'), findsOneWidget);
    expect(find.text('Couldn’t refresh upcoming events.'), findsOneWidget);
  });
}

ProfileBusinessEventsTarget _target() => ProfileBusinessEventsTarget(
  account: AccountKey('did:plc:viewer'),
  owner: AtIdentifier.parse('business.example'),
);

BusinessEvent _event(
  String rkey, {
  required String name,
  bool localPreview = false,
}) => BusinessEvent(
  did: 'did:plc:business',
  rkey: rkey,
  uri: 'at://did:plc:business/social.craftsky.business.event/$rkey',
  cid: 'bafy-$rkey',
  name: name,
  startsAt: DateTime.utc(2026, 9, 5, 9),
  endsAt: DateTime.utc(2026, 9, 5, 17),
  roles: const [BusinessOpenValue(value: 'vendor', known: true)],
  mode: const BusinessOpenValue(value: 'in-person', known: true),
  status: const BusinessOpenValue(value: 'scheduled', known: true),
  timeZone: 'Europe/London',
  isAllDay: false,
  venueName: 'Town Hall',
  image: localPreview
      ? BusinessImageView.localPreview(
          cid: 'bafy-image-$rkey',
          mime: 'image/png',
          size: _transparentPng.length,
          alt: '$name poster',
          previewBytes: _transparentPng,
        )
      : BusinessImageView(
          cid: 'bafy-image-$rkey',
          mime: 'image/jpeg',
          size: 1200,
          alt: '$name poster',
          thumb: 'https://cdn.example/$rkey-thumb.jpg',
          fullsize: 'https://cdn.example/$rkey-full.jpg',
        ),
  createdAt: DateTime.utc(2026, 8, 30),
  past: false,
  publicSuppressionReasons: const [],
  upcomingExclusionReasons: const [],
);

final Uint8List _transparentPng = base64Decode(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVQIHWP4z8DwHwAF'
  'gAI/ScL5WQAAAABJRU5ErkJggg==',
);

Future<void> _pump(
  WidgetTester tester, {
  required BusinessRepository repository,
  required Widget child,
}) => tester.pumpWidget(
  ProviderScope(
    overrides: [businessRepositoryProvider.overrideWithValue(repository)],
    child: MaterialApp(
      theme: AppTheme.lightThemeData,
      locale: const Locale('en'),
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: Scaffold(body: CustomScrollView(slivers: [child])),
    ),
  ),
);

final class _Repository extends Fake implements BusinessRepository {
  _Repository(this.pages);

  final List<Object> pages;
  final List<String?> cursors = [];

  @override
  Future<BusinessEventPage> listProfileEvents(
    AtIdentifier owner, {
    String? cursor,
    int limit = 10,
  }) async {
    cursors.add(cursor);
    final result = pages.removeAt(0);
    if (result is BusinessEventPage) return result;
    if (result is Completer<BusinessEventPage>) return result.future;
    return switch (result) {
      final Exception error => throw error,
      final Error error => throw error,
      _ => throw StateError('Unexpected fake result'),
    };
  }
}
