import 'dart:async';
import 'dart:math' as math;

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/browsing/browsing_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final _statsRevisionProvider = NotifierProvider<_StatsRevision, int>(
  _StatsRevision.new,
);

class _StatsRevision extends Notifier<int> {
  @override
  int build() => 0;

  void changed() => state++;
}

/// Count floors bridge the gap between an accepted publication and indexing.
/// Keep them for a bounded interval: an older concurrent profile read must not
/// undo a count just because another read has already observed the new post.
final profilePostStatsOverlayProvider = Provider<ProfilePostStatsOverlay>((
  ref,
) {
  final lease = ref.watch(
    sessionRegistryProvider.select(
      (value) => value.value?.activeLease?.session,
    ),
  );
  final overlay = ProfilePostStatsOverlay(
    lease: lease,
    changed: () => ref.read(_statsRevisionProvider.notifier).changed(),
  );
  ref.onDispose(overlay.dispose);
  return overlay;
});

class ProfilePostStatsOverlay {
  new({required this.lease, required this.changed});

  final AccountSessionLease? lease;
  final void Function() changed;
  final _seen = <AtUri>{};
  Profile? _floor;
  Timer? _expiry;

  void accept(Post post, Profile baseline) {
    if (post.reply != null ||
        post.author.did != lease?.account.did ||
        baseline.did != post.author.did ||
        !_seen.add(post.uri)) {
      return;
    }
    final current = apply(baseline);
    _floor = current.copyWith(
      postCount: post.project == null
          ? _increment(current.postCount)
          : current.postCount,
      postsLast7Days: _increment(current.postsLast7Days),
      projectCount: post.project != null
          ? _increment(current.projectCount)
          : current.projectCount,
    );
    _expiry?.cancel();
    _expiry = Timer(const Duration(seconds: 30), () {
      _floor = null;
      changed();
    });
    changed();
  }

  bool get hasPending => _floor != null;

  Profile apply(Profile profile) {
    final floor = _floor;
    if (floor == null || floor.did != profile.did) return profile;
    return profile.copyWith(
      postCount: _maximum(profile.postCount, floor.postCount),
      postsLast7Days: _maximum(profile.postsLast7Days, floor.postsLast7Days),
      projectCount: _maximum(profile.projectCount, floor.projectCount),
    );
  }

  void dispose() => _expiry?.cancel();
}

int? _increment(int? value) => value == null ? null : value + 1;
int? _maximum(int? value, int? floor) =>
    value == null || floor == null ? value : math.max(value, floor);

void publishProfilePostStats(Ref ref, Post post) {
  if (post.reply != null) return;
  // Both unqualified and audience-keyed profile reads can be mounted.
  for (final provider in {
    userProfileProvider(post.author.did),
    userProfileProvider(
      post.author.did,
      audience: ref.read(browsingContextProvider),
    ),
  }) {
    if (!ref.exists(provider)) continue;
    final baseline = ref.read(provider).value;
    if (baseline == null) continue;
    ref.read(profilePostStatsOverlayProvider).accept(post, baseline);
    break;
  }
}

void listenForProfilePostStats(
  Ref ref,
  Did did,
  void Function({required bool pending}) update,
) {
  ref.listen(_statsRevisionProvider, (_, _) {
    final overlay = ref.read(profilePostStatsOverlayProvider);
    if (overlay.lease?.account.did != did) return;
    update(pending: overlay.hasPending);
  });
}
