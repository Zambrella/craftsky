import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class UserProfilePage extends ConsumerWidget {
  const UserProfilePage({required this.handle, super.key});

  final String handle;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // TODO(craftsky): l10n
    return Scaffold(
      appBar: AppBar(title: Text('@$handle')),
      body: Center(child: Text('Profile for @$handle')),
    );
  }
}
