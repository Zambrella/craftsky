import 'dart:async';

import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/services/chat_recipient_controller.dart';
import 'package:craftsky_app/chat/widgets/chat_app_bar_title.dart';
import 'package:craftsky_app/chat/widgets/chat_message_effect.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
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
    final id = await widget.controller.start();
    if (mounted && id != null) widget.onOpen(id);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final controller = widget.controller;
    return ListenableBuilder(
      listenable: controller,
      builder: (context, _) => Scaffold(
        appBar: AppBar(
          leading: widget.onClose == null
              ? null
              : BackButton(onPressed: widget.onClose),
          title: ChatAppBarTitle(
            loading: controller.searching || controller.checking,
            child: Text(
              l10n.chatNewMessage,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
          ),
        ),
        body: SafeArea(
          top: false,
          child: ListView(
            children: [
              Padding(
                padding: const EdgeInsets.all(16),
                child: BrandTextField(
                  label: l10n.chatSearchCraftsky,
                  prefixIcon: const Icon(CraftskyIcons.search),
                  controller: _search,
                  enabled: !controller.starting,
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
              if (controller.failed)
                ChatMessageEffect(
                  severity: MessageSeverity.error,
                  message: l10n.chatUnavailable,
                ),
              for (final item in controller.items)
                ListTile(
                  leading: ProfileAvatar(
                    seed: item.displayName ?? item.handle.value,
                    avatarUrl: item.avatar,
                    size: ProfileAvatarSize.small,
                    customisation: item.customisation,
                  ),
                  title: Text(item.displayName ?? item.handle.value),
                  subtitle: item.displayName == null
                      ? null
                      : Text(item.handle.value),
                  selected: controller.recipient == item.did,
                  onTap: controller.starting
                      ? null
                      : () => unawaited(controller.select(item)),
                ),
              if (controller.recipient != null)
                Padding(
                  padding: const EdgeInsets.all(16),
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      if (widget.initialRecipient != null &&
                          controller.items.isEmpty)
                        Text(controller.recipient!.value),
                      if (!controller.checking &&
                          controller.availability?.canChat == false)
                        Text(l10n.chatRecipientUnavailable),
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
        ),
      ),
    );
  }
}
