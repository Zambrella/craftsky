import 'package:craftsky_app/service_status/data/service_status_repository.dart';

ServiceStatusRepository createServiceStatusRepository(Uri uri) =>
    throw UnsupportedError('Status retrieval unavailable');
bool isExpectedStatusPlatformFailure(Object error) => false;
