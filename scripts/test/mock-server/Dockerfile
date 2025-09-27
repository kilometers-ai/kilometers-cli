FROM python:3.11-slim

WORKDIR /app

# Copy the mock server script
COPY server.py .

# Create directory for test data
RUN mkdir -p /app/data

# Expose port 8080
EXPOSE 8080

# Default command
CMD ["python", "server.py", "--port", "8080"]
