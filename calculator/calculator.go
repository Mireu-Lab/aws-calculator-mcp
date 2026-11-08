package calculator

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"time"
)

type PricingPolicy struct {
	UpdatedAt           string             `json:"updated_at"`
	Version             string             `json:"version"`
	TotalProducts       int                `json:"total_products"`
	CurrencyExchange    map[string]float64 `json:"currency_exchange"`
	RegionMultipliers   map[string]float64 `json:"region_multipliers"`
	CommitmentDiscounts map[string]float64 `json:"commitment_discounts"`
	Products            []CatalogProduct   `json:"products"`
}

var (
	policyLock   sync.RWMutex
	activePolicy PricingPolicy
)

func init() {
	activePolicy = GetDefaultPolicy()
	locations := []string{
		"pricing.json",
		"/home/limmireu1214/_projects/_ARCHIVE/_GITHUB-REPO/mireu-workflow/GCP-Calculator-MCP/pricing.json",
		"/home/limmireu1214/.gemini/config/gcp_pricing.json",
		"/home/limmireu1214/aws-calculator-mcp/pricing.json",
	}
	for _, loc := range locations {
		if LoadPricingPolicy(loc) == nil {
			break
		}
	}
}

func GetDefaultPolicy() PricingPolicy {
	prods := GetFullCatalog()
	return PricingPolicy{
		UpdatedAt:        time.Now().UTC().Format(time.RFC3339),
		Version:          "8.1.0-en-usd-official",
		TotalProducts:    len(prods),
		CurrencyExchange: map[string]float64{"USD_TO_KRW": 1334.51},
		RegionMultipliers: map[string]float64{
			"us-central1":     1.00,
			"us-east1":        1.00,
			"us-west1":        1.00,
			"asia-northeast3": 1.20, // Seoul
			"asia-northeast1": 1.18, // Tokyo
			"asia-east1":      1.15, // Taiwan
			"europe-west1":    1.10, // Belgium
		},
		CommitmentDiscounts: map[string]float64{
			"none":   1.00,
			"1-year": 0.63, // 37% discount
			"3-year": 0.45, // 55% discount
		},
		Products: prods,
	}
}

func LoadPricingPolicy(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var policy PricingPolicy
	if err := json.Unmarshal(data, &policy); err != nil {
		return err
	}
	policyLock.Lock()
	activePolicy = policy
	policyLock.Unlock()
	return nil
}

func GetActivePolicy() PricingPolicy {
	policyLock.RLock()
	defer policyLock.RUnlock()
	return activePolicy
}

func GetExchangeRate() float64 {
	policy := GetActivePolicy()
	if ex, ok := policy.CurrencyExchange["USD_TO_KRW"]; ok && ex > 0 {
		return ex
	}
	return 1334.51
}

func GetRegionMultiplier(region string) float64 {
	policy := GetActivePolicy()
	if mult, ok := policy.RegionMultipliers[strings.ToLower(region)]; ok {
		return mult
	}
	return 1.00
}

func GetCommitmentMultiplier(commitment string) float64 {
	policy := GetActivePolicy()
	if mult, ok := policy.CommitmentDiscounts[strings.ToLower(commitment)]; ok {
		return mult
	}
	return 1.00
}

// Compute Engine calculation
type ComputeEngineInput struct {
	MachineType     string  `json:"machine_type"`
	Region          string  `json:"region"`
	Commitment      string  `json:"commitment"` // "none", "1-year", "3-year"
	HoursPerMonth   float64 `json:"hours_per_month"`
	IsSpot          bool    `json:"is_spot"`
	GPUType         string  `json:"gpu_type"`
	GPUCount        int     `json:"gpu_count"`
	TPUType         string  `json:"tpu_type"`
	TPUCount        int     `json:"tpu_count"`
	StorageGB       float64 `json:"storage_gb"`
	StorageType     string  `json:"storage_type"`
	ProvisionedIOPS float64 `json:"provisioned_iops"`
}

type ComputeEngineResult struct {
	MachineType     string  `json:"machine_type"`
	Region          string  `json:"region"`
	Commitment      string  `json:"commitment"`
	HourlyRate      float64 `json:"hourly_rate_usd"`
	GPUCost         float64 `json:"gpu_cost_usd"`
	TPUCost         float64 `json:"tpu_cost_usd"`
	ComputeCost     float64 `json:"compute_cost_usd"`
	StorageCost     float64 `json:"storage_cost_usd"`
	TotalMonthlyUSD float64 `json:"total_monthly_cost_usd"`
	TotalMonthlyKRW float64 `json:"total_monthly_cost_krw"`
	IsSpot          bool    `json:"is_spot"`
	SavingsTip      string  `json:"savings_tip"`
}

func CalculateComputeEngine(input ComputeEngineInput) ComputeEngineResult {
	if input.HoursPerMonth <= 0 {
		input.HoursPerMonth = 730
	}
	if input.Region == "" {
		input.Region = "us-central1"
	}
	regionMult := GetRegionMultiplier(input.Region)
	commitMult := GetCommitmentMultiplier(input.Commitment)

	machine := strings.ToLower(input.MachineType)
	if machine == "" {
		machine = "e2-micro"
	}

	rate := 0.0336 * regionMult
	if input.IsSpot {
		rate *= 0.30
	} else {
		rate *= commitMult
	}

	gpuHourly := 0.0
	if input.GPUType != "" && input.GPUCount > 0 {
		gpuHourly = 0.35 * float64(input.GPUCount) * regionMult
		if input.IsSpot {
			gpuHourly *= 0.40
		}
	}

	tpuHourly := 0.0
	if input.TPUType != "" {
		count := input.TPUCount
		if count <= 0 {
			count = 1
		}
		tpuHourly = 4.50 * float64(count) * regionMult
		if input.IsSpot {
			tpuHourly *= 0.30
		}
	}

	computeCost := rate * input.HoursPerMonth
	gpuCost := gpuHourly * input.HoursPerMonth
	tpuCost := tpuHourly * input.HoursPerMonth

	diskRate := 0.04 * regionMult // standard
	switch strings.ToLower(input.StorageType) {
	case "balanced", "pd-balanced":
		diskRate = 0.10 * regionMult
	case "ssd", "pd-ssd":
		diskRate = 0.17 * regionMult
	case "extreme", "pd-extreme":
		diskRate = 0.125 * regionMult
	}
	storageCost := input.StorageGB * diskRate
	if strings.Contains(strings.ToLower(input.StorageType), "extreme") && input.ProvisionedIOPS > 0 {
		// Provisioned IOPS: ~$0.065/IOPS in US (~86.74 KRW), ~$0.085/IOPS in Seoul (~113.43 KRW)
		iopsMonthlyUSD := 0.065 * regionMult
		storageCost += input.ProvisionedIOPS * iopsMonthlyUSD * (input.HoursPerMonth / 730.0)
	}
	totalUSD := computeCost + gpuCost + tpuCost + storageCost
	exchangeRate := GetExchangeRate()
	totalKRW := totalUSD * exchangeRate

	tip := "Consider 3-Year Committed Use Discounts (CUD) to save up to 55%."
	if input.IsSpot {
		tip = "Spot VM discount applied."
	} else if input.Commitment != "" && input.Commitment != "none" {
		tip = fmt.Sprintf("%s Committed Use Discount (CUD) applied.", input.Commitment)
	}

	return ComputeEngineResult{
		MachineType:     machine,
		Region:          input.Region,
		Commitment:      input.Commitment,
		HourlyRate:      math.Round(rate*10000) / 10000,
		GPUCost:         math.Round(gpuCost*100) / 100,
		TPUCost:         math.Round(tpuCost*100) / 100,
		ComputeCost:     math.Round(computeCost*100) / 100,
		StorageCost:     math.Round(storageCost*100) / 100,
		TotalMonthlyUSD: math.Round(totalUSD*100) / 100,
		TotalMonthlyKRW: math.Round(totalKRW),
		IsSpot:          input.IsSpot,
		SavingsTip:      tip,
	}
}

// Cloud Run
type CloudRunInput struct {
	RequestsPerMonth float64 `json:"requests_per_month"`
	CPUCores         float64 `json:"cpu_cores"`
	MemoryGB         float64 `json:"memory_gb"`
	AvgDurationMS    float64 `json:"avg_duration_ms"`
	Concurrency      float64 `json:"concurrency"`
}

type CloudRunResult struct {
	BillableRequests float64 `json:"billable_requests"`
	TotalMonthlyUSD  float64 `json:"total_monthly_cost_usd"`
	TotalMonthlyKRW  float64 `json:"total_monthly_cost_krw"`
	IsFreeTier       bool    `json:"is_free_tier"`
	SavingsTip       string  `json:"savings_tip"`
}

func CalculateCloudRun(input CloudRunInput) CloudRunResult {
	if input.CPUCores <= 0 {
		input.CPUCores = 1
	}
	if input.MemoryGB <= 0 {
		input.MemoryGB = 0.5
	}
	if input.AvgDurationMS <= 0 {
		input.AvgDurationMS = 200
	}
	if input.Concurrency <= 0 {
		input.Concurrency = 80
	}

	effectiveExecutions := math.Ceil(input.RequestsPerMonth / input.Concurrency)
	durationSec := effectiveExecutions * (input.AvgDurationMS / 1000.0)

	totalVCPUSeconds := durationSec * input.CPUCores
	totalGiBSeconds := durationSec * input.MemoryGB

	billableRequests := math.Max(0, input.RequestsPerMonth-2000000)
	billableVCPU := math.Max(0, totalVCPUSeconds-180000)
	billableMem := math.Max(0, totalGiBSeconds-360000)

	reqCost := billableRequests * (0.40 / 1000000.0)
	vcpuCost := billableVCPU * 0.00002400
	memCost := billableMem * 0.00000250

	totalUSD := reqCost + vcpuCost + memCost
	totalKRW := totalUSD * GetExchangeRate()

	return CloudRunResult{
		BillableRequests: billableRequests,
		TotalMonthlyUSD:  math.Round(totalUSD*100) / 100,
		TotalMonthlyKRW:  math.Round(totalKRW),
		IsFreeTier:       totalUSD == 0,
		SavingsTip:       "Cloud Run includes 2M requests free every month.",
	}
}

// Storage
type StorageInput struct {
	StorageGB       float64 `json:"storage_gb"`
	StorageClass    string  `json:"storage_class"`
	NetworkEgressGB float64 `json:"network_egress_gb"`
}

type StorageResult struct {
	StorageClass    string  `json:"storage_class"`
	StorageCost     float64 `json:"storage_cost_usd"`
	EgressCost      float64 `json:"egress_cost_usd"`
	TotalMonthlyUSD float64 `json:"total_monthly_cost_usd"`
	TotalMonthlyKRW float64 `json:"total_monthly_cost_krw"`
	SavingsTip      string  `json:"savings_tip"`
}

func CalculateStorage(input StorageInput) StorageResult {
	rate := 0.020
	switch strings.ToLower(input.StorageClass) {
	case "nearline":
		rate = 0.010
	case "coldline":
		rate = 0.004
	case "archive":
		rate = 0.0012
	}

	storageCost := input.StorageGB * rate
	egressBillable := math.Max(0, input.NetworkEgressGB-10)
	egressCost := egressBillable * 0.12

	totalUSD := storageCost + egressCost
	totalKRW := totalUSD * GetExchangeRate()

	return StorageResult{
		StorageClass:    input.StorageClass,
		StorageCost:     math.Round(storageCost*100) / 100,
		EgressCost:      math.Round(egressCost*100) / 100,
		TotalMonthlyUSD: math.Round(totalUSD*100) / 100,
		TotalMonthlyKRW: math.Round(totalKRW),
		SavingsTip:      "Nearline/Coldline lifecycle policies save storage costs.",
	}
}

// GKE
type GKEInput struct {
	Mode         string  `json:"mode"`
	NodeCount    int     `json:"node_count"`
	VCPUsPerNode float64 `json:"vcpus_per_node"`
	MemGBPerNode float64 `json:"mem_gb_per_node"`
	Hours        float64 `json:"hours_per_month"`
}

type GKEResult struct {
	Mode            string  `json:"mode"`
	ClusterFeeCost  float64 `json:"cluster_fee_usd"`
	WorkloadCost    float64 `json:"workload_cost_usd"`
	TotalMonthlyUSD float64 `json:"total_monthly_cost_usd"`
	TotalMonthlyKRW float64 `json:"total_monthly_cost_krw"`
	SavingsTip      string  `json:"savings_tip"`
}

func CalculateGKE(input GKEInput) GKEResult {
	if input.Hours <= 0 {
		input.Hours = 730
	}
	clusterFee := 0.10 * input.Hours
	var workloadCost float64

	if strings.ToLower(input.Mode) == "autopilot" {
		vcpuCost := input.VCPUsPerNode * 0.0445 * input.Hours
		memCost := input.MemGBPerNode * 0.004925 * input.Hours
		workloadCost = vcpuCost + memCost
	} else {
		ceInput := ComputeEngineInput{
			MachineType:   "e2-standard-4",
			HoursPerMonth: input.Hours,
			StorageGB:     100,
		}
		if input.VCPUsPerNode == 2 {
			ceInput.MachineType = "e2-standard-2"
		}
		ceRes := CalculateComputeEngine(ceInput)
		workloadCost = ceRes.TotalMonthlyUSD * float64(math.Max(1, float64(input.NodeCount)))
	}

	totalUSD := clusterFee + workloadCost
	totalKRW := totalUSD * GetExchangeRate()

	return GKEResult{
		Mode:            input.Mode,
		ClusterFeeCost:  math.Round(clusterFee*100) / 100,
		WorkloadCost:    math.Round(workloadCost*100) / 100,
		TotalMonthlyUSD: math.Round(totalUSD*100) / 100,
		TotalMonthlyKRW: math.Round(totalKRW),
		SavingsTip:      "Autopilot eliminates unallocated node capacity cost.",
	}
}

// Cloud SQL
type CloudSQLInput struct {
	InstanceType string  `json:"instance_type"`
	StorageGB    float64 `json:"storage_gb"`
	HighAvail    bool    `json:"high_availability"`
	Hours        float64 `json:"hours_per_month"`
}

type CloudSQLResult struct {
	InstanceType    string  `json:"instance_type"`
	ComputeCost     float64 `json:"compute_cost_usd"`
	StorageCost     float64 `json:"storage_cost_usd"`
	TotalMonthlyUSD float64 `json:"total_monthly_cost_usd"`
	TotalMonthlyKRW float64 `json:"total_monthly_cost_krw"`
	SavingsTip      string  `json:"savings_tip"`
}

func CalculateCloudSQL(input CloudSQLInput) CloudSQLResult {
	if input.Hours <= 0 {
		input.Hours = 730
	}
	hourly := 0.0150
	switch input.InstanceType {
	case "db-g1-small":
		hourly = 0.0300
	case "db-custom-2-7680":
		hourly = 0.1080
	case "db-custom-4-15360":
		hourly = 0.2160
	}

	mult := 1.0
	if input.HighAvail {
		mult = 2.0
	}

	computeCost := hourly * input.Hours * mult
	storageCost := input.StorageGB * 0.17
	totalUSD := computeCost + storageCost
	totalKRW := totalUSD * GetExchangeRate()

	return CloudSQLResult{
		InstanceType:    input.InstanceType,
		ComputeCost:     math.Round(computeCost*100) / 100,
		StorageCost:     math.Round(storageCost*100) / 100,
		TotalMonthlyUSD: math.Round(totalUSD*100) / 100,
		TotalMonthlyKRW: math.Round(totalKRW),
		SavingsTip:      "Use db-f1-micro or db-g1-small for dev/test environments.",
	}
}

// BigQuery
type BigQueryInput struct {
	QueryTBScanned  float64 `json:"query_tb_scanned"`
	ActiveStorageGB float64 `json:"active_storage_gb"`
}

type BigQueryResult struct {
	QueryCost       float64 `json:"query_cost_usd"`
	StorageCost     float64 `json:"storage_cost_usd"`
	TotalMonthlyUSD float64 `json:"total_monthly_cost_usd"`
	TotalMonthlyKRW float64 `json:"total_monthly_cost_krw"`
	SavingsTip      string  `json:"savings_tip"`
}

func CalculateBigQuery(input BigQueryInput) BigQueryResult {
	billableTB := math.Max(0, input.QueryTBScanned-1.0)
	queryCost := billableTB * 6.25

	billableGB := math.Max(0, input.ActiveStorageGB-10.0)
	storageCost := billableGB * 0.02

	totalUSD := queryCost + storageCost
	totalKRW := totalUSD * GetExchangeRate()

	return BigQueryResult{
		QueryCost:       math.Round(queryCost*100) / 100,
		StorageCost:     math.Round(storageCost*100) / 100,
		TotalMonthlyUSD: math.Round(totalUSD*100) / 100,
		TotalMonthlyKRW: math.Round(totalKRW),
		SavingsTip:      "BigQuery includes 1TB queries and 10GB storage FREE per month.",
	}
}

// Universal Service Calculator with Param-Aware Dynamic Evaluator
type UniversalInput struct {
	ServiceName string                 `json:"service_name"`
	Params      map[string]interface{} `json:"params"`
}

type UniversalResult struct {
	ServiceName     string           `json:"service_name"`
	TotalMonthlyUSD float64          `json:"total_monthly_cost_usd"`
	TotalMonthlyKRW float64          `json:"total_monthly_cost_krw"`
	CatalogMatches  []CatalogProduct `json:"catalog_matches,omitempty"`
	Details         interface{}      `json:"details"`
	SavingsTip      string           `json:"savings_tip"`
}

func CalculateService(input UniversalInput) UniversalResult {
	svc := strings.ToLower(input.ServiceName)
	matches := SearchCatalog(svc)

	switch svc {
	case "compute", "compute_engine", "vm", "tpu", "gpu":
		var ceInput ComputeEngineInput
		bytes, _ := json.Marshal(input.Params)
		_ = json.Unmarshal(bytes, &ceInput)
		res := CalculateComputeEngine(ceInput)
		return UniversalResult{ServiceName: "Compute Engine (VM/GPU/TPU)", TotalMonthlyUSD: res.TotalMonthlyUSD, TotalMonthlyKRW: res.TotalMonthlyKRW, CatalogMatches: matches, Details: res, SavingsTip: res.SavingsTip}

	case "cloud_run", "run":
		var crInput CloudRunInput
		bytes, _ := json.Marshal(input.Params)
		_ = json.Unmarshal(bytes, &crInput)
		res := CalculateCloudRun(crInput)
		return UniversalResult{ServiceName: "Cloud Run", TotalMonthlyUSD: res.TotalMonthlyUSD, TotalMonthlyKRW: res.TotalMonthlyKRW, CatalogMatches: matches, Details: res, SavingsTip: res.SavingsTip}

	case "app_engine", "appengine":
		var aeInput AppEngineInput
		bytes, _ := json.Marshal(input.Params)
		_ = json.Unmarshal(bytes, &aeInput)
		res := CalculateAppEngine(aeInput)
		totalKRW := res.TotalMonthlyCost * GetExchangeRate()
		return UniversalResult{ServiceName: "App Engine", TotalMonthlyUSD: res.TotalMonthlyCost, TotalMonthlyKRW: math.Round(totalKRW), CatalogMatches: matches, Details: res, SavingsTip: res.SavingsTip}

	case "storage", "cloud_storage", "gcs":
		var stInput StorageInput
		bytes, _ := json.Marshal(input.Params)
		_ = json.Unmarshal(bytes, &stInput)
		res := CalculateStorage(stInput)
		return UniversalResult{ServiceName: "Cloud Storage", TotalMonthlyUSD: res.TotalMonthlyUSD, TotalMonthlyKRW: res.TotalMonthlyKRW, CatalogMatches: matches, Details: res, SavingsTip: res.SavingsTip}

	case "filestore", "nfs":
		var fsInput FilestoreInput
		bytes, _ := json.Marshal(input.Params)
		_ = json.Unmarshal(bytes, &fsInput)
		res := CalculateFilestore(fsInput)
		totalKRW := res.TotalMonthlyCost * GetExchangeRate()
		return UniversalResult{ServiceName: "Filestore", TotalMonthlyUSD: res.TotalMonthlyCost, TotalMonthlyKRW: math.Round(totalKRW), CatalogMatches: matches, Details: res, SavingsTip: res.SavingsTip}

	case "gke", "kubernetes":
		var gkeInput GKEInput
		bytes, _ := json.Marshal(input.Params)
		_ = json.Unmarshal(bytes, &gkeInput)
		res := CalculateGKE(gkeInput)
		return UniversalResult{ServiceName: "GKE", TotalMonthlyUSD: res.TotalMonthlyUSD, TotalMonthlyKRW: res.TotalMonthlyKRW, CatalogMatches: matches, Details: res, SavingsTip: res.SavingsTip}

	case "cloud_sql", "sql":
		var sqlInput CloudSQLInput
		bytes, _ := json.Marshal(input.Params)
		_ = json.Unmarshal(bytes, &sqlInput)
		res := CalculateCloudSQL(sqlInput)
		return UniversalResult{ServiceName: "Cloud SQL", TotalMonthlyUSD: res.TotalMonthlyUSD, TotalMonthlyKRW: res.TotalMonthlyKRW, CatalogMatches: matches, Details: res, SavingsTip: res.SavingsTip}

	case "alloydb":
		var alloyInput AlloyDBInput
		bytes, _ := json.Marshal(input.Params)
		_ = json.Unmarshal(bytes, &alloyInput)
		res := CalculateAlloyDB(alloyInput)
		totalKRW := res.TotalMonthlyCost * GetExchangeRate()
		return UniversalResult{ServiceName: "AlloyDB for PostgreSQL", TotalMonthlyUSD: res.TotalMonthlyCost, TotalMonthlyKRW: math.Round(totalKRW), CatalogMatches: matches, Details: res, SavingsTip: res.SavingsTip}

	case "bigquery", "bq":
		var bqInput BigQueryInput
		bytes, _ := json.Marshal(input.Params)
		_ = json.Unmarshal(bytes, &bqInput)
		res := CalculateBigQuery(bqInput)
		return UniversalResult{ServiceName: "BigQuery", TotalMonthlyUSD: res.TotalMonthlyUSD, TotalMonthlyKRW: res.TotalMonthlyKRW, CatalogMatches: matches, Details: res, SavingsTip: res.SavingsTip}

	case "dataproc", "spark":
		var dpInput DataprocInput
		bytes, _ := json.Marshal(input.Params)
		_ = json.Unmarshal(bytes, &dpInput)
		res := CalculateDataproc(dpInput)
		totalKRW := res.TotalMonthlyCost * GetExchangeRate()
		return UniversalResult{ServiceName: "Dataproc", TotalMonthlyUSD: res.TotalMonthlyCost, TotalMonthlyKRW: math.Round(totalKRW), CatalogMatches: matches, Details: res, SavingsTip: res.SavingsTip}

	case "dataflow", "beam":
		var dfInput DataflowInput
		bytes, _ := json.Marshal(input.Params)
		_ = json.Unmarshal(bytes, &dfInput)
		res := CalculateDataflow(dfInput)
		totalKRW := res.TotalMonthlyCost * GetExchangeRate()
		return UniversalResult{ServiceName: "Dataflow", TotalMonthlyUSD: res.TotalMonthlyCost, TotalMonthlyKRW: math.Round(totalKRW), CatalogMatches: matches, Details: res, SavingsTip: res.SavingsTip}

	case "vertex_ai", "vertex", "gemini":
		var vxInput VertexAIInput
		bytes, _ := json.Marshal(input.Params)
		_ = json.Unmarshal(bytes, &vxInput)
		res := CalculateVertexAI(vxInput)
		totalKRW := res.TotalMonthlyCost * GetExchangeRate()
		return UniversalResult{ServiceName: "Vertex AI", TotalMonthlyUSD: res.TotalMonthlyCost, TotalMonthlyKRW: math.Round(totalKRW), CatalogMatches: matches, Details: res, SavingsTip: res.SavingsTip}
	}

	// Dynamic param-aware evaluation for any catalog product matched from pricing.json
	if len(matches) > 0 {
		rate := matches[0].BaseRateUSD
		unit := matches[0].PricingUnit

		// Extract custom params dynamically
		hours := 730.0
		if h, ok := input.Params["hours_per_month"].(float64); ok && h > 0 {
			hours = h
		}
		qty := 1.0
		if q, ok := input.Params["quantity"].(float64); ok && q > 0 {
			qty = q
		}
		if n, ok := input.Params["nodes"].(float64); ok && n > 0 {
			qty = n
		}

		regMult := 1.0
		if reg, ok := input.Params["region"].(string); ok {
			regMult = GetRegionMultiplier(reg)
		}

		commMult := 1.0
		if comm, ok := input.Params["commitment"].(string); ok {
			commMult = GetCommitmentMultiplier(comm)
		}

		estCostUSD := rate * hours * qty * regMult * commMult
		if strings.Contains(unit, "1M") || strings.Contains(unit, "GB-month") {
			estCostUSD = rate * qty * regMult
		}
		estCostKRW := estCostUSD * GetExchangeRate()

		return UniversalResult{
			ServiceName:     input.ServiceName,
			TotalMonthlyUSD: math.Round(estCostUSD*100) / 100,
			TotalMonthlyKRW: math.Round(estCostKRW),
			CatalogMatches:  matches,
			Details:         map[string]interface{}{"matched_product": matches[0].SubService, "pricing_unit": unit, "base_rate_usd": rate, "region_multiplier": regMult, "commitment_multiplier": commMult},
			SavingsTip:      matches[0].CostOptimizationTip,
		}
	}

	return UniversalResult{
		ServiceName:     svc,
		TotalMonthlyUSD: 0.0,
		TotalMonthlyKRW: 0.0,
		CatalogMatches:  nil,
		Details:         "Service lookup completed across 115 GCP products in pricing.json.",
		SavingsTip:      "Check pricing.json.",
	}
}

type EstimateInput struct {
	ComputeEngine *ComputeEngineInput `json:"compute_engine,omitempty"`
	CloudRun      *CloudRunInput      `json:"cloud_run,omitempty"`
	Storage       *StorageInput       `json:"storage,omitempty"`
}

type EstimateResult struct {
	ComputeEngineCost float64  `json:"compute_engine_cost_usd"`
	CloudRunCost      float64  `json:"cloud_run_cost_usd"`
	StorageCost       float64  `json:"storage_cost_usd"`
	TotalMonthlyUSD   float64  `json:"total_monthly_cost_usd"`
	TotalMonthlyKRW   float64  `json:"total_monthly_cost_krw"`
	Recommendations   []string `json:"recommendations"`
}

func EstimateInfrastructure(input EstimateInput) EstimateResult {
	var ceCost, crCost, stCost float64
	var recs []string

	if input.ComputeEngine != nil {
		ceRes := CalculateComputeEngine(*input.ComputeEngine)
		ceCost = ceRes.TotalMonthlyUSD
		recs = append(recs, ceRes.SavingsTip)
	}
	if input.CloudRun != nil {
		crRes := CalculateCloudRun(*input.CloudRun)
		crCost = crRes.TotalMonthlyUSD
		recs = append(recs, crRes.SavingsTip)
	}
	if input.Storage != nil {
		stRes := CalculateStorage(*input.Storage)
		stCost = stRes.TotalMonthlyUSD
		recs = append(recs, stRes.SavingsTip)
	}

	totalUSD := ceCost + crCost + stCost
	totalKRW := totalUSD * GetExchangeRate()
	return EstimateResult{
		ComputeEngineCost: ceCost,
		CloudRunCost:      crCost,
		StorageCost:       stCost,
		TotalMonthlyUSD:   math.Round(totalUSD*100) / 100,
		TotalMonthlyKRW:   math.Round(totalKRW),
		Recommendations:   recs,
	}
}
