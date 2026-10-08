import 'dart:convert';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/active_account_identity_provider.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/providers/business_event_detail_provider.dart';
import 'package:craftsky_app/business/providers/profile_business_events_provider.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/settings/models/settings_identity.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/diagnostic_summary.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/widgets/post_summary.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:logging/logging.dart';

void main() {
  test('UT-014 summary ignores UI placeholders that are not identifiers', () {
    final summary = const SettingsIdentity(
      primaryLabel: 'PRIVATE_LABEL',
      handleLabel: 'PRIVATE_HANDLE_PLACEHOLDER',
      avatarSeed: 'PRIVATE_AVATAR_SEED',
      avatarUrl: 'https://example.invalid/PRIVATE_URL',
      customisation: ProfileCustomisation.defaults,
    ).diagnosticSummary(publicWorkflow: true);
    expect(summary.selectedFields.containsKey('actorDid'), isFalse);
    expect(summary.selectedFields.containsKey('handle'), isFalse);
    expect(jsonEncode(summary.selectedFields), isNot(contains('PRIVATE_')));
  });
  test(
    'UT-014 public summaries retain approved references '
    'and coarse state without model serialization',
    () {
      final auth = SignedIn(did: 'did:plc:alice', handle: 'alice.test');
      const settings = SettingsIdentity(
        primaryLabel: 'private display canary',
        handleLabel: '@alice.test',
        avatarSeed: 'did:plc:alice',
        avatarUrl: 'https://cdn.example/private?token=secret',
        customisation: ProfileCustomisation.defaults,
      );
      final identity = ActiveAccountIdentity(
        lease: AccountSessionLease(
          account: AccountKey('did:plc:alice'),
          sessionGeneration: 4,
        ),
        profile: Profile(
          did: 'did:plc:alice',
          handle: 'alice.test',
          displayName: 'private display canary',
          crafts: const [],
        ),
      );
      final post = PostSummaryData(
        state: PostSummaryState.visible,
        author: PostAuthor(did: 'did:plc:alice', handle: 'alice.test'),
        text: 'private unpublished canary',
        projectTitle: 'private title canary',
      );
      final event = BusinessEvent(
        did: 'did:plc:alice',
        rkey: '3m4event',
        uri: 'at://did:plc:alice/social.craftsky.business.event/3m4event',
        cid: 'bafy-current',
        name: 'private event canary',
        startsAt: DateTime.utc(2026),
        endsAt: DateTime.utc(2027),
        roles: const [],
        status: const BusinessOpenValue(value: 'scheduled', known: true),
        isAllDay: false,
        createdAt: DateTime.utc(2026),
        past: false,
        publicSuppressionReasons: const [],
        upcomingExclusionReasons: const [],
      );
      final detail = BusinessEventDetailAvailable(event);
      final list = BusinessEventListState(
        items: [event],
        cursor: 'private cursor canary',
        refreshError: StateError('private refresh canary'),
      );
      final public = [
        auth.diagnosticSummary(publicWorkflow: true),
        settings.diagnosticSummary(publicWorkflow: true),
        identity.diagnosticSummary(publicWorkflow: true),
        post.diagnosticSummary(publicWorkflow: true),
        event.diagnosticSummary(publicWorkflow: true),
        detail.diagnosticSummary(publicWorkflow: true),
      ];
      for (final summary in public) {
        final record = selectDiagnosticRecord(
          LogRecord(
            Level.WARNING,
            'Operation failed',
            'Summary',
            null,
            null,
            null,
            DiagnosticMessage(
              'Operation failed',
              context: ReportContext(
                feature: 'Summary',
                operation: 'summary',
                classification: 'summary',
                workflow: summary,
              ),
            ),
          ),
        );
        expect(jsonEncode(record), contains('did:plc:alice'));
        expect(jsonEncode(record), isNot(contains('private')));
      }
      expect(
        event.diagnosticSummary(publicWorkflow: true).selectedFields,
        containsPair('recordUri', event.uri.toString()),
      );
      expect(
        list.diagnosticSummary(publicWorkflow: true).selectedFields,
        containsPair('itemCount', 1),
      );
      expect(
        list.diagnosticSummary(publicWorkflow: true).selectedFields,
        containsPair('hasMore', true),
      );
      final private = [
        auth.diagnosticSummary(),
        settings.diagnosticSummary(),
        identity.diagnosticSummary(),
        post.diagnosticSummary(),
        event.diagnosticSummary(),
        detail.diagnosticSummary(),
        list.diagnosticSummary(),
      ];
      for (final summary in private) {
        expect(jsonEncode(summary.selectedFields), isNot(contains('did:')));
        expect(
          jsonEncode(summary.selectedFields),
          isNot(contains('alice.test')),
        );
        expect(jsonEncode(summary.selectedFields), isNot(contains('private')));
      }
      expect(auth.toString(), 'SignedIn(<redacted>)');
      expect(post.toString(), 'PostSummaryData(<redacted>)');
    },
  );
}
