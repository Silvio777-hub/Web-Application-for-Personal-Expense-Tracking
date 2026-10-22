# Personal Expense Tracker (Zero-Trust Financial Platform)

> **Author:** Opoka John Modi Silvio  
> **Course:** Distributed Programming for Web, IoT, and Mobile Systems  
> **Degree:** MSc Software Science and Technology, Università degli Studi di Firenze  
> **Academic Year:** 2025–2026  
> **Repository:** [https://github.com/Silvio777-hub/Web-Application-for-Personal-Expense-Tracking](https://github.com/Silvio777-hub/Web-Application-for-Personal-Expense-Tracking)

[![Go Version](https://img.shields.io/badge/Go-1.24-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Router](https://img.shields.io/badge/Router-Chi_v5-black?style=flat)](https://github.com/go-chi/chi)
[![Database](https://img.shields.io/badge/Database-SQLite3_WAL-003B57?style=flat&logo=sqlite)](https://sqlite.org/)
[![Security](https://img.shields.io/badge/Security-AES--GCM_%2B_Zero--Trust-red?style=flat)](README.md)
[![PWA](https://img.shields.io/badge/PWA-Offline--First-5A0FC8?style=flat)](README.md)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

---

## 📌 Executive Summary

Commercial financial tools frequently monetize user metadata and rely on centralized, third-party cloud infrastructure. The **Personal Expense Tracker** is a production-grade web application engineered on a **Zero-Trust Architecture**. It offers complete privacy, local-first data ownership, bank-grade encryption-at-rest, and seamless offline capability via Progressive Web App (PWA) standards.

Engineered with **Go 1.24** and **SQLite 3 in WAL (Write-Ahead Logging) mode**, the system provides deterministic performance with zero CGO dependencies, sub-millisecond response times, and robust cryptographic isolation.

---

## 🌟 Key Architecture & Security Innovations

### 1. Zero-Trust Security Module
* **AES-256-GCM Encryption-at-Rest:** Sensitive records, tokens, and financial artifacts are encrypted using Galois/Counter Mode (AES-GCM), guaranteeing confidentiality and ciphertext authenticity.
* **Constant-Time Authentication:** Defeats username enumeration and side-channel timing attacks by normalizing password evaluation latency across valid and invalid identities.
* **Deep Magic-Byte Inspection:** File uploads (receipts and invoices) undergo strict MIME verification using `http.DetectContentType` to prevent polyglot attacks and malicious payload masquerading.
* **Strict Header Hardening & CSRF:** Enforces modern security headers (`Content-Security-Policy`, `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`) alongside cryptographically bound CSRF tokens and `HttpOnly`, `SameSite=Strict` cookie policies.

### 2. High-Durability Disaster Recovery
* **Atomic Hot Backups:** Utilizes SQLite's native `VACUUM INTO` engine to capture transactional consistency snapshots on live production workloads without read/write locks or service interruptions.
* **CSV Injection Shield:** All tabular exports neutralize formula injection vectors (e.g., prepending single quotes to `=, +, -, @` characters) to safeguard users against spreadsheet execution exploits.
* **WAL Concurrency:** SQLite in WAL mode allows multiple concurrent readers alongside an active writer thread, delivering scalable local throughput.

### 3. PWA & Resilient User Experience
* **Offline-First Cache:** Service worker implementation allows application loading, UI navigation, and cached asset availability under degraded connectivity.
* **5-Second Undo Buffer:** Deletion operations utilize a transactional buffer window, preventing accidental data loss through user error.
* **PII-Sanitized Audit Trail:** Complete audit logging with automated redaction of personally identifiable information (PII) before ingestion to persistent loggers.

---

## 🏗 Directory & Package Layout

```
expense-tracker/
├── cmd/
│   └── (entrypoint bootstraps configuration and HTTP listeners)
├── internal/
│   ├── api/                # Chi v5 routes, controllers, and REST handlers
│   ├── config/             # Environment variable parsing and validation
│   ├── database/           # SQLite connection lifecycle, pragma tuning, and migrations
│   ├── logger/             # Structured, PII-sanitized logger
│   ├── mail/               # Asynchronous mail notifications
│   ├── middleware/         # Security headers, rate limiting, auth, CSRF, and cache controls
│   ├── models/             # Domain entities (User, Expense, Category, Audit, Debt)
│   ├── ocr/                # Tesseract OCR engine integration for receipt processing
│   ├── repository/         # Repository pattern contracts
│   │   └── sqlite/         # SQL implementations (expenses, users, categories, debts, audits)
│   ├── service/            # Core business domain logic
│   └── utils/              # Cryptographic primitives, key providers, and validators
├── public/                 # Static assets (ES6+ modules, CSS, PWA manifests, icons)
├── templates/              # Go HTML templates (SSR layout and dynamic components)
├── uploads/                # Isolated storage for validated receipts and profiles
├── .env.example            # Sanitized configuration template
├── .gitignore              # Ignores databases, binaries, and local credentials
├── go.mod / go.sum         # Dependency definitions
├── main.go                 # Main server orchestration
└── README.md               # Technical documentation
```

---

## 🚀 Quick Start Guide

### Prerequisites
* **Go 1.24+** installed: [https://go.dev/dl/](https://go.dev/dl/)
* Git

### 1. Clone & Setup Configuration
```bash
git clone https://github.com/Silvio777-hub/Web-Application-for-Personal-Expense-Tracking.git
cd Web-Application-for-Personal-Expense-Tracking

# Copy environment template
cp .env.example .env
```

### 2. Configure Environment (`.env`)
Generate a secure 32-byte key for AES-256 and set your JWT secret:
```env
PORT=:8080
APP_BASE_URL=http://localhost:8080
DB_PATH=expense.db
JWT_SECRET=generate-a-strong-secret-key-here
DEFAULT_CURRENCY=EUR
MASTER_KEY=your-32-byte-hex-key-here
```

### 3. Run the Application
```bash
# Download dependencies
go mod download

# Launch server
go run main.go
```
The server will initialize the SQLite schema automatically and bind to **`http://localhost:8080`**.

---

## 🔑 Authentication & Roles

| Role | Default Username | Default Password | Privileges |
|---|---|---|---|
| **Administrator** | `admin` | `admin` | Full system audit logs, database hot-backup triggers, user management, system metrics. |
| **Standard User** | *(Register via UI)* | *(User Defined)* | Personal expense entry, category breakdowns, visual analytics, CSV export, receipt uploads. |

> ⚠️ **Security Warning:** Change default administrative credentials immediately upon first system initialization.

---

## 🧪 Testing & Quality Assurance

The codebase adopts automated testing across critical domain layers:
```bash
# Run all unit and integration tests
go test ./...

# Run tests with verbose output and race detection
go test -v -race ./internal/...
```

Key test suites evaluate:
* Cryptographic token verification and expiry.
* Database repository transactions and edge cases.
* Sanitization of malicious payloads and CSV formula injections.
* Constant-time string comparisons.

---

## 📈 Engineering Methodology (Agile SDLC)

1. **Threat Modeling & Requirements:** Identified data privacy vulnerabilities in existing SaaS solutions; formulated the Zero-Trust model.
2. **Interface-Driven Design:** Decoupled business logic from storage repositories via Go interfaces to ensure high testability.
3. **Defense-in-Depth Implementation:** Integrated multi-layered security (AES-GCM at storage layer, bcrypt at auth layer, CSRF/headers at transport layer).
4. **Empirical Refactoring:** Evaluated multi-factor time drift on mobile devices; transitioned from fragile client-clock TOTP to hardened HttpOnly session cookies.
5. **Durability Verification:** Tested concurrent WAL-mode database reads/writes during active backup execution without database locks.

---

## 📜 Academic Attribution & License

* Developed as part of the **MSc in Software Science and Technology** at the **University of Florence (UniFi)**.
* Distributed under the **MIT License**.
