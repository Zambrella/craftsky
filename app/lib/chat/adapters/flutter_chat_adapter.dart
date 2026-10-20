import 'dart:async';

import 'package:craftsky_app/chat/adapters/disabled_chat_media_cache.dart';
import 'package:craftsky_app/chat/models/chat_history.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_message.dart';
import 'package:craftsky_app/chat/widgets/chat_message_text.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_chat_core/flutter_chat_core.dart' as core;
import 'package:flutter_chat_ui/flutter_chat_ui.dart' as ui;
import 'package:material_ui/material_ui.dart';

/// Package state is only an in-memory presentation mirror. Business operations,
/// drafts and authenticated content resolution remain owned by CraftSky.
class FlutterChatAdapter extends StatefulWidget {
  const new({
    required this.history,
    required this.owner,
    required this.composer,
    this.onPresented,
    this.onLoadOlder,
    this.onReport,
    this.messageFooterBuilder,
    this.onLink,
    super.key,
  });
  final ChatHistory history;
  final Did owner;
  final Widget composer;
  final ValueChanged<List<ChatMessageId>>? onPresented;
  final Future<void> Function()? onLoadOlder;
  final ValueChanged<ChatMessageId>? onReport;
  final Widget Function(ChatMessage)? messageFooterBuilder;
  final ValueChanged<Uri>? onLink;

  @override
  State<FlutterChatAdapter> createState() => _FlutterChatAdapterState();
}

class _FlutterChatAdapterState extends State<FlutterChatAdapter> {
  late core.InMemoryChatController _controller;
  final _packageIds = <ChatMessageId, String>{};
  final _messages = <String, ChatMessage>{};
  final GlobalKey<State<StatefulWidget>> _viewportKey = GlobalKey();
  final _visibleKeys = <String, GlobalKey>{};
  List<ChatMessageId> _lastVisible = const [];
  bool _measurementScheduled = false;
  int _nextId = 0;
  int _generation = 0;
  bool _loadingOlder = false;

  @override
  void initState() {
    super.initState();
    _controller = core.InMemoryChatController();
  }

  @override
  void didUpdateWidget(covariant FlutterChatAdapter oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.owner != widget.owner) {
      _controller.dispose();
      _controller = core.InMemoryChatController();
      _packageIds.clear();
      _messages.clear();
      _visibleKeys.clear();
      _lastVisible = const [];
      _nextId = 0;
      _generation++;
    }
  }

  List<core.Message> _presentationMessages(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    _messages.clear();
    return [
      for (final message in widget.history.messages)
        (() {
          final local = _packageIds.putIfAbsent(
            message.id,
            () => 'message-${_nextId++}',
          );
          _messages[local] = message;
          return core.TextMessage(
            id: local,
            authorId: message.senderDid == widget.owner ? 'me' : 'peer',
            createdAt: message.sentAt,
            text: switch (message.kind) {
              ChatMessageKind.text => message.text!,
              ChatMessageKind.deleted => l10n.chatMessageDeleted,
              ChatMessageKind.unsupported => l10n.chatMessageUnsupported,
            },
          );
        })(),
    ];
  }

  void _scheduleVisible() {
    if (_measurementScheduled || widget.onPresented == null) return;
    _measurementScheduled = true;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _measurementScheduled = false;
      if (!mounted) return;
      final viewport = _viewportKey.currentContext?.findRenderObject();
      if (viewport is! RenderBox || !viewport.hasSize || !viewport.attached) {
        return;
      }
      final bounds = viewport.localToGlobal(Offset.zero) & viewport.size;
      final visible = <ChatMessageId>[];
      for (final entry in _visibleKeys.entries) {
        final box = entry.value.currentContext?.findRenderObject();
        final message = _messages[entry.key];
        if (message == null ||
            box is! RenderBox ||
            !box.hasSize ||
            !box.attached) {
          continue;
        }
        final intersection = bounds.intersect(
          box.localToGlobal(Offset.zero) & box.size,
        );
        if (intersection.width > 1 && intersection.height > 1) {
          visible.add(message.id);
        }
      }
      if (!listEquals(visible, _lastVisible)) {
        _lastVisible = visible;
        widget.onPresented?.call(List.unmodifiable(visible));
      }
    });
  }

  Future<void> _loadOlder() async {
    if (_loadingOlder) return;
    setState(() => _loadingOlder = true);
    try {
      await widget.onLoadOlder?.call();
    } finally {
      if (mounted) setState(() => _loadingOlder = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    _scheduleVisible();
    final messages = _presentationMessages(context);
    if (_controller.messages.isEmpty && messages.isNotEmpty) {
      // Before the package subscribes, setMessages updates its in-memory list.
      unawaited(_controller.setMessages(messages, animated: false));
    } else if (!_sameMessages(_controller.messages, messages)) {
      unawaited(_controller.setMessages(messages, animated: false));
    }
    final theme = Theme.of(context);
    return Column(
      children: [
        Expanded(
          child: Stack(
            fit: StackFit.expand,
            children: [
              NotificationListener<ScrollNotification>(
                key: _viewportKey,
                onNotification: (_) {
                  _scheduleVisible();
                  return false;
                },
                child: ui.Chat(
                  key: ValueKey(_generation),
                  currentUserId: 'me',
                  resolveUser: (id) async => core.User(id: id),
                  chatController: _controller,
                  crossCache: const DisabledChatMediaCache(),
                  backgroundColor: theme.colorScheme.surface,
                  builders: core.Builders(
                    chatAnimatedListBuilder: (_, itemBuilder) =>
                        ui.ChatAnimatedList(
                          itemBuilder: itemBuilder,
                          reversed: true,
                          onEndReached:
                              widget.history.hasMore &&
                                  widget.onLoadOlder != null
                              ? _loadOlder
                              : null,
                        ),
                    // Loading overlays the top padding; a changing sliver
                    // height would move the owner's current message anchor.
                    loadMoreBuilder: (_) => const SizedBox.shrink(),
                    composerBuilder: (_) => const SizedBox.shrink(),
                    linkPreviewBuilder: (_, _, _) => null,
                    textMessageBuilder:
                        (
                          _,
                          message,
                          _, {
                          required isSentByMe,
                          groupStatus,
                        }) => Align(
                          alignment: isSentByMe
                              ? Alignment.centerRight
                              : Alignment.centerLeft,
                          child: Container(
                            margin: const EdgeInsets.symmetric(
                              vertical: 4,
                              horizontal: 12,
                            ),
                            padding: const EdgeInsets.all(12),
                            constraints: const BoxConstraints(maxWidth: 560),
                            decoration: BoxDecoration(
                              color: isSentByMe
                                  ? theme.colorScheme.primaryContainer
                                  : theme.colorScheme.surfaceContainerHighest,
                              borderRadius: BorderRadius.circular(16),
                            ),
                            child: Column(
                              mainAxisSize: MainAxisSize.min,
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                if (widget.onLink != null &&
                                    _messages[message.id]?.kind ==
                                        ChatMessageKind.text)
                                  ChatMessageText(
                                    text: message.text,
                                    onLink: widget.onLink!,
                                  )
                                else
                                  SelectableText(
                                    message.text,
                                    style: theme.textTheme.bodyMedium,
                                  ),
                                if (_messages[message.id] case final original?
                                    when original.kind == ChatMessageKind.text)
                                  if (widget.messageFooterBuilder != null)
                                    widget.messageFooterBuilder!(original),
                                if (!isSentByMe &&
                                    widget.onReport != null &&
                                    _messages[message.id]?.kind ==
                                        ChatMessageKind.text)
                                  IconButton(
                                    tooltip: AppLocalizations.of(context)
                                        .chatReportTitle,
                                    icon: const Icon(Icons.flag_outlined),
                                    onPressed: () {
                                      final original = _messages[message.id];
                                      if (original != null) {
                                        widget.onReport?.call(original.id);
                                      }
                                    },
                                  ),
                              ],
                            ),
                          ),
                        ),
                    chatMessageBuilder:
                        (
                          _,
                          message,
                          _,
                          _,
                          child, {
                          required isSentByMe,
                          isRemoved,
                          groupStatus,
                        }) => isRemoved == true
                        ? child
                        : KeyedSubtree(
                            key: _visibleKeys.putIfAbsent(
                              message.id,
                              GlobalKey.new,
                            ),
                            child: child,
                          ),
                    emptyChatListBuilder: (_) => const SizedBox.shrink(),
                  ),
                ),
              ),
              if (_loadingOlder)
                IgnorePointer(
                  child: Align(
                    alignment: Alignment.topCenter,
                    child: LinearProgressIndicator(
                      semanticsLabel: AppLocalizations.of(context)
                          .chatLoadingOlder,
                    ),
                  ),
                ),
            ],
          ),
        ),
        widget.composer,
      ],
    );
  }

  bool _sameMessages(List<core.Message> a, List<core.Message> b) =>
      a.length == b.length &&
      List.generate(
        a.length,
        (index) => a[index] == b[index],
      ).every((value) => value);

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }
}
