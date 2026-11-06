import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/shared/browsing/browsing_context.dart';
import 'package:flutter_test/flutter_test.dart';

// UT-002 / FR-006, RULE-001 / AC-005, AC-006.
void main() {
  test(
    'guest capabilities permit public reads and share without an account',
    () {
      const context = GuestBrowsingContext(epoch: 1);
      expect(context.viewerDid, isNull);
      expect(context.capabilities.canReadPublic, isTrue);
      expect(context.capabilities.canShare, isTrue);
      expect(context.capabilities.canUseProjectFeedControls, isTrue);
      expect(context.capabilities.canMutate, isFalse);
      expect(context.capabilities.canReadPrivate, isFalse);
      expect(context.capabilities.canUseAutocomplete, isFalse);
      expect(context.capabilities.canViewRecentUsers, isFalse);
      expect(context.capabilities.canViewTopHashtags, isFalse);
      expect(context.capabilities.canUseAdvancedSearch, isFalse);
    },
  );
  test('member capabilities retain the real immutable account lease', () {
    final lease = ActiveAccountLease(
      session: AccountSessionLease(
        account: AccountKey('did:plc:alice'),
        sessionGeneration: 1,
      ),
      activationGeneration: 2,
    );
    final context = MemberBrowsingContext(lease: lease);
    expect(context.viewerDid, lease.session.account.did);
    expect(context.capabilities.canMutate, isTrue);
    expect(context.capabilities.canReadPrivate, isTrue);
    expect(context.capabilities.canUseAutocomplete, isTrue);
    expect(context.capabilities.canViewRecentUsers, isTrue);
    expect(context.capabilities.canViewTopHashtags, isTrue);
    expect(context, MemberBrowsingContext(lease: lease));
    expect(context, isNot(const GuestBrowsingContext(epoch: 1)));
  });
  test('loading does not grant guest or member read capabilities', () {
    const context = BrowsingLoading(epoch: 1);
    expect(context.viewerDid, isNull);
    expect(context.capabilities.canReadPublic, isFalse);
    expect(context.capabilities.canMutate, isFalse);
    expect(
      const GuestBrowsingContext(epoch: 1),
      isNot(const GuestBrowsingContext(epoch: 2)),
    );
  });
}
