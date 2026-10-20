import 'package:craftsky_app/chat/models/chat_link_target.dart';
import 'package:flutter/gestures.dart';
import 'package:material_ui/material_ui.dart';

/// Links are explicit taps. This widget performs no network operation.
class ChatMessageText extends StatefulWidget {
  const new({required this.text, required this.onLink, super.key});
  final String text;
  final ValueChanged<Uri> onLink;
  @override
  State<ChatMessageText> createState() => _ChatMessageTextState();
}

class _ChatMessageTextState extends State<ChatMessageText> {
  final _recognizers = <TapGestureRecognizer>[];
  late List<ChatTextLink> _links;
  @override
  void initState() {
    super.initState();
    _configure();
  }

  void _configure() {
    for (final recognizer in _recognizers) {
      recognizer.dispose();
    }
    _recognizers.clear();
    _links = ChatLinkTarget.textLinksIn(widget.text).toList();
    for (final link in _links) {
      _recognizers.add(
        TapGestureRecognizer()..onTap = () => widget.onLink(link.uri),
      );
    }
  }

  @override
  void didUpdateWidget(covariant ChatMessageText oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.text != widget.text) _configure();
  }

  @override
  void dispose() {
    for (final recognizer in _recognizers) {
      recognizer.dispose();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final spans = <TextSpan>[];
    var offset = 0;
    for (var index = 0; index < _links.length; index++) {
      final link = _links[index];
      if (link.start > offset) {
        spans.add(TextSpan(text: widget.text.substring(offset, link.start)));
      }
      spans.add(
        TextSpan(
          text: widget.text.substring(link.start, link.end),
          recognizer: _recognizers[index],
          style: const TextStyle(decoration: TextDecoration.underline),
        ),
      );
      offset = link.end;
    }
    if (offset < widget.text.length) {
      spans.add(TextSpan(text: widget.text.substring(offset)));
    }
    return SelectableText.rich(
      TextSpan(children: spans),
      style: Theme.of(context).textTheme.bodyMedium,
    );
  }
}
