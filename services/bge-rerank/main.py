"""
BGE Rerank Service - FastAPI wrapper for BGE Cross-Encoder reranking
Compatible with FlagEmbedding: https://github.com/FlagOpen/FlagEmbedding
"""

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field
from typing import List, Optional, Union
import logging
import os

# Fix for FlagEmbedding compatibility issue
# FlagEmbedding's trainer.py uses Optional without importing it
import builtins
if not hasattr(builtins, 'Optional'):
    import typing
    builtins.Optional = typing.Optional
    builtins.Union = typing.Union

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(
    title="BGE Rerank Service",
    description="Self-hosted BGE Cross-Encoder reranking service",
    version="1.0.0"
)


class RerankRequest(BaseModel):
    query: str = Field(..., description="Search query", min_length=1, max_length=1000)
    documents: List[str] = Field(..., description="List of documents to rerank", min_length=1, max_length=1000)
    top_n: Optional[int] = Field(default=None, description="Number of results to return", ge=1, le=100)


class RerankResult(BaseModel):
    index: int
    score: float
    text: Optional[str] = None


class RerankResponse(BaseModel):
    results: List[RerankResult]


class HealthResponse(BaseModel):
    status: str
    model: Optional[str] = None


# Global reranker instance
reranker = None


@app.on_event("startup")
async def load_model():
    """Load BGE reranker model on startup"""
    global reranker

    model_name = os.getenv("BGE_MODEL", "BAAI/bge-reranker-v2-m3")
    device = os.getenv("BGE_DEVICE", "cuda")  # cuda or cpu

    logger.info(f"Loading BGE reranker model: {model_name} on {device}")

    try:
        from FlagEmbedding import FlagReranker

        reranker = FlagReranker(
            model_name,
            use_fp16=device == "cuda",  # Use FP16 for GPU inference
        )

        logger.info(f"Model loaded successfully: {model_name}")

    except ImportError:
        logger.error("FlagEmbedding not installed. Install with: pip install FlagEmbedding")
        raise
    except Exception as e:
        logger.error(f"Failed to load model: {e}")
        raise


@app.get("/health", response_model=HealthResponse)
async def health_check():
    """Health check endpoint"""
    return HealthResponse(
        status="healthy",
        model=os.getenv("BGE_MODEL", "BAAI/bge-reranker-v2-m3")
    )


@app.post("/rerank", response_model=RerankResponse)
async def rerank(request: RerankRequest):
    """
    Rerank documents based on relevance to query using BGE Cross-Encoder

    Args:
        request: RerankRequest with query, documents, and optional top_n

    Returns:
        RerankResponse with sorted results containing index and score
    """
    if reranker is None:
        raise HTTPException(status_code=503, detail="Model not loaded")

    if not request.documents:
        return RerankResponse(results=[])

    try:
        # Compute scores using BGE reranker
        # FlagReranker.compute_score expects (query, passage) pair or list of pairs
        # Create pairs of query and each document
        pairs = [[request.query, doc] for doc in request.documents]
        scores = reranker.compute_score(pairs)

        # Handle single document case (returns float instead of list)
        if isinstance(scores, (float, int)):
            scores = [scores]

        # Create results with index and score
        results = []
        for idx, score in enumerate(scores):
            results.append(RerankResult(
                index=idx,
                score=float(score),
                text=request.documents[idx] if len(request.documents) > idx else None
            ))

        # Sort by score descending
        results.sort(key=lambda x: x.score, reverse=True)

        # Apply top_n limit
        if request.top_n and request.top_n > 0:
            results = results[:request.top_n]

        return RerankResponse(results=results)

    except Exception as e:
        logger.error(f"Rerank failed: {e}")
        raise HTTPException(status_code=500, detail=f"Rerank failed: {str(e)}")


@app.get("/models")
async def list_models():
    """List available model information"""
    return {
        "current_model": os.getenv("BGE_MODEL", "BAAI/bge-reranker-v2-m3"),
        "available_models": [
            "BAAI/bge-reranker-v2-m3",      # Multilingual, best quality
            "BAAI/bge-reranker-v2-base",    # Base version, balanced
            "BAAI/bge-reranker-v2-minico",  # Lightweight, fastest
        ]
    }


if __name__ == "__main__":
    import uvicorn

    host = os.getenv("BGE_HOST", "0.0.0.0")
    port = int(os.getenv("BGE_PORT", "8800"))

    logger.info(f"Starting BGE Rerank Service on {host}:{port}")

    uvicorn.run(app, host=host, port=port)
