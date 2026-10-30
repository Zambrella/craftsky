import 'package:craftsky_app/service_status/providers/service_status_controller.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

class StatusMessageImage extends ConsumerStatefulWidget {
  const new({required this.url, super.key});
  final String url;
  @override
  ConsumerState<StatusMessageImage> createState() => _StatusMessageImageState();
}

class _StatusMessageImageState extends ConsumerState<StatusMessageImage> {
  bool _reported = false;
  @override
  Widget build(BuildContext context) {
    return ExcludeSemantics(
      child: Image.network(
        widget.url,
        width: double.infinity,
        height: 200,
        fit: BoxFit.contain,
        errorBuilder: (_, error, stack) {
          if (!_reported) {
            _reported = true;
            ref
                .read(serviceStatusControllerProvider.notifier)
                .reportImageFailure(
                  error,
                  stack ?? StackTrace.current,
                  expectedContentFailure: error is Exception,
                );
          }
          return const SizedBox.shrink();
        },
      ),
    );
  }
}
