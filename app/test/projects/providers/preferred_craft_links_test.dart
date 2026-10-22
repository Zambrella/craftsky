import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/projects/providers/preferred_craft_links_provider.dart';
import 'package:craftsky_app/projects/providers/preferred_project_crafts_provider.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/auth_session_fakes.dart';

class _Profile extends UserProfile {
  new(this.crafts);
  final List<String> crafts;
  @override
  Future<Profile> build(Did did) async =>
      Profile(did: did.value, handle: 'test.test', crafts: crafts);
}

void main() {
  for (final entry in <List<String>, String?>{
    ['unknown', 'quilting', 'sewing']: ProjectOptionCatalogs.quiltingCraftToken,
    ['knitting', 'quilting']: ProjectOptionCatalogs.knittingCraftToken,
    []: null,
    ['unknown']: null,
  }.entries) {
    test('default is the first supported preference ${entry.key}', () async {
      final container = ProviderContainer.test(
        overrides: [
          authSessionProvider.overrideWith(SignedInAuthSession.new),
          userProfileProvider(Did.parse('did:plc:test'))
              .overrideWith(() => _Profile(entry.key)),
        ],
      );
      await container.read(authSessionProvider.future);
      container.read(defaultProjectCraftProvider);
      await container.read(
        userProfileProvider(Did.parse('did:plc:test')).future,
      );
      expect(container.read(defaultProjectCraftProvider), entry.value);
    });
  }

  final lease = ActiveAccountLease(
    session: AccountSessionLease(
      account: AccountKey('did:plc:test'),
      sessionGeneration: 1,
    ),
    activationGeneration: 1,
  );
  for (final entry in <List<String>, List<String>>{
    ['quilting', 'knitting', 'quilting', 'unknown']: ['Quilting', 'Knitting'],
    []: [],
    ['unknown']: [],
    ['quilting', 'embroidery', 'sewing', 'crochet', 'knitting']: [
      'Quilting',
      'Embroidery',
      'Sewing',
      'Crochet',
      'Knitting',
    ],
  }.entries) {
    test('UT-001 preferred-only profile order ${entry.key}', () async {
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
