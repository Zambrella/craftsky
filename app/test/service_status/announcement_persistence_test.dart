import 'package:craftsky_app/service_status/data/announcement_dismissal_store.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  // IT-002 / FR-006, FR-008 / AC-009, AC-011. Mock corroboration only.
  test('awaited dismissal is read after reload by a fresh store', () async {
    SharedPreferences.setMockInitialValues({});
    final first = PreferencesAnnouncementDismissalStore();
    await first.writeRevision('A');
    final preferences = await SharedPreferences.getInstance();
    await preferences.reload();
    expect(preferences.getString('service_status.dismissed_revision'), 'A');
    expect(await PreferencesAnnouncementDismissalStore().readRevision(), 'A');
    expect(preferences.getKeys(), {'service_status.dismissed_revision'});
  });
}
