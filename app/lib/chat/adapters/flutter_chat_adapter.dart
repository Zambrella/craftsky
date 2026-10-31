import 'dart:async';

import 'package:craftsky_app/chat/adapters/disabled_chat_media_cache.dart';
import 'package:craftsky_app/chat/models/chat_draft.dart';
import 'package:craftsky_app/chat/models/chat_history.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_link_target.dart';
import 'package:craftsky_app/chat/models/chat_message.dart';
import 'package:craftsky_app/chat/models/chat_outgoing_message.dart';
import 'package:craftsky_app/chat/widgets/chat_time_header.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
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
    this.outgoing = const [],
    this.onPresented,
    this.onLoadOlder,
    this.onReport,
    this.messageFooterBuilder,
    this.onLink,
    super.key,
  });
  final ChatHistory history;
  final List<ChatOutgoingMessage> outgoing;
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
  final _outgoingIds = <ChatOperationId, String>{};
  final _outgoingMessages = <String, ChatOutgoingMessage>{};
  final _messages = <String, ChatMessage>{};
  final GlobalKey<State<StatefulWidget>> _viewportKey = GlobalKey();
  final _visibleKeys = <String, GlobalKey>{};
  List<ChatMessageId> _lastVisible = const [];
  bool _measurementScheduled = false;
  int _nextId = 0;
  int _generation = 0;
  bool _loadingOlder = false;
  bool _menuOpen = false;

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
      _outgoingIds.clear();
      _outgoingMessages.clear();
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
    _outgoingMessages.clear();
    final historyIds = {
      for (final message in widget.history.messages) message.id,
    };
    for (final outgoing in widget.outgoing) {
      final local = _outgoingIds.putIfAbsent(
        outgoing.attempt.operationId,
        () => 'message-${_nextId++}',
      );
      if (outgoing.messageId case final id?) _packageIds[id] = local;
      if (!historyIds.contains(outgoing.messageId)) {
        _outgoingMessages[local] = outgoing;
      }
    }
    return [
      for (final (index, message) in widget.history.messages.indexed)
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
            metadata: {
              'timeSection': startsChatTimeSection(
                message.sentAt,
                index == 0 ? null : widget.history.messages[index - 1].sentAt,
              ),
            },
            text: switch (message.kind) {
              ChatMessageKind.text => message.text!,
              ChatMessageKind.deleted => l10n.chatMessageDeleted,
              ChatMessageKind.unsupported => l10n.chatMessageUnsupported,
            },
          );
        })(),
      for (final entry in _outgoingMessages.entries)
        core.TextMessage(
          id: entry.key,
          authorId: 'me',
          text: entry.value.attempt.text,
          status: entry.value.messageId != null
              ? null
              : switch (entry.value.attempt.state) {
                  ChatDraftSendState.pending => core.MessageStatus.sending,
                  ChatDraftSendState.failed => core.MessageStatus.error,
                  ChatDraftSendState.uncertain => null,
                },
        ),
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

  Future<void> _showMessageMenu(BuildContext context, String localId) async {
    final original = _messages[localId];
    if (_menuOpen || original?.kind != ChatMessageKind.text) return;
    final owner = widget.owner;
    final generation = _generation;
    final route = ModalRoute.of(context);
    bool current() {
      final message = _messages[localId];
      return mounted &&
          context.mounted &&
          widget.owner == owner &&
          _generation == generation &&
          (route?.isCurrent ?? true) &&
          message?.id == original!.id &&
          message?.revision == original.revision &&
          message?.kind == original.kind &&
          message?.text == original.text &&
          message?.senderDid == original.senderDid;
    }

    final l10n = AppLocalizations.of(context);
    final links = ChatLinkTarget.linksIn(original!.text!).toSet();
    _menuOpen = true;
    try {
      await showCraftskyContextMenu(
        context,
        position: craftskyContextMenuAnchorPosition(context),
        groups: [
          CraftskyContextMenuGroup(
            items: [
              CraftskyContextMenuItem(
                text: MaterialLocalizations.of(context).copyButtonLabel,
                icon: CraftskyIcons.copy,
                onPressed: () async {
                  if (!current()) return;
                  try {
                    await Clipboard.setData(
                      ClipboardData(text: original.text!),
                    );
                  } on PlatformException {
                    if (context.mounted && current()) {
                      context.showError(l10n.chatUnavailable);
                    }
                  }
                },
              ),
              if (widget.onLink != null)
                for (final link in links)
                  CraftskyContextMenuItem(
                    text: l10n.profileLinkOpen,
                    description: link.toString(),
                    icon: CraftskyIcons.externalLink,
                    onPressed: () {
                      if (current()) widget.onLink?.call(link);
                    },
                  ),
            ],
          ),
          if (original.senderDid != null &&
              original.senderDid != owner &&
              widget.onReport != null)
            CraftskyContextMenuGroup(
              items: [
                CraftskyContextMenuItem(
                  text: l10n.chatReportTitle,
                  icon: CraftskyIcons.report,
                  style: CraftskyContextMenuItemStyle.destructive,
                  onPressed: () {
                    if (current() && widget.onReport != null) {
                      widget.onReport!(original.id);
                    }
                  },
                ),
              ],
            ),
        ],
      );
    } finally {
      _menuOpen = false;
    }
  }

  @override
  Widget build(BuildContext context) {
    _scheduleVisible();
    final messages = _presentationMessages(context);
    // The package's mounted rows listen for update operations. A set operation
    // alone can retain a row's previous content when its stable ID is reused.
    final previous = {
      for (final message in _controller.messages) message.id: message,
    };
    for (final message in messages) {
      final old = previous[message.id];
      if (old != null && old != message) {
        unawaited(_controller.updateMessage(old, message));
      }
    }
    if (_controller.messages.isEmpty && messages.isNotEmpty) {
      // Before the package subscribes, setMessages updates its in-memory list.
      unawaited(_controller.setMessages(messages, animated: false));
    } else if (!_sameMessages(_controller.messages, messages)) {
      unawaited(_controller.setMessages(messages, animated: false));
    }
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;
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
                  theme: core.ChatTheme.fromThemeData(theme).copyWith(
                    colors: core.ChatColors.fromThemeData(theme).copyWith(
                      primary: theme.colorScheme.primaryContainer,
                      onPrimary: theme.colorScheme.onPrimaryContainer,
                      surface: theme.scaffoldBackgroundColor,
                      surfaceContainer: swatches.paper3,
                    ),
                    shape: BorderRadius.circular(radii.r3),
                  ),
                  onMessageLongPress: (
                    context,
                    message, {
                    required index,
                    required details,
                  }) => unawaited(_showMessageMenu(context, message.id)),
                  onMessageSecondaryTap: (
                    context,
                    message, {
                    required index,
                    details,
                  }) => unawaited(_showMessageMenu(context, message.id)),
                  builders: core.Builders(
                    chatAnimatedListBuilder: (_, itemBuilder) =>
                        ui.ChatAnimatedList(
                          itemBuilder: itemBuilder,
                          reversed: true,
                          messagesGroupingMode:
                              core.MessagesGroupingMode.timeDifference,
                          messageGroupingTimeoutInSeconds: 300,
                          onEndReached:
                              widget.history.hasMore &&
                                  widget.onLoadOlder != null
                              ? _loadOlder
                              : null,
                        ),
                    // The page title owns loading feedback; a changing sliver
                    // height would move the owner's current message anchor.
                    loadMoreBuilder: (_) => const SizedBox.shrink(),
                    composerBuilder: (_) => const SizedBox.shrink(),
                    linkPreviewBuilder: (_, message, _) {
                      final original = _messages[message.id];
                      if (original?.kind != ChatMessageKind.text) return null;
                      return widget.messageFooterBuilder?.call(original!);
                    },
                    textMessageBuilder:
                        (
                          _,
                          message,
                          index, {
                          required isSentByMe,
                          groupStatus,
                        }) => ui.SimpleTextMessage(
                          message: message,
                          index: index,
                          padding: EdgeInsets.all(spacing.sp3),
                          constraints: const BoxConstraints(maxWidth: 560),
                          timeStyle: theme.textTheme.labelSmall?.copyWith(
                            fontWeight: FontWeight.w400,
                            letterSpacing: 0,
                            color: theme.colorScheme.onSurfaceVariant,
                          ),
                          showTime: false,
                          showStatus:
                              _outgoingMessages.containsKey(message.id) &&
                              message.status != null,
                        ),
                    chatMessageBuilder:
                        (
                          context,
                          message,
                          index,
                          animation,
                          child, {
                          required isSentByMe,
                          isRemoved,
                          groupStatus,
                        }) => CallbackShortcuts(
                          bindings: {
                            const SingleActivator(
                              LogicalKeyboardKey.contextMenu,
                            ): () => unawaited(
                              _showMessageMenu(context, message.id),
                            ),
                            const SingleActivator(
                              LogicalKeyboardKey.f10,
                              shift: true,
                            ): () => unawaited(
                              _showMessageMenu(context, message.id),
                            ),
                          },
                          child: Focus(
                            skipTraversal:
                                _messages[message.id]?.kind !=
                                ChatMessageKind.text,
                            child: ui.ChatMessage(
                              message: message,
                              index: index,
                              animation: animation,
                              groupStatus: groupStatus,
                              isRemoved: isRemoved,
                              headerWidget:
                                  isRemoved != true &&
                                      message.metadata?['timeSection'] == true
                                  ? ChatTimeHeader(
                                      timestamp: message.createdAt!,
                                    )
                                  : null,
                              padding: EdgeInsets.fromLTRB(
                                isSentByMe ? spacing.sp6 : spacing.sp3,
                                message.metadata?['timeSection'] == true ||
                                        groupStatus?.isFirst != false
                                    ? spacing.sp3
                                    : spacing.sp1,
                                isSentByMe ? spacing.sp3 : spacing.sp6,
                                0,
                              ),
                              horizontalPadding: spacing.sp3,
                              verticalPadding: spacing.sp3,
                              verticalGroupedPadding: spacing.sp1,
                              child: isRemoved == true
                                  ? child
                                  : KeyedSubtree(
                                      key: _visibleKeys.putIfAbsent(
                                        message.id,
                                        GlobalKey.new,
                                      ),
                                      child: child,
                                    ),
                            ),
                          ),
                        ),
                    emptyChatListBuilder: (_) => const SizedBox.shrink(),
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
