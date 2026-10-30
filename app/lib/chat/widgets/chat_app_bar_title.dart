import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:material_ui/material_ui.dart';

/// Progress for explicit actions; background fetching stays silent.
class ChatAppBarTitle extends StatelessWidget {
  const new({
    required this.child,
    required this.loading,
    super.key,
  });
  final Widget child;
  final bool loading;

  @override
  Widget build(BuildContext context) => Row(
    mainAxisSize: MainAxisSize.min,
    children: [
      Flexible(child: child),
      if (loading)
        const Padding(
          padding: EdgeInsetsDirectional.only(start: 8),
          child: StitchProgressIndicator(size: 16),
        ),
    ],
  );
}

class ChatLoadingPage extends StatelessWidget {
  const new({
    required this.title,
    this.onClose,
    this.body,
    this.titleWidget,
    super.key,
  });
  final String title;
  final Widget? body;
  final Widget? titleWidget;
  final VoidCallback? onClose;

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(
      leading: onClose == null ? null : BackButton(onPressed: onClose),
      toolbarHeight: titleWidget == null
          ? null
          : MediaQuery.textScalerOf(context)
                .scale(kToolbarHeight)
                .clamp(kToolbarHeight, double.infinity),
      title:
          titleWidget ??
          Text(title, maxLines: 1, overflow: TextOverflow.ellipsis),
    ),
    body: body ?? const Center(child: StitchProgressIndicator()),
  );
}
