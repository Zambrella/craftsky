import 'package:craftsky_app/service_status/data/service_status_repository.dart';
import 'package:craftsky_app/service_status/models/service_status_document.dart';

/// Existing app tests isolate the optional status network dependency.
final class NormalServiceStatusRepository extends ServiceStatusRepository {
  const NormalServiceStatusRepository();
  @override
  StatusRequest start() => StatusRequest(
    Future.value(
      const ServiceStatusDocument(
        mode: ServiceStatusMode.normal,
        revision: 'test',
      ),
    ),
    () {},
  );
}
