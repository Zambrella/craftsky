import 'dart:convert';
import 'dart:math' as math;

import 'package:craftsky_app/profile/data/profile_link_policy.dart';
import 'package:craftsky_app/profile/models/profile_link.dart';
import 'package:craftsky_app/shared/link/external_link.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:material_ui/material_ui.dart';

class ProfileLinks extends StatefulWidget {
  const new({
    required this.links,
    this.confirmExternal = showOpenLinkDialog,
    this.launchExternal = launchExternalLink,
    super.key,
  });
  final List<ProfileLink> links;
  final ExternalLinkConfirmer confirmExternal;
  final ExternalLinkLauncher launchExternal;
  @override
  State<ProfileLinks> createState() => _ProfileLinksState();
}

class _ProfileLinksState extends State<ProfileLinks> {
  bool _expanded = false;

  @override
  void didUpdateWidget(covariant ProfileLinks oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (!listEquals(oldWidget.links, widget.links)) _expanded = false;
  }

  Future<void> _open(ProfileLink link) async {
    final uri = normalizeProfileLink(link.url);
    if (uri == null) return;
    await confirmAndLaunchExternalLink(
      context,
      uri: uri,
      launchUrl: widget.launchExternal,
      confirmOpenLink: widget.confirmExternal,
    );
  }

  @override
  Widget build(BuildContext context) {
    final safe = widget.links
        .where((link) => normalizeProfileLink(link.url) != null)
        .take(maxProfileLinks)
        .toList();
    return LayoutBuilder(
      builder: (context, constraints) {
        final widths = [
          for (final link in safe)
            profileLinkPillWidth(context, link, constraints.maxWidth),
        ];
        final count = _expanded
            ? safe.length
            : profileLinkVisibleCount(widths, constraints.maxWidth);
        return Wrap(
          spacing: 8,
          runSpacing: 8,
          children: [
            for (var i = 0; i < count; i++)
              SizedBox(
                width: widths[i],
                child: ProfileLinkPill(
                  link: safe[i],
                  onTap: () => _open(safe[i]),
                ),
              ),
            if (count < safe.length)
              ActionChip(
                key: const Key('profile-links-more'),
                label: Text('+${safe.length - count}'),
                onPressed: () => setState(() => _expanded = true),
              ),
          ],
        );
      },
    );
  }
}

// Replay wrapping with room for the overflow pill on the second line.
int profileLinkVisibleCount(List<double> widths, double available) {
  var row = 1;
  var used = 0.0;
  var secondRowStart = 0;
  for (var i = 0; i < widths.length; i++) {
    final gap = used == 0 ? 0.0 : 8.0;
    if (used + gap + widths[i] > available) {
      row++;
      used = 0;
      if (row == 2) secondRowStart = i;
    }
    if (row > 2) {
      var count = i;
      var lastRowWidth = 0.0;
      for (var j = secondRowStart; j < count; j++) {
        lastRowWidth += (j == secondRowStart ? 0 : 8) + widths[j];
      }
      while (count > secondRowStart && lastRowWidth + 64 > available) {
        count--;
        lastRowWidth -= widths[count] + (count == secondRowStart ? 0 : 8);
      }
      return count;
    }
    used += (used == 0 ? 0 : 8) + widths[i];
  }
  return widths.length;
}

double profileLinkPillWidth(
  BuildContext context,
  ProfileLink link,
  double available,
) {
  final text = TextPainter(
    text: TextSpan(
      text: link.label,
      style: Theme.of(context).textTheme.labelLarge,
    ),
    textDirection: Directionality.of(context),
    textScaler: MediaQuery.textScalerOf(context),
    maxLines: 1,
  )..layout();
  final double width = math.min(available, math.min(text.width, 180) + 68);
  text.dispose();
  return width;
}

class ProfileLinkPill extends StatelessWidget {
  const new({required this.link, this.onTap, super.key});
  final ProfileLink link;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final provider = profileLinkSupportProvider(link.url);
    final colors = Theme.of(context).colorScheme;
    return Tooltip(
      message: link.url,
      // Show the destination on hover without adding a long-press action.
      triggerMode: TooltipTriggerMode.manual,
      child: Semantics(
        link: true,
        label: '${link.label}, ${link.url}',
        child: Material(
          color: provider != null
              ? const Color(0xffd7f5df)
              : colors.surfaceContainerHighest,
          borderRadius: BorderRadius.circular(24),
          child: InkWell(
            onTap: onTap,
            borderRadius: BorderRadius.circular(24),
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  if (provider != null)
                    SvgPicture.asset(
                      'assets/profile-link-providers/${profileLinkSupportLogos[provider]}.svg',
                      width: 18,
                      height: 18,
                      colorFilter: const ColorFilter.mode(
                        Color(0xff176b32),
                        BlendMode.srcIn,
                      ),
                    )
                  else if (link.icon case final String icon
                      when icon.isNotEmpty)
                    ProfileLinkIcon(url: icon)
                  else
                    const Icon(Icons.language, size: 18),
                  const SizedBox(width: 6),
                  Flexible(
                    child: Text(
                      link.label,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: Theme.of(context).textTheme.labelLarge?.copyWith(
                        color: provider != null
                            ? const Color(0xff176b32)
                            : colors.onSurface,
                      ),
                    ),
                  ),
                  const SizedBox(width: 6),
                  Icon(
                    Icons.north_east,
                    size: 14,
                    color: provider != null
                        ? const Color(0xff176b32)
                        : colors.onSurfaceVariant,
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class ProfileLinkIcon extends StatelessWidget {
  const new({required this.url, super.key});
  final String url;
  @override
  Widget build(BuildContext context) {
    if (url.startsWith('data:image/')) {
      try {
        return Image.memory(
          base64Decode(url.split(',').last),
          width: 18,
          height: 18,
          errorBuilder: (_, _, _) => const Icon(Icons.language, size: 18),
        );
      } on FormatException {
        return const Icon(Icons.language, size: 18);
      }
    }
    final uri = Uri.tryParse(url);
    if (uri == null || uri.scheme != 'https') {
      return const Icon(Icons.language, size: 18);
    }
    return Image.network(
      url,
      width: 18,
      height: 18,
      errorBuilder: (_, _, _) => const Icon(Icons.language, size: 18),
    );
  }
}
