import 'dart:async';

import 'package:craftsky_app/service_status/data/status_actions.dart';
import 'package:craftsky_app/service_status/providers/service_status_controller.dart';
import 'package:craftsky_app/service_status/widgets/announcement_modal.dart';
import 'package:craftsky_app/service_status/widgets/maintenance_screen.dart';
import 'package:craftsky_app/service_status/widgets/update_required_screen.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

class ServiceStatusHost extends ConsumerStatefulWidget {
  const new({required this.child, this.onDestination, super.key});
  final Widget child;
  final Future<bool> Function(String destination)? onDestination;

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
    final update = controller.presentation == StatusPresentation.updateRequired;
    final covered = controller.blockingActive || notice != null;
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
                activating: state.activating,
                actionFailed: state.actionFailed,
                onActivate: notice.cta == null
                    ? null
                    : () {
                        unawaited(
                          controller.activateMessage(notice.revision, () {
                            final cta = notice.cta!;
                            if (cta.url != null) {
                              return ref.read(statusExternalLauncherProvider)(
                                Uri.parse(cta.url!),
                              );
                            }
                            return widget.onDestination?.call(
                                  cta.destination!,
                                ) ??
                                Future.value(false);
                          }),
                        );
                      },
                onDismiss: dismiss,
              ),
            ),
          ),
        ],
        if (update) const UpdateRequiredScreen(),
        if (maintenance)
          MaintenanceScreen(
            document: state.document!.maintenance!,
            fetching: state.fetching,
            onRetry: () =>
                ref.read(serviceStatusControllerProvider.notifier).refresh(),
          ),
      ],
    );
  }
}

/// Installed inside Navigator routes for initialization hosts; the ready router
/// uses its root dispatcher. Both read the same presentation selector.
class ServiceStatusBackGuard extends ConsumerWidget {
  const new({required this.child, super.key});
  final Widget child;
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    ref.watch(serviceStatusControllerProvider);
    final controller = ref.read(serviceStatusControllerProvider.notifier);
    return PopScope<Object?>(
      canPop: !controller.blockingActive && controller.announcement == null,
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop &&
            !controller.blockingActive &&
            controller.announcement != null) {
          unawaited(controller.dismissAnnouncement());
        }
      },
      child: child,
    );
  }
}
