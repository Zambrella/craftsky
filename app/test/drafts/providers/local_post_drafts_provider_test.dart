import 'dart:async';
import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_operation_canceled.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/drafts/providers/local_post_draft_repository_provider.dart';
import 'package:craftsky_app/drafts/providers/local_post_drafts_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'draft load after disposal cannot publish account-private data',
    () async {
      final account = AccountKey('did:plc:alice');
      final response = Completer<List<LocalPostDraft>>();
      final started = Completer<void>();
      final repository = _MemoryRepository([]);
      final container = ProviderContainer.test(
        overrides: [
          accountLocalPostDraftRepositoryProvider(
            account,
          ).overrideWith((ref) async => repository),
        ],
      );
      final subscription = container.listen(
        localPostDraftsProvider(account),
        (_, _) {},
      );
      await container.read(localPostDraftsProvider(account).future);
      repository.onList = () {
        started.complete();
        return response.future;
      };
      final result = container
          .read(localPostDraftsProvider(account).notifier)
          .build(account);
      final assertion = expectLater(
        result,
        throwsA(isA<AccountOperationCanceled>()),
      );
      await started.future;
      subscription.close();
      container.dispose();
      response.complete([_draft('private', account, 1)]);
      await assertion;
    },
  );

  test('sorts, refreshes, deletes, and isolates account families', () async {
    final alice = AccountKey('did:plc:alice');
    final bob = AccountKey('did:plc:bob');
    final aliceRepository = _MemoryRepository([
      _draft('older', alice, 1),
      _draft('newer', alice, 2),
    ]);
    final bobRepository = _MemoryRepository([_draft('bob', bob, 3)]);
    final container = ProviderContainer.test(
      overrides: [
        accountLocalPostDraftRepositoryProvider(
          alice,
        ).overrideWith((ref) async => aliceRepository),
        accountLocalPostDraftRepositoryProvider(
          bob,
        ).overrideWith((ref) async => bobRepository),
      ],
    );
    addTearDown(container.dispose);

    final aliceState = await container.read(
      localPostDraftsProvider(alice).future,
    );
    final bobState = await container.read(localPostDraftsProvider(bob).future);
    expect(aliceState.items.map((draft) => draft.id), ['newer', 'older']);
    expect(bobState.items.map((draft) => draft.id), ['bob']);

    await container
        .read(localPostDraftsProvider(alice).notifier)
        .delete('newer');
    expect(aliceRepository.deleted, ['newer']);
    expect(
      container
          .read(localPostDraftsProvider(alice))
          .requireValue
          .items
          .map((draft) => draft.id),
      ['older'],
    );
    expect(
      container
          .read(localPostDraftsProvider(bob))
          .requireValue
          .items
          .map((draft) => draft.id),
      ['bob'],
    );
  });
}

LocalPostDraft _draft(String id, AccountKey owner, int hour) => LocalPostDraft(
  id: id,
  owner: owner,
  kind: LocalPostDraftKind.standard,
  createdAt: DateTime.utc(2026, 8, 3, hour),
  updatedAt: DateTime.utc(2026, 8, 3, hour),
  content: StandardDraftContent(text: id, languages: const ['en']),
  schedule: const DraftScheduleIntent.now(),
  media: const [],
);

final class _MemoryRepository implements LocalPostDraftRepository {
  new(this.items);

  final List<LocalPostDraft> items;
  final List<String> deleted = [];
  Future<List<LocalPostDraft>> Function()? onList;

  @override
  Future<void> delete(String draftId) async {
    deleted.add(draftId);
    items.removeWhere((draft) => draft.id == draftId);
  }

  @override
  Future<List<LocalPostDraft>> list() async =>
      onList == null ? List.of(items) : await onList!();

  @override
  Future<LocalPostDraft> get(String draftId) async =>
      items.singleWhere((draft) => draft.id == draftId);

  @override
  Future<Uint8List> readMedia(String draftId, String mediaId) =>
      throw UnimplementedError();

  @override
  Future<LocalPostDraft> save(DraftWriteRequest request) =>
      throw UnimplementedError();
}
