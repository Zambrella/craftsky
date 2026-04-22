import 'package:dart_mappable/dart_mappable.dart';

part 'login_response.mapper.dart';

@MappableClass(caseStyle: CaseStyle.snakeCase)
class LoginResponse with LoginResponseMappable {
  const LoginResponse({required this.authUrl});

  final String authUrl;
}
