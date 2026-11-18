import 'package:craftsky_app/safety/support_search.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('offers optional support for explicit terms and separator variants', () {
    for (final query in [
      'Suicidal feelings',
      'self-harm',
      '#self_harm',
      'self–injury',
      'eating disorders',
      'ANOREXIA',
      'help with bulimia',
    ]) {
      expect(shouldOfferSearchSupport(query), isTrue, reason: query);
    }
  });
  test('avoids broad craft/body terms, acronyms and word fragments', () {
    for (final query in [
      '',
      'ED',
      'size',
      'fit',
      'diet',
      'knitting',
      'suicideberry',
      'anorexias',
      'self harmony',
    ]) {
      expect(shouldOfferSearchSupport(query), isFalse, reason: query);
    }
  });
}
