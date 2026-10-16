import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/projects/options/project_option.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_riverpod/misc.dart' show ProviderFamily;

/// Preferred-only shortcuts; keep the profile's loading and error states.
final ProviderFamily<AsyncValue<List<ProjectOption>>, ActiveAccountLease>
preferredCraftLinksProvider = Provider.autoDispose
    .family<AsyncValue<List<ProjectOption>>, ActiveAccountLease>((ref, lease) {
      final profileState = ref.watch(
        userProfileProvider(lease.session.account.did),
      );
      // Riverpod can retain a failure while automatically retrying. whenData's
      // loading mapping discards that cause, so preserve it explicitly here.
      if (profileState.hasError) {
        return AsyncError(profileState.error!, profileState.stackTrace!);
      }
      if (profileState.isLoading) return const AsyncLoading();
      return profileState.whenData((profile) {
        final selected = profile.crafts.toSet();
        return ProjectOptionCatalogs.craftTypes
            .where((option) => selected.contains(option.value.split('#').last))
            .toList(growable: false);
      });
    });
