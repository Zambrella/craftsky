import 'package:flutter/material.dart';
import 'package:flutter_svg/flutter_svg.dart';

/// A branded craft glyph resolved from either a profile craft ID or lexicon
/// token.
class CraftIcon extends StatelessWidget {
  const CraftIcon({
    required this.craft,
    this.size = 20,
    this.color,
    super.key,
  });

  static const _assetPrefix = 'assets/design/icons';

  final String craft;
  final double size;
  final Color? color;

  static String? assetPathFor(String craft) {
    final separator = craft.lastIndexOf('#');
    final id = (separator == -1 ? craft : craft.substring(separator + 1))
        .trim()
        .toLowerCase();
    return switch (id) {
      'knitting' => '$_assetPrefix/knitting.svg',
      'crochet' => '$_assetPrefix/crochet.svg',
      'sewing' => '$_assetPrefix/sewing.svg',
      'embroidery' => '$_assetPrefix/embroidery.svg',
      'quilting' => '$_assetPrefix/quilting.svg',
      _ => null,
    };
  }

  @override
  Widget build(BuildContext context) {
    final assetPath = assetPathFor(craft);
    if (assetPath == null) return const SizedBox.shrink();
    final foreground = color ?? IconTheme.of(context).color;
    return SvgPicture.asset(
      assetPath,
      key: ValueKey('craft-icon-$assetPath'),
      width: size,
      height: size,
      colorFilter: foreground == null
          ? null
          : ColorFilter.mode(foreground, BlendMode.srcIn),
      excludeFromSemantics: true,
    );
  }
}

/// A craft label with its branded glyph when that craft has one.
class CraftIconLabel extends StatelessWidget {
  const CraftIconLabel({
    required this.craft,
    required this.label,
    this.iconSize = 18,
    this.gap = 6,
    this.style,
    this.color,
    this.flexibleLabel = false,
    super.key,
  });

  final String craft;
  final String label;
  final double iconSize;
  final double gap;
  final TextStyle? style;
  final Color? color;
  final bool flexibleLabel;

  @override
  Widget build(BuildContext context) {
    final hasIcon = CraftIcon.assetPathFor(craft) != null;
    final foreground = color ?? style?.color;
    final text = Text(label, style: style?.copyWith(color: foreground));
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        if (hasIcon) ...[
          CraftIcon(craft: craft, size: iconSize, color: foreground),
          SizedBox(width: gap),
        ],
        if (flexibleLabel) Flexible(child: text) else text,
      ],
    );
  }
}
