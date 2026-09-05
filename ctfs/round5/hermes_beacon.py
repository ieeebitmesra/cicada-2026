#!/usr/bin/env python3
"""
[ PANTHEON CTF — ROUND 5: THE GATEKEEPER'S BEACON ]

An intercepted Olympus network transmission was detected broadcast over the ether.
Decrypt the signal beacon to locate the entrance to the inner realm.
"""

import sys
import time

# Transmitted bytecode payload
_PAYLOAD = [
    0x18, 0x04, 0x1c, 0x19, 0x0d, 0x16, 0x7d, 0x78, 0x33, 0x22, 0x66, 0x39, 0x76, 0x24, 0x06, 0x4f, 0x17, 0x30, 0x3c, 0x21, 0x75, 0x30, 0x59, 0x05, 0x2c, 0x1a, 0x3a, 0x39, 0x31, 0x23, 0x41, 0x0c, 0x67, 0x6a, 0x25, 0x28, 0x27, 0x7e, 0x41, 0x53, 0x2b, 0x30, 0x20, 0x28, 0x68, 0x23, 0x5d, 0x44, 0x3c, 0x24, 0x3e, 0x63, 0x2a, 0x3d, 0x40, 0x53, 0x26, 0x21, 0x37, 0x3f, 0x6b, 0x30, 0x5d, 0x5b, 0x35
]

def decode_signal(key_guess: str):
    if len(key_guess) != 8:
        return None
    k_bytes = [ord(c) for c in key_guess]
    res = []
    for i, b in enumerate(_PAYLOAD):
        res.append(chr(b ^ k_bytes[i % len(k_bytes)]))
    return "".join(res)

def main():
    print("=" * 60)
    print("    [ PANTHEON DEFENSE NETWORK — HERMES BEACON ]    ")
    print("=" * 60)
    print("[!] Transmission detected from Olympus Gateway.")
    print("[!] Frequency: 1337.42 MHz | Signal Modulation: XOR-8")
    print("[!] Enter the 8-character divine access key to align beacon:")
    
    try:
        key = input("> ").strip()
    except (EOFError, KeyboardInterrupt):
        return
        
    result = decode_signal(key)
    
    if result and result.startswith("PANTHEON{"):
        print("\n[*] Aligning phase...")
        time.sleep(0.5)
        print("[+] BEACON DECODED SUCCESSFULLY!")
        print("[+] Gateway Coordinates : https://web-secure-portal.onrender.com")
        print(f"[+] Flag Submission     : {result}\n")
    else:
        print("\n[-] Signal corrupted. Static fills the frequency...\n")

if __name__ == "__main__":
    main()
