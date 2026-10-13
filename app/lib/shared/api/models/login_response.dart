import 'package:dart_mappable/dart_mappable.dart';

part 'login_response.mapper.dart';

@MappableClass()
class LoginResponse with LoginResponseMappable {
  const new({required this.authUrl});

  final String authUrl;
}
