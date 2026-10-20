import 'dart:async';

import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/pages/messaging_consent_page.dart';
import 'package:craftsky_app/chat/services/chat_settings_controller.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/theme/craftsky_select_inputs.dart';
import 'package:material_ui/material_ui.dart';

class MessagingSettingsView extends StatelessWidget {
  const new({
    required this.controller,
    this.onClose,
    this.onAuthorize,
    super.key,
  });
  final ChatSettingsController controller;
  final VoidCallback? onClose;
  final ValueChanged<bool>? onAuthorize;

  Future<void> _reviewPermissions(BuildContext context) async {
    final push = await showDialog<bool>(
      context: context,
      builder: (context) => Dialog.fullscreen(
        child: MessagingConsentPage(
          initialPushEnabled: controller.pushEnabled ?? true,
          onClose: () => Navigator.pop(context),
          onAuthorize: (push) => Navigator.pop(context, push),
        ),
      ),
    );
    if (push != null && context.mounted) onAuthorize?.call(push);
  }

  Future<void> _disconnect(BuildContext context) async {
    final l10n = AppLocalizations.of(context);
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(l10n.chatDisconnectTitle),
        content: Text(l10n.chatDisconnectExplanation),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: Text(l10n.actionCancel),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, true),
            child: Text(l10n.chatDisconnect),
          ),
        ],
      ),
    );
    if (confirmed == true) await controller.disconnect();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return ListenableBuilder(
      listenable: controller,
      builder: (context, _) => Scaffold(
        appBar: AppBar(
          leading: onClose == null ? null : BackButton(onPressed: onClose),
          title: Text(l10n.chatSettingsTitle),
        ),
        body: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 640),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  if (controller.loading) const LinearProgressIndicator(),
                  if (controller.failed) ...[
                    Text(l10n.chatUnavailable),
                    TextButton(
                      onPressed: controller.busy
                          ? null
                          : () => unawaited(controller.load()),
                      child: Text(l10n.chatRefresh),
                    ),
                  ],
                  if (controller.disconnected)
                    Text(l10n.chatDisconnected)
                  else ...[
                    SwitchListTile(
                      title: Text(l10n.chatPushTitle),
                      subtitle: Text(l10n.chatPushDisclosure),
                      value: controller.pushEnabled ?? false,
                      onChanged:
                          controller.busy || controller.pushEnabled == null
                          ? null
                          : (value) =>
                                unawaited(controller.setPush(enabled: value)),
                    ),
                    const SizedBox(height: 24),
                    Text(l10n.chatIncomingExplanation),
                    const SizedBox(height: 12),
                    if (controller.policy case final policy?)
                      CraftskySingleSelectInput<ChatIncomingPolicy>(
                        label: l10n.chatIncomingTitle,
                        keyPrefix: 'chat-incoming',
                        value: policy.value,
                        options: [
                          CraftskySelectOption(
                            value: ChatIncomingPolicy.everyone,
                            label: l10n.chatIncomingEveryone,
                          ),
                          CraftskySelectOption(
                            value: ChatIncomingPolicy.following,
                            label: l10n.chatIncomingFollowing,
                          ),
                          CraftskySelectOption(
                            value: ChatIncomingPolicy.nobody,
                            label: l10n.chatIncomingNobody,
                          ),
                        ],
                        onChanged: controller.busy
                            ? null
                            : (value) {
                                if (value != null) {
                                  unawaited(controller.setPolicy(value));
                                }
                              },
                      )
                    else
                      Text(l10n.chatPolicyUnavailable),
                    if (onAuthorize != null)
                      TextButton(
                        onPressed: controller.busy
                            ? null
                            : () => unawaited(_reviewPermissions(context)),
                        child: Text(l10n.chatReviewPermissions),
                      ),
                    const SizedBox(height: 24),
                    OutlinedButton(
                      onPressed: controller.busy || controller.loading
                          ? null
                          : () => unawaited(_disconnect(context)),
                      child: Text(l10n.chatDisconnectTitle),
                    ),
                  ],
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
