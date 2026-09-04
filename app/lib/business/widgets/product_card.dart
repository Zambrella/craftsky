import 'dart:async';

import 'package:craftsky_app/business/models/business_formatters.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/widgets/business_image.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/link/external_link.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

class ProductCard extends StatelessWidget {
  const ProductCard({
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

    return Semantics(
      button: destination != null,
      label: destination == null
          ? product.title
          : l10n.businessProductOpen(product.title),
      child: CraftskyCard(
        child: InkWell(
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
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              if (product.image case final image?)
                Semantics(
                  image: true,
                  label: image.alt,
                  child: SizedBox.square(
                    dimension: 112,
                    child: BusinessImage(
                      image: image,
                      networkUrl: image.thumb,
                      fit: BoxFit.cover,
                    ),
                  ),
                ),
              Expanded(
                child: Padding(
                  padding: EdgeInsets.all(spacing.sp3),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(product.title, style: theme.textTheme.titleMedium),
                      if (price != null) ...[
                        SizedBox(height: spacing.sp1),
                        Text(price, style: theme.textTheme.bodyMedium),
                      ],
                    ],
                  ),
                ),
              ),
              if (destination != null)
                Padding(
                  padding: EdgeInsets.all(spacing.sp2),
                  child: const Icon(CraftskyIconsBold.externalLink),
                ),
            ],
          ),
        ),
      ),
    );
  }
}
