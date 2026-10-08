import 'dart:io';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('REG-005 session invalidation keeps permanent billing owner pinned', () {
    final registry = SessionRegistry.empty()
        .upsertAndActivate(
          token: 'alice-token',
          did: 'did:plc:alice',
          handle: 'alice.test',
        )
        .reserveBillingOwner('did:plc:alice')
        .completeBillingOwner(
          'did:plc:alice',
          '20000000-0000-4000-8000-000000000001',
        );

    final removed = registry.remove('did:plc:alice');

    expect(removed.billingOwner, same(registry.billingOwner));
    expect(removed.billingOwner?.did.value, 'did:plc:alice');
  });

  test('REG-005 account boundary invalidates subscription projections', () {
    final source = File(
      'lib/auth/providers/account_boundary_provider.dart',
    ).readAsStringSync();

    expect(source, contains('invalidate(subscriptionAccessProvider)'));
    expect(source, contains('invalidate(subscriptionPageModelProvider)'));
    expect(source, contains('invalidate(subscriptionRepositoryProvider)'));
  });
}
