import 'package:craftsky_app/feed/composer/link_preview_candidates.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('UT-002 deriveLinkPreviewCandidates', () {
    test('excludes an unfinished URL at the end of the draft', () {
      final candidates = deriveLinkPreviewCandidates(
        'First https://one.example/path then https://unfinished.example',
      );

      expect(
        candidates.map((candidate) => candidate.identity.toString()),
        ['https://one.example/path'],
      );
    });

    test('accepts whitespace and punctuation completion boundaries', () {
      final candidates = deriveLinkPreviewCandidates(
        'https://space.example/path\nhttps://comma.example/path, next',
      );

      expect(
        candidates.map((candidate) => candidate.identity.toString()),
        [
          'https://space.example/path',
          'https://comma.example/path',
        ],
      );
    });

    test('keeps the first occurrence of each normalized identity', () {
      final candidates = deriveLinkPreviewCandidates(
        'example.com/pattern#first https://EXAMPLE.COM:443/pattern#second '
        'other.example/pattern ',
      );

      expect(
        candidates.map((candidate) => candidate.identity.toString()),
        ['https://example.com/pattern', 'https://other.example/pattern'],
      );
      expect(
        candidates.first.identity,
        Uri.parse('https://example.com/pattern'),
      );
      expect(candidates.first.sourceFragment, 'first');
      expect(
        candidates.last.identity,
        Uri.parse('https://other.example/pattern'),
      );
    });

    test('returns only the first four distinct completed URLs', () {
      final candidates = deriveLinkPreviewCandidates(
        'one.example/path two.example/path three.example/path '
        'four.example/path five.example/path ',
      );

      expect(
        candidates.map((candidate) => candidate.identity.host),
        ['one.example', 'two.example', 'three.example', 'four.example'],
      );
    });

    test('derives uppercase HTTP schemes through shared facet tokens', () {
      final candidates = deriveLinkPreviewCandidates(
        'HTTP://One.Example/path HTTPS://Two.Example/pattern ',
      );

      expect(
        candidates.map((candidate) => candidate.identity.toString()),
        ['http://one.example/path', 'https://two.example/pattern'],
      );
    });
  });
}
