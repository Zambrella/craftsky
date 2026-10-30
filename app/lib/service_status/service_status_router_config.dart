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
  backButtonDispatcher: _StatusBackDispatcher(
    maintenance,
    dismissAnnouncement,
    () =>
        router.routerDelegate.navigatorKey.currentState != null &&
        router.routerDelegate.currentConfiguration.isNotEmpty,
  ),
);

final serviceStatusRouterConfigProvider =
    Provider<RouterConfig<RouteMatchList>>(
      (ref) => serviceStatusRouterConfig(
        ref.watch(goRouterProvider),
        () => ref.read(serviceStatusControllerProvider.notifier).blockingActive,
        dismissAnnouncement: () async {
          final controller = ref.read(serviceStatusControllerProvider.notifier);
          if (controller.announcement == null) return false;
          await controller.dismissAnnouncement();
          return true;
        },
      ),
    );

class _StatusBackDispatcher extends RootBackButtonDispatcher {
  new(this.maintenance, this.dismissAnnouncement, this.navigatorMounted);
  final bool Function() maintenance;
  final bool Function() navigatorMounted;
  final Future<bool> Function()? dismissAnnouncement;
  @override
  Future<bool> didPopRoute() async {
    if (maintenance()) return true;
    if (await dismissAnnouncement?.call() ?? false) return true;
    // Account initialization can temporarily replace the router subtree.
    // GoRouter cannot dispatch Back through its absent Navigator/match stack.
    if (!navigatorMounted()) return true;
    return await super.didPopRoute();
  }
}
