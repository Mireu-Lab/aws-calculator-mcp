# AWS Calculator & Catalog Engine (Golang)

[![Go Version](https://img.shields.io/badge/go-1.24%2B-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

High-performance AWS Pricing Catalog ETL and Calculator Engine for Amazon EC2, Amazon RDS, and Amazon S3.

## 🚀 Overview

- **Direct AWS Price List Bulk API Integration**: Streaming parser via `json.Decoder` preventing OOM.
- **Seoul Region (`ap-northeast-2`) Ready**: 75,000+ extracted pricing records.
- **Output Dataset**: `AWS-Comprehensive-Catalog.tsv` (17MB normalized schema).

## 🛠 Usage

```bash
go run main.go
```

## 👥 Governance
* **Project Manager**: limmireu1214@gmail.com
* **Implementation Owner**: limmireu1214@gmail.com
