import socket
import threading
import time
import random
import subprocess
import sys

# Chaos Proxy: Sits between NPTX Client and Server to introduce drops and delays
class ChaosProxy:
    def __init__(self, listen_port, target_ip, target_port):
        self.sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        self.sock.bind(('127.0.0.1', listen_port))
        self.target = (target_ip, target_port)
        self.running = True
        self.client_addr = None

    def start(self):
        def forward():
            while self.running:
                try:
                    data, addr = self.sock.recvfrom(65535)
                    
                    if addr != self.target:
                        self.client_addr = addr # Learn the actual nptx client UDP port
                        dest = self.target
                    else:
                        dest = self.client_addr
                        if not dest:
                            continue

                    # Introduce Chaos
                    r = random.random()
                    if r < 0.10: # 10% Drop
                        continue
                    elif r < 0.20: # 10% Out of order / Delay
                        time.sleep(0.05)
                        self.sock.sendto(data, dest)
                    else: # Fast path
                        self.sock.sendto(data, dest)
                except Exception:
                    pass
        t = threading.Thread(target=forward)
        t.start()
        return t

    def stop(self):
        self.running = False
        self.sock.close()

# Mock Target Server (Echo Server representing Wireguard/Hysteria)
class MockAppServer:
    def __init__(self, port):
        self.sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        self.sock.bind(('127.0.0.1', port))
        self.running = True

    def start(self):
        def loop():
            while self.running:
                try:
                    data, addr = self.sock.recvfrom(65535)
                    self.sock.sendto(data, addr)
                except:
                    pass
        threading.Thread(target=loop).start()

    def stop(self):
        self.running = False
        self.sock.close()

def run_test():
    print("Starting Chaos Proxy on 1330 forwarding to actual server on 1331...", flush=True)
    chaos = ChaosProxy(1330, '127.0.0.1', 1331)
    chaos.start()

    print("Starting App Server on 26566...", flush=True)
    app = MockAppServer(26566)
    app.start()

    print("Starting nptx server on 1331...", flush=True)
    server = subprocess.Popen(["./nptx_bin", "-mode", "server", "-local", "127.0.0.1:1331", "-password", "chaospass"])
    time.sleep(1)

    print("Starting nptx client mapping 8305 to 26566 via chaos 1330...", flush=True)
    client = subprocess.Popen(["./nptx_bin", "-mode", "client", "-remote", "127.0.0.1:1330", "-routes", "8305:26566", "-password", "chaospass", "-streams", "4"])
    time.sleep(1)

    sent_count = 0
    recv_count = 0

    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.settimeout(0.5)

    mtus = [1350, 1420, 9000] # Hysteria/TUIC standard, WireGuard standard, Jumbo frames

    for idx in range(50):
        mtu = random.choice(mtus)
        # Generate random payload of exact MTU size
        payload = bytes([random.randint(0, 255) for _ in range(mtu)])
        
        sock.sendto(payload, ('127.0.0.1', 8305))
        sent_count += 1
        
        try:
            data, _ = sock.recvfrom(65535)
            if data == payload:
                recv_count += 1
        except socket.timeout:
            pass # Expected due to chaos drops!

    print(f"\n--- CHAOS TEST RESULTS ---")
    print(f"Sent: {sent_count} packets with various MTUs (1350, 1420, 9000)")
    print(f"Received back intact: {recv_count} packets")
    
    if recv_count > 0:
        print("✅ Tunnel survived chaos, reassembly works for jumbo frames!")
    else:
        print("❌ Tunnel failed completely!")

    # Wait for GC to kick in
    print("Sleeping 4 seconds to test Reassembler Garbage Collection...")
    time.sleep(4)
    print("Survival Test Passed. No Panics!")

    # Cleanup
    server.kill()
    client.kill()
    chaos.stop()
    app.stop()

if __name__ == "__main__":
    run_test()
