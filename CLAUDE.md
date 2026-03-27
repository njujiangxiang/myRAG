# myRAG Project Guidelines

## Design System
Always read DESIGN.md before making any visual or UI changes.
All font choices, colors, spacing, and aesthetic direction are defined there.
Do not deviate without explicit user approval.

## Tech Stack
- **Backend:** Go 1.21+ with Gin framework
- **Frontend:** React + TypeScript + shadcn/ui + Tailwind CSS
- **Database:** PostgreSQL 16
- **Vector DB:** Qdrant 1.7+
- **Message Queue:** NATS JetStream
- **File Storage:** MinIO (S3-compatible)

## Project Structure (Planned)
```
myrag/
├── cmd/
│   └── server/
├── internal/
│   ├── handlers/
│   ├── services/
│   ├── models/
│   └── workers/
├── pkg/
├── web/ (Frontend)
├── migrations/
└── docker-compose.yml
```

## Design Tokens (Quick Reference)
- **Primary Color:** Coral #FF6B4A
- **Font Display:** Plus Jakarta Sans
- **Font Body:** DM Sans
- **Font Code:** JetBrains Mono
- **Border Radius:** 12px (default)
- **Spacing Base:** 8px
