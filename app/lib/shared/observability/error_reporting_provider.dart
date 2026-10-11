import 'package:craftsky_app/shared/observability/error_reporting_controller.dart';
import 'package:craftsky_app/shared/observability/sentry_config.dart';
import 'package:craftsky_app/shared/observability/sentry_error_reporter.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

final errorReportingControllerProvider =
    FutureProvider<ErrorReportingController>(
      (ref) async => ErrorReportingController(
        preferences: await SharedPreferences.getInstance(),
        config: SentryConfig.fromEnvironment(),
        adapter: const SentryFlutterBootstrapAdapter(),
        closeSdk: closeSentrySdk,
      ),
    );

final errorReportingPreferenceProvider =
    AsyncNotifierProvider<ErrorReportingPreference, bool>(
      ErrorReportingPreference.new,
    );

class ErrorReportingPreference extends AsyncNotifier<bool> {
  @override
  Future<bool> build() async =>
      (await ref.watch(errorReportingControllerProvider.future)).optedIn;

  Future<void> setEnabled({required bool enabled}) async {
    if (state.isLoading) return;
    state = const AsyncLoading<bool>();
    try {
      final controller = await ref.read(
        errorReportingControllerProvider.future,
      );
      await controller.setEnabled(enabled: enabled);
      if (ref.mounted) state = AsyncData(controller.optedIn);
    } on Object {
      final controller = await ref.read(
        errorReportingControllerProvider.future,
      );
      if (ref.mounted) state = AsyncData(controller.optedIn);
      rethrow;
    }
  }
}
