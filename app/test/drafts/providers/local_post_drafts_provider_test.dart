import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/drafts/providers/local_post_draft_repository_provider.dart';
import 'package:craftsky_app/drafts/providers/local_post_drafts_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
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
  _MemoryRepository(this.items);

  final List<LocalPostDraft> items;
  final List<String> deleted = [];

  @override
  Future<void> delete(String draftId) async {
    deleted.add(draftId);
    items.removeWhere((draft) => draft.id == draftId);
  }

  @override
  Future<List<LocalPostDraft>> list() async => List.of(items);

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
