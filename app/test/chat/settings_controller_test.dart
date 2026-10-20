import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/chat/data/chat_repository.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/pages/chat_settings_page.dart';
import 'package:craftsky_app/chat/services/chat_settings_controller.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  testWidgets(
    'UT-011 permission recovery requires the explicit consent disclosure',
    (tester) async {
      final repository = _Repository();
      final controller = ChatSettingsController(
        account: AccountKey('did:plc:owner'),
        repository: repository,
        canSetPolicy: false,
        operation: ActiveAccountOperation.withGuard(
          ownership: null,
          isCurrent: () => true,
        ),
      );
      addTearDown(controller.dispose);
      await controller.load();
      final authorizations = <bool>[];
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: MessagingSettingsView(
            controller: controller,
            onAuthorize: authorizations.add,
          ),
        ),
      );
      await tester.ensureVisible(find.text('Review messaging permissions'));
      await tester.tap(find.text('Review messaging permissions'));
      await tester.pumpAndSettle();
      expect(authorizations, isEmpty);
      expect(find.textContaining('end-to-end'), findsOneWidget);
      await tester.ensureVisible(find.text('Continue with Bluesky'));
      await tester.tap(find.text('Continue with Bluesky'));
      await tester.pumpAndSettle();
      expect(authorizations, [true]);
      expect(repository.effects, isEmpty);
    },
  );

  testWidgets('UT-011 settings explains shared scope and confirms disconnect', (
    tester,
  ) async {
    final repository = _Repository();
    final controller = ChatSettingsController(
      account: AccountKey('did:plc:owner'),
      repository: repository,
      canSetPolicy: true,
      operation: ActiveAccountOperation.withGuard(
        ownership: null,
        isCurrent: () => true,
      ),
    );
    addTearDown(controller.dispose);
    await controller.load();
    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: appLocalizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: MessagingSettingsView(controller: controller),
      ),
    );
    expect(find.textContaining('other apps'), findsOneWidget);
    expect(repository.effects, isEmpty);
    await tester.tap(find.byType(Switch));
    await tester.pumpAndSettle();
    expect(repository.effects, ['push:false']);
    await tester.ensureVisible(find.text('Disconnect messaging'));
    await tester.tap(find.text('Disconnect messaging'));
    await tester.pumpAndSettle();
    expect(find.textContaining('local drafts'), findsOneWidget);
    await tester.tap(find.text('Cancel'));
    await tester.pumpAndSettle();
    expect(repository.effects, ['push:false']);
    await tester.tap(find.text('Disconnect messaging'));
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(FilledButton, 'Disconnect'));
    await tester.pumpAndSettle();
    expect(repository.effects.last, 'connection:false');
    expect(tester.takeException(), isNull);
  });
  test(
    'UT-011 shared policy and local push change only by explicit action',
    () async {
      final repository = _Repository();
      var current = true;
      var changes = 0;
      final controller = ChatSettingsController(
        account: AccountKey('did:plc:owner'),
        repository: repository,
        canSetPolicy: true,
        operation: ActiveAccountOperation.withGuard(
          ownership: null,
          isCurrent: () => current,
        ),
        onChanged: () async {
          changes++;
        },
      );
      addTearDown(controller.dispose);
      await controller.load();
      expect(controller.policy?.value, ChatIncomingPolicy.following);
      expect(controller.pushEnabled, true);
      expect(repository.effects, isEmpty);
      await controller.setPush(enabled: false);
      expect(repository.effects, ['push:false']);
      expect(controller.policy?.value, ChatIncomingPolicy.following);
      await controller.setPolicy(ChatIncomingPolicy.nobody);
      expect(repository.effects, ['push:false', 'policy:nobody']);
      expect(controller.policy?.persisted, true);
      expect(controller.pushEnabled, false);
      await controller.disconnect();
      expect(repository.effects.last, 'connection:false');
      expect(controller.disconnected, true);
      expect(changes, 3);
      current = false;
      await controller.setPolicy(ChatIncomingPolicy.everyone);
      expect(repository.effects, hasLength(3));
    },
  );
}

class _Repository implements ChatRepository {
  final effects = <String>[];
  @override
  Future<bool> pushPreference() async => true;
  @override
  Future<ChatPolicySetting> incomingPolicy() async => ChatPolicySetting.fromMap(
    const {'allowIncoming': 'following', 'persisted': true},
  );
  @override
  Future<bool> setPushPreference({required bool enabled}) async {
    effects.add('push:$enabled');
    return enabled;
  }

  @override
  Future<ChatPolicySetting> setIncomingPolicy(ChatIncomingPolicy value) async {
    effects.add('policy:${value.name}');
    return ChatPolicySetting.fromMap({
      'allowIncoming': value.name,
      'persisted': true,
    });
  }

  @override
  Future<ChatStatus> setConnection({required bool enabled}) async {
    effects.add('connection:$enabled');
    return ChatStatus.fromMap(const {
      'generation': 2,
      'enabled': false,
      'pushEnabled': false,
      'state': 'disabled',
      'capabilities': {
        'canRead': false,
        'canSend': false,
        'canReport': false,
        'canSetPolicy': false,
      },
    });
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => throw UnimplementedError();
}
