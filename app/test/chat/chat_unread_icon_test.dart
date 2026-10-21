import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/chat/widgets/chat_unread_icon.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  testWidgets(
    'UT-004 shell badge uses filtered accepted count, excluding requests',
    (tester) async {
      final account = AccountKey('did:plc:owner');
      var accepted = 100;
      final storage = _Storage(
        SessionRegistry.empty().upsertAndActivate(
          token: 'synthetic',
          did: account.did.value,
          handle: 'owner.test',
        ),
      );
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(storage),
            chatUnreadCountsProvider(account).overrideWith(
              (ref) async => ChatUnreadCounts.fromMap({
                'accepted': accepted,
                'requests': 17,
              }),
            ),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const Scaffold(
              body: ChatUnreadIcon(icon: Icons.chat_outlined),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('99+'), findsOneWidget);
      expect(find.text('17'), findsNothing);
      final container = ProviderScope.containerOf(
        tester.element(find.byType(ChatUnreadIcon)),
      );
      accepted = 0;
      container.invalidate(chatUnreadCountsProvider(account));
      await tester.pumpAndSettle();
      expect(find.byType(Badge), findsNothing);
      expect(tester.takeException(), isNull);
    },
  );
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
