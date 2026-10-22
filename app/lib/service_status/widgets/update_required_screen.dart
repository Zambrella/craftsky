import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/service_status/data/status_actions.dart';
import 'package:craftsky_app/service_status/providers/service_status_controller.dart';
import 'package:craftsky_app/service_status/providers/status_runtime_provider.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

class UpdateRequiredScreen extends ConsumerStatefulWidget {
  const new({super.key});
  @override
  ConsumerState<UpdateRequiredScreen> createState() =>
      _UpdateRequiredScreenState();
}

class _UpdateRequiredScreenState extends ConsumerState<UpdateRequiredScreen> {
  bool _pending = false;
  bool _failed = false;
  Future<void> _activate() async {
    if (_pending) return;
    final controller = ref.read(serviceStatusControllerProvider.notifier);
    setState(() {
      _pending = true;
      _failed = false;
    });
    var success = false;
    try {
      final platform = ref.read(statusPlatformProvider);
      if (platform == StatusPlatform.web) {
        ref.read(statusReloadProvider)();
        // The document is leaving; retain the guard until it unloads.
        return;
      } else if (platform != null) {
        success = await ref.read(statusExternalLauncherProvider)(
          officialStatusStore(platform),
        );
      }
    } on Object catch (error, stack) {
      controller.reportActionFailure(error, stack);
    }
    if (mounted) {
      setState(() {
        _pending = false;
        _failed = !success;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Material(
      color: Theme.of(context).colorScheme.surface,
      child: SafeArea(
        child: Semantics(
          scopesRoute: true,
          explicitChildNodes: true,
          namesRoute: true,
          label: l10n.serviceStatusUpdateTitle,
          child: LayoutBuilder(
            builder: (context, constraints) => SingleChildScrollView(
              child: ConstrainedBox(
                constraints: BoxConstraints(minHeight: constraints.maxHeight),
                child: Center(
                  child: ConstrainedBox(
                    constraints: const BoxConstraints(maxWidth: 560),
                    child: Padding(
                      padding: const EdgeInsets.all(24),
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          const Icon(Icons.system_update, size: 64),
                          const SizedBox(height: 24),
                          Semantics(
                            headingLevel: 1,
                            child: Text(
                              l10n.serviceStatusUpdateTitle,
                              textAlign: TextAlign.center,
                              style: Theme.of(context).textTheme.headlineMedium,
                            ),
                          ),
                          const SizedBox(height: 16),
                          Text(
                            l10n.serviceStatusUpdateDescription,
                            textAlign: TextAlign.center,
                          ),
                          const SizedBox(height: 24),
                          if (_failed)
                            Text(
                              l10n.serviceStatusActionFailed,
                              textAlign: TextAlign.center,
                            ),
                          ChunkyButton(
                            autofocus: true,
                            onPressed: _pending ? null : _activate,
                            child: Text(
                              ref.watch(statusPlatformProvider) ==
                                      StatusPlatform.web
                                  ? l10n.serviceStatusReload
                                  : l10n.serviceStatusUpdate,
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
