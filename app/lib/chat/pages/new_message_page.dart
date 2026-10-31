import 'dart:async';

import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/services/chat_recipient_controller.dart';
import 'package:craftsky_app/chat/widgets/chat_app_bar_title.dart';
import 'package:craftsky_app/chat/widgets/chat_message_effect.dart';
import 'package:craftsky_app/chat/widgets/chat_skeleton.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/widgets/compact_profile_identity.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/search/models/profile_search_page.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/shared/messaging/widgets/craftsky_snack_bar.dart';
import 'package:craftsky_app/shared/widgets/action_progress.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:material_ui/material_ui.dart';

class NewMessageView extends StatefulWidget {
  const new({
    required this.controller,
    required this.onOpen,
    this.initialRecipient,
    this.onClose,
    super.key,
  });
  final ChatRecipientController controller;
  final Did? initialRecipient;
  final VoidCallback? onClose;
  final ValueChanged<ChatConversationId> onOpen;
  @override
  State<NewMessageView> createState() => _NewMessageViewState();
}

class _NewMessageViewState extends State<NewMessageView> {
  final _search = TextEditingController();
  Timer? _debounce;
  @override
  void initState() {
    super.initState();
    if (widget.initialRecipient case final did?) {
      unawaited(widget.controller.selectRecipient(did));
    }
  }

  @override
  void didUpdateWidget(covariant NewMessageView oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.initialRecipient != widget.initialRecipient ||
        oldWidget.controller != widget.controller) {
      _debounce?.cancel();
      _search.clear();
      widget.controller.beginSearch();
      if (widget.initialRecipient case final did?) {
        unawaited(widget.controller.selectRecipient(did));
      }
    }
  }

  @override
  void dispose() {
    _debounce?.cancel();
    _search.dispose();
    super.dispose();
  }

  Future<void> _start() async {
    final controller = widget.controller;
    final id = await controller.start();
    if (mounted &&
        widget.controller == controller &&
        (ModalRoute.of(context)?.isCurrent ?? true) &&
        id != null) {
      widget.onOpen(id);
    }
  }

  Future<void> _select(ProfileSearchResult profile) async {
    final controller = widget.controller;
    if (controller.checking || controller.starting) return;
    FocusScope.of(context).unfocus();
    await controller.select(profile);
    if (!mounted ||
        widget.controller != controller ||
        controller.recipient != profile.did ||
        !(ModalRoute.of(context)?.isCurrent ?? true)) {
      return;
    }
    if (controller.availability?.canChat == false) {
      context.showInfo(AppLocalizations.of(context).chatRecipientUnavailable);
      return;
    }
    await _start();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final controller = widget.controller;
    return ListenableBuilder(
      listenable: controller,
      builder: (context, _) => ChatMessageEffect(
        severity: controller.failed
            ? MessageSeverity.error
            : MessageSeverity.info,
        message: controller.failed
            ? l10n.chatUnavailable
            : widget.initialRecipient != null &&
                  controller.items.isEmpty &&
                  !controller.checking &&
                  controller.availability?.canChat == false
            ? l10n.chatRecipientUnavailable
            : null,
        child: Scaffold(
          appBar: AppBar(
            leading: widget.onClose == null
                ? null
                : BackButton(onPressed: widget.onClose),
            title: ChatAppBarTitle(
              loading: controller.checking || controller.starting,
              child: Text(
                l10n.chatNewMessage,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
            ),
          ),
          body: SafeArea(
            top: false,
            child: CustomScrollView(
              slivers: [
                SliverToBoxAdapter(
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: BrandTextField(
                      label: l10n.chatSearchCraftsky,
                      prefixIcon: const Icon(CraftskyIcons.search),
                      controller: _search,
                      autocorrect: false,
                      enableSuggestions: false,
                      enabled: !controller.checking && !controller.starting,
                      onChanged: (value) {
                        controller.beginSearch();
                        _debounce?.cancel();
                        _debounce = Timer(
                          const Duration(milliseconds: 300),
                          () => unawaited(controller.search(value)),
                        );
                      },
                    ),
                  ),
                ),
                if (controller.searching && controller.items.isEmpty)
                  const SliverFillRemaining(child: ChatListSkeleton()),
                SliverList.list(
                  children: [
                    for (final item in controller.items)
                      ListTile(
                        horizontalTitleGap: 12,
                        minLeadingWidth: 0,
                        leading: ProfileAvatar(
                          seed: item.displayName ?? item.handle.value,
                          avatarUrl: item.avatar,
                          size: ProfileAvatarSize.small,
                          customisation: item.customisation,
                        ),
                        title: CompactProfileIdentity(
                          handle: item.handle.value,
                          displayName: item.displayName,
                          crafts: item.crafts,
                        ),
                        selected: controller.recipient == item.did,
                        onTap: controller.checking || controller.starting
                            ? null
                            : () => unawaited(_select(item)),
                      ),
                    if (widget.initialRecipient != null &&
                        controller.items.isEmpty &&
                        controller.recipient != null)
                      Padding(
                        padding: const EdgeInsets.all(16),
                        child: Column(
                          mainAxisSize: MainAxisSize.min,
                          crossAxisAlignment: CrossAxisAlignment.stretch,
                          children: [
                            Text(controller.recipient!.value),
                            ChunkyButton(
                              onPressed: controller.canStart
                                  ? () => unawaited(_start())
                                  : null,
                              child: ActionProgress(
                                isLoading: controller.starting,
                                child: Text(l10n.chatStartConversation),
                              ),
                            ),
                          ],
                        ),
                      ),
                  ],
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
