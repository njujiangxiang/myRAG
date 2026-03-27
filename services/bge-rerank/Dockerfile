# BGE Rerank Service - Docker Image
# Self-hosted Cross-Encoder reranking with FlagEmbedding

FROM python:3.10-slim

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    git \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements first for better caching
COPY requirements.txt .

# Install Python dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY main.py .

# Expose port
EXPOSE 8800

# Environment variables
ENV BGE_MODEL=BAAI/bge-reranker-v2-m3
ENV BGE_DEVICE=cuda
ENV BGE_HOST=0.0.0.0
ENV BGE_PORT=8800

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:8800/health')" || exit 1

# Run the service
CMD ["python", "main.py"]
