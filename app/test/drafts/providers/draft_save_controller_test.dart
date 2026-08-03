import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/drafts/providers/draft_save_controller.dart';
import 'package:craftsky_app/drafts/providers/local_post_draft_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('saves for the captured owner and exposes bounded progress', () async {
    final account = AccountKey('did:plc:alice');
    final repository = _RecordingRepository();
    final container = ProviderContainer.test(
      overrides: [
        accountLocalPostDraftRepositoryProvider(
          account,
        ).overrideWith((ref) async => repository),
      ],
    );
    addTearDown(container.dispose);
    final request = DraftWriteRequest(
      id: '96ad7199-292f-4388-a6cd-b4f74230116b',
      owner: account,
      kind: LocalPostDraftKind.standard,
      content: const StandardDraftContent(
        text: 'unfinished',
        languages: ['en'],
      ),
      schedule: const DraftScheduleIntent.now(),
      orderedMedia: const [],
    );

    final future = container
        .read(draftSaveControllerProvider(account).notifier)
        .save(request);
    expect(
      container.read(draftSaveControllerProvider(account)).isLoading,
      isTrue,
    );

    repository.complete(_draft(account));
    final saved = await future;

    expect(saved?.id, request.id);
    expect(repository.request, same(request));
    expect(
      container.read(draftSaveControllerProvider(account)).hasValue,
      isTrue,
    );
  });
}

final class _RecordingRepository implements LocalPostDraftRepository {
  final _completion = Completer<LocalPostDraft>();
  DraftWriteRequest? request;

  void complete(LocalPostDraft draft) => _completion.complete(draft);

  @override
  Future<LocalPostDraft> save(DraftWriteRequest request) {
    this.request = request;
    return _completion.future;
  }

  @override
  Future<List<LocalPostDraft>> list() async => const [];

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

LocalPostDraft _draft(AccountKey owner) => LocalPostDraft(
  id: '96ad7199-292f-4388-a6cd-b4f74230116b',
  owner: owner,
  kind: LocalPostDraftKind.standard,
  createdAt: DateTime.utc(2026, 8, 3),
  updatedAt: DateTime.utc(2026, 8, 3),
  content: const StandardDraftContent(text: 'unfinished', languages: ['en']),
  schedule: const DraftScheduleIntent.now(),
  media: const [],
);
