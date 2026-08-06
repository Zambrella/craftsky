import 'package:dart_mappable/dart_mappable.dart';

part 'profile_pin_state.mapper.dart';

@MappableEnum()
enum ProfilePinSlot { standard, project }

ProfilePinSlot? classifyProfilePinSlot({
  required bool isReply,
  required bool isProject,
  required bool hasQuote,
}) {
  if (isReply || (isProject && hasQuote)) return null;
  return isProject ? ProfilePinSlot.project : ProfilePinSlot.standard;
}

@MappableClass()
class ProfilePinState with ProfilePinStateMappable {
  const ProfilePinState({this.standardPostUri, this.projectPostUri});

  final String? standardPostUri;
  final String? projectPostUri;

  String? postUriFor(ProfilePinSlot slot) => switch (slot) {
    ProfilePinSlot.standard => standardPostUri,
    ProfilePinSlot.project => projectPostUri,
  };
}
