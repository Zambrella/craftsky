import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/profile_pin_state.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  group('ProfilePinState', () {
    test('decodes and re-encodes both authoritative slots', () {
      final json = {
        'standardPostUri':
            'at://did:plc:alice/social.craftsky.feed.post/standard',
        'projectPostUri':
            'at://did:plc:alice/social.craftsky.feed.post/project',
      };

      final state = ProfilePinStateMapper.fromMap(json);

      expect(state.standardPostUri, json['standardPostUri']);
      expect(state.projectPostUri, json['projectPostUri']);
      expect(state.toMap(), json);
    });

    test('preserves one occupied and one empty slot', () {
      final json = {
        'standardPostUri':
            'at://did:plc:alice/social.craftsky.feed.post/standard',
        'projectPostUri': null,
      };

      final state = ProfilePinStateMapper.fromMap(json);

      expect(state.standardPostUri, json['standardPostUri']);
      expect(state.projectPostUri, isNull);
      expect(state.toMap(), json);
    });

    test('preserves both empty authoritative slots', () {
      final json = <String, dynamic>{
        'standardPostUri': null,
        'projectPostUri': null,
      };

      final state = ProfilePinStateMapper.fromMap(json);

      expect(state.standardPostUri, isNull);
      expect(state.projectPostUri, isNull);
      expect(state.toMap(), json);
    });
  });

  test('UT-005 classifies only client-visible pinnable post shapes', () {
    expect(
      classifyProfilePinSlot(
        isReply: false,
        isProject: false,
        hasQuote: false,
      ),
      ProfilePinSlot.standard,
    );
    expect(
      classifyProfilePinSlot(
        isReply: false,
        isProject: false,
        hasQuote: true,
      ),
      ProfilePinSlot.standard,
    );
    expect(
      classifyProfilePinSlot(
        isReply: false,
        isProject: true,
        hasQuote: false,
      ),
      ProfilePinSlot.project,
    );
    expect(
      classifyProfilePinSlot(
        isReply: true,
        isProject: false,
        hasQuote: false,
      ),
      isNull,
    );
    expect(
      classifyProfilePinSlot(
        isReply: false,
        isProject: true,
        hasQuote: true,
      ),
      isNull,
    );
  });
}
