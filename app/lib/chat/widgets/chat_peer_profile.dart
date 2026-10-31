import 'package:craftsky_app/chat/models/chat_conversation.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/profile/models/profile_handle.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/profile/widgets/compact_profile_identity.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar_hero.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

/// Enriches the limited chat identity with the shared, account-fenced profile
/// read. Profile loading never blocks the conversation or its message history.
class ChatPeerProfile extends ConsumerWidget {
  const new({required this.peer, required this.builder, super.key});

  final ChatActor peer;
  final Widget Function(
    BuildContext context,
    Widget avatar,
    Widget identity,
    VoidCallback openProfile,
  )
  builder;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final profile = ref.watch(userProfileProvider(peer.did)).value;
    final handle = profile?.handle.value ?? peer.handle ?? '';
    final displayName = profile?.displayName ?? peer.displayName;
    final crafts = profile?.crafts ?? const <String>[];
    final avatar = ProfileAvatar(
      seed: ProfileHandle(handle).displayLabel(
        displayName: displayName,
        unavailableLabel: AppLocalizations.of(context).chatSomeone,
      ),
      avatarUrl: profile?.avatar ?? peer.avatar,
      size: ProfileAvatarSize.small,
      customisation: profile?.customisation ?? ProfileCustomisation.defaults,
    );
    return ProfileAvatarHeroSource(
      did: peer.did,
      handle: handle,
      displayName: displayName,
      crafts: crafts,
      avatar: avatar,
      builder: (context, avatar, openProfile) => builder(
        context,
        avatar,
        CompactProfileIdentity(
          handle: handle,
          displayName: displayName,
          crafts: crafts,
        ),
        openProfile,
      ),
    );
  }
}
