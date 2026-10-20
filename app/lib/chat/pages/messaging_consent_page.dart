import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:material_ui/material_ui.dart';

/// Consent is explicit. Opening this page makes no network request and does
/// not change the shared declaration. The account controller owns the handoff.
class MessagingConsentPage extends StatefulWidget {
  const new({
    required this.onAuthorize,
    this.busy = false,
    this.initialPushEnabled = true,
    this.onClose,
    super.key,
  });

  final ValueChanged<bool> onAuthorize;
  final VoidCallback? onClose;
  final bool busy;
  final bool initialPushEnabled;

  @override
  State<MessagingConsentPage> createState() => _MessagingConsentPageState();
}

class _MessagingConsentPageState extends State<MessagingConsentPage> {
  late bool _pushEnabled;

  @override
  void initState() {
    super.initState();
    _pushEnabled = widget.initialPushEnabled;
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: AppBar(
        leading: widget.onClose == null
            ? null
            : BackButton(onPressed: widget.onClose),
        title: Text(l10n.chatEnableTitle),
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 640),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(l10n.chatSharedInboxDisclosure),
                const SizedBox(height: 16),
                Text(l10n.chatStorageDisclosure),
                const SizedBox(height: 16),
                Text(l10n.chatReportDisclosure),
                const SizedBox(height: 16),
                Text(l10n.chatPolicyDisclosure),
                const SizedBox(height: 24),
                Row(
                  children: [
                    Expanded(child: Text(l10n.chatPushTitle)),
                    Switch(
                      value: _pushEnabled,
                      onChanged: widget.busy
                          ? null
                          : (value) => setState(() => _pushEnabled = value),
                    ),
                  ],
                ),
                Text(l10n.chatPushDisclosure),
                const SizedBox(height: 24),
                FilledButton(
                  onPressed: widget.busy
                      ? null
                      : () => widget.onAuthorize(_pushEnabled),
                  child: Text(l10n.chatAuthorizeAction),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
