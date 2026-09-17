import 'dart:async';

import 'package:flutter_test/flutter_test.dart';

final class PaginationContractPage<T> {
  const PaginationContractPage({required this.items, this.cursor});

  final List<T> items;
  final String? cursor;
}

final class PaginationContractSnapshot<T> {
  const PaginationContractSnapshot({
    required this.items,
    required this.cursor,
    required this.hasMore,
    required this.hasError,
  });

  final List<T> items;
  final String? cursor;
  final bool hasMore;
  final bool hasError;
}

final class PaginationContractSubject<T> {
  const PaginationContractSubject({
    required this.initialize,
    required this.loadMore,
    required this.snapshot,
    required this.dispose,
  });

  final Future<void> Function() initialize;
  final Future<void> Function() loadMore;
  final PaginationContractSnapshot<T> Function() snapshot;
  final void Function() dispose;
}

typedef PaginationContractFetch<T> =
    Future<PaginationContractPage<T>> Function(String? cursor);

typedef PaginationContractSubjectFactory<T> =
    PaginationContractSubject<T> Function(PaginationContractFetch<T> fetch);

void paginationContract<T>({
  required String name,
  required T firstItem,
  required T secondItem,
  required Object Function(T item) itemIdentity,
  required PaginationContractSubjectFactory<T> createSubject,
  bool deduplicatesAcrossPages = false,
}) {
  group('$name pagination contract', () {
    test('loads the initial page and exposes its cursor', () async {
      final requestedCursors = <String?>[];
      final subject = createSubject((cursor) async {
        requestedCursors.add(cursor);
        return PaginationContractPage(
          items: [firstItem, secondItem],
          cursor: 'cursor-1',
        );
      });
      addTearDown(subject.dispose);

      await subject.initialize();

      final state = subject.snapshot();
      expect(requestedCursors, [isNull]);
      expect(state.items.map(itemIdentity), [
        itemIdentity(firstItem),
        itemIdentity(secondItem),
      ]);
      expect(state.cursor, 'cursor-1');
      expect(state.hasMore, isTrue);
      expect(state.hasError, isFalse);
    });

    test(
      deduplicatesAcrossPages
          ? 'forwards the cursor, de-duplicates, appends, and stops at the end'
          : 'forwards the cursor, appends, and stops at the end',
      () async {
        final requestedCursors = <String?>[];
        final subject = createSubject((cursor) async {
          requestedCursors.add(cursor);
          if (cursor == null) {
            return PaginationContractPage(
              items: [firstItem],
              cursor: 'cursor-1',
            );
          }
          return PaginationContractPage(
            items: [if (deduplicatesAcrossPages) firstItem, secondItem],
          );
        });
        addTearDown(subject.dispose);

        await subject.initialize();
        await subject.loadMore();
        await subject.loadMore();

        final state = subject.snapshot();
        expect(requestedCursors, [isNull, 'cursor-1']);
        expect(state.items.map(itemIdentity), [
          itemIdentity(firstItem),
          itemIdentity(secondItem),
        ]);
        expect(state.cursor, isNull);
        expect(state.hasMore, isFalse);
      },
    );

    test(
      'preserves visible state after failure and retries the cursor',
      () async {
        final requestedCursors = <String?>[];
        var calls = 0;
        final subject = createSubject((cursor) async {
          requestedCursors.add(cursor);
          calls++;
          if (calls == 1) {
            return PaginationContractPage(
              items: [firstItem],
              cursor: 'cursor-1',
            );
          }
          if (calls == 2) throw Exception('network down');
          return PaginationContractPage(items: [secondItem]);
        });
        addTearDown(subject.dispose);

        await subject.initialize();
        await subject.loadMore();

        final failed = subject.snapshot();
        expect(failed.hasError, isTrue);
        expect(failed.items.map(itemIdentity), [itemIdentity(firstItem)]);
        expect(failed.cursor, 'cursor-1');

        await subject.loadMore();

        final recovered = subject.snapshot();
        expect(requestedCursors, [isNull, 'cursor-1', 'cursor-1']);
        expect(recovered.items.map(itemIdentity), [
          itemIdentity(firstItem),
          itemIdentity(secondItem),
        ]);
        expect(recovered.hasError, isFalse);
      },
    );

    test('ignores a second load while the next page is in flight', () async {
      final requestedCursors = <String?>[];
      final nextPage = Completer<PaginationContractPage<T>>();
      final subject = createSubject((cursor) {
        requestedCursors.add(cursor);
        if (cursor == null) {
          return Future.value(
            PaginationContractPage(items: [firstItem], cursor: 'cursor-1'),
          );
        }
        return nextPage.future;
      });
      addTearDown(subject.dispose);

      await subject.initialize();
      final inFlight = subject.loadMore();
      await Future<void>.delayed(Duration.zero);
      await subject.loadMore();

      expect(requestedCursors, [isNull, 'cursor-1']);

      nextPage.complete(PaginationContractPage(items: [secondItem]));
      await inFlight;
      expect(subject.snapshot().items.map(itemIdentity), [
        itemIdentity(firstItem),
        itemIdentity(secondItem),
      ]);
    });
  });
}
