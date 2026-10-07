import 'dart:async';
import 'dart:convert';
import 'package:sentry_flutter/sentry_flutter.dart';

class SerializedTransport implements Transport {
  final payloads = <String>[];
  final issueReceived = Completer<void>();
  @override
  Future<SentryId?> send(SentryEnvelope envelope) async {
    for (final item in envelope.items) {
      final payload = utf8.decode(await item.dataFactory());
      payloads.add(payload);
      if (payload.contains('"exception"') && !issueReceived.isCompleted) {
        issueReceived.complete();
      }
    }
    return envelope.header.eventId;
  }
}
