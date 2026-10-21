import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/theme/chunky_icon_button.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

class ProfileMessageButton extends ConsumerWidget {
  const new({
    required this.recipient,
    required this.onOpen,
    required this.onEnable,
    this.disabled = false,
    super.key,
  });
  final Did recipient;
  final VoidCallback onOpen;
  final VoidCallback onEnable;
  final bool disabled;
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final lease = ref.watch(
      sessionRegistryProvider.select((value) => value.value?.activeLease),
    );
    if (lease == null || lease.session.account.did == recipient) {
      return const SizedBox.shrink();
    }
    final account = lease.session.account;
    final l10n = AppLocalizations.of(context);
    final state = ref.watch(chatStatusProvider(account));
    final status = state.isLoading || state.hasError ? null : state.value;
    final needsConsent =
        status?.state == ChatConnectionState.disabled ||
        status?.state == ChatConnectionState.authorizationNeeded;
    var available = false;
    var recipientUnavailable = false;
    if (status?.state == ChatConnectionState.ready && status?.canSend == true) {
      final availability = ref.watch(
        chatRecipientAvailabilityProvider(
          account,
          ChatRecipientId.fromDid(recipient),
        ),
      );
      available =
          !availability.isLoading &&
          !availability.hasError &&
          availability.value?.canChat == true;
      recipientUnavailable =
          !availability.isLoading &&
          !availability.hasError &&
          availability.value?.canChat == false;
    }
    return ChunkyIconButton(
      icon: CraftskyIconsBold.comment,
      tooltip: needsConsent ? l10n.chatEnableTitle : l10n.chatComposeLabel,
      onPressed: disabled
          ? null
          : () {
              ActiveAccountOperation.fromWidget(
                ref,
                lease,
              ).publish(() {
                if (needsConsent) {
                  onEnable();
                } else if (available) {
                  onOpen();
                } else {
                  context.showInfo(
                    recipientUnavailable
                        ? l10n.chatRecipientUnavailable
                        : l10n.chatUnavailable,
                  );
                }
              });
            },
    );
  }
}
