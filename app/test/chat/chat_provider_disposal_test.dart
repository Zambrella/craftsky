import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:dio/dio.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  for (final boundary in ['switch', 'dispose']) {
    testWidgets('pending shell chat status closes safely on $boundary', (
      tester,
    ) async {
      final alice = AccountKey('did:plc:alice');
      final bob = AccountKey('did:plc:bob');
      final registry = SessionRegistry.empty()
          .upsertAndActivate(token: 'b', did: bob.did.value, handle: 'bob.test')
          .upsertAndActivate(
            token: 'a',
            did: alice.did.value,
            handle: 'alice.test',
          );
      final delayed = Completer<Dio>();
      final dio = Dio(BaseOptions(baseUrl: 'https://appview.test'));
      var requests = 0;
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (options, handler) {
            requests++;
            handler.next(options);
          },
        ),
      );
      DioAdapter(dio: dio).onGet(
        '/v1/chat/status',
        (server) => server.reply(200, {
          'enabled': false,
          'pushEnabled': false,
          'generation': 0,
          'state': 'disabled',
          'capabilities': {
            'canRead': false,
            'canSend': false,
            'canReport': false,
            'canSetPolicy': false,
          },
        }),
      );
      final container = ProviderContainer.test(
        retry: (_, _) => null,
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _Storage(registry),
          ),
          accountDioProvider.overrideWith((ref, account) => delayed.future),
        ],
      );
      await container.read(sessionRegistryProvider.future);
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: Consumer(
            builder: (context, ref, child) {
              final account = ref
                  .watch(sessionRegistryProvider)
                  .value
                  ?.activeLease
                  ?.session
                  .account;
              if (account != null) {
                ref.watch(displayedChatUnreadCountsProvider(account));
              }
              return const SizedBox.shrink();
            },
          ),
        ),
      );
      await tester.pump();
      if (boundary == 'switch') {
        await container
            .read(sessionRegistryProvider.notifier)
            .activate(registry.leaseFor(bob)!);
        await tester.pump();
      } else {
        await tester.pumpWidget(const SizedBox.shrink());
        container.dispose();
      }
      delayed.complete(dio);
      await tester.pumpAndSettle();
      expect(tester.takeException(), isNull);
      expect(requests, boundary == 'switch' ? 1 : 0);
      await tester.pumpWidget(const SizedBox.shrink());
      dio.close(force: true);
    });
  }
}

class _Storage implements SessionRegistryStorage {
  new(this.value);
  SessionRegistry value;
  @override
  Future<SessionRegistry> read() async => value;
  @override
  Future<void> write(SessionRegistry value) async => this.value = value;
}
