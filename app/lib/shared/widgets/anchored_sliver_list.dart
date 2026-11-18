import 'package:flutter/rendering.dart';
import 'package:flutter/widgets.dart';

typedef _RowAnchor = ({int index, double offset});

/// Keeps the first surviving visible row in place before painting an update.
/// Previously visited rows reuse their heights; new prefix rows are measured
/// before correcting the viewport, so returning to the start still works.
class AnchoredSliverList extends StatefulWidget {
  const new({required this.identities, required this.itemBuilder, super.key});
  final List<Object> identities;
  final IndexedWidgetBuilder itemBuilder;

  @override
  State<AnchoredSliverList> createState() => _AnchoredSliverListState();
}

class _AnchoredSliverListState extends State<AnchoredSliverList> {
  _RowAnchor? _anchor;
  final _heights = <Object, double>{};

  @override
  void didUpdateWidget(AnchoredSliverList oldWidget) {
    super.didUpdateWidget(oldWidget);
    _anchor = null;
    if (oldWidget.identities.length == widget.identities.length &&
        Iterable<int>.generate(widget.identities.length)
            .every((i) => oldWidget.identities[i] == widget.identities[i])) {
      return;
    }
    final render = context.findRenderObject();
    final position = Scrollable.maybeOf(context)?.position;
    if (render is! RenderSliverList ||
        render.geometry == null ||
        position == null ||
        position.pixels <= position.minScrollExtent) {
      return;
    }
    final indexes = {
      for (var i = 0; i < widget.identities.length; i++)
        widget.identities[i]: i,
    };
    var child = render.firstChild;
    while (child != null) {
      final data = child.parentData! as SliverMultiBoxAdaptorParentData;
      final offset = data.layoutOffset;
      final oldIndex = data.index;
      if (offset != null &&
          oldIndex != null &&
          oldIndex < oldWidget.identities.length &&
          offset + child.size.height > render.constraints.scrollOffset) {
        final index = indexes[oldWidget.identities[oldIndex]];
        if (index != null) {
          _anchor = (index: index, offset: offset);
          return;
        }
      }
      child = render.childAfter(child);
    }
  }

  @override
  Widget build(BuildContext context) {
    final indexes = {
      for (var i = 0; i < widget.identities.length; i++)
        ValueKey(widget.identities[i]): i,
    };
    _heights.removeWhere((id, _) => !indexes.containsKey(ValueKey(id)));
    return _AnchorList(
      anchor: _anchor,
      identities: widget.identities,
      heights: _heights,
      delegate: SliverChildBuilderDelegate(
        (context, index) {
          final id = widget.identities[index];
          return KeyedSubtree(
            key: ValueKey(id),
            child: _MeasuredRow(
              onHeight: (height) => _heights[id] = height,
              child: widget.itemBuilder(context, index),
            ),
          );
        },
        childCount: widget.identities.length,
        findChildIndexCallback: (key) => indexes[key],
      ),
    );
  }
}

class _AnchorList extends SliverList {
  const new({
    required this.anchor,
    required this.identities,
    required this.heights,
    required super.delegate,
  });
  final _RowAnchor? anchor;
  final List<Object> identities;
  final Map<Object, double> heights;

  @override
  RenderSliverList createRenderObject(BuildContext context) =>
      _RenderAnchorList(
          childManager: context as SliverMultiBoxAdaptorElement,
        )
        ..anchor = anchor
        ..identities = identities
        ..heights = heights;

  @override
  void updateRenderObject(
    BuildContext context,
    _RenderAnchorList renderObject,
  ) {
    renderObject
      ..anchor = anchor
      ..identities = identities
      ..heights = heights
      ..markNeedsLayout();
  }
}

class _RenderAnchorList extends RenderSliverList {
  new({required super.childManager});
  _RowAnchor? anchor;
  List<Object> identities = const [];
  Map<Object, double> heights = {};
  double? _width;

  @override
  void performLayout() {
    if (_width != constraints.crossAxisExtent) {
      heights.clear();
      _width = constraints.crossAxisExtent;
    }
    final target = anchor;
    anchor = null;
    if (target != null) {
      var retained = firstChild;
      var leading = 0;
      while (retained != null && indexOf(retained) != target.index) {
        retained = childAfter(retained);
        leading++;
      }
      if (retained != null) collectGarbage(leading, 0);
      var offset = 0.0;
      for (var i = 0; i < target.index; i++) {
        final id = identities[i];
        if (!heights.containsKey(id)) {
          // Only newly inserted/unmeasured rows need a layout. Previously
          // visited rows retain their measured height without staying mounted.
          var measured = firstChild;
          while (measured != null && indexOf(measured) != i) {
            measured = childAfter(measured);
          }
          final temporary = measured == null;
          if (temporary) {
            RenderBox? after;
            var scan = firstChild;
            while (scan != null && indexOf(scan) < i) {
              after = scan;
              scan = childAfter(scan);
            }
            invokeLayoutCallback<SliverConstraints>((_) {
              childManager.createChild(i, after: after);
            });
            measured = after == null ? firstChild : childAfter(after);
          }
          if (measured != null && indexOf(measured) == i) {
            measured.layout(
              constraints.asBoxConstraints(),
              parentUsesSize: true,
            );
            heights[id] = measured.size.height;
            if (temporary) {
              invokeLayoutCallback<SliverConstraints>((_) {
                childManager.removeChild(measured!);
              });
            }
          }
        }
        offset += heights[id] ?? 0;
      }
      var child = firstChild;
      var preceding = 0;
      while (child != null && indexOf(child) != target.index) {
        child = childAfter(child);
        preceding++;
      }
      if (child != null) {
        collectGarbage(preceding, 0);
        (child.parentData! as SliverMultiBoxAdaptorParentData).layoutOffset =
            offset;
        final delta = offset - target.offset;
        if (delta.abs() > 0.5) {
          geometry = SliverGeometry(scrollOffsetCorrection: delta);
          return;
        }
      }
    }
    super.performLayout();
  }
}

class _MeasuredRow extends SingleChildRenderObjectWidget {
  const new({required this.onHeight, required super.child});
  final ValueChanged<double> onHeight;
  @override
  RenderObject createRenderObject(BuildContext context) =>
      _RenderMeasuredRow(onHeight);
  @override
  void updateRenderObject(
    BuildContext context,
    _RenderMeasuredRow renderObject,
  ) {
    renderObject.onHeight = onHeight;
  }
}

class _RenderMeasuredRow extends RenderProxyBox {
  new(this.onHeight);
  ValueChanged<double> onHeight;
  @override
  void performLayout() {
    super.performLayout();
    onHeight(size.height);
  }
}
