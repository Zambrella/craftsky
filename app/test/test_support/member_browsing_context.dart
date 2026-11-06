import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/shared/browsing/browsing_context.dart';
import 'package:craftsky_app/shared/browsing/browsing_provider.dart';
import 'package:flutter_riverpod/misc.dart' show Override;

export 'package:craftsky_app/shared/browsing/browsing_provider.dart'
    show browsingContextProvider;

/// An explicit member read audience for isolated repository tests. Full
/// account-transition tests use the real registry and do not use this fixture.
class TestMemberBrowsingContext extends BrowsingContextNotifier {
  @override
  BrowsingContext build() => MemberBrowsingContext(
    lease: ActiveAccountLease(
      session: AccountSessionLease(
        account: AccountKey('did:plc:test'),
        sessionGeneration: 1,
      ),
      activationGeneration: 1,
    ),
  );
}

final testMemberBrowsingAudience = MemberBrowsingContext(
  lease: ActiveAccountLease(
    session: AccountSessionLease(
      account: AccountKey('did:plc:test'),
      sessionGeneration: 0,
    ),
    activationGeneration: 0,
  ),
);
final Override testMemberBrowsingOverride = browsingContextProvider
    .overrideWithBuild((ref, notifier) => testMemberBrowsingAudience);

// Isolated business read tests use the same explicit viewer as their target.
Override memberBrowsingOverrideFor(String did) =>
    browsingContextProvider.overrideWithBuild(
      (ref, notifier) => MemberBrowsingContext(
        lease: ActiveAccountLease(
          session: AccountSessionLease(
            account: AccountKey(did),
            sessionGeneration: 0,
          ),
          activationGeneration: 0,
        ),
      ),
    );

const testGuestBrowsingAudience = GuestBrowsingContext(epoch: 0);
final Override testGuestBrowsingOverride = browsingContextProvider
    .overrideWithBuild((ref, notifier) => testGuestBrowsingAudience);
