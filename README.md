# AWS Pricing Calculator MCP Server & CLI (Golang)

[![Go Version](https://img.shields.io/badge/Go-1.24%2B-00ADD8?logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![MCP Compatible](https://img.shields.io/badge/MCP-2024--11--05-orange.svg)](https://modelcontextprotocol.io)

AWS Pricing Calculator MCP Server & CLI engineered in Go, structured exactly like [mireu-workflow/GCP-Calculator-MCP](https://github.com/mireu-workflow/GCP-Calculator-MCP).
It loads the comprehensive 75,000+ records from `AWS-Comprehensive-Catalog.tsv` and `pricing.json` covering **Amazon EC2**, **Amazon RDS**, and **Amazon S3** (Seoul `ap-northeast-2` and global regions).

---

## ✨ Key Features

- **📊 Comprehensive AWS Catalog**: 75,000+ raw entries extracted via streaming Bulk Price List API.
- **💵 Dual Currency Support (USD & KRW)**: Real-time dual valuation aligned with standard exchange rates.
- **📉 Spot & Savings Plan Modeling**: Automated Spot discount (up to 70%) and multi-AZ database cost calculation.
- **⚡ Stdio MCP Server**: Full Model Context Protocol server for Antigravity, Claude Desktop, Cursor, and Zed.

---

## 💻 CLI Usage & Examples

```bash
# 1. EC2 Compute (Instance + 100GB EBS + Spot discount)
./bin/aws-calculator-mcp ec2 -instance t4g.medium -ebs 100 -spot

# 2. RDS Managed Database (Multi-AZ + 100GB Storage)
./bin/aws-calculator-mcp rds -instance db.t4g.medium -storage 100 -multi-az

# 3. S3 Object Storage (Glacier Deep Archive 1TB)
./bin/aws-calculator-mcp s3 -gb 1000 -class deep-archive

# 4. Search Catalog
./bin/aws-calculator-mcp catalog -query "c6i"

# 5. Launch Stdio MCP Server
./bin/aws-calculator-mcp server
```

---

## 👥 Governance
* **Project Manager**: limmireu1214@gmail.com
* **Implementation Owner**: limmireu1214@gmail.com
