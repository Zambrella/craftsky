// Separate actions and assertions keep state transitions explicit.
// ignore_for_file: cascade_invocations

import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/feed/composer/link_preview_candidate.dart';
import 'package:craftsky_app/feed/composer/link_preview_controller.dart';
import 'package:craftsky_app/feed/models/link_preview.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('UT-003 LinkPreviewController', () {
    test(
      'IT-012 starts immediately, stays sequential, and caches outcomes',
      () async {
        final repository = _FakePreviewRepository();
        final container = _container(repository);
        addTearDown(container.dispose);
        final provider = linkPreviewControllerProvider(
          'composer',
          AccountKey('did:plc:alice'),
        );
        final subscription = container.listen(provider, (_, _) {});
        addTearDown(subscription.close);
        final controller = container.read(provider.notifier);

        controller.updateText('one.example/path two.example/path ');
        expect(repository.urls, ['https://one.example/path']);
        expect(container.read(provider).inFlightIdentity?.host, 'one.example');

        repository.completeNext(_preview('https://one.final/path'));
        await _flush();
        expect(repository.urls, [
          'https://one.example/path',
          'https://two.example/path',
        ]);
        repository.failNext();
        await _flush();

        controller.updateText('two.example/path one.example/path ');
        await _flush();
        expect(repository.urls, hasLength(2));
        expect(container.read(provider).selectedIdentity?.host, 'one.example');
      },
    );

    test(
      'IT-012 selection follows identity and source fragment follows reorder',
      () async {
        final repository = _FakePreviewRepository();
        final container = _container(repository);
        addTearDown(container.dispose);
        final provider = linkPreviewControllerProvider(
          'composer',
          AccountKey('did:plc:alice'),
        );
        final subscription = container.listen(provider, (_, _) {});
        addTearDown(subscription.close);
        final controller = container.read(provider.notifier);

        controller.updateText(
          'one.example/path#first two.example/path#two '
          'one.example/path#later ',
        );
        repository.completeNext(_preview('https://one.final/path'));
        await _flush();
        repository.completeNext(_preview('https://two.final/path#redirect'));
        await _flush();
        controller.selectNext();
        expect(container.read(provider).selectedIdentity?.host, 'two.example');

        controller.updateText(
          'one.example/path#later two.example/path#changed ',
        );
        expect(container.read(provider).selectedIdentity?.host, 'two.example');
        expect(
          controller.selected?.navigationUri.toString(),
          'https://two.final/path#redirect',
        );
        controller.selectPrevious();
        expect(
          controller.selected?.navigationUri.toString(),
          'https://one.final/path#later',
        );
        expect(repository.urls, hasLength(2));
      },
    );

    test(
      'IT-012 suppression cancels and restoration resumes uncached work',
      () async {
        final repository = _FakePreviewRepository();
        final container = _container(repository);
        addTearDown(container.dispose);
        final provider = linkPreviewControllerProvider(
          'composer',
          AccountKey('did:plc:alice'),
        );
        final subscription = container.listen(provider, (_, _) {});
        addTearDown(subscription.close);
        final controller = container.read(provider.notifier);

        controller.updateText('one.example/path ');
        final firstToken = repository.tokens.single;
        controller.setSuppressed(value: true);
        expect(firstToken.isCancelled, isTrue);
        expect(container.read(provider).suppressed, isTrue);

        controller.setSuppressed(value: false);
        expect(repository.urls, hasLength(2));
        repository.completeNext(_preview('https://one.final/path'));
        await _flush();
        expect(controller.selected, isNotNull);
      },
    );

    test(
      'IT-012 dismissal resumes only through Undo and survives Undo expiry',
      () async {
        final repository = _FakePreviewRepository();
        final container = _container(repository);
        addTearDown(container.dispose);
        final provider = linkPreviewControllerProvider(
          'composer',
          AccountKey('did:plc:alice'),
        );
        final subscription = container.listen(provider, (_, _) {});
        addTearDown(subscription.close);
        final controller = container.read(provider.notifier);

        controller.updateText('one.example/path ');
        final dismissGeneration = controller.dismiss();
        expect(repository.tokens.single.isCancelled, isTrue);
        controller.expireUndo(dismissGeneration);
        expect(container.read(provider).dismissed, isTrue);
        expect(container.read(provider).canUndoDismiss, isFalse);
        expect(repository.urls, hasLength(1));

        controller.undoDismiss();
        expect(repository.urls, hasLength(1));

        final second = linkPreviewControllerProvider(
          'second',
          AccountKey('did:plc:alice'),
        );
        final secondSubscription = container.listen(second, (_, _) {});
        addTearDown(secondSubscription.close);
        final secondController = container.read(second.notifier);
        secondController.updateText('two.example/path ');
        secondController.dismiss();
        secondController.undoDismiss();
        expect(repository.urls.last, 'https://two.example/path');
        expect(repository.urls, hasLength(3));
      },
    );

    test(
      'IT-012 ignores stale completions and cancels work on disposal',
      () async {
        final repository = _FakePreviewRepository();
        final container = _container(repository);
        final provider = linkPreviewControllerProvider(
          'composer',
          AccountKey('did:plc:alice'),
        );
        final subscription = container.listen(provider, (_, _) {});
        final controller = container.read(provider.notifier);

        controller.updateText('one.example/path ');
        final pending = repository.pending.single;
        controller.updateText('two.example/path ');
        expect(pending.token.isCancelled, isTrue);
        pending.completer.complete(_preview('https://stale.final/path'));
        await _flush();
        expect(controller.selected, isNull);
        expect(repository.urls.last, 'https://two.example/path');

        final active = repository.pending.last.token;
        subscription.close();
        container.dispose();
        expect(active.isCancelled, isTrue);
      },
    );
  });

  test(
    'IT-012 account boundary cancels the old fragmentless request',
    () async {
      final repository = _FakePreviewRepository();
      final container = _container(repository);
      addTearDown(container.dispose);
      final alice = linkPreviewControllerProvider(
        'composer',
        AccountKey('did:plc:alice'),
      );
      final aliceSubscription = container.listen(alice, (_, _) {});
      container.read(alice.notifier).updateText('one.example/path#source ');
      expect(repository.urls.single, 'https://one.example/path');
      final aliceRequest = repository.pending.single;

      aliceSubscription.close();
      await container.pump();
      expect(aliceRequest.token.isCancelled, isTrue);

      final bob = linkPreviewControllerProvider(
        'composer',
        AccountKey('did:plc:bob'),
      );
      final bobSubscription = container.listen(bob, (_, _) {});
      addTearDown(bobSubscription.close);
      container.read(bob.notifier).updateText('two.example/path ');
      expect(repository.urls.last, 'https://two.example/path');
      aliceRequest.completer.complete(_preview('https://stale.example/path'));
      await _flush();
      expect(container.read(bob.notifier).selected, isNull);
    },
  );

  test('AT-005 frozen schedule seeds without refetching its source', () {
    final repository = _FakePreviewRepository();
    final container = _container(repository);
    addTearDown(container.dispose);
    final provider = linkPreviewControllerProvider(
      'scheduled',
      AccountKey('did:plc:alice'),
    );
    final controller = container.read(provider.notifier);
    final candidate = LinkPreviewCandidate.parse('https://source.example/a');

    controller.seed(
      SelectedLinkPreview(
        candidate: candidate,
        preview: LinkPreview(
          url: Uri.parse('https://final.example/a'),
          title: 'Frozen',
          description: 'Frozen description',
        ),
      ),
    );
    controller.updateText(
      'https://source.example/a https://new.example/b ',
    );

    expect(controller.selected?.preview.title, 'Frozen');
    expect(repository.urls, ['https://new.example/b']);
  });

  test('IT-017 reopened draft starts a fresh preview session', () {
    final repository = _FakePreviewRepository();
    final container = _container(repository);
    addTearDown(container.dispose);
    final account = AccountKey('did:plc:alice');
    final oldProvider = linkPreviewControllerProvider('old-draft', account);
    final old = container.read(oldProvider.notifier);
    old.updateText('https://source.example/a ');
    old.dismiss();

    final reopenedProvider = linkPreviewControllerProvider(
      'reopened-draft',
      account,
    );
    final reopened = container.read(reopenedProvider.notifier);
    reopened.updateText('https://source.example/a ');

    expect(reopened.state.dismissed, isFalse);
    expect(repository.urls, [
      'https://source.example/a',
      'https://source.example/a',
    ]);
    expect(repository.tokens.first.isCancelled, isTrue);
  });
}

ProviderContainer _container(_FakePreviewRepository repository) =>
    ProviderContainer.test(
      overrides: [linkPreviewRepositoryProvider.overrideWithValue(repository)],
    );

LinkPreview _preview(String url) => LinkPreview(
  url: Uri.parse(url),
  title: 'Pattern',
  description: 'Description',
);

Future<void> _flush() async {
  await Future<void>.delayed(Duration.zero);
  await Future<void>.delayed(Duration.zero);
}

final class _PendingFetch {
  _PendingFetch(this.url, this.token);

  final Uri url;
  final CancelToken token;
  final completer = Completer<LinkPreview>();
}

final class _FakePreviewRepository implements LinkPreviewRepository {
  final pending = <_PendingFetch>[];

  List<String> get urls => [
    for (final request in pending) request.url.toString(),
  ];
  List<CancelToken> get tokens => [
    for (final request in pending) request.token,
  ];

  @override
  Future<LinkPreview> fetch(Uri url, CancelToken cancelToken) {
    final request = _PendingFetch(url, cancelToken);
    pending.add(request);
    return request.completer.future;
  }

  void completeNext(LinkPreview preview) {
    pending
        .firstWhere(
          (request) =>
              !request.completer.isCompleted && !request.token.isCancelled,
        )
        .completer
        .complete(preview);
  }

  void failNext() {
    pending
        .firstWhere(
          (request) =>
              !request.completer.isCompleted && !request.token.isCancelled,
        )
        .completer
        .completeError(Exception('rate limited'));
  }
}
