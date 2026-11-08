package calculator

import (
	"math"
	"strings"
)

// AppEngineInput holds inputs for App Engine calculations.
type AppEngineInput struct {
	Env           string  `json:"env"`            // "standard", "flexible"
	InstanceClass string  `json:"instance_class"` // e.g. "F1", "F2", "B1", "B2"
	InstanceCount int     `json:"instance_count"`
	HoursPerMonth float64 `json:"hours_per_month"`
}

type AppEngineResult struct {
	Env              string  `json:"env"`
	InstanceClass    string  `json:"instance_class"`
	TotalMonthlyCost float64 `json:"total_monthly_cost_usd"`
	SavingsTip       string  `json:"savings_tip"`
}

func CalculateAppEngine(input AppEngineInput) AppEngineResult {
	if input.HoursPerMonth <= 0 {
		input.HoursPerMonth = 730
	}
	if input.InstanceCount <= 0 {
		input.InstanceCount = 1
	}

	hourlyRate := 0.06 // F1 base rate
	switch strings.ToUpper(input.InstanceClass) {
	case "F2", "B2":
		hourlyRate = 0.12
	case "F4", "B4":
		hourlyRate = 0.24
	case "F4_1G":
		hourlyRate = 0.30
	}
	if strings.ToLower(input.Env) == "flexible" {
		hourlyRate = 0.135 // Custom vCPU/RAM flex base
	}

	total := hourlyRate * float64(input.InstanceCount) * input.HoursPerMonth
	return AppEngineResult{
		Env:              input.Env,
		InstanceClass:    input.InstanceClass,
		TotalMonthlyCost: math.Round(total*100) / 100,
		SavingsTip:       "Standard Environment includes 28 free instance-hours per day for F1 instances.",
	}
}

// FilestoreInput holds inputs for Filestore calculations.
type FilestoreInput struct {
	Tier       string  `json:"tier"` // "basic_hdd", "basic_ssd", "high_scale", "enterprise"
	CapacityTB float64 `json:"capacity_tb"`
}

type FilestoreResult struct {
	Tier             string  `json:"tier"`
	TotalMonthlyCost float64 `json:"total_monthly_cost_usd"`
	SavingsTip       string  `json:"savings_tip"`
}

func CalculateFilestore(input FilestoreInput) FilestoreResult {
	if input.CapacityTB <= 0 {
		input.CapacityTB = 1.0
	}

	ratePerGB := 0.16 // basic_hdd per GB-mo ($160/TB)
	switch strings.ToLower(input.Tier) {
	case "basic_ssd":
		ratePerGB = 0.24
	case "high_scale", "highscale":
		ratePerGB = 0.30
	case "enterprise":
		ratePerGB = 0.36
	}

	totalGB := input.CapacityTB * 1024
	total := totalGB * ratePerGB
	return FilestoreResult{
		Tier:             input.Tier,
		TotalMonthlyCost: math.Round(total*100) / 100,
		SavingsTip:       "Basic HDD Filestore is optimal for media sharing and development NFS shares.",
	}
}

// AlloyDBInput holds inputs for AlloyDB calculations.
type AlloyDBInput struct {
	VCPUs         int     `json:"vcpus"`
	MemoryGB      float64 `json:"memory_gb"`
	StorageGB     float64 `json:"storage_gb"`
	HoursPerMonth float64 `json:"hours_per_month"`
}

type AlloyDBResult struct {
	ComputeCost      float64 `json:"compute_cost_usd"`
	StorageCost      float64 `json:"storage_cost_usd"`
	TotalMonthlyCost float64 `json:"total_monthly_cost_usd"`
	SavingsTip       string  `json:"savings_tip"`
}

func CalculateAlloyDB(input AlloyDBInput) AlloyDBResult {
	if input.VCPUs <= 0 {
		input.VCPUs = 2
	}
	if input.MemoryGB <= 0 {
		input.MemoryGB = 16
	}
	if input.HoursPerMonth <= 0 {
		input.HoursPerMonth = 730
	}

	vcpuRate := 0.0544  // per vCPU-hour
	memRate := 0.00942  // per GB-hour
	storageRate := 0.30 // per GB-month

	computeCost := (float64(input.VCPUs)*vcpuRate + input.MemoryGB*memRate) * input.HoursPerMonth
	storageCost := input.StorageGB * storageRate
	total := computeCost + storageCost

	return AlloyDBResult{
		ComputeCost:      math.Round(computeCost*100) / 100,
		StorageCost:      math.Round(storageCost*100) / 100,
		TotalMonthlyCost: math.Round(total*100) / 100,
		SavingsTip:       "AlloyDB columnar engine accelerates analytical SQL without separate ETL.",
	}
}

// DataprocInput holds inputs for Dataproc calculations.
type DataprocInput struct {
	Mode          string  `json:"mode"` // "compute", "serverless"
	WorkerNodes   int     `json:"worker_nodes"`
	HoursPerMonth float64 `json:"hours_per_month"`
}

type DataprocResult struct {
	Mode             string  `json:"mode"`
	TotalMonthlyCost float64 `json:"total_monthly_cost_usd"`
	SavingsTip       string  `json:"savings_tip"`
}

func CalculateDataproc(input DataprocInput) DataprocResult {
	if input.HoursPerMonth <= 0 {
		input.HoursPerMonth = 730
	}
	if input.WorkerNodes <= 0 {
		input.WorkerNodes = 2
	}

	dataprocPremium := 0.01 // $0.01 per vCPU hour premium over Compute Engine
	ceCost := CalculateComputeEngine(ComputeEngineInput{
		MachineType:   "n1-standard-4",
		HoursPerMonth: input.HoursPerMonth,
	}).TotalMonthlyUSD

	total := (ceCost + (dataprocPremium * 4 * input.HoursPerMonth)) * float64(input.WorkerNodes)
	if strings.ToLower(input.Mode) == "serverless" {
		total = 0.20 * 4 * input.HoursPerMonth // Serverless Spark vCPU/hour rate
	}

	return DataprocResult{
		Mode:             input.Mode,
		TotalMonthlyCost: math.Round(total*100) / 100,
		SavingsTip:       "Dataproc Serverless eliminates cluster provisioning and idle worker node costs.",
	}
}

// DataflowInput holds inputs for Dataflow calculations.
type DataflowInput struct {
	JobType       string  `json:"job_type"` // "batch", "streaming"
	VCPUs         float64 `json:"vcpus"`
	MemoryGB      float64 `json:"memory_gb"`
	HoursPerMonth float64 `json:"hours_per_month"`
}

type DataflowResult struct {
	JobType          string  `json:"job_type"`
	TotalMonthlyCost float64 `json:"total_monthly_cost_usd"`
	SavingsTip       string  `json:"savings_tip"`
}

func CalculateDataflow(input DataflowInput) DataflowResult {
	if input.HoursPerMonth <= 0 {
		input.HoursPerMonth = 730
	}
	if input.VCPUs <= 0 {
		input.VCPUs = 2
	}
	if input.MemoryGB <= 0 {
		input.MemoryGB = 8
	}

	vcpuRate := 0.056 // Batch vCPU per hour
	memRate := 0.00355
	if strings.ToLower(input.JobType) == "streaming" {
		vcpuRate = 0.069
		memRate = 0.00455
	}

	total := (input.VCPUs*vcpuRate + input.MemoryGB*memRate) * input.HoursPerMonth
	return DataflowResult{
		JobType:          input.JobType,
		TotalMonthlyCost: math.Round(total*100) / 100,
		SavingsTip:       "Dataflow Prime autoscales compute and memory independently to reduce streaming cost.",
	}
}

// VertexAIInput holds inputs for Vertex AI calculations.
type VertexAIInput struct {
	TaskType       string  `json:"task_type"` // "gemini_flash", "gemini_pro", "custom_training", "search_rag"
	InputTokens1k  float64 `json:"input_tokens_1k"`
	OutputTokens1k float64 `json:"output_tokens_1k"`
	NodeHours      float64 `json:"node_hours"`
}

type VertexAIResult struct {
	TaskType         string  `json:"task_type"`
	TotalMonthlyCost float64 `json:"total_monthly_cost_usd"`
	SavingsTip       string  `json:"savings_tip"`
}

func CalculateVertexAI(input VertexAIInput) VertexAIResult {
	var total float64

	switch strings.ToLower(input.TaskType) {
	case "gemini_flash", "flash":
		inputCost := input.InputTokens1k * 0.000075
		outputCost := input.OutputTokens1k * 0.00030
		total = inputCost + outputCost
	case "gemini_pro", "pro":
		inputCost := input.InputTokens1k * 0.00125
		outputCost := input.OutputTokens1k * 0.00375
		total = inputCost + outputCost
	default:
		if input.NodeHours <= 0 {
			input.NodeHours = 100
		}
		total = input.NodeHours * 0.22
	}

	return VertexAIResult{
		TaskType:         input.TaskType,
		TotalMonthlyCost: math.Round(total*100) / 100,
		SavingsTip:       "Use Gemini 1.5 Flash for high-throughput batch text and multimodal processing.",
	}
}
