import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'onboarding_status_provider.g.dart';

/// Per-DID onboarding completion flag. Backed by `SharedPreferences`;
/// survives relaunch but not reinstall on Android (clear-app-data
/// semantics). First-run for a new DID defaults to `false`.
///
/// `@riverpod` codegen exposes the family arg as an instance field
/// (`did`) on the generated notifier base class, so both `build` and
/// `finish` reference `did` directly.
@riverpod
class OnboardingStatus extends _$OnboardingStatus {
  @override
  bool build(Did did) {
    final prefs = ref.watch(sharedPreferencesProvider);
    return prefs.getBool(_keyFor(did)) ?? false;
  }

  Future<void> finish() async {
    final prefs = ref.read(sharedPreferencesProvider);
    await prefs.setBool(_keyFor(did), true);
    if (!ref.mounted) return;
    state = true;
  }

  static String _keyFor(Did did) => 'onboarded_$did';
}
