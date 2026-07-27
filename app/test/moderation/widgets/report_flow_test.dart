import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/moderation/widgets/report_flow.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

class _RecordingNavigatorObserver extends NavigatorObserver {
  final pushedRoutes = <Route<dynamic>>[];

  @override
  void didPush(Route<dynamic> route, Route<dynamic>? previousRoute) {
    super.didPush(route, previousRoute);
    pushedRoutes.add(route);
  }
}

Post _post() => Post(
  uri: 'at://did:plc:bob/social.craftsky.feed.post/report',
  cid: 'bafy_report',
  rkey: 'report',
  text: 'Reported post',
  tags: const [],
  createdAt: DateTime.utc(2026, 5, 31, 12),
  indexedAt: DateTime.utc(2026, 5, 31, 12, 1),
  author: PostAuthor(did: 'did:plc:bob', handle: 'bob.craftsky.social'),
  likeCount: 0,
  repostCount: 0,
  replyCount: 0,
  viewerHasLiked: false,
  viewerHasReposted: false,
  viewerHasSaved: false,
);

Future<void> _pumpNestedNavigatorHarness(
  WidgetTester tester, {
  required void Function(BuildContext context, WidgetRef ref) showSheet,
  required _RecordingNavigatorObserver rootObserver,
  required _RecordingNavigatorObserver nestedObserver,
}) async {
  await tester.pumpWidget(
    ProviderScope(
      child: MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        navigatorObservers: [rootObserver],
        home: Navigator(
          observers: [nestedObserver],
          onGenerateRoute: (_) => MaterialPageRoute<void>(
            builder: (_) => Consumer(
              builder: (context, ref, _) => Scaffold(
                body: Center(
                  child: FilledButton(
                    onPressed: () => showSheet(context, ref),
                    child: const Text('Open report sheet'),
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    ),
  );
}

void main() {
  group('report flow', () {
    testWidgets(
      'showPostReportSheet presents the sheet on the root navigator',
      (
        tester,
      ) async {
        final rootObserver = _RecordingNavigatorObserver();
        final nestedObserver = _RecordingNavigatorObserver();

        await _pumpNestedNavigatorHarness(
          tester,
          rootObserver: rootObserver,
          nestedObserver: nestedObserver,
          showSheet: (context, ref) =>
              showPostReportSheet(context, ref, _post()),
        );
        rootObserver.pushedRoutes.clear();
        nestedObserver.pushedRoutes.clear();

        await tester.tap(find.text('Open report sheet'));
        await tester.pumpAndSettle();

        expect(find.text('Report post'), findsOneWidget);
        expect(rootObserver.pushedRoutes, hasLength(1));
        expect(
          rootObserver.pushedRoutes.single,
          isA<MaterialPageRoute<void>>(),
        );
        expect(
          (rootObserver.pushedRoutes.single as MaterialPageRoute<void>)
              .fullscreenDialog,
          isTrue,
        );
        expect(nestedObserver.pushedRoutes, isEmpty);
      },
    );

    testWidgets(
      'showProfileReportSheet presents the sheet on the root navigator',
      (tester) async {
        final rootObserver = _RecordingNavigatorObserver();
        final nestedObserver = _RecordingNavigatorObserver();

        await _pumpNestedNavigatorHarness(
          tester,
          rootObserver: rootObserver,
          nestedObserver: nestedObserver,
          showSheet: (context, ref) => showProfileReportSheet(
            context,
            ref,
            'bob.craftsky.social',
          ),
        );
        rootObserver.pushedRoutes.clear();
        nestedObserver.pushedRoutes.clear();

        await tester.tap(find.text('Open report sheet'));
        await tester.pumpAndSettle();

        expect(find.text('Report profile'), findsOneWidget);
        expect(rootObserver.pushedRoutes, hasLength(1));
        expect(
          rootObserver.pushedRoutes.single,
          isA<MaterialPageRoute<void>>(),
        );
        expect(
          (rootObserver.pushedRoutes.single as MaterialPageRoute<void>)
              .fullscreenDialog,
          isTrue,
        );
        expect(nestedObserver.pushedRoutes, isEmpty);
      },
    );
  });
}
