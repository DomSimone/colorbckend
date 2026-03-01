# Greens Marketplace Backend

Go-based backend API for the Greens Marketplace application.

## Docker Deployment

This backend is configured for deployment to Render using Docker.

### Building the Docker Image

```bash
cd greens-marketplace/backend
docker build -t greens-marketplace-backend .
```

### Running the Container

```bash
docker run -d \
  --name greens-marketplace-backend \
  -p 8080:8080 \
  -e DB_HOST=your-db-host \
  -e DB_USER=your-db-user \
  -e DB_PASSWORD=your-db-password \
  -e DB_NAME=your-db-name \
  -e REDIS_HOST=your-redis-host \
  -e JWT_SECRET=your-jwt-secret \
  -e OPENAI_API_KEY=your-openai-key \
  greens-marketplace-backend
```

### Environment Variables

Required environment variables for production:

- `DB_HOST`: PostgreSQL database host
- `DB_USER`: PostgreSQL database user
- `DB_PASSWORD`: PostgreSQL database password
- `DB_NAME`: PostgreSQL database name
- `REDIS_HOST`: Redis host
- `JWT_SECRET`: JWT signing secret (must be secure in production)
- `OPENAI_API_KEY`: OpenAI API key for semantic search

Optional environment variables:

- `DB_PORT`: PostgreSQL port (default: 5432)
- `DB_SSLMODE`: PostgreSQL SSL mode (default: disable)
- `REDIS_PORT`: Redis port (default: 6379)
- `REDIS_PASSWORD`: Redis password (default: empty)
- `REDIS_DB`: Redis database number (default: 0)
- `JWT_EXPIRATION`: JWT expiration in hours (default: 24)
- `OPENAI_MODEL`: OpenAI model (default: text-embedding-ada-002)
- `OPENAI_MAX_TOKENS`: Maximum tokens for OpenAI (default: 1000)
- `OPENAI_TEMPERATURE`: OpenAI temperature (default: 0.7)

### Health Check

The container includes a health check endpoint at `/health` that returns:
```json
{
  "status": "healthy",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

### Render Deployment

1. Push to your repository
2. In Render Dashboard, create a new Web Service
3. Connect to your repository
4. Set the build command: `cd backend && docker build -t greens-marketplace-backend .`
5. Set the start command: `docker run --rm -p 8080:8080 greens-marketplace-backend`
6. Configure environment variables in Render Dashboard
7. Set the health check path to `/health`

### Database Migrations

Database migrations are included in the Docker image at `/app/migrations/`.
Ensure your PostgreSQL database is set up and run the migrations manually
or through your deployment process.

### Security Notes

- The application runs as a non-root user (appuser)
- All sensitive configuration is handled through environment variables
- The binary is built with stripped symbols for smaller image size
- Health check ensures the application is responsive