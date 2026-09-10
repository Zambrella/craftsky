package subscriptions

import "errors"

type ProductMapping struct {
	AppID string
	Tier  Tier
}

type CatalogConfig struct {
	ProjectID string
	AppIDs    []string
	Products  map[string]ProductMapping
}

type Catalog struct {
	projectID string
	apps      map[string]struct{}
	products  map[string]ProductMapping
}

func NewCatalog(config CatalogConfig) (*Catalog, error) {
	if config.ProjectID == "" {
		return nil, errors.New("subscription catalog project ID is required")
	}
	catalog := &Catalog{
		projectID: config.ProjectID,
		apps:      make(map[string]struct{}, len(config.AppIDs)),
		products:  make(map[string]ProductMapping, len(config.Products)),
	}
	for _, appID := range config.AppIDs {
		if appID != "" {
			catalog.apps[appID] = struct{}{}
		}
	}
	for productID, mapping := range config.Products {
		catalog.products[productID] = mapping
	}
	return catalog, nil
}

func (c *Catalog) Authorize(projectID, environment, productID string) (ProductMapping, bool) {
	if c == nil || projectID != c.projectID || environment != "production" {
		return ProductMapping{}, false
	}
	mapping, ok := c.products[productID]
	if !ok {
		return ProductMapping{}, false
	}
	if _, ok := c.apps[mapping.AppID]; !ok {
		return ProductMapping{}, false
	}
	if mapping.Tier != TierPlus && mapping.Tier != TierBusiness {
		return ProductMapping{}, false
	}
	return mapping, true
}
