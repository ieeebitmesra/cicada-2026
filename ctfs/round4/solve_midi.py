#!/usr/bin/env python3
"""
solve_midi.py — Reads a 2-track MIDI file, decodes the melody pitch sequence
into the plaintext password, and extracts the password-protected zip file.
"""

import struct
import sys
import zipfile

def parse_midi_notes(midi_path):
    with open(midi_path, 'rb') as f:
        data = f.read()

    # Read Header
    if data[:4] != b'MThd':
        raise ValueError("Invalid MIDI header")
    
    header_len = struct.unpack('>I', data[4:8])[0]
    fmt, num_tracks, division = struct.unpack('>HHH', data[8:8+header_len])
    
    print(f"[*] MIDI Format: {fmt}, Tracks: {num_tracks}, Division: {division}")

    offset = 8 + header_len
    all_tracks = []

    for t_idx in range(num_tracks):
        if data[offset:offset+4] != b'MTrk':
            break
        track_len = struct.unpack('>I', data[offset+4:offset+8])[0]
        track_data = data[offset+8:offset+8+track_len]
        offset += 8 + track_len

        # Parse track events
        t_offset = 0
        notes = []
        track_name = ""

        while t_offset < len(track_data):
            # Read delta time (VLQ)
            delta = 0
            while True:
                b = track_data[t_offset]
                t_offset += 1
                delta = (delta << 7) | (b & 0x7F)
                if not (b & 0x80):
                    break

            if t_offset >= len(track_data):
                break

            status = track_data[t_offset]
            t_offset += 1

            if status == 0xFF: # Meta event
                meta_type = track_data[t_offset]
                t_offset += 1
                # Meta length
                meta_len = 0
                while True:
                    b = track_data[t_offset]
                    t_offset += 1
                    meta_len = (meta_len << 7) | (b & 0x7F)
                    if not (b & 0x80):
                        break
                meta_payload = track_data[t_offset:t_offset+meta_len]
                t_offset += meta_len
                if meta_type == 0x03:
                    track_name = meta_payload.decode('utf-8', errors='ignore')
            elif (status & 0xF0) == 0x90: # Note On
                note = track_data[t_offset]
                velocity = track_data[t_offset+1]
                t_offset += 2
                if velocity > 0:
                    notes.append(note)
            elif (status & 0xF0) == 0x80: # Note Off
                t_offset += 2
            elif (status & 0xF0) in (0xC0, 0xD0): # Program Change, Channel Pressure (1 param)
                t_offset += 1
            elif (status & 0xF0) in (0xA0, 0xB0, 0xE0): # 2 params
                t_offset += 2

        all_tracks.append({"index": t_idx + 1, "name": track_name, "notes": notes})

    return all_tracks

def decode_password(notes, base_pitch=60):
    chars = []
    for n in notes:
        idx = n - base_pitch
        if 0 <= idx < 26:
            chars.append(chr(ord('A') + idx))
        else:
            chars.append('?')
    return ''.join(chars)

def main():
    midi_file = sys.argv[1] if len(sys.argv) > 1 else "ancient_hymn.mid"
    tracks = parse_midi_notes(midi_file)

    print("\n" + "="*50)
    for t in tracks:
        print(f"Track #{t['index']}: '{t['name']}' -> {len(t['notes'])} notes: {t['notes']}")
    
    # Melody is in Track 1
    melody_notes = tracks[0]['notes']
    password = decode_password(melody_notes)
    print("="*50)
    print(f"\n[+] Decoded Password from Track 1 (Base Pitch 60=A): \033[1;32m{password}\033[0m")

    # If a zip file exists, test unlocking it
    zip_file = "flag_archive.zip"
    if len(sys.argv) > 2:
        zip_file = sys.argv[2]
    
    try:
        with zipfile.ZipFile(zip_file) as zf:
            zf.extractall(pwd=password.encode('utf-8'))
            print(f"[+] Successfully extracted {zip_file} using decoded password '{password}'!")
    except Exception as e:
        print(f"[*] Zip extraction test ({zip_file}): {e}")

if __name__ == '__main__':
    main()
