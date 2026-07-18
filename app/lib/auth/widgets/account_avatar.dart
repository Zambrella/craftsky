import 'package:flutter/material.dart';

class AccountAvatar extends StatelessWidget {
  const AccountAvatar({
    required this.avatarUrl,
    this.size = 32,
    this.selected = false,
    super.key,
  });

  final String? avatarUrl;
  final double size;
  final bool selected;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    final borderColor = selected ? colors.primary : Colors.transparent;
    return Semantics(
      image: true,
      selected: selected,
      child: Container(
        width: size,
        height: size,
        padding: const EdgeInsets.all(2),
        decoration: BoxDecoration(
          shape: BoxShape.circle,
          color: selected ? colors.primaryContainer : Colors.transparent,
          border: Border.all(color: borderColor, width: 2),
        ),
        child: ClipOval(child: _image(context)),
      ),
    );
  }

  Widget _image(BuildContext context) {
    final url = avatarUrl;
    if (url == null || url.isEmpty) return _fallback(context);
    return Image.network(
      url,
      fit: BoxFit.cover,
      errorBuilder: (context, error, stackTrace) => _fallback(context),
    );
  }

  Widget _fallback(BuildContext context) => ColoredBox(
    color: Theme.of(context).colorScheme.surfaceContainerHighest,
    child: const Center(child: Icon(Icons.person, size: 18)),
  );
}
