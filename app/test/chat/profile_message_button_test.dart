import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/chat/data/chat_repository.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/chat/widgets/profile_message_button.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_icon_button.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  test(
    'IT-018 private availability provider diagnostics redact recipient routing',
    () {
      final account = AccountKey('did:plc:owner');
      final peer = Did.parse('did:plc:private-peer');
      expect(
        chatRecipientAvailabilityProvider(
          account,
          ChatRecipientId.fromDid(peer),
        ).toString(),
        isNot(contains(peer.value)),
      );
    },
  );

  testWidgets(
    'IT-003 profile availability is read-only and explicit Message opens '
    'selection',
    (tester) async {
      final account = AccountKey('did:plc:owner');
      final peer = Did.parse('did:plc:peer');
      final repository = _Repository();
      var opened = 0;
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _Storage(
                SessionRegistry.empty().upsertAndActivate(
                  token: 'synthetic',
                  did: account.did.value,
                  handle: 'owner.test',
                ),
              ),
            ),
            accountChatRepositoryProvider(account)
                .overrideWith((ref) async => repository),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: Scaffold(
              body: ProfileMessageButton(
                recipient: peer,
                onOpen: () => opened++,
                onEnable: () {},
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(repository.checks, 1);
      expect(opened, 0);
      expect(
        tester
            .widget<ChunkyIconButton>(find.byType(ChunkyIconButton))
            .onPressed,
        isNull,
      );
      final container = ProviderScope.containerOf(
        tester.element(find.byType(ProfileMessageButton)),
      );
      repository.available = true;
      container.invalidate(
        chatRecipientAvailabilityProvider(
          account,
          ChatRecipientId.fromDid(peer),
        ),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.byTooltip('Message'));
      expect(opened, 1);
      expect(repository.checks, 2);
      expect(tester.takeException(), isNull);
    },
  );
}

class _Repository implements ChatRepository {
  int checks = 0;
  bool available = false;
  @override
  Future<ChatAvailability> availability(Did recipient) async {
    checks++;
    return ChatAvailability.fromMap({'canChat': available});
  }

  @override
  Future<ChatStatus> status() async => ChatStatus.fromMap(const {
    'generation': 1,
    'enabled': true,
    'pushEnabled': true,
    'state': 'ready',
    'capabilities': {
      'canRead': true,
      'canSend': true,
      'canReport': true,
      'canSetPolicy': true,
    },
  });
  @override
  dynamic noSuchMethod(Invocation invocation) => throw UnimplementedError();
}

class _Storage implements SessionRegistryStorage {
  new(this.value);
  SessionRegistry value;
  @override
  Future<SessionRegistry> read() async => value;
  @override
  Future<void> write(SessionRegistry value) async {
    this.value = value;
  }
}
