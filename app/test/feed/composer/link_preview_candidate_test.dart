import 'package:craftsky_app/feed/composer/link_preview_candidate.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('UT-001 LinkPreviewCandidate', () {
    test('normalizes identity and fragmentless transport URI', () {
      final https = LinkPreviewCandidate.parse(
        'HTTPS://Example.COM:443/pattern?id=7#size-chart',
      );
      final http = LinkPreviewCandidate.parse(
        'HTTP://Example.COM:80/pattern?id=7#materials',
      );
      final bare = LinkPreviewCandidate.parse('Example.COM/pattern?id=7#notes');

      expect(https.identity, Uri.parse('https://example.com/pattern?id=7'));
      expect(https.transportUri, https.identity);
      expect(https.sourceFragment, 'size-chart');
      expect(http.identity, Uri.parse('http://example.com/pattern?id=7'));
      expect(bare.identity, Uri.parse('https://example.com/pattern?id=7'));
      expect(bare.sourceFragment, 'notes');
    });

    test(
      'equivalent occurrences share identity but retain source fragments',
      () {
        final first = LinkPreviewCandidate.parse(
          'https://example.com/pattern#materials',
        );
        final second = LinkPreviewCandidate.parse(
          'HTTPS://EXAMPLE.COM:443/pattern#sizes',
        );
        final withoutFragment = LinkPreviewCandidate.parse(
          'https://example.com/pattern',
        );

        expect(first.identity, second.identity);
        expect(first.identity, withoutFragment.identity);
        expect(
          first.navigationUri(Uri.parse('https://final.example/pattern')),
          Uri.parse('https://final.example/pattern#materials'),
        );
        expect(
          second.navigationUri(Uri.parse('https://final.example/pattern')),
          Uri.parse('https://final.example/pattern#sizes'),
        );
        expect(
          withoutFragment.navigationUri(
            Uri.parse('https://final.example/pattern'),
          ),
          Uri.parse('https://final.example/pattern'),
        );
      },
    );

    test(
      'current first occurrence changes navigation without changing identity',
      () {
        final originalFirst = LinkPreviewCandidate.parse(
          'https://example.com/pattern#first',
        );
        final reorderedFirst = LinkPreviewCandidate.parse(
          'https://example.com/pattern#second',
        );
        final finalUri = Uri.parse('https://final.example/pattern?lang=en');

        expect(originalFirst.identity, reorderedFirst.identity);
        expect(
          originalFirst.navigationUri(finalUri),
          Uri.parse('https://final.example/pattern?lang=en#first'),
        );
        expect(
          reorderedFirst.navigationUri(finalUri),
          Uri.parse('https://final.example/pattern?lang=en#second'),
        );
      },
    );

    test('AppView redirect fragment takes precedence', () {
      final candidate = LinkPreviewCandidate.parse(
        'https://example.com/pattern#source',
      );
      final redirected = Uri.parse(
        'https://final.example/pattern#redirect-supplied',
      );

      expect(candidate.navigationUri(redirected), redirected);
    });
  });
}
