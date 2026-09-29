import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

final class AsyncSubmitContractSubject<T> {
  const AsyncSubmitContractSubject({
    required this.submit,
    required this.state,
    required this.dispose,
  });

  final Future<void> Function() submit;
  final AsyncValue<T?> Function() state;
  final void Function() dispose;
}

typedef AsyncSubmitContractSubjectFactory<T> =
    AsyncSubmitContractSubject<T> Function(Future<T> Function() operation);

void asyncSubmitContract<T>({
  required String name,
  required T successValue,
  required T retryValue,
  required AsyncSubmitContractSubjectFactory<T> createSubject,
}) {
  group('$name async-submit contract', () {
    test('starts idle with no result', () {
      final subject = createSubject(() async => successValue);
      addTearDown(subject.dispose);

      expect(subject.state(), AsyncData<T?>(null));
    });

    test(
      'suppresses duplicate in-flight submits and exposes success',
      () async {
        var calls = 0;
        final pending = Completer<T>();
        final subject = createSubject(() {
          calls++;
          return pending.future;
        });
        addTearDown(subject.dispose);

        final first = subject.submit();
        await Future<void>.delayed(Duration.zero);
        await subject.submit();

        expect(calls, 1);
        expect(subject.state().isLoading, isTrue);

        pending.complete(successValue);
        await first;
        expect(subject.state().requireValue, successValue);
      },
    );

    test('exposes failure and permits a successful retry', () async {
      var calls = 0;
      final subject = createSubject(() async {
        calls++;
        if (calls == 1) throw Exception('network down');
        return retryValue;
      });
      addTearDown(subject.dispose);

      await subject.submit();
      expect(subject.state().hasError, isTrue);

      await subject.submit();
      expect(calls, 2);
      expect(subject.state().requireValue, retryValue);
    });
  });
}
