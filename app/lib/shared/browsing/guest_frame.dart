import 'dart:async';

import 'package:craftsky_app/auth/pages/welcome_page.dart';
import 'package:craftsky_app/router/app_shell.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/browsing/guest_session.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';

Future<void> showGuestSignIn(BuildContext context, WidgetRef ref) async {
  final session = ref.read(guestSessionProvider);
  if (ref.read(guestPromptOpenProvider)) return;
  final prompt = ref.read(guestPromptOpenProvider.notifier)..open();
  try {
    await showDialog<void>(
      context: context,
      builder: (dialogContext) => Dialog(
        insetPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 24),
        backgroundColor: Colors.transparent,
        surfaceTintColor: Colors.transparent,
        elevation: 0,
        child: ConstrainedBox(
          constraints: BoxConstraints(
            maxWidth: 640,
            maxHeight: MediaQuery.sizeOf(context).height * .9,
          ),
          child: WelcomePage(
            embedded: true,
            onExplore: () => Navigator.of(dialogContext).pop(),
            onStartAuth: () => session.prepareSignIn(
              GoRouter.of(context).routeInformationProvider.value.uri
                  .toString(),
            ),
          ),
        ),
      ),
    );
  } finally {
    session.explore();
    prompt.close();
  }
}

/// Public chrome has no account runtime, drawers, notifications or composer.
class GuestFrame extends ConsumerStatefulWidget {
  const new({required this.child, super.key});
  final Widget child;
  @override
  ConsumerState<GuestFrame> createState() => _GuestFrameState();
}

class _GuestFrameState extends ConsumerState<GuestFrame> {
  Future<void> _showSignIn() => showGuestSignIn(context, ref);

  bool _scheduled = false;
  bool _authScheduled = false;
  @override
  Widget build(BuildContext context) {
    final delegate = GoRouter.of(context).routerDelegate;
    return ListenableBuilder(
      listenable: delegate,
      builder: (context, _) =>
          _buildLocation(context, delegate.currentConfiguration.uri),
    );
  }

  Widget _buildLocation(BuildContext context, Uri uri) {
    final explicit = uri.queryParameters['auth'] == '1';
    if (!explicit) _authScheduled = false;
    if ((!_scheduled &&
            ref.read(guestSessionProvider).shouldPrompt(uri.path)) ||
        (explicit && !_authScheduled)) {
      _scheduled = true;
      _authScheduled = explicit;
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) return;
        if (explicit) const FeedRoute().go(context);
        unawaited(_showSignIn());
      });
    }
    return GuestNavigationFrame(
      selectedIndex: uri.path.startsWith('/search') ? 1 : 0,
      onDestinationSelected: (index) =>
          context.go(index == 0 ? '/feed' : '/search'),
      child: Scaffold(
        body: Column(
          children: [
            Expanded(child: widget.child),
            if (!ref.watch(guestPromptOpenProvider) &&
                !ref.read(guestSessionProvider).shouldPrompt(uri.path) &&
                !explicit)
              SafeArea(
                top: false,
                child: Semantics(
                  container: true,
                  label: 'Join CraftSky',
                  child: Padding(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 16,
                      vertical: 8,
                    ),
                    child: Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: [
                        const Flexible(
                          flex: 9,
                          child: Text(
                            'Your craft. Your community.',
                            style: TextStyle(fontWeight: FontWeight.w700),
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                          ),
                        ),
                        const Spacer(),
                        const SizedBox(width: 12),
                        ChunkyButton(
                          onPressed: _showSignIn,
                          child: const Text('Join'),
                        ),
                      ],
                    ),
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }
}
