import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/chat/data/chat_repository.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/pages/chat_settings_page.dart';
import 'package:craftsky_app/chat/services/chat_settings_controller.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/cupertino.dart' show CupertinoSwitch;
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

import '../fakes/recording_messenger.dart';

void main() {
  testWidgets('settings centers its initial loader and keeps refresh beside '
      'the title with Material switches on iOS', (tester) async {
    final repository = _Repository()..pendingPush = Completer<bool>();
    final controller = ChatSettingsController(
      account: AccountKey('did:plc:owner'),
      repository: repository,
      status: _status(canSetPolicy: false),
      operation: ActiveAccountOperation.withGuard(
        ownership: null,
        isCurrent: () => true,
      ),
    );
    addTearDown(controller.dispose);
    final initial = controller.load();
    await tester.pumpWidget(
      MaterialApp(
        builder: (context, child) =>
            MessengerScope(messenger: RecordingMessenger(), child: child!),
        theme: AppTheme.lightThemeData.copyWith(platform: TargetPlatform.iOS),
        localizationsDelegates: appLocalizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: MessagingSettingsView(controller: controller),
      ),
    );
    final titleSpinner = find.descendant(
      of: find.byType(AppBar),
      matching: find.byType(StitchProgressIndicator),
    );
    expect(titleSpinner, findsNothing);
    final spinner = find.byType(StitchProgressIndicator);
    expect(spinner, findsOneWidget);
    final scaffold = tester.widget<Scaffold>(find.byType(Scaffold));
    expect(
      tester.getCenter(spinner),
      tester.getCenter(find.byWidget(scaffold.body!)),
    );
    expect(find.byType(Switch), findsNothing);
    repository.pendingPush!.complete(true);
    await initial;
    await tester.pumpAndSettle();
    expect(find.byType(Switch), findsNWidgets(2));
    expect(find.byType(CupertinoSwitch), findsNothing);

    repository.pendingPush = Completer<bool>();
    final refresh = controller.load();
    await tester.pump();
    expect(titleSpinner, findsOneWidget);
    expect(tester.widget<StitchProgressIndicator>(titleSpinner).size, 16);
    expect(find.byType(StitchProgressIndicator), findsOneWidget);
    expect(find.byType(Switch), findsNWidgets(2));
    repository.pendingPush!.complete(true);
    await refresh;
    await tester.pumpAndSettle();
    expect(find.byType(StitchProgressIndicator), findsNothing);
  });

  testWidgets(
    'SETUP-001 missing authority offers explicit recovery without enabling',
    (tester) async {
      final repository = _Repository();
      final controller = ChatSettingsController(
        account: AccountKey('did:plc:owner'),
        repository: repository,
        status: _status(enabled: false, authorizationNeeded: true, push: false),
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
          builder: (context, child) =>
              MessengerScope(messenger: RecordingMessenger(), child: child!),
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: MessagingSettingsView(
            controller: controller,
            onAuthorize: authorizations.add,
          ),
        ),
      );
      expect(authorizations, isEmpty);
      expect(find.textContaining('end-to-end'), findsOneWidget);
      expect(find.text('Push notifications'), findsNothing);
      await tester.ensureVisible(find.text('Review messaging permissions'));
      await tester.tap(find.text('Review messaging permissions'));
      await tester.pumpAndSettle();
      expect(authorizations, [false]);
      expect(repository.effects, isEmpty);
      expect(controller.enabled, false);
    },
  );

  testWidgets(
    'SETUP-001 settings explains scope and preserves push across toggles',
    (
      tester,
    ) async {
      final repository = _Repository();
      final controller = ChatSettingsController(
        account: AccountKey('did:plc:owner'),
        repository: repository,
        status: _status(),
        operation: ActiveAccountOperation.withGuard(
          ownership: null,
          isCurrent: () => true,
        ),
      );
      addTearDown(controller.dispose);
      await controller.load();
      await tester.pumpWidget(
        MaterialApp(
          builder: (context, child) =>
              MessengerScope(messenger: RecordingMessenger(), child: child!),
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: MessagingSettingsView(controller: controller),
        ),
      );
      expect(find.textContaining('other apps'), findsOneWidget);
      expect(repository.effects, isEmpty);
      await tester.tap(find.byType(Switch).last);
      await tester.pumpAndSettle();
      expect(repository.effects, ['push:false']);
      await tester.ensureVisible(find.byType(Switch).first);
      await tester.tap(find.byType(Switch).first);
      await tester.pumpAndSettle();
      expect(repository.effects, ['push:false', 'connection:false']);
      expect(find.text('Push notifications'), findsNothing);
      await tester.ensureVisible(find.byType(Switch));
      await tester.tap(find.byType(Switch));
      await tester.pumpAndSettle();
      expect(repository.effects.last, 'connection:true');
      expect(tester.widget<Switch>(find.byType(Switch).last).value, false);
      expect(tester.takeException(), isNull);
    },
  );
  test(
    'UT-011 shared policy and local push change only by explicit action',
    () async {
      final repository = _Repository();
      var current = true;
      var changes = 0;
      final controller = ChatSettingsController(
        account: AccountKey('did:plc:owner'),
        repository: repository,
        status: _status(),
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
      expect(changes, 2);
      await controller.setEnabled(enabled: true);
      expect(controller.policy?.value, ChatIncomingPolicy.nobody);
      expect(controller.pushEnabled, false);
      expect(repository.effects.last, 'connection:true');
      current = false;
      await controller.setPolicy(ChatIncomingPolicy.everyone);
      expect(repository.effects, hasLength(4));
    },
  );
  test(
    'ROUTE-001 enablement losing permission does not publish ready access',
    () async {
      final repository = _Repository()
        ..pushFailure = const ApiBadRequest('chat_authorization_required');
      final published = <ChatStatus>[];
      final controller = ChatSettingsController(
        account: AccountKey('did:plc:owner'),
        repository: repository,
        status: _status(enabled: false),
        operation: ActiveAccountOperation.withGuard(
          ownership: null,
          isCurrent: () => true,
        ),
        onConnectionChanged: published.add,
      );
      addTearDown(controller.dispose);
      await controller.load();
      await controller.setEnabled(enabled: true);
      expect(controller.enabled, true);
      expect(controller.needsAuthorization, true);
      expect(published.single.state, ChatConnectionState.authorizationNeeded);
      expect(published.single.canRead, false);
      expect(repository.effects, ['connection:true']);
    },
  );

  for (final dispose in [false, true]) {
    test(
      'ROUTE-001 pending enablement discards stale owner results '
      'dispose=$dispose',
      () async {
        final pending = Completer<ChatStatus>();
        final repository = _Repository()..pendingConnection = pending;
        var current = true;
        final published = <ChatStatus>[];
        final controller = ChatSettingsController(
          account: AccountKey('did:plc:owner'),
          repository: repository,
          status: _status(enabled: false),
          operation: ActiveAccountOperation.withGuard(
            ownership: null,
            isCurrent: () => current,
          ),
          onConnectionChanged: published.add,
        );
        if (!dispose) addTearDown(controller.dispose);
        await controller.load();
        final enable = controller.setEnabled(enabled: true);
        await Future<void>.delayed(Duration.zero);
        expect(repository.effects, ['connection:true']);
        if (dispose) {
          controller.dispose();
        } else {
          current = false;
        }
        pending.complete(_status());
        await enable;
        expect(published, isEmpty);
        expect(controller.enabled, false);
        expect(controller.policy, isNull);
        expect(repository.pushReads, 0);
        expect(repository.effects, ['connection:true']);
      },
    );
  }

  test(
    'SETUP-001 denied enablement stays off with explicit recovery',
    () async {
      final repository = _Repository()
        ..connectionFailure = const ApiBadRequest(
          'chat_authorization_required',
        );
      final controller = ChatSettingsController(
        account: AccountKey('did:plc:owner'),
        repository: repository,
        status: _status(enabled: false),
        operation: ActiveAccountOperation.withGuard(
          ownership: null,
          isCurrent: () => true,
        ),
      );
      addTearDown(controller.dispose);
      await controller.load();
      await controller.setEnabled(enabled: true);
      expect(controller.enabled, false);
      expect(controller.needsAuthorization, true);
      expect(controller.failed, true);
      expect(controller.policy, isNull);
      expect(repository.pushReads, 0);
      expect(repository.effects, ['connection:true']);
    },
  );
}

class _Repository implements ChatRepository {
  final effects = <String>[];
  Completer<bool>? pendingPush;
  Exception? pushFailure;
  Exception? connectionFailure;
  Completer<ChatStatus>? pendingConnection;
  int pushReads = 0;
  bool push = true;
  ChatIncomingPolicy incoming = ChatIncomingPolicy.following;
  @override
  Future<bool> pushPreference() async {
    pushReads++;
    if (pushFailure case final failure?) throw failure;
    if (pendingPush case final pending?) return await pending.future;
    return push;
  }

  @override
  Future<ChatPolicySetting> incomingPolicy() async => ChatPolicySetting.fromMap(
    {'allowIncoming': incoming.name, 'persisted': true},
  );
  @override
  Future<bool> setPushPreference({required bool enabled}) async {
    effects.add('push:$enabled');
    push = enabled;
    return enabled;
  }

  @override
  Future<ChatPolicySetting> setIncomingPolicy(ChatIncomingPolicy value) async {
    effects.add('policy:${value.name}');
    incoming = value;
    return ChatPolicySetting.fromMap({
      'allowIncoming': value.name,
      'persisted': true,
    });
  }

  @override
  Future<ChatStatus> setConnection({required bool enabled}) async {
    effects.add('connection:$enabled');
    if (connectionFailure case final failure?) throw failure;
    if (pendingConnection case final pending?) return await pending.future;
    return _status(enabled: enabled, push: push);
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => throw UnimplementedError();
}

ChatStatus _status({
  bool enabled = true,
  bool canSetPolicy = true,
  bool authorizationNeeded = false,
  bool push = true,
}) => ChatStatus.fromMap({
  'generation': 1,
  'enabled': enabled,
  'pushEnabled': push,
  'state': authorizationNeeded
      ? 'authorizationNeeded'
      : enabled
      ? 'ready'
      : 'disabled',
  'capabilities': {
    'canRead': enabled,
    'canSend': enabled,
    'canReport': enabled,
    'canSetPolicy': enabled && canSetPolicy,
  },
});
