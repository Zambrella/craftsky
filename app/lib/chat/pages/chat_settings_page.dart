import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/chat/services/chat_settings_controller.dart';
import 'package:craftsky_app/chat/widgets/chat_app_bar_title.dart';
import 'package:craftsky_app/chat/widgets/chat_content_gate.dart';
import 'package:craftsky_app/chat/widgets/chat_message_effect.dart';
import 'package:craftsky_app/chat/widgets/messaging_explanation.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/message_action.dart';
import 'package:craftsky_app/shared/messaging/widgets/craftsky_snack_bar.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_select_inputs.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

/// Shared by the disabled Messages destination and the settings route.
class MessagingSettingsPage extends ConsumerWidget {
  const new({
    required this.account,
    this.onClose,
    this.onAuthorize,
    this.onOpenMessages,
    super.key,
  });
  final AccountKey account;
  final VoidCallback? onClose;
  final ValueChanged<bool>? onAuthorize;
  final VoidCallback? onOpenMessages;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final family = chatSettingsControllerProvider(account);
    return ref
        .watch(family)
        .when(
          loading: () => ChatLoadingPage(
            title: AppLocalizations.of(context).chatSettingsTitle,
            onClose: onClose,
          ),
          error: (_, _) => Scaffold(
            appBar: AppBar(
              leading: onClose == null ? null : BackButton(onPressed: onClose),
              title: Text(AppLocalizations.of(context).chatSettingsTitle),
            ),
            body: ChatFailurePanel(
              message: AppLocalizations.of(context).chatUnavailable,
              onRetry: () => ref.invalidate(family),
            ),
          ),
          data: (controller) => MessagingSettingsView(
            controller: controller,
            onClose: onClose,
            onAuthorize: onAuthorize,
            onOpenMessages: onOpenMessages,
          ),
        );
  }
}

class MessagingSettingsView extends StatelessWidget {
  const new({
    required this.controller,
    this.onClose,
    this.onAuthorize,
    this.onOpenMessages,
    super.key,
  });
  final ChatSettingsController controller;
  final VoidCallback? onClose;
  final ValueChanged<bool>? onAuthorize;
  final VoidCallback? onOpenMessages;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return ListenableBuilder(
      listenable: controller,
      builder: (context, _) => controller.loading && !controller.initialized
          ? ChatLoadingPage(title: l10n.chatSettingsTitle, onClose: onClose)
          : Scaffold(
              appBar: AppBar(
                leading: onClose == null
                    ? null
                    : BackButton(onPressed: onClose),
                title: ChatAppBarTitle(
                  loading: controller.busy,
                  child: Text(
                    l10n.chatSettingsTitle,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
              ),
              body: SingleChildScrollView(
                padding: const EdgeInsets.all(24),
                child: Center(
                  child: ConstrainedBox(
                    constraints: const BoxConstraints(maxWidth: 640),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        if (controller.failed &&
                            !controller.needsAuthorization &&
                            !controller.restricted)
                          ChatMessageEffect(
                            severity: MessageSeverity.error,
                            message: l10n.chatUnavailable,
                            action: MessageAction(
                              label: l10n.chatRefresh,
                              onPressed: () => unawaited(controller.load()),
                            ),
                            actionEnabled: !controller.busy,
                          ),
                        if (controller.restricted) Text(l10n.chatRestricted),
                        if (!controller.enabled) ...[
                          const MessagingExplanation(),
                          const SizedBox(height: 24),
                        ],
                        SwitchListTile(
                          contentPadding: EdgeInsets.zero,
                          title: Text(l10n.chatEnableTitle),
                          subtitle: Text(l10n.chatEnableExplanation),
                          value: controller.enabled,
                          onChanged: controller.busy || controller.loading
                              ? null
                              : (value) => unawaited(
                                  controller.setEnabled(enabled: value),
                                ),
                        ),
                        if (controller.needsAuthorization) ...[
                          Text(l10n.chatAuthorizationNeeded),
                          if (onAuthorize != null)
                            TextButton(
                              onPressed: controller.busy
                                  ? null
                                  : () => onAuthorize!(
                                      controller.pushEnabled ?? true,
                                    ),
                              child: Text(l10n.chatReviewPermissions),
                            ),
                        ],
                        if (controller.enabled) ...[
                          const SizedBox(height: 24),
                          SwitchListTile(
                            contentPadding: EdgeInsets.zero,
                            title: Text(l10n.chatPushTitle),
                            subtitle: Text(l10n.chatPushDisclosure),
                            value: controller.pushEnabled ?? false,
                            onChanged:
                                controller.busy ||
                                    controller.loading ||
                                    controller.pushEnabled == null
                                ? null
                                : (value) => unawaited(
                                    controller.setPush(enabled: value),
                                  ),
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

                          if (onOpenMessages != null && controller.canRead)
                            Padding(
                              padding: const EdgeInsets.only(top: 24),
                              child: ChunkyButton(
                                onPressed: controller.busy || controller.loading
                                    ? null
                                    : onOpenMessages,
                                child: Text(l10n.chatOpenMessages),
                              ),
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
