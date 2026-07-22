import 'dart:io';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_switcher_state.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/handoff_api_client_provider.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_account.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_suggestion.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_verification.dart';
import 'package:craftsky_app/notifications/models/account_subscription_id.dart';
import 'package:craftsky_app/notifications/models/foreground_notification_event.dart';
import 'package:craftsky_app/notifications/models/notification_effect.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/services/notification_routing_storage.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-014 provider diagnostics redact handoff credentials', () {
    const tokenSentinel = 'token-sentinel-private';
    const deviceSentinel = 'device-sentinel-private';
    final provider = handoffApiClientProvider(
      const HandoffClientKey(
        token: tokenSentinel,
        deviceId: deviceSentinel,
      ),
    );

    final diagnostic = '$provider ${provider.argument}';
    expect(diagnostic, isNot(contains(tokenSentinel)));
    expect(diagnostic, isNot(contains(deviceSentinel)));
  });

  test('UT-014 account and routing diagnostics redact every sentinel', () {
    const token = 'private-token-sentinel';
    const routing = 'routing_id_sentinel_123';
    const did = 'did:plc:identitysentinel';
    const handle = 'identity-sentinel.test';
    const displayName = 'Identity Sentinel';
    const avatar = 'https://identity-sentinel.test/avatar.jpg';
    var registry = SessionRegistry.empty().upsertAndActivate(
      token: token,
      did: did,
      handle: handle,
      cachedDisplayName: displayName,
      cachedAvatarUrl: avatar,
    );
    final lease = registry.leaseFor(AccountKey(did))!;
    registry = registry.saveRoutingBinding(lease, routing);
    final attempt = NotificationOpenAttempt.fromProviderData({
      'payloadVersion': '1',
      'type': 'everythingElse',
      'accountSubscriptionId': routing,
    });
    final foreground = ForegroundNotificationEvent(
      title: displayName,
      body: handle,
      openAttempt: attempt,
    );
    final resolution = NotificationRoutingStorage(
      () => registry,
    ).resolve(AccountSubscriptionId.parse(routing));
    final recipient = NotificationRecipientIdentity(
      lease: lease,
      handle: handle,
      avatarUrl: avatar,
    );
    final diagnostics = [
      registry,
      registry.sessions.values.single,
      registry.activeLease,
      lease,
      AccountKey(did),
      AccountSwitcherState.fromRegistry(registry),
      ...AccountSwitcherState.fromRegistry(registry).rows,
      attempt,
      foreground,
      resolution,
      recipient,
      NotificationBannerEffect(
        foreground,
        resolution: resolution,
        recipient: recipient,
      ),
      AccountSubscriptionId.parse(routing),
    ].join(' ');

    for (final sentinel in [token, routing, did, handle, displayName, avatar]) {
      expect(diagnostics, isNot(contains(sentinel)));
    }
  });

  test('REG-010 route diagnostics stay disabled for callback URLs', () {
    final routerSource = File('lib/router/router.dart').readAsStringSync();
    expect(routerSource, isNot(contains('debugLogDiagnostics: true')));
  });

  test('Sentry and auth secrets are not committed in app source/config', () {
    final paths = [
      'pubspec.yaml',
      ...Directory(
        'lib',
      ).listSync(recursive: true).whereType<File>().map((file) => file.path),
    ];
    final forbidden = RegExp(
      r'(SENTRY_AUTH_TOKEN\s*=|Authorization:\s*Bearer|Cookie:\s*|pds[_-]?token|appview[_-]?session[_-]?token)',
      caseSensitive: false,
    );
    final offenders = <String>[];

    for (final path in paths) {
      final text = File(path).readAsStringSync();
      if (forbidden.hasMatch(text)) offenders.add(path);
    }

    expect(offenders, isEmpty);
  });

  test('REG-002 notification source has no direct diagnostic sink', () {
    final files = Directory(
      'lib/notifications',
    ).listSync(recursive: true).whereType<File>();
    final forbiddenSink = RegExp(
      r'\b(print|debugPrint|log)\s*\(|Sentry\.|addBreadcrumb\s*\(|captureException\s*\(|analytics\.',
    );

    final offenders = [
      for (final file in files)
        if (forbiddenSink.hasMatch(file.readAsStringSync())) file.path,
    ];

    expect(offenders, isEmpty);
  });

  test('REG-002 notification stringification redacts IDs and payload copy', () {
    const routingSentinel = 'routing_sentinel';
    const titleSentinel = 'private-title-sentinel';
    const bodySentinel = 'private-body-sentinel';
    const subjectSentinel =
        'at://did:plc:subject/social.craftsky.feed.post/private-subject';
    const focusSentinel =
        'at://did:plc:source/social.craftsky.feed.post/private-focus';
    final event = NotificationOpenAttempt.fromProviderData(
      {
        'payloadVersion': '1',
        'type': 'reply',
        'accountSubscriptionId': routingSentinel,
        'subjectUri': subjectSentinel,
        'sourceUri': focusSentinel,
      },
      source: NotificationOpenSource.foregroundBanner,
    );
    final foregroundEvent = ForegroundNotificationEvent(
      title: titleSentinel,
      body: bodySentinel,
      openAttempt: event,
    );

    final diagnostics = '$event $foregroundEvent';
    for (final sentinel in [
      routingSentinel,
      subjectSentinel,
      focusSentinel,
      titleSentinel,
      bodySentinel,
    ]) {
      expect(diagnostics, isNot(contains(sentinel)));
    }
  });

  test('UT-015 Instagram diagnostics redact every private-data canary', () {
    const challenge = 'CSKY-PRIV-ATE1-CODE-X';
    const webhookBody = '{"private":"instagram-webhook-body-canary"}';
    const username = 'private.instagram.username';
    const igsid = '17841400000000000';
    const importedHandle = 'private-imported-handle';
    const metaToken = 'EAAG-private-meta-token';
    const exportPayload = 'private-export-payload';
    const upstreamResponse = 'private-upstream-response';
    final now = DateTime.utc(2026, 7, 19, 20);
    final account = InstagramAccountLink(
      state: InstagramAccountLinkState.active,
      username: username,
      discoverable: true,
      conflictPending: false,
      reactivationRequired: false,
      verifiedAt: now,
    );
    final models = <Object>[
      InstagramVerificationAttempt(
        verificationId: '00000000-0000-0000-0000-000000000001',
        state: InstagramVerificationState.pendingDm,
        expiresAt: now,
        challenge: challenge,
        dmUrl: Uri.parse('https://www.instagram.com/direct/t/synthetic'),
      ),
      account,
      InstagramAccountStatus(integrationAvailable: true, account: account),
      InstagramVerificationConfirmation(
        state: InstagramVerificationState.confirmed,
        account: account,
      ),
      const InstagramImportEntry(
        username: importedHandle,
        direction: InstagramRelationshipDirection.following,
      ),
      InstagramImportRequest(
        sourceType: InstagramImportSourceType.manual,
        retainUnmatched: false,
        entries: const [
          InstagramImportEntry(
            username: importedHandle,
            direction: InstagramRelationshipDirection.following,
          ),
        ],
      ),
      const InstagramSuggestionProfile(
        did: 'did:plc:synthetic-instagram-target',
        handle: importedHandle,
        displayName: username,
      ),
    ];
    final diagnostics = models.join(' ');
    for (final canary in [
      challenge,
      webhookBody,
      username,
      igsid,
      importedHandle,
      metaToken,
      exportPayload,
      upstreamResponse,
    ]) {
      expect(diagnostics, isNot(contains(canary)));
    }
  });

  test(
    'REG-007 Instagram source has no diagnostic sinks or private URL data',
    () {
      final files = Directory(
        'lib/instagram_migration',
      ).listSync(recursive: true).whereType<File>();
      final forbiddenSink = RegExp(
        r'\b(print|debugPrint|log)\s*\(|Sentry\.|addBreadcrumb\s*\(|captureException\s*\(|analytics\.',
      );
      final privateUrlData = RegExp(
        r'[?&](challenge|username|igsid|handle|access_token|message|body)=',
        caseSensitive: false,
      );
      final sinkOffenders = <String>[];
      final urlOffenders = <String>[];
      for (final file in files) {
        final source = file.readAsStringSync();
        if (forbiddenSink.hasMatch(source)) sinkOffenders.add(file.path);
        if (privateUrlData.hasMatch(source)) urlOffenders.add(file.path);
      }
      expect(sinkOffenders, isEmpty);
      expect(urlOffenders, isEmpty);
    },
  );
}
