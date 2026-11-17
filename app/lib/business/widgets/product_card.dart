import 'dart:async';

import 'package:craftsky_app/business/models/business_formatters.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/post_image_carousel.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/link/external_link.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:material_ui/material_ui.dart';

class ProductCard extends StatelessWidget {
  const new({
    required this.product,
    this.launchExternal = launchExternalLink,
    this.confirmExternal = showOpenLinkDialog,
    super.key,
  });

  final BusinessProductView product;
  final ExternalLinkLauncher launchExternal;
  final ExternalLinkConfirmer confirmExternal;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final l10n = AppLocalizations.of(context);
    final destination = hydratedExternalActionUri(product.uri);
    final price = BusinessFormatters.money(product.price, l10n.localeName);

    final images = [
      for (final image in product.images)
        PostImage(
          cid: image.cid.toString(),
          mime: image.mime,
          size: image.size,
          alt: image.alt,
          thumb: image.thumb.isEmpty ? null : image.thumb,
          fullsize: image.fullsize.isEmpty ? null : image.fullsize,
          aspectRatio: image.aspectRatio == null
              ? null
              : PostImageAspectRatio(
                  width: image.aspectRatio!.width,
                  height: image.aspectRatio!.height,
                ),
        ),
    ];
    return CraftskyCard(
      child: Semantics(
        button: destination != null,
        label: destination == null
            ? product.title
            : l10n.businessProductOpen(product.title),
        child: InkWell(
          key: const Key('product-open-link'),
          onTap: destination == null
              ? null
              : () => unawaited(
                  confirmAndLaunchExternalAction(
                    context,
                    uri: destination,
                    launchUrl: launchExternal,
                    confirmOpenLink: confirmExternal,
                  ),
                ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              if (images.isNotEmpty)
                PostImageCarousel(
                  images: images,
                  previewBytesFor: (index) =>
                      product.images[index].previewBytes,
                  showOutline: false,
                ),
              Padding(
                padding: EdgeInsets.all(spacing.sp4),
                child: Row(
                  children: [
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            product.title,
                            style: theme.textTheme.titleMedium,
                          ),
                          if (price != null) ...[
                            SizedBox(height: spacing.sp1),
                            Text(price, style: theme.textTheme.bodyMedium),
                          ],
                        ],
                      ),
                    ),
                    if (destination != null) ...[
                      SizedBox(width: spacing.sp3),
                      const Icon(CraftskyIconsBold.externalLink),
                    ],
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
