package subscriptions

import "errors"

type ProductMapping struct {
	AppID string
	Tier  Tier
}

type CatalogConfig struct {
	ProjectID   string
	Environment string
	AppIDs      []string
	Products    map[string]ProductMapping
}

type Catalog struct {
	projectID   string
	environment string
	apps        map[string]struct{}
	products    map[string]ProductMapping
}

func NewCatalog(config CatalogConfig) (*Catalog, error) {
	if config.ProjectID == "" {
		return nil, errors.New("subscription catalog project ID is required")
	}
	environment := config.Environment
	if environment == "" {
		environment = "production"
	}
	if environment != "production" && environment != "sandbox" {
		return nil, errors.New("subscription catalog environment must be production or sandbox")
	}
	catalog := &Catalog{
		projectID:   config.ProjectID,
		environment: environment,
		apps:        make(map[string]struct{}, len(config.AppIDs)),
		products:    make(map[string]ProductMapping, len(config.Products)),
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
	if c == nil || projectID != c.projectID || environment != c.environment {
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
