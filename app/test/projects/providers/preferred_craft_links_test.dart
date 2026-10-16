import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/projects/providers/preferred_craft_links_provider.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

class _Profile extends UserProfile {
  new(this.crafts);
  final List<String> crafts;
  @override
  Future<Profile> build(Did did) async =>
      Profile(did: did.value, handle: 'test.test', crafts: crafts);
}

void main() {
  final lease = ActiveAccountLease(
    session: AccountSessionLease(
      account: AccountKey('did:plc:test'),
      sessionGeneration: 1,
    ),
    activationGeneration: 1,
  );
  for (final entry in <List<String>, List<String>>{
    ['quilting', 'knitting', 'quilting', 'unknown']: ['Knitting', 'Quilting'],
    []: [],
    ['unknown']: [],
    ['quilting', 'embroidery', 'sewing', 'crochet', 'knitting']: [
      'Knitting',
      'Crochet',
      'Sewing',
      'Embroidery',
      'Quilting',
    ],
  }.entries) {
    test('UT-001 preferred-only catalog order ${entry.key}', () async {
      final container = ProviderContainer.test(
        overrides: [
          userProfileProvider(lease.session.account.did)
              .overrideWith(() => _Profile(entry.key)),
        ],
      );
      final sub = container.listen(
        preferredCraftLinksProvider(lease),
        (_, _) {},
      );
      addTearDown(sub.close);
      await container.read(
        userProfileProvider(lease.session.account.did).future,
      );
      expect(
        container
            .read(preferredCraftLinksProvider(lease))
            .requireValue
            .map((option) => option.label),
        entry.value,
      );
    });
  }
}
