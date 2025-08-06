#!/bin/bash

# Kilometers CLI Development Environment Manager
# Provides easy commands to manage the Docker development environment

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
COMPOSE_FILE="$PROJECT_DIR/docker-compose.dev.yml"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed or not in PATH"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose is not installed or not in PATH"
        exit 1
    fi
    
    if [ ! -d "$PROJECT_DIR/../kilometers-api" ]; then
        log_error "kilometers-api repository not found at ../kilometers-api"
        log_warning "Please ensure both repositories are in the same parent directory"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Start the development environment
start_dev_env() {
    log_info "Starting development environment..."
    
    cd "$PROJECT_DIR"
    
    if [ "$1" = "--with-tools" ]; then
        log_info "Starting with pgAdmin..."
        docker-compose -f docker-compose.dev.yml --profile tools up -d
    else
        docker-compose -f docker-compose.dev.yml up -d
    fi
    
    log_success "Development environment started"
    log_info "Services:"
    log_info "  🐘 PostgreSQL: localhost:5432"
    log_info "  🚀 API: http://localhost:5000"
    
    if [ "$1" = "--with-tools" ]; then
        log_info "  🔧 pgAdmin: http://localhost:5050 (dev@kilometers.ai / devpassword)"
    fi
    
    log_info ""
    log_info "Waiting for services to be ready..."
    sleep 10
    
    # Check if API is ready
    for i in {1..30}; do
        if curl -f http://localhost:5000/health &> /dev/null; then
            log_success "API is ready!"
            break
        fi
        echo -n "."
        sleep 2
    done
    
    echo ""
    log_info "Environment ready! Set these variables to use with CLI:"
    log_info "  export KM_API_URL=http://localhost:5000"
    log_info "  export KM_API_KEY=your-api-key-here"
}

# Stop the development environment
stop_dev_env() {
    log_info "Stopping development environment..."
    cd "$PROJECT_DIR"
    docker-compose -f docker-compose.dev.yml down
    log_success "Development environment stopped"
}

# Clean the development environment
clean_dev_env() {
    log_warning "This will remove all data and volumes!"
    read -p "Are you sure? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log_info "Cleaning development environment..."
        cd "$PROJECT_DIR"
        docker-compose -f docker-compose.dev.yml down -v --remove-orphans
        log_success "Development environment cleaned"
    else
        log_info "Cancelled"
    fi
}

# Show logs
show_logs() {
    cd "$PROJECT_DIR"
    if [ -n "$1" ]; then
        docker-compose -f docker-compose.dev.yml logs -f "$1"
    else
        docker-compose -f docker-compose.dev.yml logs -f
    fi
}

# Show status
show_status() {
    cd "$PROJECT_DIR"
    docker-compose -f docker-compose.dev.yml ps
    
    echo ""
    log_info "Health checks:"
    
    # Check PostgreSQL
    if docker exec kilometers-dev-postgres pg_isready -U postgres &> /dev/null; then
        log_success "PostgreSQL: Ready"
    else
        log_error "PostgreSQL: Not ready"
    fi
    
    # Check API
    if curl -f http://localhost:5000/health &> /dev/null; then
        log_success "API: Ready (http://localhost:5000)"
    else
        log_error "API: Not ready"
    fi
}

# Restart service
restart_service() {
    if [ -z "$1" ]; then
        log_error "Please specify a service to restart (api, postgres)"
        exit 1
    fi
    
    log_info "Restarting $1..."
    cd "$PROJECT_DIR"
    docker-compose -f docker-compose.dev.yml restart "$1"
    log_success "$1 restarted"
}

# Rebuild service
rebuild_service() {
    if [ -z "$1" ]; then
        log_error "Please specify a service to rebuild (api)"
        exit 1
    fi
    
    log_info "Rebuilding $1..."
    cd "$PROJECT_DIR"
    docker-compose -f docker-compose.dev.yml up -d --build "$1"
    log_success "$1 rebuilt"
}

# Test CLI integration
test_cli() {
    log_info "Testing CLI integration..."
    
    # Check if API is running
    if ! curl -f http://localhost:5000/health &> /dev/null; then
        log_error "API is not running. Start the dev environment first."
        exit 1
    fi
    
    # Set environment variables
    export KM_API_URL=http://localhost:5000
    export KM_DEBUG=true
    
    log_info "Testing with environment:"
    log_info "  KM_API_URL=$KM_API_URL"
    log_info "  KM_API_KEY=${KM_API_KEY:-not-set}"
    
    # Build CLI if not exists
    if [ ! -f "$PROJECT_DIR/km" ]; then
        log_info "Building CLI..."
        cd "$PROJECT_DIR"
        go build -o km ./cmd/main.go
    fi
    
    # Test basic CLI functionality
    log_info "Testing CLI..."
    cd "$PROJECT_DIR"
    ./km --version
    
    if [ -n "$KM_API_KEY" ]; then
        log_info "Testing with API key..."
        ./km plugins list
    else
        log_warning "KM_API_KEY not set, skipping authenticated tests"
    fi
    
    log_success "CLI integration test completed"
}

# Print usage
print_usage() {
    echo "Kilometers CLI Development Environment Manager"
    echo ""
    echo "Usage: $0 <command> [options]"
    echo ""
    echo "Commands:"
    echo "  start [--with-tools]  Start the development environment"
    echo "  stop                  Stop the development environment"
    echo "  clean                 Stop and remove all data (destructive)"
    echo "  status                Show service status and health"
    echo "  logs [service]        Show logs (all or specific service)"
    echo "  restart <service>     Restart a specific service"
    echo "  rebuild <service>     Rebuild and restart a service"
    echo "  test-cli              Test CLI integration with dev environment"
    echo "  check                 Check prerequisites"
    echo "  help                  Show this help message"
    echo ""
    echo "Services: api, postgres"
    echo ""
    echo "Examples:"
    echo "  $0 start              # Start basic environment"
    echo "  $0 start --with-tools # Start with pgAdmin"
    echo "  $0 logs api           # Show API logs"
    echo "  $0 restart api        # Restart API service"
    echo "  $0 rebuild api        # Rebuild API from source"
}

# Main command dispatcher
case "${1:-}" in
    "start")
        check_prerequisites
        start_dev_env "$2"
        ;;
    "stop")
        stop_dev_env
        ;;
    "clean")
        clean_dev_env
        ;;
    "status")
        show_status
        ;;
    "logs")
        show_logs "$2"
        ;;
    "restart")
        restart_service "$2"
        ;;
    "rebuild")
        rebuild_service "$2"
        ;;
    "test-cli")
        test_cli
        ;;
    "check")
        check_prerequisites
        ;;
    "help"|"--help"|"-h")
        print_usage
        ;;
    *)
        log_error "Unknown command: ${1:-}"
        echo ""
        print_usage
        exit 1
        ;;
esac
