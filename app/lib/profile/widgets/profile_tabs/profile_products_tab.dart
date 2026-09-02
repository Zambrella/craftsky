import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/widgets/product_card.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
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
        child: Center(
          child: Padding(
            padding: EdgeInsets.all(spacing.sp6),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(
                  isOwnProfile
                      ? l10n.businessProductsOwnerEmpty
                      : l10n.businessProductsVisitorEmpty,
                  textAlign: TextAlign.center,
                ),
                if (isOwnProfile && onManage != null) ...[
                  SizedBox(height: spacing.sp2),
                  TextButton(
                    onPressed: onManage,
                    child: Text(l10n.businessProductsManageAction),
                  ),
                ],
              ],
            ),
          ),
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
