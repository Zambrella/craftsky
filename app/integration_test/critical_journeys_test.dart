import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/providers/notification_runtime_provider.dart';
import 'package:craftsky_app/profile/widgets/profile_route_presentation.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/service_status/data/announcement_dismissal_store.dart';
import 'package:craftsky_app/service_status/providers/service_status_controller.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'support/critical_journey_harness.dart';

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  configureCriticalJourneyPreferences();

  late CriticalJourneyHarness harness;

  setUp(() async {
    harness = await CriticalJourneyHarness.start();
    addTearDown(harness.close);
  });

  // IT-002 / FR-006, FR-008 / AC-009, AC-011; IT-001 real native socket corroboration.
  testWidgets(
    'status dismissal persists after actual preferences reload '
    'and fresh controller',
    (tester) async {
      harness.status.document = {
        'schemaVersion': 1,
        'mode': 'announcement',
        'revision': 'A',
        'title': 'Notice',
        'message': 'Public test notice',
      };
      ProviderContainer fresh() => ProviderContainer(
        overrides: [
          serviceStatusRepositoryProvider.overrideWithValue(
            harness.status.repository,
          ),
        ],
      );
      final first = fresh();
      final controller = first.read(serviceStatusControllerProvider.notifier)
        ..start();
      await _pumpUntil(tester, () => controller.announcement != null);
      expect(await controller.dismissAnnouncement(), isTrue);
      first.dispose();
      final preferences = await SharedPreferences.getInstance();
      await preferences.reload();
      expect(
        preferences.getString(PreferencesAnnouncementDismissalStore.key),
        'A',
      );
      final second = fresh();
      addTearDown(second.dispose);
      final restarted = second.read(serviceStatusControllerProvider.notifier)
        ..start();
      await _pumpUntil(
        tester,
        () =>
            second.read(serviceStatusControllerProvider).dismissalReady &&
            second.read(serviceStatusControllerProvider).document?.revision ==
                'A',
      );
      expect(restarted.announcement, isNull);
      harness.status.document = {...harness.status.document, 'revision': 'B'};
      await restarted.refresh();
      expect(restarted.announcement?.revision, 'B');
      harness.status.document = {
        ...harness.status.document,
        'mode': 'maintenance',
      };
      await restarted.refresh();
      expect(restarted.maintenanceActive, isTrue);
      expect(harness.status.requests.length, greaterThanOrEqualTo(4));
      for (final headers in harness.status.requests) {
        expect(headers['authorization'], isNull);
        expect(headers['cookie'], isNull);
        expect(headers['x-device-id'], isNull);
      }
    },
  );

  testWidgets(
    'authenticated read switches retained accounts and reloads account data',
    (tester) async {
      final dependencies = await harness.loadPlatformDependencies();
      expect(dependencies.packageInfo.version, isNotEmpty);
      expect(dependencies.deviceInfo.platform, anyOf('Android', 'iOS'));

      await _mountApp(tester, harness);
      await _pumpUntilFound(tester, find.text('Alice timeline'));

      expect(
        harness.server.requests,
        contains(
          isA<RecordedRequest>()
              .having((request) => request.path, 'path', '/v1/feed/timeline')
              .having(
                (request) => request.authorization,
                'authorization',
                'Bearer alice-token',
              ),
        ),
      );

      await tester.longPress(
        find.byTooltip('Switch account').hitTestable().first,
      );
      await _pumpUntilFound(tester, find.text('Bob'));
      await tester.pumpAndSettle();
      expect(find.text('@bob.test'), findsOneWidget);
      await tester.tap(find.text('Bob'));
      await _pumpUntilFound(tester, find.text('Bob timeline'));

      expect(
        harness.container
            .read(sessionRegistryProvider)
            .requireValue
            .activeDid
            ?.value,
        'did:plc:bob',
      );
      expect(find.text('Alice timeline'), findsNothing);
      expect(
        harness.server.requests,
        contains(
          isA<RecordedRequest>()
              .having((request) => request.path, 'path', '/v1/feed/timeline')
              .having(
                (request) => request.authorization,
                'authorization',
                'Bearer bob-token',
              ),
        ),
      );
    },
  );

  testWidgets(
    'composer publishes through the AppView client and updates the live feed',
    (tester) async {
      await harness.loadPlatformDependencies();
      await _mountApp(tester, harness);
      await _pumpUntilFound(tester, find.text('Alice timeline'));

      await tester.tap(find.byIcon(CraftskyIconsBold.add).hitTestable().first);
      await _pumpUntilFound(tester, find.text('Regular post'));
      await tester.tap(find.text('Regular post'));
      await _pumpUntilFound(tester, find.byType(PostComposerSheet));

      await tester.enterText(
        find.byType(EditableText).first,
        'Integration write journey',
      );
      FocusManager.instance.primaryFocus?.unfocus();
      await tester.pumpAndSettle();
      // REG-003 / FR-009, RULE-004 / AC-008, AC-012: accepted HTTP write continues under cover.
      final finishWrite = Completer<void>();
      harness.server.createGate = finishWrite.future;
      addTearDown(() {
        if (!finishWrite.isCompleted) finishWrite.complete();
      });
      await _pumpUntilFound(
        tester,
        find.byKey(const Key('post-composer-primary-action')).hitTestable(),
      );
      await tester.tap(
        find.byKey(const Key('post-composer-primary-action')).hitTestable(),
      );
      await _pumpUntil(
        tester,
        () => harness.server.requests.any(
          (request) => request.method == 'POST' && request.path == '/v1/posts',
        ),
      );
      harness.status.document = {
        'schemaVersion': 1,
        'mode': 'maintenance',
        'revision': 'maintenance-A',
        'title': 'Maintenance',
        'message': 'Public integration notice',
      };
      await harness.container
          .read(serviceStatusControllerProvider.notifier)
          .refresh();
      await tester.pump();
      expect(find.text('Maintenance'), findsOneWidget);
      expect(harness.server.createBodies, isEmpty);
      finishWrite.complete();
      await _pumpUntil(
        tester,
        () => harness.server.createBodies.isNotEmpty,
      );
      await _pumpUntilAbsent(tester, find.byType(PostComposerSheet));
      expect(find.text('Maintenance'), findsOneWidget);
      harness.status.document = {
        'schemaVersion': 1,
        'mode': 'normal',
        'revision': 'normal-B',
      };
      await harness.container
          .read(serviceStatusControllerProvider.notifier)
          .refresh();
      await _pumpUntilAbsent(tester, find.text('Maintenance'));
      await _pumpUntilFound(tester, find.text('Integration write journey'));

      expect(harness.server.createBodies, hasLength(1));
      expect(
        harness.server.createBodies.single['text'],
        'Integration write journey',
      );
      expect(harness.server.createBodies.single['langs'], ['en']);
      expect(
        harness.server.requests,
        contains(
          isA<RecordedRequest>()
              .having((request) => request.method, 'method', 'POST')
              .having((request) => request.path, 'path', '/v1/posts')
              .having(
                (request) => request.authorization,
                'authorization',
                'Bearer alice-token',
              ),
        ),
      );
    },
  );

  testWidgets(
    'notification open traverses the runtime and pushes a profile deep link',
    (tester) async {
      await harness.loadPlatformDependencies();
      await _mountApp(tester, harness);
      await _pumpUntilFound(tester, find.text('Alice timeline'));
      expect(harness.notificationService.initialized, isTrue);
      final runtime = harness.container.read(notificationRuntimeProvider);
      await runtime.start();
      await runtime.updateReadiness(
        did: AccountKey('did:plc:alice').did,
        onboarded: true,
      );
      await runtime.receiveOpen(
        NotificationOpenAttempt.fromProviderData({
          'payloadVersion': '1',
          'type': 'follow',
          'accountSubscriptionId': 'integration-alice-binding',
          'actorDid': 'did:plc:notified',
        }),
      );

      await _pumpUntilFound(tester, find.byType(ProfileRoutePresentation));
      await _pumpUntilFound(tester, find.text('Notified Maker'));
      expect(
        harness.container.read(goRouterProvider).state.uri.path,
        '/profiles/did%3Aplc%3Anotified',
      );
      expect(
        harness.container.read(sessionRegistryProvider).requireValue.activeDid,
        AccountKey('did:plc:alice').did,
      );
    },
  );
}

Future<void> _mountApp(
  WidgetTester tester,
  CriticalJourneyHarness harness,
) async {
  await tester.pumpWidget(const SizedBox.shrink());
  tester.view
    ..devicePixelRatio = 1
    ..physicalSize = const Size(402, 874);
  addTearDown(tester.view.reset);
  addTearDown(() async {
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump();
  });
  await tester.pumpWidget(harness.app);
}

Future<void> _pumpUntil(
  WidgetTester tester,
  bool Function() condition, {
  Duration timeout = const Duration(seconds: 15),
}) async {
  final deadline = DateTime.now().add(timeout);
  while (!condition() && DateTime.now().isBefore(deadline)) {
    await tester.pump(const Duration(milliseconds: 100));
  }
  expect(condition(), isTrue);
}

Future<void> _pumpUntilAbsent(
  WidgetTester tester,
  Finder finder, {
  Duration timeout = const Duration(seconds: 15),
}) => _pumpUntil(tester, () => finder.evaluate().isEmpty, timeout: timeout);

Future<void> _pumpUntilFound(
  WidgetTester tester,
  Finder finder, {
  Duration timeout = const Duration(seconds: 15),
}) async {
  final deadline = DateTime.now().add(timeout);
  while (finder.evaluate().isEmpty && DateTime.now().isBefore(deadline)) {
    await tester.pump(const Duration(milliseconds: 100));
  }
  expect(finder, findsWidgets);
}
