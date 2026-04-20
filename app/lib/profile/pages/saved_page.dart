import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class SavedPage extends ConsumerWidget {
  const SavedPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // TODO(craftsky): l10n — page titles will move to AppLocalizations
    // when real UI lands.
    return Scaffold(
      appBar: AppBar(title: const Text('Saved')),
      body: const Center(child: Text('Saved')),
    );
  }
}
