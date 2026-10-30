import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/projects/options/project_option.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Profile order is preference order; unknown crafts are skipped by consumers.
final _profileCraftsProvider = Provider<List<String>>((ref) {
  final auth = ref.watch(authSessionProvider).value;
  return switch (auth) {
    SignedIn(:final did) =>
      ref.watch(userProfileProvider(Did.parse(did))).value?.crafts ??
          const <String>[],
    _ => const <String>[],
  };
});

/// Selected crafts in profile order, followed by the remaining catalog entries.
final preferredProjectCraftsProvider = Provider<List<ProjectOption>>((ref) {
  return ProjectOptionCatalogs.craftTypesForPreferences(
    ref.watch(_profileCraftsProvider),
  );
});

/// Empty/unsupported preferences leave new composers unselected.
final defaultProjectCraftProvider = Provider<String?>((ref) {
  final crafts = ref.watch(_profileCraftsProvider).toSet();
  for (final option in ProjectOptionCatalogs.craftTypesForPreferences(crafts)) {
    if (crafts.contains(option.value.split('#').last)) return option.value;
  }
  return null;
});
