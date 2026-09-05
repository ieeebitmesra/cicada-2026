#!/usr/bin/env python3
"""
generate_midi.py — Generates a 2-track Type-1 MIDI file where Track 1 encodes
a secret password through musical pitch mapping (A=60, B=61, ..., Z=85),
accompanied by Track 2 harmonic bass/chords.
"""

import struct
import zipfile
import pyminizip  # fallback or zipfile with AES/PKWARE
import os

def write_varlen(value):
    """Encodes an integer into variable-length quantity (MIDI VLQ)."""
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

def make_meta_tempo(bpm=110):
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

def generate_midi_puzzle(password="BABYLONIANHARMONY", output_file="ancient_hymn.mid"):
    """
    Encodes password into a 2-track MIDI song:
    - Pitch Mapping: Note = 60 + (ord(char.upper()) - ord('A'))
      Range: 60 (Middle C) to 85 (C#6) -> exactly 26 distinct pitches.
    - Track 1 (Melody / Lead): Orchestral Harp (Program 46)
    - Track 2 (Harmony / Bass): Cello (Program 42)
    """
    ticks_per_beat = 480
    note_duration = 420   # Note sound duration
    gap = 60              # Silence gap before next note (total = 480 ticks = 1 beat)
    base_pitch = 60       # MIDI 60 = Middle C = 'A'

    # -------------------------------------------------------------
    # Track 1: Melody (Spell out the password)
    # -------------------------------------------------------------
    t1_events = [
        make_meta_track_name("Melody - The Oracle's Voice"),
        make_meta_tempo(100),
        program_change(channel=0, program=46, delta=0), # Orchestral Harp
    ]

    first_note = True
    for char in password.upper():
        if 'A' <= char <= 'Z':
            pitch = base_pitch + (ord(char) - ord('A'))
            delta_on = 0 if first_note else gap
            t1_events.append(note_on(channel=0, note=pitch, velocity=95, delta=delta_on))
            t1_events.append(note_off(channel=0, note=pitch, velocity=64, delta=note_duration))
            first_note = False
        elif char == ' ':
            delta_on = 0 if first_note else gap
            t1_events.append(note_on(channel=0, note=0, velocity=0, delta=delta_on))
            t1_events.append(note_off(channel=0, note=0, velocity=0, delta=note_duration))
            first_note = False

    track1 = build_track(t1_events)

    # -------------------------------------------------------------
    # Track 2: Ancient Lyre & Bass Accompaniment
    # -------------------------------------------------------------
    t2_events = [
        make_meta_track_name("Harmony - Babylonian Lyre"),
        program_change(channel=1, program=42, delta=0), # Cello / Ancient Lyre
    ]

    # Ancient Dorian bass progression: D2, F2, A2, G2
    bass_notes = [38, 41, 45, 43, 38, 45, 41, 38]
    first_bass = True
    
    for i in range(len(password)):
        bass_pitch = bass_notes[i % len(bass_notes)]
        delta_on = 0 if first_bass else gap
        t2_events.append(note_on(channel=1, note=bass_pitch, velocity=75, delta=delta_on))
        t2_events.append(note_off(channel=1, note=bass_pitch, velocity=50, delta=note_duration))
        first_bass = False

    track2 = build_track(t2_events)

    # -------------------------------------------------------------
    # Header Chunk: Format 1, 2 Tracks, 480 ticks/quarter note
    # -------------------------------------------------------------
    header = b'MThd' + struct.pack('>IHHH', 6, 1, 2, ticks_per_beat)

    midi_bytes = header + track1 + track2
    with open(output_file, 'wb') as f:
        f.write(midi_bytes)

    print(f"[+] Generated MIDI: {output_file} ({len(midi_bytes)} bytes)")
    print(f"[+] Password Encoded: {password}")
    print(f"[+] Pitch Range: {base_pitch} ('A') to {base_pitch + 25} ('Z')")
    return output_file

if __name__ == '__main__':
    generate_midi_puzzle()
