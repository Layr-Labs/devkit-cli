#!/bin/bash

#Linux testing script using Docker

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "🐧 Running Linux compatibility tests..."
echo "This will test ALL aspects of devkit-cli in a Linux environment"
echo ""

cd "$PROJECT_ROOT"

# Function to cleanup
cleanup() {
    echo "🧹 Cleaning up test containers..."
    docker container prune -f >/dev/null 2>&1 || true
    docker image prune -f >/dev/null 2>&1 || true
}

# Trap cleanup on exit
trap cleanup EXIT


# Build the Docker image with all tests
# Each RUN command in the Dockerfile is a test - if any fail, the build stops
if docker build --add-host=host.docker.internal:host-gateway -f docker/linux-test/Dockerfile -t devkit-linux-test .; then
    echo ""
    echo "ALL LINUX COMPATIBILITY TESTS PASSED!"
    echo ""
    echo "devkit-cli works correctly on Linux!"
    echo ""
    echo "🔍 To manually test in the Linux environment, run:"
    echo "   docker run -it --rm --add-host=host.docker.internal:host-gateway -v \$(pwd):/workspace -w /workspace devkit-linux-test"
    echo ""
    echo "📝 To see detailed test results:"
    echo "   docker run --rm --add-host=host.docker.internal:host-gateway devkit-linux-test cat test-results.txt"
    
    # Show the test results
    echo ""
    echo "📋 Test Summary:"
    docker run --rm --add-host=host.docker.internal:host-gateway devkit-linux-test cat test-results.txt
    
else
    echo ""
    echo "❌ LINUX COMPATIBILITY TESTS FAILED!"
    echo ""
    echo "The Docker build failed, which means there are Linux-specific issues."
    echo "Check the error output above to see which test failed."
    echo ""
    echo "🔧 To debug:"
    echo "1. Look at the last successful RUN command in the output"
    echo "2. Run intermediate stages manually:"
    echo "   docker build --target=cli-basic-test -f docker/linux-test/Dockerfile ."
    echo "3. Start an interactive session:"
    echo "   docker run -it --rm -v \$(pwd):/workspace -w /workspace golang:1.24-bookworm bash"
    
    exit 1
fi

echo ""
echo "🚀 Testing complete!CLI is Linux-compatible." 