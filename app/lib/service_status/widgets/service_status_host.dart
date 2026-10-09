import 'dart:async';

import 'package:craftsky_app/service_status/providers/service_status_controller.dart';
import 'package:craftsky_app/service_status/widgets/announcement_modal.dart';
import 'package:craftsky_app/service_status/widgets/maintenance_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class ServiceStatusHost extends ConsumerStatefulWidget {
  const ServiceStatusHost({required this.child, super.key});
  final Widget child;

  @override
  ConsumerState<ServiceStatusHost> createState() => _ServiceStatusHostState();
}

class _ServiceStatusHostState extends ConsumerState<ServiceStatusHost> {
  bool _wasCovered = false;
  FocusNode? _retainedFocus;

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(serviceStatusControllerProvider);
    final controller = ref.watch(serviceStatusControllerProvider.notifier);
    final maintenance = controller.maintenanceActive;
    final notice = controller.announcement;
    final covered = maintenance || notice != null;
    if (covered && !_wasCovered) {
      _retainedFocus = FocusManager.instance.primaryFocus;
    } else if (!covered && _wasCovered) {
      final retained = _retainedFocus;
      _retainedFocus = null;
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted &&
            !_wasCovered &&
            retained?.context != null &&
            retained!.canRequestFocus) {
          retained.requestFocus();
        }
      });
    }
    _wasCovered = covered;
    void dismiss() {
      if (!state.dismissing) {
        unawaited(
          ref
              .read(serviceStatusControllerProvider.notifier)
              .dismissAnnouncement(),
        );
      }
    }

    return Stack(
      fit: StackFit.expand,
      children: [
        IgnorePointer(
          ignoring: covered,
          child: ExcludeFocus(
            excluding: covered,
            child: ExcludeSemantics(excluding: covered, child: widget.child),
          ),
        ),
        if (notice != null) ...[
          ModalBarrier(
            color: Colors.black54,
            dismissible: !state.dismissing,
            onDismiss: dismiss,
            semanticsLabel: MaterialLocalizations.of(
              context,
            ).modalBarrierDismissLabel,
          ),
          FocusScope(
            key: ValueKey(notice.revision),
            autofocus: true,
            child: CallbackShortcuts(
              bindings: {
                const SingleActivator(LogicalKeyboardKey.escape): dismiss,
              },
              child: AnnouncementModal(
                document: notice,
                dismissing: state.dismissing,
                onDismiss: dismiss,
              ),
            ),
          ),
        ],
        if (maintenance)
          MaintenanceScreen(
            document: state.document!,
            fetching: state.fetching,
            onRetry: () =>
                ref.read(serviceStatusControllerProvider.notifier).refresh(),
          ),
      ],
    );
  }
}
