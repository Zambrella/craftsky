import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/widgets/product_card.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/widgets/craftsky_empty_state.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

class ProfileProductsTab extends StatelessWidget {
  const ProfileProductsTab({
    required this.products,
    required this.isOwnProfile,
    this.onManage,
    super.key,
  });

  final List<BusinessProductView> products;
  final bool isOwnProfile;
  final VoidCallback? onManage;

  @override
  Widget build(BuildContext context) {
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    if (products.isEmpty) {
      final l10n = AppLocalizations.of(context);
      return SliverFillRemaining(
        hasScrollBody: false,
        child: CraftskyEmptyState(
          icon: CraftskyIcons.storefront,
          title: l10n.businessProductsSettingsTitle,
          subtitle: isOwnProfile
              ? l10n.businessProductsOwnerEmpty
              : l10n.businessProductsVisitorEmpty,
          actionLabel: isOwnProfile && onManage != null
              ? l10n.businessProductsManageAction
              : null,
          onAction: isOwnProfile ? onManage : null,
        ),
      );
    }

    return SliverPadding(
      padding: EdgeInsets.all(spacing.sp4),
      sliver: SliverList.separated(
        itemCount: products.length,
        itemBuilder: (context, index) => ProductCard(product: products[index]),
        separatorBuilder: (_, _) => SizedBox(height: spacing.sp3),
      ),
    );
  }
}
