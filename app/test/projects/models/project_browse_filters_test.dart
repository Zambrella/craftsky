import 'package:craftsky_app/projects/models/project_browse_filters.dart';
import 'package:craftsky_app/search/models/search_sort.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('ProjectBrowseQuery', () {
    test('tokens constructor unwraps typed token values', () {
      final query = ProjectBrowseQuery.tokens(
        craftTypes: const [CraftTypeFilterToken('knitting')],
        filters: ProjectBrowseFilters.tokens(
          projectType: const [ProjectTypeFilterToken('sweater')],
          patternDifficulty: const [PatternDifficultyFilterToken('easy')],
          designTag: const [DesignTagFilterToken('floral')],
        ),
        sort: SearchSort.popular,
      );

      expect(query.craftTypes, ['knitting']);
      expect(query.filters.projectType, ['sweater']);
      expect(query.filters.patternDifficulty, ['easy']);
      expect(query.filters.designTag, ['floral']);
      expect(query.sort, SearchSort.popular);
    });
  });

  group('ProjectBrowseFilters', () {
    test(
      'toQueryParameters omits empty families and keeps selected values',
      () {
        const filters = ProjectBrowseFilters(
          projectType: ['quilt'],
          color: ['blue', 'green'],
          material: ['cotton'],
        );

        expect(filters.toQueryParameters(), {
          'projectType': ['quilt'],
          'color': ['blue', 'green'],
          'material': ['cotton'],
        });
      },
    );

    test('valuesFor returns each family list', () {
      const filters = ProjectBrowseFilters(
        projectType: ['garment'],
        patternDifficulty: ['beginner'],
        color: ['red'],
        material: ['linen'],
        designTag: ['striped'],
        projectTag: ['gift'],
      );

      expect(
        filters.valuesFor(ProjectBrowseFilterFamily.projectType),
        ['garment'],
      );
      expect(
        filters.valuesFor(ProjectBrowseFilterFamily.patternDifficulty),
        ['beginner'],
      );
      expect(filters.valuesFor(ProjectBrowseFilterFamily.color), ['red']);
      expect(filters.valuesFor(ProjectBrowseFilterFamily.material), ['linen']);
      expect(
        filters.valuesFor(ProjectBrowseFilterFamily.designTag),
        ['striped'],
      );
      expect(filters.valuesFor(ProjectBrowseFilterFamily.projectTag), ['gift']);
    });

    test('toggleValue adds absent values and removes present values', () {
      const empty = ProjectBrowseFilters();

      final added = empty.toggleValue(ProjectBrowseFilterFamily.color, 'blue');
      final removed = added.toggleValue(
        ProjectBrowseFilterFamily.color,
        'blue',
      );

      expect(added.color, ['blue']);
      expect(removed.color, isEmpty);
    });

    test('withValue preserves identity when value already exists', () {
      const filters = ProjectBrowseFilters(material: ['wool']);

      final result = filters.withValue(
        ProjectBrowseFilterFamily.material,
        'wool',
      );

      expect(identical(result, filters), isTrue);
    });

    test('withoutValue removes every matching value', () {
      const filters = ProjectBrowseFilters(projectTag: ['gift', 'gift', 'hat']);

      final result = filters.withoutValue(
        ProjectBrowseFilterFamily.projectTag,
        'gift',
      );

      expect(result.projectTag, ['hat']);
    });

    test('withValues replaces only the requested family', () {
      const filters = ProjectBrowseFilters(color: ['blue'], material: ['wool']);

      final result = filters.withValues(
        ProjectBrowseFilterFamily.color,
        ['green'],
      );

      expect(result.color, ['green']);
      expect(result.material, ['wool']);
    });
  });
}
