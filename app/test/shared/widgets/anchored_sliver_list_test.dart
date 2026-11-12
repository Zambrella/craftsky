import 'package:craftsky_app/shared/widgets/anchored_sliver_list.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  for (final removeAnchor in [false, true]) {
    testWidgets(
      'keeps the surviving visible row on insertion/removal: $removeAnchor',
      (tester) async {
        final controller = ScrollController();
        addTearDown(controller.dispose);
        var rows = List.generate(30, (i) => 'row$i');
        late StateSetter update;
        await tester.pumpWidget(
          Directionality(
            textDirection: TextDirection.ltr,
            child: StatefulBuilder(
              builder: (context, setState) {
                update = setState;
                return CustomScrollView(
                  controller: controller,
                  slivers: [
                    AnchoredSliverList(
                      identities: rows,
                      itemBuilder: (_, i) =>
                          SizedBox(height: 150, child: Text(rows[i])),
                    ),
                  ],
                );
              },
            ),
          ),
        );
        controller.jumpTo(1000);
        await tester.pump();
        final before = tester.getTopLeft(find.text('row7')).dy;
        update(() {
          rows = [
            ...List.generate(15, (i) => 'new$i'),
            ...rows.where((id) => id != (removeAnchor ? 'row6' : 'row1')),
          ];
        });
        await tester.pump();
        expect(tester.getTopLeft(find.text('row7')).dy, closeTo(before, 0.01));
        await tester.pump();
        expect(tester.getTopLeft(find.text('row7')).dy, closeTo(before, 0.01));
        controller.jumpTo(controller.position.minScrollExtent);
        await tester.pumpAndSettle();
        expect(tester.getTopLeft(find.text('new0')).dy, closeTo(0, 0.01));
      },
    );
  }
}
