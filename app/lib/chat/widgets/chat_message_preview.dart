import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/chat/models/chat_conversation.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_link_target.dart';
import 'package:craftsky_app/chat/models/chat_message.dart';
import 'package:craftsky_app/chat/models/chat_preview.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/chat/widgets/chat_peer_profile.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/widgets/post_summary.dart';
import 'package:craftsky_app/shared/widgets/quote_embed_surface.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

class ChatMessagePreview extends ConsumerStatefulWidget {
  const new({
    required this.account,
    required this.conversationId,
    required this.message,
    super.key,
  });
  final AccountKey account;
  final ChatConversationId conversationId;
  final ChatMessage message;
  @override
  ConsumerState<ChatMessagePreview> createState() => _ChatMessagePreviewState();
}

class _ChatMessagePreviewState extends ConsumerState<ChatMessagePreview> {
  @override
  void didUpdateWidget(covariant ChatMessagePreview oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.message.revision != widget.message.revision ||
        oldWidget.message.text != widget.message.text) {
      ref.invalidate(
        chatPreviewProvider(
          widget.account,
          widget.conversationId,
          widget.message.id,
        ),
      );
    }
  }

  void _open(ChatPreview preview, {VoidCallback? openProfile}) {
    final lease = ref.read(sessionRegistryProvider).value?.activeLease;
    if (lease?.session.account != widget.account) return;
    ActiveAccountOperation.fromWidget(ref, lease).publish(() {
      final target = preview.target;
      switch (target.kind) {
        case ChatLinkKind.post:
          unawaited(
            PostThreadRoute(
              did: target.did.value,
              rkey: target.rkey!.value,
              focus: target.focus?.value,
            ).push<void>(context),
          );
        case ChatLinkKind.profile:
          if (openProfile != null) {
            openProfile();
          } else {
            unawaited(UserProfileRoute(did: target.did).push<void>(context));
          }
        case ChatLinkKind.event:
          unawaited(
            BusinessEventRoute(
              did: target.did.value,
              rkey: target.rkey!.value,
            ).push<void>(context),
          );
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(
      chatPreviewProvider(
        widget.account,
        widget.conversationId,
        widget.message.id,
      ),
    );
    if (state.isLoading || state.hasError) return const SizedBox.shrink();
    final preview = state.value;
    if (preview == null) return const SizedBox.shrink();
    if (preview.target.kind == ChatLinkKind.post &&
        preview.postSummary != null) {
      return Padding(
        padding: const EdgeInsets.only(top: 8),
        child: QuoteEmbedSurface(
          child: PostSummary(
            data: preview.postSummary!,
            onTap: () => _open(preview),
            enableAuthorNavigation: true,
          ),
        ),
      );
    }
    final localizations = MaterialLocalizations.of(context);
    final start = preview.startsAt?.toLocal();
    final formattedTime = start == null
        ? null
        : localizations.formatTimeOfDay(TimeOfDay.fromDateTime(start));
    Widget card({
      Widget? profileAvatar,
      Widget? profileIdentity,
      VoidCallback? openProfile,
    }) => Padding(
      padding: const EdgeInsets.only(top: 8),
      child: QuoteEmbedSurface(
        child: InkWell(
          onTap: () => _open(preview, openProfile: openProfile),
          child: Padding(
            padding: const EdgeInsets.all(12),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                if (profileAvatar != null) ...[
                  profileAvatar,
                  const SizedBox(width: 12),
                ] else if (preview.thumbnail case final thumbnail?) ...[
                  Image.network(
                    thumbnail.toString(),
                    width: 64,
                    height: 64,
                    fit: BoxFit.cover,
                    errorBuilder: (_, _, _) => const SizedBox.shrink(),
                  ),
                  const SizedBox(width: 12),
                ],
                Expanded(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      if (profileIdentity != null)
                        profileIdentity
                      else
                        Text(
                          preview.title,
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                          style: Theme.of(context).textTheme.titleSmall,
                        ),
                      if (preview.excerpt.isNotEmpty)
                        Text(
                          preview.excerpt,
                          maxLines: 3,
                          overflow: TextOverflow.ellipsis,
                        ),
                      if (start != null)
                        Text(
                          '${localizations.formatCompactDate(start)} '
                          '$formattedTime',
                        ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
    if (preview.target.kind == ChatLinkKind.profile) {
      return ChatPeerProfile(
        peer: ChatActor.fromMap({
          'did': preview.target.did.value,
          'displayName': preview.title,
          'avatar': preview.thumbnail?.toString(),
        }),
        builder: (context, avatar, identity, openProfile) => card(
          profileAvatar: avatar,
          profileIdentity: identity,
          openProfile: openProfile,
        ),
      );
    }
    return card();
  }
}
