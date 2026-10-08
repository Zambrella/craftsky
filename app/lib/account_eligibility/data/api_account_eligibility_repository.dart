import 'package:craftsky_app/account_eligibility/data/account_eligibility_repository.dart';
import 'package:craftsky_app/account_eligibility/models/account_eligibility_status.dart';
import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:dio/dio.dart';

final class ApiAccountEligibilityRepository
    implements AccountEligibilityRepository {
  const ApiAccountEligibilityRepository(this._dio);

  final Dio _dio;

  @override
  Future<AccountEligibilityStatus> readStatus() => unwrapApi(() async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/v1/account/eligibility',
    );
    return AccountEligibilityStatus.fromJson(response.data!);
  });
}
