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
          status: ['finished'],
          projectType: ['quilt'],
          projectSubtype: ['throw'],
          color: ['blue', 'green'],
          yarnWeight: ['dk'],
          selfDrafted: true,
        );

        expect(filters.toQueryParameters(), {
          'status': ['finished'],
          'projectType': ['quilt'],
          'projectSubtype': ['throw'],
          'color': ['blue', 'green'],
          'yarnWeight': ['dk'],
          'selfDrafted': true,
        });
      },
    );

    test('valuesFor returns each family list', () {
      const filters = ProjectBrowseFilters(
        projectType: ['garment'],
        patternDifficulty: ['beginner'],
        color: ['red'],
        designTag: ['striped'],
        quiltingMethod: ['tied'],
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
      expect(
        filters.valuesFor(ProjectBrowseFilterFamily.designTag),
        ['striped'],
      );
      expect(
        filters.valuesFor(ProjectBrowseFilterFamily.quiltingMethod),
        ['tied'],
      );
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
      const filters = ProjectBrowseFilters(yarnWeight: ['worsted']);

      final result = filters.withValue(
        ProjectBrowseFilterFamily.yarnWeight,
        'worsted',
      );

      expect(identical(result, filters), isTrue);
    });

    test('withoutValue removes every matching value', () {
      const filters = ProjectBrowseFilters(
        projectSubtype: ['hat', 'hat', 'bag'],
      );

      final result = filters.withoutValue(
        ProjectBrowseFilterFamily.projectSubtype,
        'hat',
      );

      expect(result.projectSubtype, ['bag']);
    });

    test('withValues replaces only the requested family', () {
      const filters = ProjectBrowseFilters(
        color: ['blue'],
        yarnWeight: ['worsted'],
      );

      final result = filters.withValues(
        ProjectBrowseFilterFamily.color,
        ['green'],
      );

      expect(result.color, ['green']);
      expect(result.yarnWeight, ['worsted']);
    });
  });
}
