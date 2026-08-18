import 'dart:async';

import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/services/notification_delivery_dedupe_store.dart';
import 'package:craftsky_app/notifications/services/notification_delivery_envelope.dart';
import 'package:craftsky_app/notifications/services/notification_presentation_eligibility.dart';

// Public constructor parameter names intentionally do not expose private
// implementation field names.
// ignore_for_file: prefer_initializing_formals

final class NotificationPresentation {
  const NotificationPresentation({
    required this.id,
    required this.tag,
    required this.title,
    required this.body,
    required this.payload,
  });

  final int id;
  final String tag;
  final String title;
  final String body;
  final String payload;
}

abstract interface class NotificationPresentationGateway {
  Future<void> initialize({required void Function(String? payload) onOpen});
  Future<void> present(NotificationPresentation presentation);
  Future<String?> takeInitialOpenPayload();
}

final class NotificationLocalPresenter {
  NotificationLocalPresenter({
    required NotificationPresentationGateway gateway,
    required NotificationDeliveryDedupeStore dedupe,
    required NotificationPresentationEligibility eligibility,
  }) : _gateway = gateway,
       _dedupe = dedupe,
       _eligibility = eligibility;

  /// A fixed type-scoped integer paired with the full UUID Android string tag.
  /// Distinct notifications therefore never rely on a truncated UUID hash.
  static const int androidTypeId = 0x43534b59;

  final NotificationPresentationGateway _gateway;
  final NotificationDeliveryDedupeStore _dedupe;
  final NotificationPresentationEligibility _eligibility;
  final _opened = StreamController<NotificationOpenAttempt>.broadcast();
  bool _initialized = false;
  bool _disposed = false;

  Stream<NotificationOpenAttempt> get openedNotifications => _opened.stream;

  Future<void> initialize() async {
    if (_initialized || _disposed) return;
    _initialized = true;
    await _gateway.initialize(
      onOpen: (payload) => unawaited(_receiveOpen(payload)),
    );
  }

  Future<void> present(NotificationDeliveryEnvelope envelope) async {
    final firstPresentation = await _claim(
      envelope,
      NotificationDeliveryStage.presented,
    );
    if (!firstPresentation) return;

    // The persisted claim deliberately precedes the OS call. This suppresses
    // ordinary retries but is not atomic with Android presentation: a crash or
    // plugin failure in this gap may lose the local banner.
    await _gateway.present(
      NotificationPresentation(
        id: androidTypeId,
        tag: envelope.androidTag,
        title: envelope.title,
        body: envelope.body,
        payload: envelope.localOpenPayload,
      ),
    );
  }

  Future<bool> claimForegroundEffect(
    NotificationDeliveryEnvelope envelope,
  ) => _claim(envelope, NotificationDeliveryStage.foregroundEffectEmitted);

  Future<NotificationOpenAttempt?> claimProviderOpen(
    NotificationDeliveryEnvelope envelope,
  ) async => await _claim(envelope, NotificationDeliveryStage.opened)
      ? envelope.openAttempt
      : null;

  Future<NotificationOpenAttempt?> takeInitialOpen() async {
    final payload = await _gateway.takeInitialOpenPayload();
    return _claimOpen(
      payload,
      source: NotificationOpenSource.initialOpen,
    );
  }

  Future<void> _receiveOpen(String? payload) async {
    final attempt = await _claimOpen(payload);
    if (!_disposed && attempt != null) _opened.add(attempt);
  }

  Future<NotificationOpenAttempt?> _claimOpen(
    String? payload, {
    NotificationOpenSource source = NotificationOpenSource.backgroundOpen,
  }) async {
    final envelope = NotificationDeliveryEnvelope.tryParseLocalPayload(
      payload,
      source: source,
    );
    if (envelope == null ||
        !await _claim(envelope, NotificationDeliveryStage.opened)) {
      return null;
    }
    return envelope.openAttempt;
  }

  Future<bool> _claim(
    NotificationDeliveryEnvelope envelope,
    NotificationDeliveryStage stage,
  ) async {
    final account = envelope.openAttempt.accountSubscriptionId!;
    if (!await _eligibility.allows(account)) return false;
    final claimed = await _dedupe.claim(
      notificationId: envelope.notificationId,
      accountPartition: account.wireValue,
      stage: stage,
    );
    if (!claimed) return false;

    // Account removal and a background delivery can overlap while SQLite is
    // performing the persisted compare-and-set. Re-read the authoritative
    // secure registry after that await before releasing any visible effect.
    return _eligibility.allows(account);
  }

  Future<void> dispose() async {
    if (_disposed) return;
    _disposed = true;
    await _opened.close();
  }
}
