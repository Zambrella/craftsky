import 'package:craftsky_app/business/data/api_business_repository.dart';
import 'package:craftsky_app/business/data/business_api_client.dart';
import 'package:craftsky_app/business/data/business_repository.dart';
import 'package:craftsky_app/business/services/business_time_zone_service.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'business_repository_provider.g.dart';

@Riverpod(keepAlive: true)
BusinessRepository businessRepository(Ref ref) => ApiBusinessRepository(
  BusinessApiClient(ref.watch(dioProvider)),
  ref.watch(businessTimeZoneServiceProvider),
);
