import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/service_status/providers/service_status_controller.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

RouterConfig<RouteMatchList> serviceStatusRouterConfig(
  GoRouter router,
  bool Function() maintenance, {
  Future<bool> Function()? dismissAnnouncement,
}) => RouterConfig<RouteMatchList>(
  routeInformationProvider: router.routeInformationProvider,
  routeInformationParser: router.routeInformationParser,
  routerDelegate: router.routerDelegate,
  backButtonDispatcher: _StatusBackDispatcher(maintenance, dismissAnnouncement),
);

final serviceStatusRouterConfigProvider =
    Provider<RouterConfig<RouteMatchList>>(
      (ref) => serviceStatusRouterConfig(
        ref.watch(goRouterProvider),
        () => ref
            .read(serviceStatusControllerProvider.notifier)
            .maintenanceActive,
        dismissAnnouncement: () async {
          final controller = ref.read(serviceStatusControllerProvider.notifier);
          if (controller.announcement == null) return false;
          await controller.dismissAnnouncement();
          return true;
        },
      ),
    );

class _StatusBackDispatcher extends RootBackButtonDispatcher {
  _StatusBackDispatcher(this.maintenance, this.dismissAnnouncement);
  final bool Function() maintenance;
  final Future<bool> Function()? dismissAnnouncement;
  @override
  Future<bool> didPopRoute() async {
    if (maintenance()) return true;
    if (await dismissAnnouncement?.call() ?? false) return true;
    return super.didPopRoute();
  }
}
