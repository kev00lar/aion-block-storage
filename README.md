# Block Storage Engine

A Go-based microservice designed to handle document ingestion using a strict **1MB block-level storage** constraint. This prototype implements a Virtual File System layer that abstracts physical blocks into logical documents.

## Project Overview

This prototype addresses the challenge of storing large documents by "shredding" them into standardized 1MB segments. It ensures memory efficiency and storage optimization through Content-Addressable Storage (CAS).

### Core Requirements (Task 1)
* **Block Constraint:** Maximum storage unit is 1MB.
* **Interface:** Exposed as a RESTful Microservice via the Gin framework.
* **Storage Strategy:** Content-Addressable Storage for built-in deduplication.

---

## Data Architecture

The system uses a "Store & Map" strategy to manage files larger than the 1MB limit.

1. **The Shredder:** Incoming file streams are read into a fixed 1MB buffer.
2. **The CAS Layer:** Each block is hashed using **SHA-256**. The hash serves as the block's unique ID and filename.
3. **The Manifest:** A metadata file (`.txt`) stores the ordered sequence of Block IDs required to reconstruct the original document.



---

## Technical Implementation

### Key Logic Flow
1. **Ingestion:** The client streams a file to the `/upload` endpoint.
2. **Chunking:** The server fills a 1MB buffer.
3. **Deduplication:** Before writing to disk, the server checks if a block with that hash already exists.
4. **Persistence:** Unique blocks are saved to `./data/blocks`.
5. **Registration:** A manifest is saved to `./data/manifests` only after the final block is successfully written.

---

## Edge Cases Handled

| Edge Case | Risk | Resolution |
| :--- | :--- | :--- |
| **File > 1MB** | RAM Exhaustion | **Streaming Buffer:** Reuses a fixed 1MB RAM buffer regardless of total file size. |
| **Tail Blocks** | Data Corruption | **Slice Tracking:** The final chunk is sliced to the exact byte count (`n`), avoiding zero-padding. |
| **Duplicate Content** | Disk Bloat | **CAS Hashing:** Identical blocks across different files are stored only once. |
| **Partial Uploads** | Corrupt Files | **Atomic Manifests:** The manifest is only created upon successful completion of the stream. |



---

## Getting Started

### Prerequisites
* Go 1.18+
* Gin Web Framework: `go get github.com/gin-gonic/gin`

### Running the Service
```bash
go run main.go
