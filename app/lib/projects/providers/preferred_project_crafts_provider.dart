import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/projects/options/project_option.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Keeps preferred crafts first, retaining catalog order within each group.
final preferredProjectCraftsProvider = Provider<List<ProjectOption>>((ref) {
  final auth = ref.watch(authSessionProvider).value;
  final crafts = switch (auth) {
    SignedIn(:final did) =>
      ref.watch(userProfileProvider(Did.parse(did))).value?.crafts ??
          const <String>[],
    _ => const <String>[],
  };
  return ProjectOptionCatalogs.craftTypesForPreferences(crafts);
});
