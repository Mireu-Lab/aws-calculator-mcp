package calculator

import (
	"encoding/json"
	"os"
	"strings"
)

// CatalogProduct represents a single GCP product with detailed pricing and cost optimization metadata.
type CatalogProduct struct {
	ID                  string  `json:"id"`
	Category            string  `json:"category"`
	MainProduct         string  `json:"main_product"`
	SubService          string  `json:"sub_service"`
	ServiceType         string  `json:"service_type"`
	Description         string  `json:"description"`
	PricingUnit         string  `json:"pricing_unit"`
	PricingModel        string  `json:"pricing_model"`
	BaseRateUSD         float64 `json:"base_rate_usd"`
	FreeTierAllowance   string  `json:"free_tier_allowance"`
	CostOptimizationTip string  `json:"cost_optimization_tip"`
}

type PricingFileFormat struct {
	UpdatedAt     string           `json:"updated_at"`
	Version       string           `json:"version"`
	TotalProducts int              `json:"total_products"`
	Products      []CatalogProduct `json:"products"`
}

var loadedProducts []CatalogProduct

func init() {
	ReloadProductsFromPricingJSON()
}

// ReloadProductsFromPricingJSON loads the catalog and pricing rates directly from pricing.json.
func ReloadProductsFromPricingJSON() {
	locations := []string{
		"pricing.json",
		"/home/limmireu1214/_projects/_ARCHIVE/_GITHUB-REPO/mireu-workflow/GCP-Calculator-MCP/pricing.json",
		"/home/limmireu1214/.gemini/config/gcp_pricing.json",
		"/home/limmireu1214/aws-calculator-mcp/pricing.json",
	}

	for _, loc := range locations {
		data, err := os.ReadFile(loc)
		if err == nil {
			var pf PricingFileFormat
			if json.Unmarshal(data, &pf) == nil && len(pf.Products) > 0 {
				loadedProducts = pf.Products
				return
			}
		}
	}
}

// GetFullCatalog returns all 115 products loaded from pricing.json.
func GetFullCatalog() []CatalogProduct {
	if len(loadedProducts) == 0 {
		ReloadProductsFromPricingJSON()
	}
	return loadedProducts
}

// SearchCatalog searches for matching products by keyword.
func SearchCatalog(query string) []CatalogProduct {
	products := GetFullCatalog()
	query = strings.ToLower(query)
	var results []CatalogProduct

	for _, p := range products {
		if strings.Contains(strings.ToLower(p.Category), query) ||
			strings.Contains(strings.ToLower(p.MainProduct), query) ||
			strings.Contains(strings.ToLower(p.SubService), query) ||
			strings.Contains(strings.ToLower(p.ServiceType), query) ||
			strings.Contains(strings.ToLower(p.Description), query) {
			results = append(results, p)
		}
	}
	return results
}
