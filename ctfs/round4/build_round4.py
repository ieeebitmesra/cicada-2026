#!/usr/bin/env python3
"""
build_round4.py — Complete generator for Round 4 Challenge:
1. Generates 2-track MIDI file 'ancient_hymn.mid' (Track 1 = Melody/Key, Track 2 = Harmony).
2. Creates password-protected 'secret_archive.zip' containing 'flag.txt' encrypted with the MIDI key.
3. Packages everything into 'round4_challenge.zip'.
4. Encrypts payload for the obfuscated Round 4 static site (PIN: 20092002194507165301101201319690815).
"""

import struct
import zlib
import os
import zipfile
import base64
import hashlib
import subprocess

# -------------------------------------------------------------
# Configuration
# -------------------------------------------------------------
PIN = "20092002194507165301101201319690815"
PASSWORD = "BABYLONIANHARMONY"
FLAG = "PANTHEON{B4byl0n14n_M1d1_H4rm0ny_D3c0d3d}"

# -------------------------------------------------------------
# 1. MIDI Generation (2 Tracks, 26 Pitches A=60..Z=85)
# -------------------------------------------------------------
def write_varlen(value):
    buf = bytearray()
    buf.append(value & 0x7F)
    value >>= 7
    while value > 0:
        buf.append((value & 0x7F) | 0x80)
        value >>= 7
    return bytes(reversed(buf))

def make_meta_track_name(name):
    encoded = name.encode('utf-8')
    return b'\x00\xFF\x03' + write_varlen(len(encoded)) + encoded

def make_meta_tempo(bpm=100):
    us_per_beat = int(60000000 / bpm)
    return b'\x00\xFF\x51\x03' + struct.pack('>I', us_per_beat)[1:]

def make_end_of_track():
    return b'\x00\xFF\x2F\x00'

def note_on(channel, note, velocity, delta=0):
    return write_varlen(delta) + bytes([0x90 | (channel & 0x0F), note & 0x7F, velocity & 0x7F])

def note_off(channel, note, velocity=64, delta=0):
    return write_varlen(delta) + bytes([0x80 | (channel & 0x0F), note & 0x7F, velocity & 0x7F])

def program_change(channel, program, delta=0):
    return write_varlen(delta) + bytes([0xC0 | (channel & 0x0F), program & 0x7F])

def build_track(events):
    data = b''.join(events) + make_end_of_track()
    return b'MTrk' + struct.pack('>I', len(data)) + data

def generate_midi(password, output_path):
    ticks_per_beat = 480
    note_duration = 420
    gap = 60
    base_pitch = 60 # C4 = Middle C = 'A'

    # Track 1: Melody (Letters A-Z mapped to pitches 60-85)
    t1_events = [
        make_meta_track_name("Melody - The Oracle's Voice"),
        make_meta_tempo(100),
        program_change(channel=0, program=46, delta=0), # Orchestral Harp
    ]

    first = True
    for char in password.upper():
        if 'A' <= char <= 'Z':
            pitch = base_pitch + (ord(char) - ord('A'))
            delta = 0 if first else gap
            t1_events.append(note_on(channel=0, note=pitch, velocity=95, delta=delta))
            t1_events.append(note_off(channel=0, note=pitch, velocity=64, delta=note_duration))
            first = False

    track1 = build_track(t1_events)

    # Track 2: Ancient Lyre & Bass Accompaniment
    t2_events = [
        make_meta_track_name("Harmony - Babylonian Lyre"),
        program_change(channel=1, program=42, delta=0), # Cello / Ancient Lyre
    ]

    bass_notes = [38, 41, 45, 43, 38, 45, 41, 38]
    first = True
    for i in range(len(password)):
        bass_pitch = bass_notes[i % len(bass_notes)]
        delta = 0 if first else gap
        t2_events.append(note_on(channel=1, note=bass_pitch, velocity=75, delta=delta))
        t2_events.append(note_off(channel=1, note=bass_pitch, velocity=50, delta=note_duration))
        first = False

    track2 = build_track(t2_events)

    header = b'MThd' + struct.pack('>IHHH', 6, 1, 2, ticks_per_beat)
    with open(output_path, 'wb') as f:
        f.write(header + track1 + track2)
    print(f"[+] Created MIDI file: {output_path}")

# -------------------------------------------------------------
# 2. Password-Protected Zip Generation (Pure Python ZipCrypto)
# -------------------------------------------------------------
class ZipCrypto:
    def __init__(self, password: str):
        self.keys = [0x12345678, 0x23456789, 0x34567890]
        for c in password.encode('utf-8'):
            self.update_keys(c)

    def update_keys(self, char_val: int):
        self.keys[0] = (~zlib.crc32(bytes([char_val]), ~self.keys[0])) & 0xFFFFFFFF
        self.keys[1] = ((self.keys[1] + (self.keys[0] & 0xFF)) * 134775813 + 1) & 0xFFFFFFFF
        self.keys[2] = (~zlib.crc32(bytes([(self.keys[1] >> 24) & 0xFF]), ~self.keys[2])) & 0xFFFFFFFF

    def magic_byte(self):
        temp = (self.keys[2] | 2) & 0xFFFF
        return ((temp * (temp ^ 1)) >> 8) & 0xFF

    def encrypt(self, data: bytes) -> bytes:
        out = bytearray()
        for b in data:
            k = self.magic_byte()
            out.append(b ^ k)
            self.update_keys(b)
        return bytes(out)

def create_encrypted_zip(zip_path, filename_inside, content_bytes, password):
    zc = ZipCrypto(password)
    crc = zlib.crc32(content_bytes) & 0xFFFFFFFF
    header = os.urandom(11) + bytes([(crc >> 24) & 0xFF])
    enc_header = zc.encrypt(header)
    
    compressor = zlib.compressobj(zlib.Z_DEFAULT_COMPRESSION, zlib.DEFLATED, -15)
    compressed = compressor.compress(content_bytes) + compressor.flush()
    enc_data = zc.encrypt(compressed)
    full_payload = enc_header + enc_data
    
    fn_bytes = filename_inside.encode('utf-8')
    lfh = struct.pack('<4sHHHHHIIIHH',
        b'PK\x03\x04', 20, 1, 8, 0, 0,
        crc, len(full_payload), len(content_bytes), len(fn_bytes), 0
    ) + fn_bytes
    
    cdh = struct.pack('<4sHHHHHHIIIHHHHHII',
        b'PK\x01\x02', 20, 20, 1, 8, 0, 0,
        crc, len(full_payload), len(content_bytes), len(fn_bytes),
        0, 0, 0, 0, 0, 0
    ) + fn_bytes
    
    eocd = struct.pack('<4sHHHHIIH',
        b'PK\x05\x06', 0, 0, 1, 1,
        len(cdh), len(lfh) + len(full_payload), 0
    )
    
    with open(zip_path, 'wb') as f:
        f.write(lfh + full_payload + cdh + eocd)
    print(f"[+] Created encrypted zip: {zip_path} (Password: {password})")

# -------------------------------------------------------------
# 3. Main Builder
# -------------------------------------------------------------
def main():
    cur_dir = os.path.dirname(os.path.abspath(__file__))
    os.chdir(cur_dir)

    # 1. Generate MIDI
    midi_path = "ancient_hymn.mid"
    generate_midi(PASSWORD, midi_path)

    # 2. Generate inner secret archive
    inner_zip_path = "secret_archive.zip"
    flag_content = f"""PANTHEON ARCHIVES — CLASSIFIED REWARD
==================================================
Congratulations, Seeker!
You deciphered the musical pitches of the ancient Babylonian Lyre.

FLAG: {FLAG}
==================================================
""".encode('utf-8')
    create_encrypted_zip(inner_zip_path, "flag.txt", flag_content, PASSWORD)

    # 3. Generate parchment scroll lore
    scroll_content = """THE SCROLL OF THE BABYLONIAN LYRE
==================================================
"To open the sealed archives of the Pantheon, one must listen
 closely to the 26 voices of the Oracle's scale.
 From the foundational C4 (A) rising upward to C#6 (Z),
 the harp sings the true name of power."

- Track 1: The Oracle's Voice (Melody)
- Track 2: Babylonian Lyre (Harmony)
==================================================
"""
    with open("scroll.txt", "w", encoding="utf-8") as f:
        f.write(scroll_content)

    # 4. Package outer challenge zip
    outer_zip = "round4_challenge.zip"
    with zipfile.ZipFile(outer_zip, "w", compression=zipfile.ZIP_DEFLATED) as zf:
        zf.write(midi_path, "ancient_hymn.mid")
        zf.write(inner_zip_path, "secret_archive.zip")
        zf.write("scroll.txt", "scroll.txt")
    print(f"[+] Packaged outer bundle: {outer_zip}")

    # 5. Encrypt payload for Round 4 static site using Node.js built-in crypto (Web Crypto compatible AES-GCM + PBKDF2)
    node_script = f"""
    const fs = require('fs');
    const crypto = require('crypto');

    const pin = "{PIN}";
    const zipBytes = fs.readFileSync('round4_challenge.zip');

    const salt = crypto.randomBytes(16);
    const iv = crypto.randomBytes(12);
    const key = crypto.pbkdf2Sync(pin, salt, 100000, 32, 'sha256');

    const cipher = crypto.createCipheriv('aes-256-gcm', key, iv);
    const encrypted = Buffer.concat([cipher.update(zipBytes), cipher.final()]);
    const tag = cipher.getAuthTag();

    const blob = Buffer.concat([salt, iv, encrypted, tag]);
    fs.writeFileSync('payload.b64', blob.toString('base64'));
    """
    subprocess.run(["node", "-e", node_script], check=True)
    print(f"[+] Updated encrypted payload for static site: payload.b64")

    # 6. Compute Flag Hash for portal
    flag_hash = hashlib.sha256(FLAG.encode('utf-8')).hexdigest()
    print("\n" + "="*50)
    print(f"Round 4 Flag: {FLAG}")
    print(f"Round 4 Flag Hash (SHA-256): {flag_hash}")
    print("="*50)

if __name__ == '__main__':
    main()
