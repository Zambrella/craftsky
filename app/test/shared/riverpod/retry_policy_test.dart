import 'package:craftsky_app/bootstrap.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('appProviderRetry disables Riverpod automatic retry', () {
    expect(appProviderRetry(0, StateError('boom')), isNull);
    expect(appProviderRetry(3, Exception('still failing')), isNull);
  });
}
