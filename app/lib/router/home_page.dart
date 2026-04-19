import 'package:craftsky_app/app_dependencies.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class HomePage extends ConsumerWidget {
  const HomePage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final version = ref.watch(packageInfoProvider).version;

    return Scaffold(
      appBar: AppBar(title: const Text('Craftsky')),
      body: Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Icon(Icons.palette_outlined, size: 64, color: theme.colorScheme.primary),
              const SizedBox(height: 16),
              Text('Craftsky', style: theme.textTheme.headlineMedium),
              const SizedBox(height: 8),
              Text(
                'Scaffold ready',
                style: theme.textTheme.titleMedium?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
              const SizedBox(height: 8),
              Text('v$version', style: theme.textTheme.bodySmall),
            ],
          ),
        ),
      ),
    );
  }
}
