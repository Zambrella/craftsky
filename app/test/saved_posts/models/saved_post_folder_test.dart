import 'package:craftsky_app/saved_posts/models/saved_post_folder.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-003 validates names and identifies folders by opaque ID', () {
    expect(normalizeSavedPostFolderName('  Ideas  '), 'Ideas');
    expect(normalizeSavedPostFolderName('\u00a0Ideas\u00a0'), 'Ideas');
    expect(normalizeSavedPostFolderName('🧶' * 100), '🧶' * 100);

    void expectInvalid(String name, SavedPostFolderNameError error) {
      expect(
        () => normalizeSavedPostFolderName(name),
        throwsA(
          isA<SavedPostFolderNameException>().having(
            (exception) => exception.error,
            'error',
            error,
          ),
        ),
      );
    }

    expectInvalid('', SavedPostFolderNameError.empty);
    expectInvalid(' \t\n ', SavedPostFolderNameError.empty);
    expectInvalid('🧶' * 101, SavedPostFolderNameError.tooLong);
    expectInvalid('Ideas/2026', SavedPostFolderNameError.slash);
    expectInvalid(r'Ideas\2026', SavedPostFolderNameError.slash);
    expectInvalid('Ide\nas', SavedPostFolderNameError.control);
    expectInvalid('Ide\u0085as', SavedPostFolderNameError.control);

    final createdAt = DateTime.utc(2026, 7, 21, 9);
    final updatedAt = DateTime.utc(2026, 7, 21, 10);
    final ideasA = SavedPostFolder(
      id: 'folder-a',
      name: 'Ideas',
      createdAt: createdAt,
      updatedAt: updatedAt,
    );
    final ideasB = SavedPostFolder(
      id: 'folder-b',
      name: 'Ideas',
      createdAt: createdAt,
      updatedAt: updatedAt,
    );
    final caseVariant = SavedPostFolder(
      id: 'folder-c',
      name: 'IDEAS',
      createdAt: createdAt,
      updatedAt: updatedAt,
    );
    final renamedA = SavedPostFolder(
      id: 'folder-a',
      name: 'Later',
      createdAt: createdAt,
      updatedAt: updatedAt.add(const Duration(minutes: 1)),
    );

    expect({ideasA, ideasB, caseVariant}, hasLength(3));
    expect(ideasA, isNot(ideasB));
    expect(ideasA, isNot(caseVariant));
    expect(ideasA, renamedA);
    expect(ideasA.hashCode, renamedA.hashCode);
  });
}
