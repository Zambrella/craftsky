import 'package:craftsky_app/shared/widgets/auto_paginated_list_view.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  for (final axis in Axis.values) {
    testWidgets('nested ${axis.name} scrolling does not load outer pages', (
      tester,
    ) async {
      var loads = 0;
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: _list(
              onNearEnd: () => loads++,
              firstRow: ListView(
                key: const Key('nested-scroll'),
                scrollDirection: axis,
                children: [
                  for (var index = 0; index < 2; index++)
                    SizedBox(
                      width: 800,
                      height: 200,
                      child: Text('Nested row $index'),
                    ),
                ],
              ),
            ),
          ),
        ),
      );

      await tester.drag(
        find.byKey(const Key('nested-scroll')),
        axis == Axis.horizontal
            ? const Offset(-1600, 0)
            : const Offset(0, -400),
      );
      await tester.pumpAndSettle();

      expect(loads, 0);
    });
  }

  testWidgets('own vertical scrolling near the end loads another page', (
    tester,
  ) async {
    var loads = 0;
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(body: _list(onNearEnd: () => loads++)),
      ),
    );

    await tester.drag(find.byType(ListView), const Offset(0, -6000));
    await tester.pumpAndSettle();

    expect(loads, greaterThan(0));
  });
}

Widget _list({required VoidCallback onNearEnd, Widget? firstRow}) =>
    AutoPaginatedListView(
      itemCount: 30,
      emptyState: const Text('Empty'),
      isLoadingMore: false,
      hasLoadMoreError: false,
      onNearEnd: onNearEnd,
      itemBuilder: (_, index) => SizedBox(
        height: 200,
        child: index == 0 && firstRow != null ? firstRow : Text('Row $index'),
      ),
    );
