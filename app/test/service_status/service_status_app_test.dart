import 'dart:async';
import 'dart:io';

import 'package:craftsky_app/app.dart';
import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/auth/models/active_account_initialization.dart';
import 'package:craftsky_app/auth/providers/active_account_initialization_provider.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/initialization_error_screen.dart';
import 'package:craftsky_app/initialization_loading_screen.dart';
import 'package:craftsky_app/service_status/models/service_status_document.dart';
import 'package:craftsky_app/service_status/providers/service_status_controller.dart';
import 'package:craftsky_app/service_status/widgets/maintenance_screen.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/auth_session_fakes.dart';
import 'service_status_controller_test.dart'
    show FakeStatusRepository, maintenance;

void main() {
  // AT-001 / BR-001, FR-001, FR-005 / AC-001; UT-003 startup wiring.
  for (final signedIn in [false, true]) {
    for (final failed in [false, true]) {
      testWidgets(
        'maintenance before ${signedIn ? 'signed-in' : 'signed-out'} '
        '${failed ? 'failed' : 'pending'} initialization',
        (tester) async {
          final repository = FakeStatusRepository();
          final dependency = Completer<AppDependencies>();
          final account = Completer<ActiveAccountInitialization?>();
          var splashRemovals = 0;
          await tester.pumpWidget(
            ProviderScope(
              overrides: [
                appDependenciesProvider.overrideWith(
                  (ref) => failed
                      ? Future.error(StateError('Initialization failed'))
                      : dependency.future,
                ),
                activeAccountInitializationProvider.overrideWith(
                  (ref) => account.future,
                ),
                authSessionProvider.overrideWith(
                  signedIn ? SignedInAuthSession.new : SignedOutAuthSession.new,
                ),
                serviceStatusRepositoryProvider.overrideWithValue(repository),
              ],
              child: App(onInitializationResolved: () => splashRemovals++),
            ),
          );
          await tester.pump();
          expect(repository.requests, hasLength(1));
          repository.requests.single.complete(maintenance);
          await tester.pump();
          await tester.pump();
          expect(find.text('Maintenance'), findsOneWidget);
          expect(find.text('Public message'), findsOneWidget);
          expect(splashRemovals, 1);
          final context = tester.element(find.byType(App));
          final container = ProviderScope.containerOf(context);
          final retry = container
              .read(serviceStatusControllerProvider.notifier)
              .refresh();
          repository.requests.last.complete(
            const ServiceStatusDocument(
              mode: ServiceStatusMode.normal,
              revision: 'B',
            ),
          );
          await tester.pump();
          await retry;
          expect(find.text('Maintenance'), findsNothing);
          if (!failed) {
            expect(find.byType(InitializationLoadingScreen), findsOneWidget);
          }
          expect(splashRemovals, 1);
          await tester.pumpWidget(const SizedBox.shrink());
        },
      );
    }
  }
  // AT-005 / FR-007, NFR-001, RULE-001 / AC-010.
  for (final statusFailure in [
    const SocketException('Offline'),
    const FormatException('Proxy HTML'),
    const ApiServerError('502'),
  ]) {
    testWidgets(
      'unknown status failure retains startup retry: '
      '${statusFailure.runtimeType}',
      (tester) async {
        final repository = FakeStatusRepository();
        var initializations = 0;
        await tester.pumpWidget(
          ProviderScope(
            retry: (_, _) => null,
            overrides: [
              appDependenciesProvider.overrideWith((ref) {
                initializations++;
                return Future.error(const ApiServerError('503'));
              }),
              activeAccountInitializationProvider.overrideWith((ref) => null),
              authSessionProvider.overrideWith(SignedOutAuthSession.new),
              serviceStatusRepositoryProvider.overrideWithValue(repository),
            ],
            child: App(onInitializationResolved: () {}),
          ),
        );
        await tester.pump();
        repository.requests.single.completeError(statusFailure);
        await tester.pump();
        expect(find.byType(MaintenanceScreen), findsNothing);
        expect(find.byType(InitializationErrorScreen), findsOneWidget);
        await tester.tap(find.text('Retry'));
        await tester.pump();
        expect(initializations, 2);
        expect(find.byType(MaintenanceScreen), findsNothing);
        await tester.pumpWidget(const SizedBox.shrink());
      },
    );
  }
}
