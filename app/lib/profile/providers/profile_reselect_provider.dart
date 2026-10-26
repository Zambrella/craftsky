import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Signals an explicit tap on Profile while the profile branch is open.
final profileReselectProvider = NotifierProvider<ProfileReselect, int>(
  ProfileReselect.new,
);

class ProfileReselect extends Notifier<int> {
  @override
  int build() => 0;

  void reselect() => state++;
}
