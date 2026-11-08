package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// Price List Bulk API 리전별 엔드포인트 기본 경로
const baseURL = "https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws"

// 추출 대상 서비스 목록
var targetServices = []string{"AmazonEC2", "AmazonRDS", "AmazonS3"}

// 정규화된 카탈로그 레코드 구조체
type CatalogRecord struct {
	SKU             string
	ServiceCode     string
	ProductFamily   string
	RegionCode      string
	InstanceType    string
	VCPU            string
	MemoryGiB       string
	StorageType     string
	UsageType       string
	Operation       string
	PurchaseOption  string
	Unit            string
	PricePerUnitUSD string
	Description     string
}

// AWS Bulk Price List 내부 JSON 구조 매핑
type BulkOfferPayload struct {
	FormatVersion string                 `json:"formatVersion"`
	Disclaimer    string                 `json:"disclaimer"`
	OfferCode     string                 `json:"offerCode"`
	Products      map[string]ProductItem `json:"products"`
	Terms         TermsContainer         `json:"terms"`
}

type ProductItem struct {
	SKU           string                 `json:"sku"`
	ProductFamily string                 `json:"productFamily"`
	Attributes    map[string]interface{} `json:"attributes"`
}

type TermsContainer struct {
	OnDemand map[string]map[string]TermOffer `json:"OnDemand"`
}

type TermOffer struct {
	OfferTermCode   string                    `json:"offerTermCode"`
	SKU             string                    `json:"sku"`
	PriceDimensions map[string]PriceDimension `json:"priceDimensions"`
}

type PriceDimension struct {
	RateCode     string            `json:"rateCode"`
	Description  string            `json:"description"`
	BeginRange   string            `json:"beginRange"`
	EndRange     string            `json:"endRange"`
	Unit         string            `json:"unit"`
	PricePerUnit map[string]string `json:"pricePerUnit"`
}

func main() {
	outputFilename := "AWS-Comprehensive-Catalog.tsv"
	region := "ap-northeast-2" // 서울 리전 기본 설정

	outFile, err := os.Create(outputFilename)
	if err != nil {
		log.Fatalf("출력 파일 생성 실패: %v", err)
	}
	defer outFile.Close()

	tsvWriter := csv.NewWriter(outFile)
	tsvWriter.Comma = '\t'
	defer tsvWriter.Flush()

	// TSV 헤더 정의
	headers := []string{
		"sku", "service_code", "product_family", "region_code",
		"instance_type", "vcpu", "memory_gib", "storage_type",
		"usage_type", "operation", "purchase_option", "unit",
		"price_per_unit_usd", "description",
	}
	if err := tsvWriter.Write(headers); err != nil {
		log.Fatalf("헤더 작성 실패: %v", err)
	}

	client := &http.Client{Timeout: 10 * time.Minute}

	for _, service := range targetServices {
		log.Printf("[%s] 리전(%s) 요금 데이터 다운로드 및 파싱 시작...", service, region)
		targetURL := fmt.Sprintf("%s/%s/current/%s/index.json", baseURL, service, region)

		resp, err := client.Get(targetURL)
		if err != nil {
			log.Printf("[%s] 요청 실패: %v", service, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			log.Printf("[%s] 잘못된 응답 코드: %d", service, resp.StatusCode)
			continue
		}

		records, err := processOfferPayload(resp.Body, service, region)
		resp.Body.Close()
		if err != nil {
			log.Printf("[%s] 데이터 파싱 실패: %v", service, err)
			continue
		}

		log.Printf("[%s] %d개 레코드 추출 완료. TSV 기록 중...", service, len(records))
		for _, r := range records {
			row := []string{
				r.SKU, r.ServiceCode, r.ProductFamily, r.RegionCode,
				r.InstanceType, r.VCPU, r.MemoryGiB, r.StorageType,
				r.UsageType, r.Operation, r.PurchaseOption, r.Unit,
				r.PricePerUnitUSD, r.Description,
			}
			if err := tsvWriter.Write(row); err != nil {
				log.Printf("행 기록 실패 (SKU: %s): %v", r.SKU, err)
			}
		}
		tsvWriter.Flush()
	}

	log.Printf("전체 카탈로그 생성 완료: %s", outputFilename)
}

func processOfferPayload(reader io.Reader, serviceCode, defaultRegion string) ([]CatalogRecord, error) {
	var payload BulkOfferPayload
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&payload); err != nil {
		return nil, err
	}

	var results []CatalogRecord

	// OnDemand 요금 매핑
	for sku, termMap := range payload.Terms.OnDemand {
		prod, exists := payload.Products[sku]
		if !exists {
			continue
		}

		attr := prod.Attributes
		regionCode := getStringAttr(attr, "regionCode")
		if regionCode == "" {
			regionCode = defaultRegion
		}

		instanceType := getStringAttr(attr, "instanceType")
		vcpu := getStringAttr(attr, "vcpu")
		memory := getStringAttr(attr, "memory")
		storageType := getStringAttr(attr, "volumeType")
		if storageType == "" {
			storageType = getStringAttr(attr, "storageClass")
		}
		usageType := getStringAttr(attr, "usagetype")
		operation := getStringAttr(attr, "operation")

		for _, term := range termMap {
			for _, dim := range term.PriceDimensions {
				priceUSD := dim.PricePerUnit["USD"]
				if priceUSD == "" {
					continue
				}

				// 정규화된 레코드 생성
				record := CatalogRecord{
					SKU:             sku,
					ServiceCode:     serviceCode,
					ProductFamily:   prod.ProductFamily,
					RegionCode:      regionCode,
					InstanceType:    instanceType,
					VCPU:            vcpu,
					MemoryGiB:       normalizeMemory(memory),
					StorageType:     storageType,
					UsageType:       usageType,
					Operation:       operation,
					PurchaseOption:  "OnDemand",
					Unit:            dim.Unit,
					PricePerUnitUSD: priceUSD,
					Description:     strings.TrimSpace(dim.Description),
				}
				results = append(results, record)
			}
		}
	}

	return results, nil
}

func getStringAttr(attrs map[string]interface{}, key string) string {
	if val, ok := attrs[key]; ok && val != nil {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

func normalizeMemory(memStr string) string {
	memStr = strings.TrimSpace(memStr)
	memStr = strings.ReplaceAll(memStr, "GiB", "")
	memStr = strings.ReplaceAll(memStr, "GB", "")
	return strings.TrimSpace(memStr)
}
