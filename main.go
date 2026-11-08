package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"aws-calculator-mcp/calculator"
	"aws-calculator-mcp/mcp"
)

func main() {
	if len(os.Args) > 1 {
		subcommand := os.Args[1]
		switch subcommand {
		case "ec2", "compute":
			runEC2CLI(os.Args[2:])
			return
		case "rds", "database":
			runRDSCLI(os.Args[2:])
			return
		case "s3", "storage":
			runS3CLI(os.Args[2:])
			return
		case "catalog", "search":
			runCatalogCLI(os.Args[2:])
			return
		case "pricing", "policy":
			runPolicyCLI()
			return
		case "server":
			runMCPServer()
			return
		case "help", "-h", "--help":
			printHelp()
			return
		default:
			fmt.Printf("Unknown command '%s'. Use --help for usage.\n", subcommand)
			os.Exit(1)
		}
	}

	// Default to MCP server mode if no args provided
	runMCPServer()
}

func runMCPServer() {
	server := mcp.NewServer(os.Stdin, os.Stdout)
	if err := server.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Server failed: %v\n", err)
		os.Exit(1)
	}
}

func runEC2CLI(args []string) {
	fs := flag.NewFlagSet("ec2", flag.ExitOnError)
	instanceType := fs.String("instance", "t4g.medium", "EC2 Instance type (e.g., t4g.medium, m6i.xlarge)")
	count := fs.Int("count", 1, "Number of instances")
	hours := fs.Float64("hours", 730, "Monthly running hours")
	ebsGB := fs.Float64("ebs", 50, "EBS storage in GB")
	isSpot := fs.Bool("spot", false, "Apply Spot discount")
	fs.Parse(args)

	// Calculation using active policy rates
	policy := calculator.GetActivePolicy()
	rate := 0.0416 // default t4g.medium
	for _, p := range policy.Products {
		if p.SubService == *instanceType && p.BaseRateUSD > 0 {
			rate = p.BaseRateUSD
			break
		}
	}
	if *isSpot {
		rate *= 0.35 // spot discount
	}

	computeUSD := rate * (*hours) * float64(*count)
	ebsUSD := (*ebsGB) * 0.096 * float64(*count)
	totalUSD := computeUSD + ebsUSD
	krwRate := policy.CurrencyExchange["USD_TO_KRW"]
	if krwRate == 0 {
		krwRate = 1350.0
	}

	res := map[string]interface{}{
		"service":                "Amazon EC2 & EBS",
		"instance_type":          *instanceType,
		"instance_count":         *count,
		"hours_per_month":        *hours,
		"ebs_gb":                 *ebsGB,
		"is_spot":                *isSpot,
		"monthly_cost_usd":       totalUSD,
		"monthly_cost_krw":       int(totalUSD * krwRate),
		"exchange_rate_usd_krw":  krwRate,
		"cost_optimization_tip": "Use Compute Savings Plans or Spot instances for up to 70% savings",
	}
	printJSON(res)
}

func runRDSCLI(args []string) {
	fs := flag.NewFlagSet("rds", flag.ExitOnError)
	instance := fs.String("instance", "db.t4g.medium", "RDS DB instance type")
	storageGB := fs.Float64("storage", 100, "Allocated storage GB")
	multiAZ := fs.Bool("multi-az", false, "Enable Multi-AZ deployment")
	fs.Parse(args)

	policy := calculator.GetActivePolicy()
	baseUSD := 0.068 * 730.0 // db.t4g.medium single AZ approx
	if *multiAZ {
		baseUSD *= 2.0
	}
	storageUSD := (*storageGB) * 0.115
	totalUSD := baseUSD + storageUSD
	krwRate := policy.CurrencyExchange["USD_TO_KRW"]
	if krwRate == 0 {
		krwRate = 1350.0
	}

	res := map[string]interface{}{
		"service":               "Amazon RDS",
		"db_instance":           *instance,
		"multi_az":              *multiAZ,
		"storage_gb":            *storageGB,
		"monthly_cost_usd":      totalUSD,
		"monthly_cost_krw":      int(totalUSD * krwRate),
		"exchange_rate_usd_krw": krwRate,
	}
	printJSON(res)
}

func runS3CLI(args []string) {
	fs := flag.NewFlagSet("s3", flag.ExitOnError)
	gb := fs.Float64("gb", 1000, "Total storage in GB")
	class := fs.String("class", "standard", "standard, glacier, deep-archive")
	fs.Parse(args)

	policy := calculator.GetActivePolicy()
	rate := 0.025
	switch *class {
	case "glacier":
		rate = 0.0045
	case "deep-archive":
		rate = 0.0020
	}

	totalUSD := (*gb) * rate
	krwRate := policy.CurrencyExchange["USD_TO_KRW"]
	if krwRate == 0 {
		krwRate = 1350.0
	}

	res := map[string]interface{}{
		"service":               "Amazon S3",
		"storage_class":         *class,
		"storage_gb":            *gb,
		"monthly_cost_usd":      totalUSD,
		"monthly_cost_krw":      int(totalUSD * krwRate),
		"exchange_rate_usd_krw": krwRate,
	}
	printJSON(res)
}

func runCatalogCLI(args []string) {
	fs := flag.NewFlagSet("catalog", flag.ExitOnError)
	query := fs.String("query", "", "Filter query")
	fs.Parse(args)

	products := calculator.GetFullCatalog()
	if *query != "" {
		products = calculator.SearchCatalog(*query)
	}
	printJSON(products)
}

func runPolicyCLI() {
	policy := calculator.GetActivePolicy()
	printJSON(policy)
}

func printJSON(v interface{}) {
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(b))
}

func printHelp() {
	helpText := `AWS Pricing Calculator MCP Server & CLI Tool (Golang)

Usage:
  aws-calculator-mcp [command] [flags]

Commands:
  server      Run as MCP stdio server (default)
  ec2         Calculate EC2 Compute costs (Instances, EBS, Spot)
  rds         Calculate RDS Database costs
  s3          Calculate S3 Storage costs
  catalog     Search products in catalog
  pricing     Display active pricing policy rates
`
	fmt.Println(helpText)
}
