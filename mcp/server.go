package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"aws-calculator-mcp/calculator"
)

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Server struct {
	reader *bufio.Reader
	writer io.Writer
}

func NewServer(r io.Reader, w io.Writer) *Server {
	return &Server{
		reader: bufio.NewReader(r),
		writer: w,
	}
}

func (s *Server) Start() error {
	for {
		line, err := s.reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if len(line) == 0 {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			s.sendError(nil, -32700, "Parse error")
			continue
		}

		s.handleRequest(&req)
	}
}

func (s *Server) handleRequest(req *JSONRPCRequest) {
	switch req.Method {
	case "initialize":
		s.sendResult(req.ID, map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "aws-calculator-mcp",
				"version": "2.0.0",
			},
		})
	case "notifications/initialized":
	case "ping":
		s.sendResult(req.ID, map[string]interface{}{})
	case "tools/list":
		s.handleToolsList(req.ID)
	case "tools/call":
		s.handleToolCall(req.ID, req.Params)
	default:
		if req.ID != nil {
			s.sendError(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
		}
	}
}

func (s *Server) handleToolsList(id interface{}) {
	tools := []map[string]interface{}{
		{
			"name":        "calculate_compute_engine",
			"description": "Calculate monthly cost for GCP Compute Engine VM instances, Spot discounts, GPUs (NVIDIA T4, L4, V100, A100, H100), Cloud TPUs (v2, v3, v4, v5e, v5p), and persistent disks.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"machine_type":     map[string]interface{}{"type": "string", "default": "e2-micro"},
					"region":           map[string]interface{}{"type": "string", "default": "us-central1", "description": "e.g. us-central1, us-west1, asia-northeast3 (Seoul)"},
					"commitment":       map[string]interface{}{"type": "string", "default": "none", "description": "none, 1-year, 3-year"},
					"hours_per_month":  map[string]interface{}{"type": "number", "default": 730},
					"is_spot":          map[string]interface{}{"type": "boolean", "default": false},
					"gpu_type":         map[string]interface{}{"type": "string", "description": "e.g. nvidia-tesla-t4, nvidia-l4, nvidia-tesla-a100-80gb, nvidia-h100-80gb"},
					"gpu_count":        map[string]interface{}{"type": "number", "default": 0},
					"tpu_type":         map[string]interface{}{"type": "string", "description": "e.g. tpu-v2-8, tpu-v3-8, tpu-v4-pod-8, tpu-v5litepod-8, tpu-v5p-8"},
					"tpu_count":        map[string]interface{}{"type": "number", "default": 0},
					"storage_gb":       map[string]interface{}{"type": "number", "default": 20},
					"storage_type":     map[string]interface{}{"type": "string", "default": "standard", "description": "standard, balanced, ssd, extreme"},
					"provisioned_iops": map[string]interface{}{"type": "number", "default": 0, "description": "Provisioned IOPS for Extreme PD or Hyperdisk"},
				},
			},
		},
		{
			"name":        "calculate_cloud_run",
			"description": "Calculate monthly cost for GCP Cloud Run serverless application with free tier.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"requests_per_month": map[string]interface{}{"type": "number", "default": 1000000},
					"cpu_cores":          map[string]interface{}{"type": "number", "default": 1},
					"memory_gb":          map[string]interface{}{"type": "number", "default": 0.5},
					"avg_duration_ms":    map[string]interface{}{"type": "number", "default": 200},
					"concurrency":        map[string]interface{}{"type": "number", "default": 80},
				},
			},
		},
		{
			"name":        "calculate_storage",
			"description": "Calculate monthly cost for GCP Cloud Storage and network egress.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"storage_gb":        map[string]interface{}{"type": "number", "default": 50},
					"storage_class":     map[string]interface{}{"type": "string", "default": "standard"},
					"network_egress_gb": map[string]interface{}{"type": "number", "default": 5},
				},
			},
		},
		{
			"name":        "calculate_gke",
			"description": "Calculate monthly cost for Google Kubernetes Engine (GKE Autopilot / Standard).",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"mode":            map[string]interface{}{"type": "string", "description": "'autopilot' or 'standard'", "default": "autopilot"},
					"node_count":      map[string]interface{}{"type": "number", "default": 3},
					"vcpus_per_node":  map[string]interface{}{"type": "number", "default": 2},
					"mem_gb_per_node": map[string]interface{}{"type": "number", "default": 4},
				},
			},
		},
		{
			"name":        "calculate_cloud_sql",
			"description": "Calculate monthly cost for GCP Cloud SQL relational database.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"instance_type":     map[string]interface{}{"type": "string", "default": "db-f1-micro"},
					"storage_gb":        map[string]interface{}{"type": "number", "default": 10},
					"high_availability": map[string]interface{}{"type": "boolean", "default": false},
				},
			},
		},
		{
			"name":        "calculate_bigquery",
			"description": "Calculate monthly cost for BigQuery data warehouse queries and storage.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query_tb_scanned":  map[string]interface{}{"type": "number", "default": 2.0},
					"active_storage_gb": map[string]interface{}{"type": "number", "default": 50.0},
				},
			},
		},
		{
			"name":        "calculate_gcp_service",
			"description": "Universal GCP Product Cost Calculator for any GCP service (compute, cloud_run, storage, gke, cloud_sql, bigquery, pubsub, functions, vertex_ai, memorystore, spanner, bigtable).",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"service_name": map[string]interface{}{"type": "string", "description": "Target GCP product name"},
					"params":       map[string]interface{}{"type": "object", "description": "Service parameters"},
				},
				"required": []string{"service_name"},
			},
		},
		{
			"name":        "estimate_infrastructure",
			"description": "Comprehensive multi-product GCP infrastructure cost estimate & optimization recommendations.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"compute_engine": map[string]interface{}{"type": "object"},
					"cloud_run":      map[string]interface{}{"type": "object"},
					"storage":        map[string]interface{}{"type": "object"},
				},
			},
		},
	}

	s.sendResult(id, map[string]interface{}{
		"tools": tools,
	})
}

func (s *Server) handleToolCall(id interface{}, paramsRaw json.RawMessage) {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(paramsRaw, &params); err != nil {
		s.sendError(id, -32602, "Invalid params")
		return
	}

	var textResult string

	switch params.Name {
	case "calculate_compute_engine":
		var input calculator.ComputeEngineInput
		if len(params.Arguments) > 0 {
			_ = json.Unmarshal(params.Arguments, &input)
		}
		res := calculator.CalculateComputeEngine(input)
		bytes, _ := json.MarshalIndent(res, "", "  ")
		textResult = string(bytes)

	case "calculate_cloud_run":
		var input calculator.CloudRunInput
		if len(params.Arguments) > 0 {
			_ = json.Unmarshal(params.Arguments, &input)
		}
		res := calculator.CalculateCloudRun(input)
		bytes, _ := json.MarshalIndent(res, "", "  ")
		textResult = string(bytes)

	case "calculate_storage":
		var input calculator.StorageInput
		if len(params.Arguments) > 0 {
			_ = json.Unmarshal(params.Arguments, &input)
		}
		res := calculator.CalculateStorage(input)
		bytes, _ := json.MarshalIndent(res, "", "  ")
		textResult = string(bytes)

	case "calculate_gke":
		var input calculator.GKEInput
		if len(params.Arguments) > 0 {
			_ = json.Unmarshal(params.Arguments, &input)
		}
		res := calculator.CalculateGKE(input)
		bytes, _ := json.MarshalIndent(res, "", "  ")
		textResult = string(bytes)

	case "calculate_cloud_sql":
		var input calculator.CloudSQLInput
		if len(params.Arguments) > 0 {
			_ = json.Unmarshal(params.Arguments, &input)
		}
		res := calculator.CalculateCloudSQL(input)
		bytes, _ := json.MarshalIndent(res, "", "  ")
		textResult = string(bytes)

	case "calculate_bigquery":
		var input calculator.BigQueryInput
		if len(params.Arguments) > 0 {
			_ = json.Unmarshal(params.Arguments, &input)
		}
		res := calculator.CalculateBigQuery(input)
		bytes, _ := json.MarshalIndent(res, "", "  ")
		textResult = string(bytes)

	case "calculate_gcp_service":
		var input calculator.UniversalInput
		if len(params.Arguments) > 0 {
			_ = json.Unmarshal(params.Arguments, &input)
		}
		res := calculator.CalculateService(input)
		bytes, _ := json.MarshalIndent(res, "", "  ")
		textResult = string(bytes)

	case "estimate_infrastructure":
		var input calculator.EstimateInput
		if len(params.Arguments) > 0 {
			_ = json.Unmarshal(params.Arguments, &input)
		}
		res := calculator.EstimateInfrastructure(input)
		bytes, _ := json.MarshalIndent(res, "", "  ")
		textResult = string(bytes)

	default:
		s.sendError(id, -32601, fmt.Sprintf("Tool not found: %s", params.Name))
		return
	}

	s.sendResult(id, map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": textResult,
			},
		},
	})
}

func (s *Server) sendResult(id interface{}, result interface{}) {
	resp := JSONRPCResponse{JSONRPC: "2.0", ID: id, Result: result}
	s.sendResponse(resp)
}

func (s *Server) sendError(id interface{}, code int, message string) {
	resp := JSONRPCResponse{JSONRPC: "2.0", ID: id, Error: &RPCError{Code: code, Message: message}}
	s.sendResponse(resp)
}

func (s *Server) sendResponse(resp JSONRPCResponse) {
	bytes, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling response: %v\n", err)
		return
	}
	bytes = append(bytes, '\n')
	_, _ = s.writer.Write(bytes)
}
