import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

import '../test_support/source_scan.dart';

void main() {
  test('REG-009 superseded business mutation overlay has no caller', () {
    const legacyPath =
        'lib/business/providers/business_projection_overlay_provider.dart';
    expect(File(legacyPath).existsSync(), isFalse);

    final violations = forbiddenSourceMatches(scanDartSources('lib'), [
      'business_projection_overlay_provider.dart',
      'businessProjectionOverlayProvider',
      'BusinessProjectionOverlayController',
    ]);
    expect(violations, isEmpty, reason: violations.join('\n'));
  });

  test('UT-019 migrated production callers cannot mutate feature caches', () {
    const privateMutationExclusions = {
      'lib/business/providers/account_type_controller.dart',
      'lib/profile/providers/profile_customisation_provider.dart',
      'lib/profile/providers/profile_relationship_provider.dart',
      'lib/profile/providers/user_profile_provider.dart',
    };
    final sources = scanDartSources(
      'lib',
    ).where((source) => !privateMutationExclusions.contains(source.path));
    final violations = _obsoleteMutationCacheMatches(sources);
    expect(violations, isEmpty, reason: violations.join('\n'));
  });

  test('REG-009 guard rejects a reintroduced mutation cache helper', () {
    final violations = _obsoleteMutationCacheMatches([
      DartSourceFile.fromSource(
        'lib/feed/pages/regression.dart',
        'void replace(Post post) {}',
      ),
    ]);

    expect(violations, isNotEmpty);
  });
}

List<String> _obsoleteMutationCacheMatches(
  Iterable<DartSourceFile> sources,
) => forbiddenSourceMatches(sources, [
  RegExp(r'\b(?:update|prepend|removeFrom)Live\w*Cache'),
  'publishProfileCache(',
  '.setCached(',
  '_suppressLoadedSurfaces(',
  RegExp(
    r'\bvoid\s+(?:prepend|prependOrReplace|replace|removeByRkey|removeByUri)'
    r'\s*\(\s*Post\b',
  ),
  RegExp(
    r'\.notifier\)\s*\.(?:prepend|prependOrReplace|replace|removeByRkey|removeByUri)\s*\(',
  ),
]);
