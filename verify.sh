#!/bin/bash

# Ensure clean state
rm -f client1.log client2.log echo.log teapot-client

export PORT=3000
export READ_SECRET="asdfghjklç"
export WRITE_SECRET="qwertyuiop"

# Kill any process running on port $PORT
PID=$(lsof -t -i:$PORT)
if [ -n "$PID" ]; then
  echo "Killing existing process $PID on port $PORT"
  kill -9 $PID
fi

echo "Starting server on port $PORT..."
# Start server in background
go run main.go &
SERVER_PID=$!
# Wait for server to be ready
sleep 3

echo "Building client..."
cd client
go build -o ../teapot-client main.go
cd ..

echo "Sending requests via curl (Legacy Client 1)..."
# Client 1 loop (using curl with WRITE_SECRET)
(
  for i in {1..5}; do
    curl -s -u "$WRITE_SECRET:client1" -d "client1 log $i" http://localhost:$PORT/log
    sleep 0.5
  done
) &
C1_PID=$!

echo "Running teapot client (Client 2)..."
# Client 2 using the new CLI
# We use --insecure because localhost is HTTP
# Command is "echo" for simplicity, running in a loop
(
  for i in {1..3}; do
    ./teapot-client -s "$WRITE_SECRET" -h "http://localhost:$PORT" --insecure echo "client2 log $i"
    sleep 0.5
  done
) &
C2_PID=$!

wait $C1_PID
wait $C2_PID

# Allow async writes to finish
sleep 2

echo "--- Verifying client1 logs (Expected: 5 lines) ---"
curl -s -H "Authorization: $READ_SECRET" "http://localhost:$PORT/logs/client1"
echo -e "\n----------------------------------------------"

echo "--- Verifying client2 logs (Expected: 3 lines) ---"
# Note: The client uses the command name as the client name.
# Here the command is "echo". So logs should be in "echo.log".
# Wait, the user said: "The client must "accumulate" the logs from the application it wraps... and send it to the server"
# And "Authorization header for saving the logs... WRITE_SECRET:clientName".
# In my implementation: req.SetBasicAuth(cfg.Secret, cfg.Command)
# So clientName is the command name.
curl -s -H "Authorization: $READ_SECRET" "http://localhost:$PORT/logs/echo"
echo -e "\n----------------------------------------------"

echo "--- Testing GET /logs/client1?n=2&skip=1 (Expected: lines 3 and 4) ---"
curl -s -H "Authorization: $READ_SECRET" "http://localhost:$PORT/logs/client1?n=2&skip=1"
echo -e "\n------------------------------------------------------------------"

echo "--- Testing Unauthorized Access (Expected: 401) ---"
curl -s -o /dev/null -w "%{http_code}" "http://localhost:$PORT/logs/client1"
echo -e "\n---------------------------------------------------"

echo "Stopping server..."
kill $SERVER_PID
wait $SERVER_PID
