import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:shared_preferences/shared_preferences.dart';

abstract interface class AnnouncementDismissalStore {
  Future<String?> readRevision();
  Future<void> writeRevision(String revision);
}

final class PreferencesAnnouncementDismissalStore
    implements AnnouncementDismissalStore {
  static const key = 'service_status.dismissed_revision';
  @override
  Future<String?> readRevision() async {
    final preferences = await SharedPreferences.getInstance();
    await preferences.reload();
    return preferences.getString(key);
  }

  @override
  Future<void> writeRevision(String revision) async {
    final preferences = await SharedPreferences.getInstance();
    if (!await preferences.setString(key, revision)) {
      throw DiagnosticStateError('Status dismissal could not be saved');
    }
  }
}
