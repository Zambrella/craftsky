import 'dart:async';

import 'package:craftsky_app/auth/providers/active_account_identity_provider.dart';
import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/providers/products_controller.dart';
import 'package:craftsky_app/business/widgets/product_editor.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/image/craftsky_image_attachment_preview.dart';
import 'package:craftsky_app/shared/widgets/craftsky_empty_state.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/craftsky_floating_action_button.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class ProductsSettingsPage extends ConsumerWidget {
  const ProductsSettingsPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final identity = ref.watch(activeAccountIdentityProvider);
    return Scaffold(
      appBar: AppBar(title: Text(l10n.businessProductsSettingsTitle)),
      body: identity.when(
        loading: () => Center(
          child: CircularProgressIndicator(
            semanticsLabel: l10n.businessLoading,
          ),
        ),
        error: (_, _) => _LoadError(
          onRetry: () => ref.invalidate(activeAccountIdentityProvider),
        ),
        data: (value) {
          if (value == null ||
              value.profile.accountType != AccountType.business) {
            return Center(child: Text(l10n.businessProductsUnavailable));
          }
          return const _ProductsManager();
        },
      ),
      floatingActionButton:
          identity.value?.profile.accountType == AccountType.business
          ? const _ProductActions()
          : null,
    );
  }
}

class _ProductsManager extends ConsumerWidget {
  const _ProductsManager();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final products = ref.watch(productsControllerProvider);
    return products.when(
      loading: () => Center(
        child: CircularProgressIndicator(
          semanticsLabel: AppLocalizations.of(context).businessLoading,
        ),
      ),
      error: (_, _) => _LoadError(
        onRetry: () => ref.invalidate(productsControllerProvider),
      ),
      data: (state) => _ProductsContent(state: state),
    );
  }
}

class _ProductsContent extends ConsumerWidget {
  const _ProductsContent({required this.state});

  final ProductsState state;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final controller = ref.read(productsControllerProvider.notifier);
    final busy = state.status == ProductsStatus.saving;
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    return SafeArea(
      child: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 760),
          child: Padding(
            padding: EdgeInsets.all(spacing.sp4),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Semantics(
                  liveRegion: true,
                  label: l10n.businessProductsCount(
                    state.products.length,
                    businessProductLimit,
                  ),
                  child: Text(
                    l10n.businessProductsCount(
                      state.products.length,
                      businessProductLimit,
                    ),
                  ),
                ),
                if (state.status == ProductsStatus.conflict)
                  Card(
                    color: Theme.of(context).colorScheme.errorContainer,
                    child: Padding(
                      padding: const EdgeInsets.all(16),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(l10n.businessProductsConflict),
                          TextButton(
                            onPressed: controller.reloadAfterConflict,
                            child: Text(l10n.businessProductsReload),
                          ),
                        ],
                      ),
                    ),
                  )
                else if (state.status == ProductsStatus.error)
                  Text(
                    l10n.businessProductsSaveError,
                    style: TextStyle(
                      color: Theme.of(context).colorScheme.error,
                    ),
                  ),
                if (state.imageErrorProductId != null)
                  Text(
                    l10n.businessProductsUploadError,
                    style: TextStyle(
                      color: Theme.of(context).colorScheme.error,
                    ),
                  ),
                SizedBox(height: spacing.sp3),
                Expanded(
                  child: state.products.isEmpty
                      ? CraftskyEmptyState(
                          icon: CraftskyIcons.storefront,
                          title: l10n.businessProductsSettingsTitle,
                          subtitle: l10n.businessProductsEmpty,
                        )
                      : ReorderableListView.builder(
                          itemCount: state.products.length,
                          onReorderItem: (oldIndex, newIndex) => unawaited(
                            controller.reorder(oldIndex, newIndex),
                          ),
                          itemBuilder: (context, index) {
                            final product = state.products[index];
                            final editLabel =
                                l10n.businessProductEditorEditTitle;
                            return CraftskyCard(
                              key: ValueKey(product.id),
                              margin: EdgeInsets.only(bottom: spacing.sp3),
                              child: InkWell(
                                onTap: busy
                                    ? null
                                    : () => _openEditor(
                                        context,
                                        controller,
                                        product,
                                      ),
                                child: Padding(
                                  padding: EdgeInsets.all(spacing.sp3),
                                  child: Row(
                                    children: [
                                      SizedBox.square(
                                        dimension: 88,
                                        child: CraftskyImageAttachmentPreview(
                                          aspectRatio: 1,
                                          bytes: product.image.previewBytes,
                                          imageUrl: product.image.previewUrl,
                                          placeholderIcon: CraftskyIcons.image,
                                        ),
                                      ),
                                      SizedBox(width: spacing.sp3),
                                      Expanded(
                                        child: Column(
                                          crossAxisAlignment:
                                              CrossAxisAlignment.start,
                                          children: [
                                            Text(
                                              product.title,
                                              style: Theme.of(
                                                context,
                                              ).textTheme.titleMedium,
                                            ),
                                            SizedBox(height: spacing.sp1),
                                            Text(
                                              product.destination,
                                              maxLines: 1,
                                              overflow: TextOverflow.ellipsis,
                                              style: Theme.of(
                                                context,
                                              ).textTheme.bodySmall,
                                            ),
                                            if (product.amount.isNotEmpty &&
                                                product
                                                    .currency
                                                    .isNotEmpty) ...[
                                              SizedBox(height: spacing.sp1),
                                              Text(
                                                '${product.amount} '
                                                '${product.currency}',
                                                style: Theme.of(
                                                  context,
                                                ).textTheme.labelMedium,
                                              ),
                                            ],
                                          ],
                                        ),
                                      ),
                                      CraftskyContextMenuButton(
                                        tooltip: l10n.businessProductEdit(
                                          product.title,
                                        ),
                                        enabled: !busy,
                                        groups: [
                                          CraftskyContextMenuGroup(
                                            items: [
                                              CraftskyContextMenuItem(
                                                text: editLabel,
                                                icon: CraftskyIconsBold.edit,
                                                onPressed: () => _openEditor(
                                                  context,
                                                  controller,
                                                  product,
                                                ),
                                              ),
                                              if (index > 0)
                                                CraftskyContextMenuItem(
                                                  text: l10n
                                                      .businessProductMoveUp(
                                                        product.title,
                                                      ),
                                                  icon:
                                                      CraftskyIconsBold.moveUp,
                                                  onPressed: () => unawaited(
                                                    controller.move(
                                                      product.id,
                                                      -1,
                                                    ),
                                                  ),
                                                ),
                                              if (index <
                                                  state.products.length - 1)
                                                CraftskyContextMenuItem(
                                                  text: l10n
                                                      .businessProductMoveDown(
                                                        product.title,
                                                      ),
                                                  icon: CraftskyIconsBold
                                                      .moveDown,
                                                  onPressed: () => unawaited(
                                                    controller.move(
                                                      product.id,
                                                      1,
                                                    ),
                                                  ),
                                                ),
                                            ],
                                          ),
                                          CraftskyContextMenuGroup(
                                            items: [
                                              CraftskyContextMenuItem(
                                                text: l10n
                                                    .businessProductRemove(
                                                      product.title,
                                                    ),
                                                icon: CraftskyIconsBold.delete,
                                                style:
                                                    CraftskyContextMenuItemStyle
                                                        .destructive,
                                                onPressed: () => unawaited(
                                                  _removeProduct(
                                                    context,
                                                    controller,
                                                    product,
                                                  ),
                                                ),
                                              ),
                                            ],
                                          ),
                                        ],
                                      ),
                                    ],
                                  ),
                                ),
                              ),
                            );
                          },
                        ),
                ),
                SizedBox(
                  key: const Key('products-manager-bottom-safe-space'),
                  height: spacing.sp9,
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Future<void> _openEditor(
    BuildContext context,
    ProductsController controller,
    ProductDraft? initial,
  ) async {
    await showProductEditorSheet(
      context,
      initial: initial,
      persist: initial == null ? controller.add : controller.editProduct,
      destinationExists: (destination) => state.products.any(
        (product) =>
            product.id != initial?.id && product.destination == destination,
      ),
    );
  }

  Future<void> _removeProduct(
    BuildContext context,
    ProductsController controller,
    ProductDraft product,
  ) async {
    final l10n = AppLocalizations.of(context);
    final remove = await showCraftskyDestructiveConfirmDialog(
      context,
      title: l10n.businessProductRemoveConfirmTitle,
      message: l10n.businessProductRemoveConfirmMessage(product.title),
      confirmLabel: l10n.businessProductRemoveConfirm,
      cancelLabel: l10n.businessProductRemoveCancel,
    );
    if (remove) await controller.remove(product.id);
  }
}

class _ProductActions extends ConsumerWidget {
  const _ProductActions();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final state = ref.watch(productsControllerProvider).value;
    if (state == null) return const SizedBox.shrink();
    final l10n = AppLocalizations.of(context);
    final controller = ref.read(productsControllerProvider.notifier);
    final busy = state.status == ProductsStatus.saving;
    return CraftskyFloatingActionButton.extended(
      onPressed: busy || state.products.length >= businessProductLimit
          ? null
          : () => _openNewProduct(context, controller, state.products),
      icon: const Icon(CraftskyIconsBold.add),
      label: Text(l10n.businessProductsAdd),
    );
  }

  Future<void> _openNewProduct(
    BuildContext context,
    ProductsController controller,
    List<ProductDraft> products,
  ) async {
    await showProductEditorSheet(
      context,
      persist: controller.add,
      destinationExists: (destination) => products.any(
        (product) => product.destination == destination,
      ),
    );
  }
}

class _LoadError extends StatelessWidget {
  const _LoadError({required this.onRetry});

  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(l10n.businessProductsLoadError),
          TextButton(
            onPressed: onRetry,
            child: Text(l10n.businessProductsRetry),
          ),
        ],
      ),
    );
  }
}
