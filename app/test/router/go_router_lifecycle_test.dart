import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/router/router.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/auth_session_fakes.dart';
import '../test_support/session_registry_storage.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('disposing the router provider releases its router resources', () async {
    final container = ProviderContainer.test(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          InMemorySessionRegistryStorage(SessionRegistry.empty()),
        ),
        authSessionProvider.overrideWith(SignedOutAuthSession.new),
      ],
    );
    await container.read(sessionRegistryProvider.future);
    final subscription = container.listen(goRouterProvider, (_, _) {});
    final router = subscription.read();
    void listener() {}
    router.routerDelegate
      ..addListener(listener)
      ..removeListener(listener);

    subscription.close();
    await container.pump();

    expect(
      () => router.routerDelegate.addListener(listener),
      throwsFlutterError,
    );
    expect(
      () => router.routeInformationProvider.addListener(listener),
      throwsFlutterError,
    );
  });
}
