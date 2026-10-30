import 'package:pub_semver/pub_semver.dart';

/// Strict status versions with build-independent precedence. Numeric components
/// are limited to 15 digits for exact, consistent native/web comparisons.
final class StatusAppVersion implements Comparable<StatusAppVersion> {
  const new _(this._version);

  factory parse(String value) {
    final match = syntax.firstMatch(value);
    if (match == null || match.end != value.length) {
      throw const FormatException('Status version');
    }
    // pub_semver orders builds; status policy deliberately ignores them.
    return StatusAppVersion._(Version.parse(value.split('+').first));
  }

  static final syntax = RegExp(
    r'^(0|[1-9][0-9]{0,14})\.(0|[1-9][0-9]{0,14})\.(0|[1-9][0-9]{0,14})(?:-((?:0|[1-9][0-9]{0,14}|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9][0-9]{0,14}|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*))?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$',
  );
  final Version _version;

  @override
  int compareTo(StatusAppVersion other) => _version.compareTo(other._version);
}
