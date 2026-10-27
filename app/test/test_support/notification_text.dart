import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

// Notification actor names are inline widgets so they can animate separately
// from the action text. Read their accessible labels along with the sentence.
Finder notificationText(String text) => find.byWidgetPredicate(
  (widget) =>
      widget is Text && (widget.data ?? _plainText(widget.textSpan)) == text,
);

String _plainText(InlineSpan? span) => switch (span) {
  TextSpan(:final text, :final children) =>
    (text ?? '') + (children?.map(_plainText).join() ?? ''),
  WidgetSpan(child: Semantics(:final properties)) => properties.label ?? '',
  _ => '',
};
