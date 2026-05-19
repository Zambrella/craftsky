import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class DesignPlaygroundPage extends ConsumerWidget {
  const DesignPlaygroundPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final version = ref.watch(packageInfoProvider).version;
    final sp = theme.extension<SpacingTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;

    return Scaffold(
      backgroundColor: swatches.paper,
      appBar: AppBar(
        title: Text(l10n.appTitle, style: theme.textTheme.titleLarge),
        backgroundColor: swatches.paper,
        elevation: 0,
        scrolledUnderElevation: 0,
        shape: Border(
          bottom: BorderSide(color: theme.colorScheme.onSurface, width: 1.5),
        ),
      ),
      body: ListView(
        padding: EdgeInsets.fromLTRB(sp.sp5, sp.sp5, sp.sp5, sp.sp8),
        children: [
          _HomeHeader(
            subtitle: l10n.homeSubtitle,
            versionLabel: l10n.homeVersionLabel(version),
          ),
          SizedBox(height: sp.sp6),
          const _PlaygroundSection(
            eyebrow: 'Typography',
            child: _TypographySample(),
          ),
          SizedBox(height: sp.sp7),
          const _PlaygroundSection(
            eyebrow: 'Buttons',
            child: _ButtonsSample(),
          ),
          SizedBox(height: sp.sp7),
          const _PlaygroundSection(
            eyebrow: 'Progress',
            child: _ProgressSample(),
          ),
          SizedBox(height: sp.sp7),
          const _PlaygroundSection(
            eyebrow: 'Chips',
            child: _ChipsSample(),
          ),
          SizedBox(height: sp.sp7),
          const _PlaygroundSection(
            eyebrow: 'Text fields',
            child: _TextFieldsSample(),
          ),
          SizedBox(height: sp.sp7),
          const _PlaygroundSection(
            eyebrow: 'Cards',
            child: _CardsSample(),
          ),
          SizedBox(height: sp.sp7),
          const _PlaygroundSection(
            eyebrow: 'Dialogs',
            child: _DialogsSample(),
          ),
          SizedBox(height: sp.sp7),
          const _PlaygroundSection(
            eyebrow: 'Swatches',
            child: _SwatchesSample(),
          ),
        ],
      ),
    );
  }
}

class _HomeHeader extends StatelessWidget {
  const _HomeHeader({
    required this.subtitle,
    required this.versionLabel,
  });

  final String subtitle;
  final String versionLabel;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final sp = theme.extension<SpacingTheme>()!;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          'Paper cutout',
          style: theme.textTheme.labelSmall?.copyWith(
            color: theme.colorScheme.primary,
          ),
        ),
        SizedBox(height: sp.sp2),
        Text(
          'Design playground',
          style: theme.textTheme.displaySmall,
        ),
        SizedBox(height: sp.sp3),
        Text(
          'Poking the atoms before assembling the product.',
          style: theme.textTheme.bodyLarge,
        ),
        SizedBox(height: sp.sp2),
        Text(subtitle, style: theme.textTheme.bodySmall),
        SizedBox(height: sp.sp1),
        Text(versionLabel, style: theme.textTheme.bodySmall),
      ],
    );
  }
}

class _PlaygroundSection extends StatelessWidget {
  const _PlaygroundSection({
    required this.eyebrow,
    required this.child,
  });

  final String eyebrow;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final sp = theme.extension<SpacingTheme>()!;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(eyebrow, style: theme.textTheme.labelSmall),
        SizedBox(height: sp.sp2),
        Container(
          height: 2.5,
          color: theme.colorScheme.onSurface,
        ),
        SizedBox(height: sp.sp4),
        child,
      ],
    );
  }
}

class _TypographySample extends StatelessWidget {
  const _TypographySample();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final sp = theme.extension<SpacingTheme>()!;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('Editorial', style: theme.textTheme.displayMedium),
        SizedBox(height: sp.sp2),
        Text('Heavy headline', style: theme.textTheme.headlineMedium),
        SizedBox(height: sp.sp3),
        Text(
          'Body copy sits at 16px with a roomy 1.5 line-height. '
          "It's quietly confident — the display type is doing the shouting.",
          style: theme.textTheme.bodyLarge,
        ),
        SizedBox(height: sp.sp3),
        Text('Metadata · subdued', style: theme.textTheme.bodySmall),
      ],
    );
  }
}

class _ButtonsSample extends StatelessWidget {
  const _ButtonsSample();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final sp = theme.extension<SpacingTheme>()!;

    return Wrap(
      spacing: sp.sp3,
      runSpacing: sp.sp3,
      children: [
        ChunkyButton(
          onPressed: () {},
          child: const Text('Share'),
        ),
        OutlinedButton(
          onPressed: () {},
          child: const Text('Follow'),
        ),
        ChunkyButton(
          onPressed: () {},
          backgroundColor: theme.colorScheme.secondary,
          foregroundColor: theme.colorScheme.onSecondary,
          child: const Text('Report'),
        ),
        TextButton(
          onPressed: () {},
          child: const Text('Cancel'),
        ),
      ],
    );
  }
}

class _ChipsSample extends StatelessWidget {
  const _ChipsSample();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final sp = theme.extension<SpacingTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final ink = theme.colorScheme.onSurface;

    Widget chip(
      String label, {
      required Color background,
      Color? foreground,
    }) {
      return Container(
        padding: EdgeInsets.symmetric(horizontal: sp.sp3, vertical: sp.sp1),
        decoration: BoxDecoration(
          color: background,
          borderRadius: BorderRadius.circular(999),
          border: Border.all(color: ink, width: 1.5),
        ),
        child: Text(
          label,
          style: theme.textTheme.labelMedium?.copyWith(
            color: foreground ?? ink,
          ),
        ),
      );
    }

    return Wrap(
      spacing: sp.sp2,
      runSpacing: sp.sp2,
      children: [
        chip('Work in progress', background: swatches.wip),
        chip(
          'Finished',
          background: swatches.done,
          foreground: Colors.white,
        ),
        chip(
          'Business account',
          background: theme.colorScheme.primary,
          foreground: theme.colorScheme.onPrimary,
        ),
        chip(
          'Sponsored',
          background: theme.colorScheme.secondary,
          foreground: theme.colorScheme.onSecondary,
        ),
        chip('Sewing · Linen', background: swatches.paper3),
        chip('Quilting', background: swatches.sky),
      ],
    );
  }
}

class _TextFieldsSample extends StatelessWidget {
  const _TextFieldsSample();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final sp = theme.extension<SpacingTheme>()!;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const BrandTextField(
          label: 'Pattern name',
          hintText: 'e.g. Wiksten Haori',
          prefixIcon: Icon(Icons.search),
        ),
        SizedBox(height: sp.sp5),
        const BrandTextField(
          label: 'Fabric or yarn',
          hintText: 'e.g. Merchant & Mills 185 linen, indigo',
          helperText: 'What did you use? Brand and colour help other makers.',
        ),
        SizedBox(height: sp.sp5),
        const BrandTextField(
          label: 'Modifications',
          hintText: 'What did you change?',
          maxLines: 3,
          minLines: 3,
        ),
        SizedBox(height: sp.sp5),
        const BrandTextField(
          label: 'Cover image',
          hintText: 'Paste an image URL',
          errorText: 'Image needs to be under 20 MB.',
        ),
      ],
    );
  }
}

class _CardsSample extends StatelessWidget {
  const _CardsSample();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final sp = theme.extension<SpacingTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    final shadows = theme.extension<BrandShadowTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final ink = theme.colorScheme.onSurface;

    return Column(
      children: [
        _HardShadowCard(
          title: 'Wiksten Haori',
          meta: 'Sewing · WIP · 2 days',
          body:
              'Linen mid-weight, indigo. Lengthened the sleeves, '
              'swapped in a shell button closure.',
          swatchColor: swatches.clay,
        ),
        SizedBox(height: sp.sp5),
        Container(
          padding: EdgeInsets.all(sp.sp4),
          decoration: BoxDecoration(
            color: swatches.paper3,
            borderRadius: BorderRadius.circular(radii.r3),
            border: Border.all(color: swatches.borderHair),
            boxShadow: shadows.paper1,
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('Soft-paper surface', style: theme.textTheme.titleMedium),
              SizedBox(height: sp.sp2),
              Text(
                'For floating sheets — modals, popovers — papery not glassy. '
                'Hairline border replaces the heavy ink rule.',
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: ink,
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }
}

class _HardShadowCard extends StatelessWidget {
  const _HardShadowCard({
    required this.title,
    required this.meta,
    required this.body,
    required this.swatchColor,
  });

  final String title;
  final String meta;
  final String body;
  final Color swatchColor;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final sp = theme.extension<SpacingTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    final shadows = theme.extension<BrandShadowTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final ink = theme.colorScheme.onSurface;

    return Container(
      decoration: BoxDecoration(
        color: swatches.paper3,
        borderRadius: BorderRadius.circular(radii.r3),
        border: Border.all(color: ink, width: 1.5),
        boxShadow: shadows.drop,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Coloured-paper swatch with the signature 4–12px of paper visible
          // around the "image".
          Padding(
            padding: EdgeInsets.fromLTRB(sp.sp2, sp.sp2, sp.sp2, 0),
            child: AspectRatio(
              aspectRatio: 16 / 9,
              child: Container(
                decoration: BoxDecoration(
                  color: swatchColor,
                  borderRadius: BorderRadius.circular(radii.r1),
                ),
                alignment: Alignment.center,
                child: Text(
                  'Image',
                  style: theme.textTheme.labelSmall?.copyWith(
                    color: Colors.white,
                  ),
                ),
              ),
            ),
          ),
          Padding(
            padding: EdgeInsets.all(sp.sp4),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(title, style: theme.textTheme.headlineSmall),
                SizedBox(height: sp.sp1),
                Text(
                  meta,
                  style: theme.textTheme.bodySmall,
                ),
                SizedBox(height: sp.sp3),
                Text(body, style: theme.textTheme.bodyMedium),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _SwatchesSample extends StatelessWidget {
  const _SwatchesSample();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final sp = theme.extension<SpacingTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;

    final entries = <(String, Color)>[
      ('Paper', swatches.paper),
      ('Paper 2', swatches.paper2),
      ('Butter', swatches.butter),
      ('Clay', swatches.clay),
      ('Moss', swatches.moss),
      ('Sky', swatches.sky),
      ('Lilac', swatches.lilac),
      ('Cobalt', theme.colorScheme.primary),
      ('Red', theme.colorScheme.secondary),
    ];

    return Wrap(
      spacing: sp.sp3,
      runSpacing: sp.sp3,
      children: [
        for (final (label, color) in entries)
          _SwatchTile(label: label, color: color),
      ],
    );
  }
}

class _SwatchTile extends StatelessWidget {
  const _SwatchTile({required this.label, required this.color});

  final String label;
  final Color color;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final sp = theme.extension<SpacingTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    final ink = theme.colorScheme.onSurface;

    return SizedBox(
      width: 96,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            height: 64,
            decoration: BoxDecoration(
              color: color,
              borderRadius: BorderRadius.circular(radii.r2),
              border: Border.all(color: ink, width: 1.5),
            ),
          ),
          SizedBox(height: sp.sp1),
          Text(label, style: theme.textTheme.labelMedium),
        ],
      ),
    );
  }
}

class _DialogsSample extends StatelessWidget {
  const _DialogsSample();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;

    return Wrap(
      spacing: spacing.sp3,
      runSpacing: spacing.sp3,
      children: [
        ChunkyButton(
          onPressed: () async {
            final result = await showCraftskyConfirmDialog(
              context,
              title: 'Discard draft?',
              message: 'Your changes will be lost.',
              confirmLabel: 'Discard',
              cancelLabel: 'Keep editing',
            );
            if (!context.mounted) return;
            ScaffoldMessenger.of(context).showSnackBar(
              SnackBar(content: Text('Confirm result: $result')),
            );
          },
          child: const Text('Show neutral confirm'),
        ),
        ChunkyButton(
          backgroundColor: theme.colorScheme.error,
          onPressed: () async {
            final result = await showCraftskyDestructiveConfirmDialog(
              context,
              title: 'Delete this post?',
              message: 'This cannot be undone.',
              confirmLabel: 'Delete',
              cancelLabel: 'Cancel',
            );
            if (!context.mounted) return;
            ScaffoldMessenger.of(context).showSnackBar(
              SnackBar(content: Text('Destructive result: $result')),
            );
          },
          child: const Text('Show destructive confirm'),
        ),
        ChunkyButton(
          onPressed: () async {
            await showCraftskyAlertDialog(
              context,
              title: 'Saved',
              message: 'Your profile is live.',
              dismissLabel: 'Got it',
            );
            if (!context.mounted) return;
            ScaffoldMessenger.of(context).showSnackBar(
              const SnackBar(content: Text('Alert dismissed')),
            );
          },
          child: const Text('Show alert'),
        ),
        ChunkyButton(
          onPressed: () async {
            final result = await showCraftskyConfirmDialog(
              context,
              title: 'Sync draft?',
              message: 'Pretends to do work for 1.5s, throws ~50% of the time.',
              confirmLabel: 'Sync',
              cancelLabel: 'Cancel',
              onConfirm: () async {
                await Future<void>.delayed(const Duration(milliseconds: 1500));
                if (DateTime.now().millisecondsSinceEpoch.isEven) {
                  throw StateError('Pretend network error');
                }
              },
            );
            if (!context.mounted) return;
            ScaffoldMessenger.of(context).showSnackBar(
              SnackBar(content: Text('Async result: $result')),
            );
          },
          child: const Text('Show async confirm'),
        ),
      ],
    );
  }
}

class _ProgressSample extends StatelessWidget {
  const _ProgressSample();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final sp = theme.extension<SpacingTheme>()!;
    final text = theme.textTheme.labelSmall;

    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceAround,
      crossAxisAlignment: CrossAxisAlignment.end,
      children: [
        _ProgressSize(label: '18 px', size: 18, sp: sp, labelStyle: text),
        _ProgressSize(label: '36 px', size: 36, sp: sp, labelStyle: text),
        _ProgressSize(label: '64 px', size: 64, sp: sp, labelStyle: text),
      ],
    );
  }
}

class _ProgressSize extends StatelessWidget {
  const _ProgressSize({
    required this.label,
    required this.size,
    required this.sp,
    required this.labelStyle,
  });

  final String label;
  final double size;
  final SpacingTheme sp;
  final TextStyle? labelStyle;

  @override
  Widget build(BuildContext context) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        StitchProgressIndicator(size: size),
        SizedBox(height: sp.sp2),
        Text(label, style: labelStyle),
      ],
    );
  }
}
