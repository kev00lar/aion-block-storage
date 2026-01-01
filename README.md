# Document Intelligence Prototype: Block Storage & Search Engine

A high-performance Go microservice that handles document ingestion by shredding files into **strict 1MB blocks** and providing **Intelligence-driven keyword search** across the document library.

## Project Overview

This system solves two primary challenges:
1.  **Task 1 (Storage):** Abstracting large files into a Content-Addressable Storage (CAS) layer using 1MB blocks.
2.  **Task 2 (Intelligence):** Providing instant keyword search across all documents using a memory-mapped Inverted Index.

---

## Technical Architecture

### 1. Storage Layer (The Shredder)
* **1MB Block Constraint:** Files are processed through a fixed-size memory buffer. A 10MB file becomes 10 independent blocks.
* **Content-Addressable Storage (CAS):** Each block is named after its **SHA-256 hash**. 
* **Deduplication:** If two different files contain the same 1MB of data, the block is stored only once on disk.
* **Manifest System:** A `.txt` manifest maps the original filename to the ordered sequence of block hashes for reassembly.



### 2. Intelligence Layer (Inverted Index)
The search engine does not scan files during a query. Instead, it builds an **Inverted Index** during the upload phase.
* **Logic:** Instead of `File -> Words`, we store `Word -> [Files]`.
* **Advanced Tokenization:** The engine splits text using a custom `FieldsFunc` (delimiters: `, : ; " _ \n \t`) to correctly index data in JSON and CSV formats.
* **Search Complexity:** $O(1)$ constant time lookup via memory map.



---

## Implementation Details & Edge Cases

| Feature | Handling Logic | Why? |
| :--- | :--- | :--- |
| **Tail Blocks** | `buffer[:n]` slicing | Ensures the final chunk of a file isn't padded with empty bytes. |
| **JSON/CSV Search** | Delimiter-based splitting | Correctly identifies words in strings. |
| **Memory Ceiling** | Streaming ingestion | The service never loads the full file into RAM; it only ever "sees" 1MB at a time. |
| **Case Sensitivity** | Case Folding | All keywords are lowercased to ensure `UUID` and `uuid` yield the same results. |

---

## Getting Started

### Prerequisites
* Go 1.18+
* Gin Framework: `go get github.com/gin-gonic/gin`

### API Usage

#### 1. Ingest a Document (Task 1 & 2)
Uploads, shreds, and indexes the document.
```bash
curl -F "document=@stageUserInfo.json" http://localhost:8080/upload
```
#### 2. Intelligence Search (Task 2)
Search for any keyword
```bash
curl "http://localhost:8080/search?q=uuid"
```
#### 3. Reassemble and Download
Reconstructs the original file from the 1MB blocks.
```bash
curl http://localhost:8080/download/stageUserInfo.json --output downloaded.json
```
---
### Logic Assumptions
**Text Processing: The intelligence engine assumes files are UTF-8 compatible. Binary files (like JPEGs) will be indexed as gibberish.**

**In-Memory Index: For this prototype, the search index is held in RAM. (For production, this would be persisted to a Key-Value store).**

**Keyword Length: Words shorter than 3 characters are ignored to prevent index bloat from common words (e.g., "is", "a", "the").**