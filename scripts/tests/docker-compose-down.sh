#!/bin/bash

# Docker Compose Down Script for Kilometers CLI Testing
# Usage: ./docker-compose-down.sh [environment]
# Environments: dev, test, shared

set -euo pipefail

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TIMESTAMP=$(date '+%Y%m%d-%H%M%S')
LOG_DIR="$REPO_ROOT/logs"
LOG_FILE="$LOG_DIR/docker-compose-down-$TIMESTAMP.log"

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
    echo "Usage: $0 [environment] [options]"
    echo ""
    echo "Available environments:"
    for env in "${!COMPOSE_FILES[@]}"; do
        echo "  $env - ${ENVIRONMENT_DESCRIPTIONS[$env]}"
    done
    echo ""
    echo "Options:"
    echo "  -v, --volumes    Remove volumes as well (destructive)"
    echo "  -f, --force      Force removal without confirmation"
    echo "  --all           Stop all environments"
    echo ""
    echo "Examples:"
    echo "  $0 dev          # Stop development environment"
    echo "  $0 shared -v    # Stop shared environment and remove volumes"
    echo "  $0 --all        # Stop all environments"
    echo ""
    echo "If no environment is specified, you'll be prompted to choose."
}

# Function to prompt for environment selection
prompt_environment() {
    echo "Available environments to stop:"
    local i=1
    local envs=()
    local running_envs=()
    
    # Check which environments are actually running
    for env in $(printf '%s\n' "${!COMPOSE_FILES[@]}" | sort); do
        local project_name="kilometers-cli-$env"
        local running_containers=$(docker ps --filter "label=com.docker.compose.project=$project_name" --format "{{.Names}}" 2>/dev/null || true)
        
        envs+=("$env")
        if [[ -n "$running_containers" ]]; then
            running_envs+=("$env")
            echo "  $i) $env - ${ENVIRONMENT_DESCRIPTIONS[$env]} [RUNNING]"
        else
            echo "  $i) $env - ${ENVIRONMENT_DESCRIPTIONS[$env]} [STOPPED]"
        fi
        ((i++))
    done
    
    # Add option to stop all running environments
    if [[ ${#running_envs[@]} -gt 0 ]]; then
        echo "  $i) all - Stop all running environments"
        envs+=("all")
    fi
    
    echo ""
    
    if [[ ${#running_envs[@]} -eq 0 ]]; then
        log_info "No environments are currently running"
        exit 0
    fi
    
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

# Function to check running containers for environment
check_running_containers() {
    local environment="$1"
    local project_name="kilometers-cli-$environment"
    
    local running_containers=$(docker ps --filter "label=com.docker.compose.project=$project_name" --format "{{.Names}}" 2>/dev/null || true)
    
    if [[ -z "$running_containers" ]]; then
        log_warn "No running containers found for environment: $environment"
        return 1
    fi
    
    log_info "Found running containers for environment '$environment':"
    echo "$running_containers" | while read -r container; do
        log_info "  - $container"
    done
    
    return 0
}

# Function to stop docker compose
stop_compose() {
    local environment="$1"
    local remove_volumes="$2"
    local force="$3"
    local compose_file="${COMPOSE_FILES[$environment]}"
    local project_name="kilometers-cli-$environment"
    
    log_info "Stopping Docker Compose for environment: $environment"
    log_info "Using compose file: $compose_file"
    log_info "Project name: $project_name"
    log_info "Remove volumes: $remove_volumes"
    
    # Check if environment is running
    if ! check_running_containers "$environment"; then
        if [[ "$force" == "true" ]]; then
            log_info "Forcing cleanup even though no containers are running"
        else
            return 0
        fi
    fi
    
    cd "$REPO_ROOT"
    
    # Confirm destructive operations
    if [[ "$remove_volumes" == "true" && "$force" != "true" ]]; then
        echo ""
        log_warn "WARNING: This will remove volumes and permanently delete data!"
        echo ""
        while true; do
            read -p "Are you sure you want to remove volumes? (yes/no): " choice
            case "$choice" in
                [Yy][Ee][Ss] ) 
                    log_info "User confirmed volume removal"
                    break
                    ;;
                [Nn][Oo] ) 
                    log_info "User cancelled volume removal"
                    remove_volumes="false"
                    break
                    ;;
                * ) 
                    echo "Please answer yes or no."
                    ;;
            esac
        done
    fi
    
    # Build docker compose command
    local compose_cmd="docker compose -f $compose_file -p $project_name down"
    
    if [[ "$remove_volumes" == "true" ]]; then
        compose_cmd="$compose_cmd -v"
        log_info "Will remove volumes"
    fi
    
    # Stop services
    log_info "Executing: $compose_cmd"
    if eval "$compose_cmd"; then
        log_info "Successfully stopped Docker Compose services for: $environment"
    else
        log_error "Failed to stop Docker Compose services for: $environment"
        return 1
    fi
    
    # Clean up orphaned containers
    log_info "Cleaning up orphaned containers..."
    docker compose -f "$compose_file" -p "$project_name" down --remove-orphans >/dev/null 2>&1 || true
    
    # Verify containers are stopped
    local remaining_containers=$(docker ps --filter "label=com.docker.compose.project=$project_name" --format "{{.Names}}" 2>/dev/null || true)
    if [[ -n "$remaining_containers" ]]; then
        log_warn "Some containers are still running:"
        echo "$remaining_containers" | while read -r container; do
            log_warn "  - $container"
        done
    else
        log_info "All containers stopped successfully"
    fi
    
    return 0
}

# Function to stop all environments
stop_all_environments() {
    local remove_volumes="$1"
    local force="$2"
    local stopped_count=0
    local error_count=0
    
    log_info "Stopping all environments..."
    
    for env in "${!COMPOSE_FILES[@]}"; do
        log_info "Processing environment: $env"
        if stop_compose "$env" "$remove_volumes" "$force"; then
            ((stopped_count++))
        else
            ((error_count++))
        fi
        echo ""
    done
    
    log_info "Stopped $stopped_count environments"
    if [[ $error_count -gt 0 ]]; then
        log_warn "Failed to stop $error_count environments"
    fi
}

# Function to clean up docker resources
cleanup_docker_resources() {
    local force="$1"
    
    if [[ "$force" == "true" ]]; then
        log_info "Cleaning up unused Docker resources..."
        
        # Remove unused networks
        log_info "Removing unused networks..."
        docker network prune -f >/dev/null 2>&1 || true
        
        # Remove unused images (only kilometers-related)
        log_info "Removing unused kilometers images..."
        docker image prune -f --filter "label=org.opencontainers.image.title=kilometers*" >/dev/null 2>&1 || true
        
        log_info "Docker cleanup completed"
    fi
}

# Function to show post-shutdown information
show_post_shutdown_info() {
    local environment="$1"
    local remove_volumes="$2"
    
    echo ""
    log_info "=== Docker Compose Stopped Successfully ==="
    log_info "Environment: $environment"
    log_info "Volumes removed: $remove_volumes"
    log_info "Log file: $LOG_FILE"
    echo ""
    
    echo "Environment is now stopped."
    echo ""
    echo "To start again:"
    echo "  ./scripts/tests/docker-compose-up.sh $environment"
    echo ""
    echo "To view logs from stopped containers:"
    echo "  docker compose -f ${COMPOSE_FILES[$environment]} logs"
}

# Main execution
main() {
    log_info "Starting Docker Compose Down script"
    log_info "Script: $0"
    log_info "Args: $*"
    log_info "Repository root: $REPO_ROOT"
    
    # Parse arguments
    local environment=""
    local remove_volumes="false"
    local force="false"
    local stop_all="false"
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_usage
                exit 0
                ;;
            -v|--volumes)
                remove_volumes="true"
                shift
                ;;
            -f|--force)
                force="true"
                shift
                ;;
            --all)
                stop_all="true"
                shift
                ;;
            -*)
                log_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
            *)
                if [[ -z "$environment" ]]; then
                    environment="$1"
                else
                    log_error "Too many arguments provided"
                    show_usage
                    exit 1
                fi
                shift
                ;;
        esac
    done
    
    # Handle --all option
    if [[ "$stop_all" == "true" ]]; then
        check_docker
        stop_all_environments "$remove_volumes" "$force"
        cleanup_docker_resources "$force"
        log_info "All environments stopped"
        exit 0
    fi
    
    # Prompt for environment if not provided
    if [[ -z "$environment" ]]; then
        environment=$(prompt_environment)
        if [[ "$environment" == "all" ]]; then
            check_docker
            stop_all_environments "$remove_volumes" "$force"
            cleanup_docker_resources "$force"
            log_info "All environments stopped"
            exit 0
        fi
    fi
    
    # Validate environment
    if [[ -z "${COMPOSE_FILES[$environment]:-}" ]]; then
        log_error "Invalid environment: $environment"
        show_usage
        exit 1
    fi
    
    # Pre-flight checks
    check_docker
    
    # Stop docker compose
    if stop_compose "$environment" "$remove_volumes" "$force"; then
        show_post_shutdown_info "$environment" "$remove_volumes"
        log_info "Docker Compose Down script completed successfully"
    else
        log_error "Docker Compose Down script failed"
        exit 1
    fi
}

# Handle script interruption
trap 'log_warn "Script interrupted by user"; exit 130' INT TERM

# Run main function
main "$@"