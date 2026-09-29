import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';

final class InMemorySessionRegistryStorage implements SessionRegistryStorage {
  InMemorySessionRegistryStorage([SessionRegistry? initialRegistry])
    : registry = initialRegistry ?? SessionRegistry.empty();

  SessionRegistry registry;

  @override
  Future<SessionRegistry> read() async => registry;

  @override
  Future<void> write(SessionRegistry registry) async {
    this.registry = registry;
  }
}
