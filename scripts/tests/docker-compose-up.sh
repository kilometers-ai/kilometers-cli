#!/bin/bash

# Docker Compose Up Script for Kilometers CLI Testing
# Usage: ./docker-compose-up.sh [environment]
# Environments: dev, test, shared

set -euo pipefail

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TIMESTAMP=$(date '+%Y%m%d-%H%M%S')
LOG_DIR="$REPO_ROOT/logs"
LOG_FILE="$LOG_DIR/docker-compose-up-$TIMESTAMP.log"

# Ensure log directory exists
mkdir -p "$LOG_DIR"

# Logging function
log() {
    local level="$1"
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[$timestamp] [$level] $message" | tee -a "$LOG_FILE"
}

log_info() { log "INFO" "$@"; }
log_warn() { log "WARN" "$@"; }
log_error() { log "ERROR" "$@"; }

# Available environments and their compose files
declare -A COMPOSE_FILES=(
    ["dev"]="docker-compose.dev.yml"
    ["test"]="docker-compose.test.yml"
    ["shared"]="docker-compose.shared.yml"
)

# Available environment descriptions
declare -A ENVIRONMENT_DESCRIPTIONS=(
    ["dev"]="Development environment - standalone with local database"
    ["test"]="Test environment - isolated testing with mock services"
    ["shared"]="Shared environment - uses shared API from kilometers-api repo"
)

# Function to show usage
show_usage() {
    echo "Usage: $0 [environment]"
    echo ""
    echo "Available environments:"
    for env in "${!COMPOSE_FILES[@]}"; do
        echo "  $env - ${ENVIRONMENT_DESCRIPTIONS[$env]}"
    done
    echo ""
    echo "Examples:"
    echo "  $0 dev     # Start development environment"
    echo "  $0 shared  # Start shared environment"
    echo "  $0 test    # Start test environment"
    echo ""
    echo "If no environment is specified, you'll be prompted to choose."
}

# Function to prompt for environment selection
prompt_environment() {
    echo "Available environments:"
    local i=1
    local envs=()
    for env in $(printf '%s\n' "${!COMPOSE_FILES[@]}" | sort); do
        envs+=("$env")
        echo "  $i) $env - ${ENVIRONMENT_DESCRIPTIONS[$env]}"
        ((i++))
    done
    echo ""
    
    while true; do
        read -p "Select environment (1-${#envs[@]}) or 'q' to quit: " choice
        
        if [[ "$choice" == "q" || "$choice" == "Q" ]]; then
            log_info "User cancelled environment selection"
            exit 0
        fi
        
        if [[ "$choice" =~ ^[0-9]+$ ]] && [ "$choice" -ge 1 ] && [ "$choice" -le "${#envs[@]}" ]; then
            local selected_env="${envs[$((choice-1))]}"
            echo "$selected_env"
            return 0
        fi
        
        echo "Invalid selection. Please enter a number between 1 and ${#envs[@]}, or 'q' to quit."
    done
}

# Function to check if docker is running
check_docker() {
    if ! docker info >/dev/null 2>&1; then
        log_error "Docker is not running or not accessible"
        log_error "Please start Docker and try again"
        exit 1
    fi
    log_info "Docker is running"
}

# Function to check if compose file exists
check_compose_file() {
    local compose_file="$1"
    local full_path="$REPO_ROOT/$compose_file"
    
    if [[ ! -f "$full_path" ]]; then
        log_error "Compose file not found: $full_path"
        exit 1
    fi
    log_info "Found compose file: $compose_file"
}

# Function to check for existing containers
check_existing_containers() {
    local environment="$1"
    local compose_file="$2"
    
    log_info "Checking for existing containers..."
    
    # Get running containers for this compose project
    local project_name="kilometers-cli-$environment"
    local running_containers=$(docker ps --filter "label=com.docker.compose.project=$project_name" --format "{{.Names}}" 2>/dev/null || true)
    
    if [[ -n "$running_containers" ]]; then
        log_warn "Found running containers for environment '$environment':"
        echo "$running_containers" | while read -r container; do
            log_warn "  - $container"
        done
        echo ""
        
        while true; do
            read -p "Stop existing containers and continue? (y/n): " choice
            case "$choice" in
                [Yy]* ) 
                    log_info "Stopping existing containers..."
                    docker compose -f "$REPO_ROOT/$compose_file" -p "$project_name" down
                    break
                    ;;
                [Nn]* ) 
                    log_info "User chose not to stop existing containers"
                    exit 0
                    ;;
                * ) 
                    echo "Please answer yes or no."
                    ;;
            esac
        done
    fi
}

# Function to start docker compose
start_compose() {
    local environment="$1"
    local compose_file="$2"
    local project_name="kilometers-cli-$environment"
    
    log_info "Starting Docker Compose for environment: $environment"
    log_info "Using compose file: $compose_file"
    log_info "Project name: $project_name"
    
    cd "$REPO_ROOT"
    
    # Pull latest images
    log_info "Pulling latest images..."
    if docker compose -f "$compose_file" -p "$project_name" pull; then
        log_info "Successfully pulled latest images"
    else
        log_warn "Failed to pull some images, continuing with cached versions"
    fi
    
    # Start services
    log_info "Starting services..."
    if docker compose -f "$compose_file" -p "$project_name" up -d; then
        log_info "Successfully started Docker Compose services"
    else
        log_error "Failed to start Docker Compose services"
        exit 1
    fi
    
    # Wait for services to be ready
    log_info "Waiting for services to start..."
    sleep 5
    
    # Show running containers
    log_info "Running containers:"
    docker compose -f "$compose_file" -p "$project_name" ps
    
    # Environment-specific post-startup actions
    case "$environment" in
        "shared")
            log_info "Checking shared API health..."
            check_shared_api_health
            ;;
        "dev")
            log_info "Checking development API health..."
            check_dev_api_health
            ;;
        "test")
            log_info "Test environment started successfully"
            ;;
    esac
}

# Function to check shared API health
check_shared_api_health() {
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s http://localhost:5194/health >/dev/null 2>&1; then
            log_info "Shared API is healthy (attempt $attempt/$max_attempts)"
            return 0
        fi
        
        log_info "Waiting for shared API to be ready (attempt $attempt/$max_attempts)..."
        sleep 2
        ((attempt++))
    done
    
    log_warn "Shared API health check failed after $max_attempts attempts"
    log_warn "API may still be starting up. Check logs with: docker compose -f docker-compose.shared.yml logs -f api"
}

# Function to check dev API health
check_dev_api_health() {
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s http://localhost:5194/health >/dev/null 2>&1; then
            log_info "Development API is healthy (attempt $attempt/$max_attempts)"
            return 0
        fi
        
        log_info "Waiting for development API to be ready (attempt $attempt/$max_attempts)..."
        sleep 2
        ((attempt++))
    done
    
    log_warn "Development API health check failed after $max_attempts attempts"
    log_warn "API may still be starting up. Check logs with: docker compose -f docker-compose.dev.yml logs -f api"
}

# Function to show post-startup information
show_post_startup_info() {
    local environment="$1"
    
    echo ""
    log_info "=== Docker Compose Started Successfully ==="
    log_info "Environment: $environment"
    log_info "Log file: $LOG_FILE"
    echo ""
    
    case "$environment" in
        "shared")
            echo "Shared Environment Endpoints:"
            echo "  - API: http://localhost:5194"
            echo "  - Swagger UI: http://localhost:5194/swagger"
            echo "  - Health: http://localhost:5194/health"
            echo ""
            echo "Useful commands:"
            echo "  - View API logs: docker compose -f docker-compose.shared.yml logs -f api"
            echo "  - View DB logs: docker compose -f docker-compose.shared.yml logs -f postgres"
            echo "  - Stop environment: ./scripts/tests/docker-compose-down.sh shared"
            ;;
        "dev")
            echo "Development Environment Endpoints:"
            echo "  - API: http://localhost:5194"
            echo "  - Swagger UI: http://localhost:5194/swagger"
            echo "  - Health: http://localhost:5194/health"
            echo ""
            echo "Useful commands:"
            echo "  - View API logs: docker compose -f docker-compose.dev.yml logs -f api"
            echo "  - View DB logs: docker compose -f docker-compose.dev.yml logs -f postgres"
            echo "  - Stop environment: ./scripts/tests/docker-compose-down.sh dev"
            ;;
        "test")
            echo "Test Environment Started"
            echo ""
            echo "Useful commands:"
            echo "  - View logs: docker compose -f docker-compose.test.yml logs -f"
            echo "  - Stop environment: ./scripts/tests/docker-compose-down.sh test"
            ;;
    esac
    
    echo ""
    echo "Next steps:"
    echo "  - Run E2E tests: ./scripts/tests/test-plugin-e2e.sh"
    echo "  - Test CLI: export KM_API_ENDPOINT=\"http://localhost:5194\" && ./km auth status"
}

# Main execution
main() {
    log_info "Starting Docker Compose Up script"
    log_info "Script: $0"
    log_info "Args: $*"
    log_info "Repository root: $REPO_ROOT"
    
    # Parse arguments
    local environment=""
    
    if [[ $# -eq 0 ]]; then
        environment=$(prompt_environment)
    elif [[ $# -eq 1 ]]; then
        if [[ "$1" == "-h" || "$1" == "--help" ]]; then
            show_usage
            exit 0
        fi
        environment="$1"
    else
        log_error "Too many arguments provided"
        show_usage
        exit 1
    fi
    
    # Validate environment
    if [[ -z "${COMPOSE_FILES[$environment]:-}" ]]; then
        log_error "Invalid environment: $environment"
        show_usage
        exit 1
    fi
    
    local compose_file="${COMPOSE_FILES[$environment]}"
    
    # Pre-flight checks
    check_docker
    check_compose_file "$compose_file"
    check_existing_containers "$environment" "$compose_file"
    
    # Start docker compose
    start_compose "$environment" "$compose_file"
    
    # Show post-startup information
    show_post_startup_info "$environment"
    
    log_info "Docker Compose Up script completed successfully"
}

# Handle script interruption
trap 'log_warn "Script interrupted by user"; exit 130' INT TERM

# Run main function
main "$@"