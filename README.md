# Tiny Ledger

A lightweight ledger API for tracking financial transactions, built with Go. This service provides a minimal yet robust implementation for recording deposits and withdrawals, viewing balances, and retrieving transaction history.

## Features

* Record transactions (Deposit / Withdrawal)
* Retrieve transaction history with pagination
* View current balance
* Thread-safe in-memory storage
* Input validation
* Time-range filtering for transaction history

## Quick Start

### Prerequisites

* Go 1.21+
* Docker (optional)

### Run Locally

```bash
# Run directly
go run cmd/server/main.go

# Or using the Go toolchain
go run ./...
```

### Docker Deployment

```bash
# Build image
docker build -t tiny-ledger .

# Run container
docker run -p 8080:8080 tiny-ledger
```

### Run Tests

```bash
go test ./... -race
```

## API Endpoints

### Record a Transaction

```
POST /users/{userId}/transactions
```

**Request Body:**
```json
{
    "type": "deposit|withdrawal",
    "amount": 100.0,
    "description": "Transaction description"
}
```

**Response:** The created transaction record with timestamp and ID.

### Get Current Balance

```
GET /users/{userId}/balance
```

**Response:**
```json
{
"balance": 250.0
}
```

### Get Transaction History

```
GET /users/{userId}/transactions
```

**Query Parameters:**
- `start`: Optional start time filter (RFC3339 format)
- `end`: Optional end time filter (RFC3339 format)
- `page`: Page number (default: 1)
- `pageSize`: Items per page (default: 10, max: 100)

**Response:**
```json
{
"transactions": [
{
"id": "uuid-string",
"amount": 100.0,
"type": "deposit",
"timestamp": "2023-07-01T12:00:00Z",
"description": "Initial deposit"
},
...
],
"pagination": {
"page": 1,
"pageSize": 10,
"totalItems": 45,
"totalPages": 5
}
}
```

## Example Usage

```bash
# Create a deposit
curl -X POST http://localhost:8080/users/saradorri/transactions \
-H "Content-Type: application/json" \
-d '{"type":"deposit", "amount":100, "description":"Initial deposit"}'

# Check balance
curl http://localhost:8080/users/saradorri/balance

# Get transaction history with time filtering
curl "http://localhost:8080/users/saradorri/transactions?start=2024-01-01T00:00:00Z&end=2025-01-01T00:00:00Z"

# Get paginated transactions (page 2 with 5 items per page)
curl "http://localhost:8080/users/saradorri/transactions?page=2&pageSize=5"
```

## Implementation Details

### Project Architecture

The implementation follows clean architecture principles with a clear separation of concerns:

```
cmd/
server/           # Main application entry point
internal/
handlers/         # HTTP API handlers
services/         # Business logic
store/            # In-memory thread-safe data store
models/           # Data models
```

### Efficient Pagination

The transaction history API uses server-side pagination to efficiently retrieve only the requested page of data:

- **Memory-efficient**: Only retrieves and processes the data needed for the current page
- **Source-level filtering**: Time and pagination filters are applied at the data source
- **Consistent pagination metadata**: Total counts and page information are calculated accurately

### Input Validation

All inputs are validated for:
- **User IDs**: Alphanumeric format with length restrictions
- **Amounts**: Positive values with maximum limits
- **Transaction types**: Valid enumeration values
- **Descriptions**: Length checks

### Thread Safety

The ledger uses an in-memory store with proper mutex locking to ensure thread safety for concurrent operations from multiple users.

## Design Considerations and Production Alternatives

This implementation uses in-memory data structures as requested in the requirements, which allows for a quick implementation that can be completed in a few hours. However, in a real-world production environment, several alternative approaches would be more suitable:

### Event Sourcing

For a financial ledger system in production, an event sourcing pattern would be more appropriate:
- Store all transactions as immutable events
- Derive the current state (balances) from the event history
- Ensures complete audit trail and data integrity
- Allows for point-in-time reconstruction of state

### Persistent Storage

A real-world implementation would use:
- A relational database for ACID transactions (PostgreSQL, MySQL)
- Or a distributed database for scalability (CockroachDB)
- Proper indexing for efficient querying

### Distributed Consistency

For a distributed system:
- Distributed locks
- Transaction isolation levels

The current in-memory implementation provides a good demonstration of the core functionality while meeting the time constraints of the assignment.

## License

MIT