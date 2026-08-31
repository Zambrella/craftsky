import 'dart:async';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/active_account_identity_provider.dart';
import 'package:craftsky_app/auth/providers/unsaved_work_guard_provider.dart';
import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/providers/products_controller.dart';
import 'package:craftsky_app/business/widgets/product_editor.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
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
    );
  }
}

class _ProductsManager extends ConsumerStatefulWidget {
  const _ProductsManager();

  @override
  ConsumerState<_ProductsManager> createState() => _ProductsManagerState();
}

class _ProductsManagerState extends ConsumerState<_ProductsManager> {
  late final UnsavedWorkGuard _unsavedGuard;
  UnsavedWorkRegistration? _unsavedRegistration;
  AccountSessionLease? _unsavedOwner;
  bool _allowPop = false;

  @override
  void initState() {
    super.initState();
    _unsavedGuard = ref.read(unsavedWorkGuardProvider);
  }

  @override
  void dispose() {
    _clearUnsavedRegistration();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final products = ref.watch(productsControllerProvider);
    _syncUnsavedRegistration(products.value);
    return PopScope<Object?>(
      canPop: _allowPop || products.value?.dirty != true,
      onPopInvokedWithResult: (didPop, _) async {
        if (didPop) return;
        final owner = _unsavedOwner;
        if (owner != null) await _unsavedGuard.confirmLeave(owner);
      },
      child: products.when(
        loading: () => Center(
          child: CircularProgressIndicator(
            semanticsLabel: AppLocalizations.of(context).businessLoading,
          ),
        ),
        error: (_, _) => _LoadError(
          onRetry: () => ref.invalidate(productsControllerProvider),
        ),
        data: (state) => _ProductsContent(state: state),
      ),
    );
  }

  void _syncUnsavedRegistration(ProductsState? products) {
    final owner = ref.read(activeAccountIdentityProvider).value?.lease;
    if (products?.dirty != true || owner == null) {
      _clearUnsavedRegistration();
      return;
    }
    if (owner == _unsavedOwner && _unsavedRegistration != null) return;
    _unsavedOwner = owner;
    _unsavedRegistration = _unsavedGuard.replace(
      _unsavedRegistration,
      owner: owner,
      isDirty: () =>
          mounted &&
          ref.read(activeAccountIdentityProvider).value?.lease == owner &&
          ref.read(productsControllerProvider).value?.dirty == true,
      confirmAndClose: () => _confirmAndClose(owner),
    );
  }

  void _clearUnsavedRegistration() {
    _unsavedGuard.unregister(_unsavedRegistration);
    _unsavedRegistration = null;
    _unsavedOwner = null;
  }

  Future<bool> _confirmAndClose(AccountSessionLease owner) async {
    if (!mounted ||
        ref.read(activeAccountIdentityProvider).value?.lease != owner) {
      return true;
    }
    final l10n = AppLocalizations.of(context);
    final discard = await showCraftskyConfirmDialog(
      context,
      title: l10n.editProfileDiscardTitle,
      message: l10n.editProfileDiscardMessage,
      confirmLabel: l10n.editProfileDiscardConfirm,
      cancelLabel: l10n.editProfileDiscardCancel,
    );
    if (!discard || !mounted) return false;
    _clearUnsavedRegistration();
    setState(() => _allowPop = true);
    await WidgetsBinding.instance.endOfFrame;
    if (mounted && Navigator.of(context).canPop()) Navigator.of(context).pop();
    return true;
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
    return SafeArea(
      child: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 760),
          child: Padding(
            padding: const EdgeInsets.all(16),
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
                const SizedBox(height: 12),
                Expanded(
                  child: state.products.isEmpty
                      ? Center(child: Text(l10n.businessProductsEmpty))
                      : ReorderableListView.builder(
                          itemCount: state.products.length,
                          onReorderItem: controller.reorder,
                          itemBuilder: (context, index) {
                            final product = state.products[index];
                            return Card(
                              key: ValueKey(product.id),
                              child: ListTile(
                                title: Text(product.title),
                                subtitle: Text(
                                  product.destination,
                                  maxLines: 1,
                                  overflow: TextOverflow.ellipsis,
                                ),
                                onTap: busy
                                    ? null
                                    : () => _openEditor(
                                        context,
                                        controller,
                                        product,
                                      ),
                                trailing: Wrap(
                                  children: [
                                    Semantics(
                                      button: true,
                                      label: l10n.businessProductMoveUp(
                                        product.title,
                                      ),
                                      excludeSemantics: true,
                                      child: IconButton(
                                        tooltip: l10n.businessProductMoveUp(
                                          product.title,
                                        ),
                                        onPressed: busy || index == 0
                                            ? null
                                            : () => controller.move(
                                                product.id,
                                                -1,
                                              ),
                                        icon: const Icon(Icons.arrow_upward),
                                      ),
                                    ),
                                    Semantics(
                                      button: true,
                                      label: l10n.businessProductMoveDown(
                                        product.title,
                                      ),
                                      excludeSemantics: true,
                                      child: IconButton(
                                        tooltip: l10n.businessProductMoveDown(
                                          product.title,
                                        ),
                                        onPressed:
                                            busy ||
                                                index ==
                                                    state.products.length - 1
                                            ? null
                                            : () => controller.move(
                                                product.id,
                                                1,
                                              ),
                                        icon: const Icon(Icons.arrow_downward),
                                      ),
                                    ),
                                    Semantics(
                                      button: true,
                                      label: l10n.businessProductRemove(
                                        product.title,
                                      ),
                                      excludeSemantics: true,
                                      child: IconButton(
                                        tooltip: l10n.businessProductRemove(
                                          product.title,
                                        ),
                                        onPressed: busy
                                            ? null
                                            : () =>
                                                  controller.remove(product.id),
                                        icon: const Icon(Icons.delete_outline),
                                      ),
                                    ),
                                  ],
                                ),
                              ),
                            );
                          },
                        ),
                ),
                const SizedBox(height: 12),
                Wrap(
                  alignment: WrapAlignment.end,
                  spacing: 8,
                  runSpacing: 8,
                  children: [
                    OutlinedButton.icon(
                      onPressed:
                          busy || state.products.length >= businessProductLimit
                          ? null
                          : () => _openEditor(context, controller, null),
                      icon: const Icon(Icons.add),
                      label: Text(l10n.businessProductsAdd),
                    ),
                    FilledButton.icon(
                      onPressed: busy || !state.dirty
                          ? null
                          : () => unawaited(controller.save()),
                      icon: busy
                          ? SizedBox.square(
                              dimension: 18,
                              child: CircularProgressIndicator(
                                strokeWidth: 2,
                                semanticsLabel: l10n.businessSaving,
                              ),
                            )
                          : const Icon(Icons.save_outlined),
                      label: Text(l10n.businessProductsSave),
                    ),
                  ],
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
    await showDialog<void>(
      context: context,
      builder: (dialogContext) => Dialog(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 560),
          child: ProductEditor(
            initial: initial,
            onCancel: () => Navigator.pop(dialogContext),
            onSave: (product) {
              if (initial == null) {
                controller.add(product);
              } else {
                controller.editProduct(product);
              }
              Navigator.pop(dialogContext);
            },
          ),
        ),
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
