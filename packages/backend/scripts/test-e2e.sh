#!/bin/bash

# Kill any existing server process
pkill -f "server"

# Start the server in the background
echo "Starting server..."
cd cmd/server && go run main.go &
SERVER_PID=$!

# Wait for server to start
echo "Waiting for server to start..."
sleep 5

# Run the tests
echo "Running end-to-end tests..."
cd ../test && go test -v -run TestEndToEndFlow

# Cleanup
echo "Cleaning up..."
kill $SERVER_PID

echo "Done!" 