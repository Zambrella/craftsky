import 'package:craftsky_app/profile/models/profile_handle.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('ProfileHandle (UT-014)', () {
    test('recognizes sentinel and absent current handles as unavailable', () {
      expect(const ProfileHandle('handle.invalid').isAvailable, isFalse);
      expect(const ProfileHandle(null).isAvailable, isFalse);
      expect(const ProfileHandle('').isAvailable, isFalse);
      expect(const ProfileHandle('not a handle').isAvailable, isFalse);
    });

    test('presents only the authoritative current handle', () {
      expect(
        const ProfileHandle('alice.example').currentLabel(
          unavailableLabel: 'Handle unavailable',
        ),
        '@alice.example',
      );
      expect(
        const ProfileHandle('handle.invalid').currentLabel(
          unavailableLabel: 'Handle unavailable',
        ),
        'Handle unavailable',
      );
    });

    test('uses display identity without exposing an unavailable handle', () {
      const handle = ProfileHandle('handle.invalid');

      expect(
        handle.displayLabel(
          displayName: 'Alice',
          unavailableLabel: 'Handle unavailable',
        ),
        'Alice',
      );
      expect(
        handle.displayLabel(unavailableLabel: 'Handle unavailable'),
        'Handle unavailable',
      );
    });

    test('allows only a current valid handle to cross an alias boundary', () {
      expect(const ProfileHandle('alice.example').aliasInput, 'alice.example');
      expect(const ProfileHandle('handle.invalid').aliasInput, isNull);
      expect(const ProfileHandle(null).aliasInput, isNull);
      expect(const ProfileHandle('not a handle').aliasInput, isNull);
    });
  });
}
