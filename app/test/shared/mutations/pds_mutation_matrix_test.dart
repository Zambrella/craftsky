import 'package:craftsky_app/shared/mutations/pds_mutation_matrix.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'matrix declares every reviewed included, AppView-only, and excluded row',
    () {
      expect(pdsMutationMatrix.map((row) => row.name).toSet(), {
        'like/unlike',
        'repost/unrepost',
        'follow/unfollow',
        'block/unblock',
        'Instagram suggestion acceptance',
        'immediate post create',
        'immediate post delete',
        'business event create',
        'business event update/delete',
        'business profile put/delete',
        'personal profile save',
        'scheduled final publication',
        'mutes',
        'drafts',
        'saves/pins',
        'schedule CRUD',
        'schedule media staging',
        'blob/video upload',
        'profile customisation/account type',
        'moderation',
        'push tokens',
        'permanent account deletion',
      });
    },
  );

  test('included rows declare the complete shared-controller contract', () {
    final included = pdsMutationMatrix.where(
      (row) => row.disposition == PdsMutationDisposition.included,
    );
    expect(included, hasLength(11));
    for (final row in included) {
      expect(row.operationKeyRequired, isTrue, reason: row.name);
      expect(row.identityStrategy, isNotEmpty, reason: row.name);
      expect(row.casStrategy, isNotEmpty, reason: row.name);
      expect(row.stepStrategy, isNotEmpty, reason: row.name);
      expect(row.controllerStrategy, isNotEmpty, reason: row.name);
      expect(
        row.migration,
        {
              'like/unlike',
              'repost/unrepost',
              'follow/unfollow',
              'block/unblock',
              'Instagram suggestion acceptance',
              'immediate post create',
              'immediate post delete',
              'business event create',
              'business event update/delete',
              'business profile put/delete',
              'personal profile save',
            }.contains(row.name)
            ? PdsMutationMigration.migrated
            : PdsMutationMigration.pending,
        reason: row.name,
      );
    }
  });

  test('every command-backed row has completed migration', () {
    final commandBacked = pdsMutationMatrix.where(
      (row) => row.disposition != PdsMutationDisposition.excluded,
    );
    for (final row in commandBacked) {
      expect(
        row.migration,
        PdsMutationMigration.migrated,
        reason: row.name,
      );
    }
  });

  test(
    'scheduled publication is AppView-only and excluded rows stay outside',
    () {
      final scheduled = pdsMutationMatrix.singleWhere(
        (row) => row.name == 'scheduled final publication',
      );
      expect(scheduled.disposition, PdsMutationDisposition.appViewOnly);
      expect(scheduled.operationKeyRequired, isFalse);
      expect(scheduled.controllerStrategy, isEmpty);

      final excluded = pdsMutationMatrix.where(
        (row) => row.disposition == PdsMutationDisposition.excluded,
      );
      expect(excluded, hasLength(10));
      for (final row in excluded) {
        expect(row.operationKeyRequired, isFalse, reason: row.name);
        expect(row.controllerStrategy, isEmpty, reason: row.name);
        expect(
          row.migration,
          PdsMutationMigration.notApplicable,
          reason: row.name,
        );
      }
    },
  );
}
