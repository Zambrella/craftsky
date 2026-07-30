import 'package:craftsky_app/languages/models/post_language_selection.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('keeps one to three ordered distinct composer languages', () {
    var selection = PostLanguageSelection.fromPrimary('en');
    expect(selection.values, ['en']);

    selection = selection.add('fr').add('cy');
    expect(selection.values, ['en', 'fr', 'cy']);
    expect(() => selection.add('de'), throwsStateError);
    expect(() => selection.add('fr'), throwsStateError);

    selection = selection.remove('fr');
    expect(selection.values, ['en', 'cy']);
    selection = selection.remove('en');
    expect(selection.values, ['cy']);
    expect(() => selection.remove('cy'), throwsStateError);
  });

  test('a new composer starts from current Primary only', () {
    final first = PostLanguageSelection.fromPrimary('en').add('fr');
    final second = PostLanguageSelection.fromPrimary('cy');

    expect(first.values, ['en', 'fr']);
    expect(second.values, ['cy']);
  });
}
