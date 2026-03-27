#!/bin/bash
set -e

echo "======================================"
echo "myRAG - Startup Script"
echo "======================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if .env exists
if [ ! -f .env ]; then
    print_warn ".env file not found. Copying from .env.example..."
    cp .env.example .env
    print_info "Please edit .env file with your configuration before running again."
    exit 1
fi

# Load environment variables
set -a
source .env
set +a

echo ""
echo "1. Checking prerequisites..."
echo "-------------------------------------------"

# Check if Ollama is running
if curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
    print_info "Ollama is running"

    # Check if embedding model exists
    if curl -s http://localhost:11434/api/tags | grep -q "$EMBEDDING_MODEL"; then
        print_info "Embedding model '$EMBEDDING_MODEL' found"
    else
        print_error "Embedding model '$EMBEDDING_MODEL' not found in Ollama"
        print_info "Run: ollama pull $EMBEDDING_MODEL"
        exit 1
    fi

    # Check if LLM model exists
    if curl -s http://localhost:11434/api/tags | grep -q "$LLM_MODEL"; then
        print_info "LLM model '$LLM_MODEL' found"
    else
        print_warn "LLM model '$LLM_MODEL' not found in Ollama"
        print_info "Run: ollama pull $LLM_MODEL"
    fi
else
    print_error "Ollama is not running. Please start Ollama first:"
    echo "       ollama serve"
    exit 1
fi

# Check Docker
if command -v docker &> /dev/null; then
    print_info "Docker is available"
else
    print_error "Docker is not installed"
    exit 1
fi

echo ""
echo "2. Starting infrastructure services..."
echo "-------------------------------------------"

# Start infrastructure (PostgreSQL, Qdrant, NATS, MinIO)
docker compose up -d postgres qdrant nats minio

print_info "Waiting for services to be ready..."
sleep 5

# Check PostgreSQL
if docker compose ps postgres | grep -q "Up"; then
    print_info "PostgreSQL is running"
else
    print_error "PostgreSQL failed to start"
    docker compose logs postgres
    exit 1
fi

# Check Qdrant
if docker compose ps qdrant | grep -q "Up"; then
    print_info "Qdrant is running"

    # Wait for Qdrant to be fully ready
    sleep 3

    # Check if collection exists and has correct dimension
    COLLECTION_INFO=$(curl -s http://localhost:6333/collections/documents 2>/dev/null)
    if echo "$COLLECTION_INFO" | grep -q "4096"; then
        print_info "Qdrant collection 'documents' exists with correct dimension (4096)"
    elif echo "$COLLECTION_INFO" | grep -q "1536"; then
        print_warn "Qdrant collection has old dimension (1536). Recreating..."
        curl -X DELETE "http://localhost:6333/collections/documents"
        sleep 2
    else
        print_info "Qdrant collection will be created on first run"
    fi
else
    print_error "Qdrant failed to start"
    docker compose logs qdrant
    exit 1
fi

# Check NATS
if docker compose ps nats | grep -q "Up"; then
    print_info "NATS is running"
else
    print_error "NATS failed to start"
    docker compose logs nats
    exit 1
fi

# Check MinIO
if docker compose ps minio | grep -q "Up"; then
    print_info "MinIO is running"
    print_info "MinIO Console: http://localhost:9001 (admin123/password123)"
else
    print_error "MinIO failed to start"
    docker compose logs minio
    exit 1
fi

echo ""
echo "3. Building and starting application..."
echo "-------------------------------------------"

# Build and start the application
docker compose up -d --build app

if docker compose ps app | grep -q "Up"; then
    print_info "Application is running"
    print_info "API: http://localhost:8080"
    print_info "Health check: curl http://localhost:8080/health"
else
    print_error "Application failed to start"
    docker compose logs app
    exit 1
fi

echo ""
echo "======================================"
echo -e "${GREEN}All services started successfully!${NC}"
echo "======================================"
echo ""
echo "Services:"
echo "  - API:          http://localhost:8080"
echo "  - MinIO:        http://localhost:9000"
echo "  - MinIO Admin:  http://localhost:9001"
echo "  - Qdrant:       http://localhost:6333"
echo "  - PostgreSQL:   localhost:5432"
echo "  - NATS:         localhost:4222"
echo "  - Ollama:       http://localhost:11434"
echo ""
echo "To view logs: docker compose logs -f app"
echo "To stop:      docker compose down"
echo ""
