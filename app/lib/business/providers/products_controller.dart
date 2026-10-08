import 'dart:async';
import 'dart:convert';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/active_account_identity_provider.dart';
import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/providers/business_record_overlay.dart';
import 'package:craftsky_app/business/providers/business_repository_provider.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:craftsky_app/shared/mutations/pds_record_reconciliation.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'products_controller.g.dart';

typedef ProductsProfileLoader = Future<Profile> Function();

final productsProfileLoaderProvider = Provider<ProductsProfileLoader>((ref) {
  return () async {
    final identity = await ref.read(activeAccountIdentityProvider.future);
    if (identity == null) throw const ProductsUnavailableException();
    return ref
        .read(profileRepositoryProvider)
        .fetch(identity.profile.did.toString());
  };
});

enum ProductsStatus { ready, saving, ambiguous, conflict, error }

@immutable
class ProductsState {
  const ProductsState({
    required this.declaration,
    required this.products,
    this.status = ProductsStatus.ready,
    this.validationErrors = const {},
    this.imageErrorProductId,
  });

  final BusinessDeclarationDraft declaration;
  final List<ProductDraft> products;
  final ProductsStatus status;
  final Set<ProductDraftError> validationErrors;
  final String? imageErrorProductId;

  ProductsState copyWith({
    BusinessDeclarationDraft? declaration,
    List<ProductDraft>? products,
    ProductsStatus? status,
    Set<ProductDraftError>? validationErrors,
    Object? imageErrorProductId = _unset,
  }) => ProductsState(
    declaration: declaration ?? this.declaration,
    products: products ?? this.products,
    status: status ?? this.status,
    validationErrors: validationErrors ?? this.validationErrors,
    imageErrorProductId: identical(imageErrorProductId, _unset)
        ? this.imageErrorProductId
        : imageErrorProductId as String?,
  );
}

const _unset = Object();

class ProductsUnavailableException implements Exception {
  const ProductsUnavailableException();
}

@Riverpod(keepAlive: true)
class ProductsController extends _$ProductsController {
  late Did _ownerDid;

  @override
  Future<ProductsState> build() async {
    final identity = await ref.watch(activeAccountIdentityProvider.future);
    if (identity == null ||
        identity.profile.accountType != AccountType.business) {
      throw const ProductsUnavailableException();
    }
    _ownerDid = identity.profile.did;
    return _stateFromProfile(
      identity.profile.copyWith(
        business: applyBusinessProfileOverlay(
          ref,
          identity.lease,
          _ownerDid,
          identity.profile.business,
        ),
      ),
    );
  }

  Future<bool> replaceProducts(List<ProductDraft> products) =>
      _persist(products);

  Future<bool> add(ProductDraft product) {
    final current = state.value;
    if (current == null || current.products.length >= businessProductLimit) {
      return Future.value(false);
    }
    return _persist([...current.products, product]);
  }

  Future<bool> editProduct(ProductDraft product) {
    final current = state.value;
    if (current == null ||
        !current.products.any((existing) => existing.id == product.id)) {
      return Future.value(false);
    }
    return _persist([
      for (final existing in current.products)
        if (existing.id == product.id) product else existing,
    ]);
  }

  Future<bool> remove(String id) {
    final current = state.value;
    if (current == null ||
        !current.products.any((product) => product.id == id)) {
      return Future.value(false);
    }
    return _persist(
      current.products.where((product) => product.id != id).toList(),
    );
  }

  Future<bool> reorder(int oldIndex, int newIndex) {
    final current = state.value;
    if (current == null ||
        oldIndex < 0 ||
        oldIndex >= current.products.length) {
      return Future.value(false);
    }
    final products = [...current.products];
    if (newIndex < 0 || newIndex >= products.length) {
      return Future.value(false);
    }
    final product = products.removeAt(oldIndex);
    products.insert(newIndex, product);
    return _persist(products);
  }

  Future<bool> move(String id, int delta) {
    final current = state.value;
    if (current == null) return Future.value(false);
    final from = current.products.indexWhere((product) => product.id == id);
    final to = from + delta;
    if (from < 0 || to < 0 || to >= current.products.length) {
      return Future.value(false);
    }
    final products = [...current.products];
    final product = products.removeAt(from);
    products.insert(to, product);
    return _persist(products);
  }

  Future<bool> replaceImage(String id, BusinessImageDraft image) {
    final current = state.value;
    if (current == null) return Future.value(false);
    final product = current.products.firstWhere((value) => value.id == id);
    return editProduct(
      ProductDraft(
        id: product.id,
        title: product.title,
        destination: product.destination,
        image: image,
        amount: product.amount,
        currency: product.currency,
      ),
    );
  }

  void imageUploadFailed(String id) => _retainImageAfterUpload(id);

  void imageUploadCancelled(String id) => _retainImageAfterUpload(id);

  Future<bool> _persist(List<ProductDraft> products) async {
    final current = state.value;
    if (current == null ||
        current.status == ProductsStatus.saving ||
        current.status == ProductsStatus.conflict) {
      return false;
    }
    final errors = validateProductDrafts(products);
    if (errors.isNotEmpty) {
      state = AsyncData(
        current.copyWith(
          status: ProductsStatus.error,
          validationErrors: errors,
        ),
      );
      return false;
    }

    final ownership = captureActiveAccountOperation(ref);
    final lease =
        ownership?.session ??
        AccountSessionLease(
          account: AccountKey(_ownerDid.toString()),
          sessionGeneration: 0,
        );
    final body = Map<String, dynamic>.from(
      jsonDecode(
            jsonEncode(
              current.declaration.toJson(
                productDrafts: products,
                preserveUnknownCatalogValues: true,
              ),
            ),
          )
          as Map,
    );
    const endpoint = '/v1/profiles/me/business';
    final immutableBody = jsonEncode(body);
    final commandController = ref.read(pdsRecordOperationControllerProvider);
    final commandToken = commandController.beginOrRetry(
      scope: businessProfileMutationScope(lease, _ownerDid),
      endpoint: endpoint,
      immutableBody: immutableBody,
      newOperationKey: newPdsMutationOperationKey,
    );
    final operationKey = commandToken.operationKey;
    final startedAt = ref.read(pdsMutationNowProvider)();
    var retryIndex = 0;
    state = AsyncData(
      current.copyWith(
        status: ProductsStatus.saving,
        validationErrors: const {},
      ),
    );
    while (true) {
      late final RecordMutationResult result;
      try {
        result = await ref
            .read(businessRepositoryProvider)
            .putBusinessProfile(
              body,
              operationKey: operationKey,
              expectedCid: current.declaration.expectedCid,
            );
      } on PdsMutationAmbiguousException catch (error) {
        if (!isActiveAccountOperationCurrent(ref, ownership)) return false;
        commandController.markAmbiguous(
          commandToken,
          retryAfterSeconds: error.retryAfterSeconds,
        );
        final delay = const PdsMutationRetryPolicy().nextDelay(
          retryIndex: retryIndex,
          retryAfterSeconds: error.retryAfterSeconds,
          elapsed: ref.read(pdsMutationNowProvider)().difference(startedAt),
          jitterMillis: ref.read(pdsMutationJitterProvider),
        );
        if (delay == null) {
          state = AsyncData(
            current.copyWith(status: ProductsStatus.ambiguous),
          );
          return false;
        }
        retryIndex++;
        await ref.read(pdsMutationDelayProvider)(delay);
        if (!isActiveAccountOperationCurrent(ref, ownership) ||
            !commandController.canRetry(
              commandToken,
              operationKey: operationKey,
              endpoint: endpoint,
              immutableBody: immutableBody,
            )) {
          return false;
        }
        continue;
      } on ApiBadRequest catch (error) {
        if (!isActiveAccountOperationCurrent(ref, ownership)) return false;
        commandController.markFailed(commandToken);
        state = AsyncData(
          current.copyWith(
            status:
                error.code == 'pds_record_conflict' ||
                    error.details.statusCode == 409
                ? ProductsStatus.conflict
                : ProductsStatus.error,
          ),
        );
        return false;
      } on Object {
        if (!isActiveAccountOperationCurrent(ref, ownership)) return false;
        commandController.markFailed(commandToken);
        state = AsyncData(current.copyWith(status: ProductsStatus.error));
        return false;
      }
      if (!isActiveAccountOperationCurrent(ref, ownership)) return false;
      final declaration = current.declaration.withExpectedCid(result.cid);
      final accepted = _acceptedProfile(
        declaration,
        products,
        current.declaration.products,
      );
      final reconciliation = PdsFixedKeyReconciliation(
        uri: 'at://$_ownerDid/social.craftsky.business.profile/self',
        controlledContent: businessProfileProjection(
          _ownerDid,
          accepted,
        ).content,
      );
      if (!commandController.markAccepted(
        commandToken,
        optimisticValue: accepted,
        agrees: (value) => reconciliation.agrees(
          value is PdsRecordProjection ? value : null,
        ),
        refresh: _invalidateBusinessProfileReads,
      )) {
        return false;
      }
      state = AsyncData(
        current.copyWith(
          declaration: declaration,
          products: List.unmodifiable(products),
          status: ProductsStatus.ready,
          imageErrorProductId: null,
        ),
      );
      return true;
    }
  }

  void _invalidateBusinessProfileReads() {
    if (!ref.mounted) return;
    ref.invalidate(userProfileProvider);
  }

  Future<void> reloadAfterConflict() async {
    final current = state.value;
    if (current == null || current.status != ProductsStatus.conflict) return;
    state = const AsyncLoading<ProductsState>();
    try {
      final profile = await ref.read(productsProfileLoaderProvider)();
      if (profile.did != _ownerDid ||
          profile.accountType != AccountType.business) {
        throw const ProductsUnavailableException();
      }
      state = AsyncData(_stateFromProfile(profile));
    } on Object catch (error, stackTrace) {
      state = AsyncError(error, stackTrace);
    }
  }

  void _retainImageAfterUpload(String id) {
    final current = state.value;
    if (current == null) return;
    state = AsyncData(current.copyWith(imageErrorProductId: id));
  }

  ProductsState _stateFromProfile(Profile profile) {
    final declaration = BusinessDeclarationDraft.fromProfile(profile.business);
    return ProductsState(
      declaration: declaration,
      products: List.unmodifiable([
        for (var index = 0; index < declaration.products.length; index++)
          ProductDraft.fromView(
            declaration.products[index],
            'saved-${declaration.expectedCid ?? 'new'}-$index',
          ),
      ]),
    );
  }

  BusinessProfile _acceptedProfile(
    BusinessDeclarationDraft declaration,
    List<ProductDraft> products,
    List<BusinessProductView> previousProducts,
  ) => BusinessProfile(
    cid: declaration.expectedCid.toString(),
    businessTypes: declaration.businessTypes,
    offerings: declaration.offerings,
    tagline: declaration.tagline,
    hoursNote: declaration.hoursNote,
    serviceArea: declaration.serviceArea,
    location: declaration.location,
    primaryAction: declaration.primaryAction,
    products: [
      for (final product in products)
        BusinessProductView(
          title: product.title,
          uri: product.destination,
          image: _acceptedImage(product.image, previousProducts),
          price: product.amount.isEmpty
              ? null
              : BusinessPrice(
                  amount: product.amount,
                  currency: product.currency,
                ),
        ),
    ],
  );

  BusinessImageView? _acceptedImage(
    BusinessImageDraft image,
    List<BusinessProductView> previousProducts,
  ) {
    if (image case ExistingBusinessImageDraft(:final cid)) {
      for (final product in previousProducts) {
        if (product.image?.cid.toString() == cid) return product.image;
      }
    }
    if (image case UploadedBusinessImageDraft(
      :final cid,
      :final mime,
      :final size,
      :final alt,
      :final aspectRatio,
      previewBytes: final previewBytes?,
    )) {
      return BusinessImageView.localPreview(
        cid: cid,
        mime: mime,
        size: size,
        alt: alt,
        aspectRatio: aspectRatio,
        previewBytes: previewBytes,
      );
    }
    return null;
  }
}
