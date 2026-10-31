import 'dart:async';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:craftsky_app/theme/chunky_icon_button.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

class ProfileMessageButton extends ConsumerStatefulWidget {
  const new({
    required this.recipient,
    required this.onOpen,
    required this.onEnable,
    this.disabled = false,
    super.key,
  });
  final Did recipient;
  final ValueChanged<ChatConversationId> onOpen;
  final VoidCallback onEnable;
  final bool disabled;
  @override
  ConsumerState<ProfileMessageButton> createState() =>
      _ProfileMessageButtonState();
}

class _ProfileMessageButtonState extends ConsumerState<ProfileMessageButton> {
  bool _starting = false;

  Future<void> _start(ActiveAccountLease lease) async {
    if (_starting) return;
    final recipient = widget.recipient;
    final authority = ActiveAccountOperation.fromWidget(ref, lease);
    final operation = ActiveAccountOperation.withGuard(
      ownership: lease,
      isCurrent: () =>
          authority.isCurrent &&
          widget.recipient == recipient &&
          (ModalRoute.of(context)?.isCurrent ?? true),
    );
    final account = lease.session.account;
    final reporter = GuardedErrorReporter(ref.read(errorReporterProvider));
    setState(() => _starting = true);
    try {
      final repository = await operation.step(
        () => ref.read(messagingChatRepositoryProvider(account).future),
      );
      final conversation = await operation.step(
        () => repository.start(recipient),
      );
      operation.publish(() => widget.onOpen(conversation.id));
    } on AccountOperationDiscarded {
      return;
    } on Object catch (error, stack) {
      if (mounted && operation.isCurrent) {
        unawaited(
          reporter.captureException(
            error,
            stackTrace: stack,
            context: ReportContext(
              feature: 'Chat',
              operation: 'start',
              classification: 'chat.failure',
              workflow: PrivateOperationalFailureContext(
                accountDid: account.did.value,
              ),
            ),
          ),
        );
        context.showError(AppLocalizations.of(context).chatUnavailable);
      }
    } finally {
      if (mounted) setState(() => _starting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final lease = ref.watch(
      sessionRegistryProvider.select((value) => value.value?.activeLease),
    );
    if (lease == null || lease.session.account.did == widget.recipient) {
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
          ChatRecipientId.fromDid(widget.recipient),
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
      isLoading: _starting,
      tooltip: needsConsent ? l10n.chatEnableTitle : l10n.chatComposeLabel,
      onPressed: widget.disabled || _starting
          ? null
          : () {
              ActiveAccountOperation.fromWidget(
                ref,
                lease,
              ).publish(() {
                if (needsConsent) {
                  widget.onEnable();
                } else if (available) {
                  unawaited(_start(lease));
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
