#!/bin/bash
export PATH=$PATH:~/.local/go/bin:/usr/local/go/bin

echo "Building nptx..."
go build -o nptx_bin ./cmd/nptx/main.go

echo "Starting UDP Echo Server on port 25566..."
python3 -c "
import socket
s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
s.bind(('127.0.0.1', 25566))
while True:
    data, addr = s.recvfrom(1024)
    s.sendto(b'ECHO: ' + data, addr)
" &
ECHO_PID=$!

echo "Starting nptx server on 1230..."
./nptx_bin -mode server -local 127.0.0.1:1230 -password testpass &
SERVER_PID=$!
sleep 1

echo "Starting nptx client mapping 7305 to 25566 via 1230..."
./nptx_bin -mode client -remote 127.0.0.1:1230 -password testpass -routes 7305:25566 -streams 4 &
CLIENT_PID=$!
sleep 1

echo "Sending test payload to client local port 7305..."
python3 -c "
import socket
try:
    s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    s.settimeout(3.0)
    s.sendto(b'HelloTunnel', ('127.0.0.1', 7305))
    data, _ = s.recvfrom(1024)
    print('SUCCESS_RECEIVED:', data.decode())
except Exception as e:
    print('ERROR:', e)
"

echo "Cleaning up..."
kill $ECHO_PID $SERVER_PID $CLIENT_PID
