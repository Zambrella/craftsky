import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/scheduled_posts/models/schedule_time.dart';
import 'package:craftsky_app/scheduled_posts/models/scheduled_post.dart';
import 'package:craftsky_app/scheduled_posts/pages/scheduled_posts_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('AT-014 keeps management actions accessible at large text', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(320, 568);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final semantics = tester.ensureSemantics();

    await tester.pumpWidget(
      MaterialApp(
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        builder: (context, child) => MediaQuery(
          data: MediaQuery.of(
            context,
          ).copyWith(textScaler: const TextScaler.linear(2)),
          child: child!,
        ),
        home: Scaffold(
          body: ScheduledPostsPageContent(
            items: [
              _item('editable', ScheduledPostStatus.scheduled),
              _item('locked', ScheduledPostStatus.publishing),
            ],
            onRefresh: () async {},
            onEdit: (_) async {},
            onDelete: (_) async {},
          ),
        ),
      ),
    );

    expect(tester.takeException(), isNull);
    expect(find.byTooltip('Edit scheduled post'), findsOneWidget);
    expect(find.byTooltip('Delete scheduled post'), findsOneWidget);
    await tester.scrollUntilVisible(find.text('Publishing'), 100);
    final lock = find.byIcon(Icons.lock_outline);
    expect(lock, findsOneWidget);
    expect(tester.getSemantics(lock).label, contains('Publishing lock'));
    expect(
      tester.getSemantics(find.text('Publishing')).flagsCollection.isLiveRegion,
      isTrue,
    );
    semantics.dispose();
  });
}

ScheduledPostSummary _item(String id, ScheduledPostStatus status) =>
    ScheduledPostSummary(
      id: id,
      kind: ScheduledPostKind.standard,
      status: status,
      text: '$id scheduled post with a long preview that wraps safely',
      scheduledAt: ScheduledInstant(DateTime.utc(2026, 8, 2, 12)),
    );
