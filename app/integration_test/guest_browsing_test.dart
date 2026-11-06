import 'package:integration_test/integration_test.dart';

import '../test/shared/browsing/guest_router_test.dart' as guest_journeys;

/// Runs the same guest router/widget contract on a native runtime. Repositories
/// are isolated fixtures; this does not exercise live OAuth or a real PDS.
void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  guest_journeys.main();
}
