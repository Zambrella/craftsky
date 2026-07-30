import 'dart:async';

import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/pages/profile_page.dart';
import 'package:craftsky_app/profile/providers/toggle_follow_profile_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/profile/widgets/profile_card.dart';
import 'package:craftsky_app/profile/widgets/profile_customisation_theme.dart';
import 'package:craftsky_app/profile/widgets/profile_presentation_page.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Owns both visual states of a profile route.
class ProfileRoutePresentation extends ConsumerStatefulWidget {
  const ProfileRoutePresentation({
    required this.handle,
    required this.startsCompact,
    this.primaryColor,
    this.backgroundIllustration,
    this.avatarFrame,
    super.key,
  });

  final String handle;
  final bool startsCompact;
  final Color? primaryColor;
  final ProfileBackgroundIllustration? backgroundIllustration;
  final ProfileAvatarFrame? avatarFrame;

  static const expansionDuration = Duration(milliseconds: 600);

  @override
  ConsumerState<ProfileRoutePresentation> createState() =>
      _ProfileRoutePresentationState();
}

class _ProfileRoutePresentationState
    extends ConsumerState<ProfileRoutePresentation>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;
  late bool _expanded;
  bool _transitioning = false;
  final GlobalKey _surfaceMeasurementKey = GlobalKey(
    debugLabel: 'profile-responsive-surface',
  );
  double? _compactHeight;

  @override
  void initState() {
    super.initState();
    _expanded = !widget.startsCompact;
    _controller =
        AnimationController(
          vsync: this,
          duration: ProfileRoutePresentation.expansionDuration,
          value: _expanded ? 1 : 0,
        )..addStatusListener((status) {
          if (status == AnimationStatus.completed && mounted) {
            setState(() => _transitioning = false);
          }
        });
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (_expanded && !_transitioning) {
      return KeyedSubtree(
        key: const Key('profile-route-expanded'),
        child: _fullProfile(),
      );
    }

    final profileAsync = ref.watch(userProfileProvider(widget.handle));
    final toggleState = ref.watch(toggleFollowProfileProvider);
    final auth = ref.watch(authSessionProvider).value;
    ref.listen(toggleFollowProfileProvider, (previous, next) {
      if (previous is AsyncLoading && next is AsyncError) {
        context.showError(
          AppLocalizations.of(context).profileFollowToggleError,
        );
        ref.read(toggleFollowProfileProvider.notifier).reset();
      }
    });

    return ProfileCustomisationTheme(
      primaryColor: widget.primaryColor,
      child: AnimatedBuilder(
        key: _transitioning
            ? const Key('profile-route-expansion')
            : const Key('profile-route-compact'),
        animation: _controller,
        builder: (context, child) {
          final value = _controller.value;
          final progress = Curves.easeInOutCubicEmphasized.transform(value);
          return AbsorbPointer(
            absorbing: _transitioning,
            child: Stack(
              fit: StackFit.expand,
              children: [
                if (!_expanded || _transitioning)
                  GestureDetector(
                    key: const Key('profile-route-barrier'),
                    behavior: HitTestBehavior.opaque,
                    onTap: () => Navigator.of(context).pop(),
                    child: ColoredBox(
                      color: Colors.black.withValues(
                        alpha: 0.54 * (1 - progress),
                      ),
                    ),
                  ),
                if (!_expanded || _transitioning)
                  _compactProfile(
                    profileAsync: profileAsync,
                    toggleState: toggleState,
                    auth: auth,
                    expansionProgress: progress,
                    transitionProgress: value,
                  ),
              ],
            ),
          );
        },
      ),
    );
  }

  Widget _fullProfile() => ProfilePage(
    handle: widget.handle,
    primaryColor: widget.primaryColor,
    backgroundIllustration: widget.backgroundIllustration,
    avatarFrame: widget.avatarFrame,
  );

  Widget _compactProfile({
    required AsyncValue<Profile> profileAsync,
    required AsyncValue<Profile?> toggleState,
    required AuthState? auth,
    required double expansionProgress,
    required double transitionProgress,
  }) {
    return switch (profileAsync) {
      AsyncValue(:final value?) => ProfileCard(
        profile: value,
        primaryColor: widget.primaryColor,
        backgroundIllustration: widget.backgroundIllustration,
        avatarFrame: widget.avatarFrame,
        isOwnProfile: _isOwnProfile(auth, value.did.toString()),
        isPrimaryActionBusy: toggleState.isLoading,
        expansionProgress: expansionProgress,
        transitionProgress: transitionProgress,
        compactHeight: _compactHeight,
        surfaceMeasurementKey: _surfaceMeasurementKey,
        onClose: () => Navigator.of(context).pop(),
        onVisitProfile: _expand,
        onPrimaryAction: () {
          unawaited(
            ref
                .read(toggleFollowProfileProvider.notifier)
                .toggle(cacheKey: widget.handle, profile: value),
          );
        },
      ),
      AsyncError() => Center(
        child: CraftskyDialog(
          title: AppLocalizations.of(context).profileLoadErrorTitle,
          body: const SizedBox.shrink(),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(context).pop(),
              child: Text(
                MaterialLocalizations.of(context).closeButtonLabel,
              ),
            ),
            ChunkyButton(
              onPressed: () =>
                  ref.invalidate(userProfileProvider(widget.handle)),
              child: Text(
                AppLocalizations.of(context).profileLoadErrorRetry,
              ),
            ),
          ],
        ),
      ),
      _ => const Center(child: StitchProgressIndicator()),
    };
  }

  void _expand() {
    if (_expanded) return;
    final surface =
        _surfaceMeasurementKey.currentContext?.findRenderObject() as RenderBox?;
    ProfilePresentationPage.markExpanded(context);
    setState(() {
      _compactHeight = surface?.size.height;
      _expanded = true;
      _transitioning = true;
    });
    unawaited(_controller.forward());
  }

  bool _isOwnProfile(AuthState? auth, String profileDid) {
    return switch (auth) {
      SignedIn(:final did) => did.toString() == profileDid,
      _ => false,
    };
  }
}
