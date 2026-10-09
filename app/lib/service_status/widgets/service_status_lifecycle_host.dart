import 'dart:async';

import 'package:craftsky_app/service_status/providers/service_status_controller.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class ServiceStatusLifecycleHost extends ConsumerStatefulWidget {
  const ServiceStatusLifecycleHost({required this.child, super.key});
  final Widget child;
  @override
  ConsumerState<ServiceStatusLifecycleHost> createState() => _LifecycleState();
}

class _LifecycleState extends ConsumerState<ServiceStatusLifecycleHost>
    with WidgetsBindingObserver {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    unawaited(
      Future.microtask(() {
        if (!mounted) return;
        ref.read(serviceStatusControllerProvider.notifier)
          ..setForeground(
            foreground:
                WidgetsBinding.instance.lifecycleState == null ||
                WidgetsBinding.instance.lifecycleState ==
                    AppLifecycleState.resumed,
          )
          ..start();
      }),
    );
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) => ref
      .read(serviceStatusControllerProvider.notifier)
      .setForeground(foreground: state == AppLifecycleState.resumed);

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => widget.child;
}
