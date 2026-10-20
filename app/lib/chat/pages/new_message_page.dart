import 'dart:async';

import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/services/chat_recipient_controller.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
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
          title: Text(l10n.chatNewMessage),
        ),
        body: Column(
          children: [
            Padding(
              padding: const EdgeInsets.all(16),
              child: TextField(
                controller: _search,
                enabled: !controller.starting,
                decoration: InputDecoration(labelText: l10n.chatSearchCraftsky),
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
            if (controller.searching ||
                controller.checking ||
                controller.starting)
              const LinearProgressIndicator(),
            if (controller.failed)
              Padding(
                padding: const EdgeInsets.all(16),
                child: Text(l10n.chatUnavailable),
              ),
            Expanded(
              child: ListView(
                children: [
                  for (final item in controller.items)
                    ListTile(
                      title: Text(item.displayName ?? item.handle.value),
                      subtitle: item.displayName == null
                          ? null
                          : Text(item.handle.value),
                      selected: controller.recipient == item.did,
                      onTap: controller.starting
                          ? null
                          : () => unawaited(controller.select(item)),
                    ),
                ],
              ),
            ),
            if (controller.recipient != null)
              Flexible(
                child: SingleChildScrollView(
                  child: SafeArea(
                    top: false,
                    child: Padding(
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
                          FilledButton(
                            onPressed: controller.canStart
                                ? () => unawaited(_start())
                                : null,
                            child: Text(l10n.chatStartConversation),
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }
}
