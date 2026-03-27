-- Add rag_type column to knowledge_bases table
-- Migration: 002_add_rag_type.sql

ALTER TABLE knowledge_bases
ADD COLUMN IF NOT EXISTS rag_type VARCHAR(50) DEFAULT 'vector'
CHECK (rag_type IN ('vector', 'graph', 'hybrid', 'keyword'));

-- Add index for rag_type filtering
CREATE INDEX IF NOT EXISTS idx_kbs_rag_type ON knowledge_bases(rag_type);

-- Comment
COMMENT ON COLUMN knowledge_bases.rag_type IS 'RAG strategy type: vector (default), graph, hybrid, or keyword';
