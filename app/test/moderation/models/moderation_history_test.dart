import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/moderation/models/account_moderation.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  test('history parses the exact AppView owner-safe effect contract', () {
    final page = ModerationHistoryPageMapper.fromMap({
      'items': [
        {
          'caseReference': 'mod-550E8400-E29B-41D4-A716-446655440000',
          'eventType': 'effectChanged',
          'reason': 'future_reason',
          'userSafeDetail': 'A safe explanation.',
          'effects': [
            {
              'type': 'strike',
              'action': 'expire',
              'dueAt': '2026-09-10T12:00:00Z',
              'futureEffectField': true,
            },
            {'type': 'futureEffect', 'action': 'futureAction'},
          ],
          'safeSnapshot': {
            'type': 'post',
            'did': 'did:plc:owner',
            'uri': 'at://did:plc:owner/social.craftsky.feed.post/abc',
            'futureSnapshotField': 'ignored',
          },
          'appealStatus': 'futureAppealState',
          'occurredAt': '2026-09-10T12:00:00Z',
          'futureEntryField': 1,
        },
      ],
      'cursor': 'opaque:next',
      'futurePageField': true,
    });

    final entry = page.items.single;
    expect(entry.caseReference, 'MOD-550e8400-e29b-41d4-a716-446655440000');
    expect(entry.reason, ModerationReason.unknown);
    expect(entry.appealStatus, ModerationAppealState.unknown);
    expect(entry.effects.first.action, ModerationEffectAction.expire);
    expect(entry.effects.last.type, ModerationEffectType.unknown);
    expect(entry.effects.last.action, ModerationEffectAction.unknown);
    expect(page.cursor, 'opaque:next');
  });

  test('missing appeal status maps to no confirmed appeal', () {
    final entry = ModerationHistoryEntryMapper.fromMap({
      'caseReference': 'MOD-550e8400-e29b-41d4-a716-446655440000',
      'eventType': 'decision',
      'reason': 'spam',
      'effects': <Map<String, Object?>>[],
      'safeSnapshot': {'type': 'account', 'did': 'did:plc:owner'},
      'occurredAt': '2026-09-10T12:00:00Z',
    });

    expect(entry.appealStatus, ModerationAppealState.none);
    expect(entry.canStartAppeal, isTrue);
  });
}
