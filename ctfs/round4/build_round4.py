import struct
import zlib
import os
import zipfile
import hashlib
import subprocess
import urllib.request
from PIL import Image, ImageEnhance
import io

PIN = "20092002194507165301101201319690815"
PASSWORD = "BABYLONIANHARMONY"
FLAG = "PANTHEON{B4byl0n14n_M1d1_H4rm0ny_D3c0d3d}"

# -------------------------------------------------------------
# 1. Generate Ramanujan ASCII Art from Wikimedia
# -------------------------------------------------------------
def get_ramanujan_ascii_art():
    url = "https://upload.wikimedia.org/wikipedia/commons/c/c1/Srinivasa_Ramanujan_-_OPC_-_1.jpg"
    headers = {"User-Agent": "Mozilla/5.0"}
    req = urllib.request.Request(url, headers=headers)
    with urllib.request.urlopen(req) as resp:
        img_bytes = resp.read()
    
    img = Image.open(io.BytesIO(img_bytes)).convert("L")
    w, h = img.size
    img = img.crop((int(w * 0.05), int(h * 0.04), int(w * 0.95), int(h * 0.94)))

    enhancer = ImageEnhance.Contrast(img)
    img = enhancer.enhance(1.4)

    width = 72
    aspect_ratio = img.height / img.width
    height = int(aspect_ratio * width * 0.48)
    img = img.resize((width, height), Image.Resampling.LANCZOS)

    chars = ["@", "%", "#", "*", "+", "=", "-", ":", ".", " "]
    pixels = list(img.getdata())
    
    lines = []
    for y in range(height):
        row = ""
        for x in range(width):
            val = pixels[y * width + x]
            idx = min(int(val / 256 * len(chars)), len(chars) - 1)
            row += chars[idx]
        lines.append(row)
    
    return "\n".join(lines)

# -------------------------------------------------------------
# 2. MIDI Generation (2 Tracks, Track 1 encodes BABYLONIANHARMONY)
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
        program_change(channel=1, program=42, delta=0), # Ancient Lyre
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
# 3. Pure Python ZipCrypto Encrypted Zip Creation
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
# 4. Build Pipeline
# -------------------------------------------------------------
def main():
    target_dir = r"E:\ieee_ctf\ctfs\round4"
    os.chdir(target_dir)

    print("[*] Generating Ramanujan ASCII art...")
    ascii_art = get_ramanujan_ascii_art()

    article_text = f"""
{ascii_art}


I am the third of my kind, yet I arrived first to your eye.
My brothers did not ride with me — they hid in plain sight,
long before you heard my engine.

One dressed himself in pixels, wearing width like a crown
and height like a throne. You saw him but did not count him.
You measured him but did not name him.

The other slept inside a silence older than your calendar,
a passenger in the bones of light itself,
waiting in the frame you passed through
before the frequencies ever sang to you.

We are three. We share one secret —
every one of us can be broken into two pairs of cubes,
and every one of us knows the same address.

I am already in your hand.
Go back. The other two are not lost.
They were never hidden from you —
only unrecognised.

When all three stand together, separated by nothing but a dot,
append our kingdom and knock.
The door has always been open.
"""

    # 1. Write the inner text file (contains only the art, article, and riddle)
    txt_filename = "flag.txt"
    with open(txt_filename, "w", encoding="utf-8") as f:
        f.write(article_text)
    print(f"[+] Wrote {txt_filename} with Ramanujan ASCII Art and Enigma Riddle.")

    # 2. Generate MIDI file
    midi_path = "ancient_hymn.mid"
    generate_midi(PASSWORD, midi_path)

    # 3. Generate inner password protected zip containing ONLY the text file
    inner_zip_path = "secret_archive.zip"
    create_encrypted_zip(inner_zip_path, txt_filename, article_text.encode('utf-8'), PASSWORD)

    # 4. Generate outer zip containing ONLY ancient_hymn.mid and secret_archive.zip
    outer_zip = "round4_challenge.zip"
    with zipfile.ZipFile(outer_zip, "w", compression=zipfile.ZIP_DEFLATED) as zf:
        zf.write(midi_path, "ancient_hymn.mid")
        zf.write(inner_zip_path, "secret_archive.zip")
    print(f"[+] Outer bundle ({outer_zip}) packaged with EXACTLY 2 items: {zf.namelist()}")

    # 5. Encrypt payload for Round 4 static site matching app.js parameters
    node_script = f"""
    const fs = require('fs');
    const crypto = require('crypto');

    const pin = "{PIN}";
    const zipBytes = fs.readFileSync('round4_challenge.zip');

    const salt = Buffer.from([112, 97, 110, 116, 104, 101, 111, 110, 95, 114, 111, 117, 110, 100, 52, 33]);
    const iv = Buffer.from([26, 63, 123, 156, 78, 130, 208, 101, 17, 148, 239, 40]);
    const key = crypto.pbkdf2Sync(pin, salt, 100000, 32, 'sha256');

    const cipher = crypto.createCipheriv('aes-256-gcm', key, iv);
    const encrypted = Buffer.concat([cipher.update(zipBytes), cipher.final(), cipher.getAuthTag()]);
    const b64 = encrypted.toString('base64');
    fs.writeFileSync('payload.b64', b64);

    let appJs = fs.readFileSync('app.js', 'utf8');
    appJs = appJs.replace(/const _0xCIPHER = '[^']*';/, `const _0xCIPHER = '${{b64}}';`);
    fs.writeFileSync('app.js', appJs);
    console.log('[+] Encrypted payload and updated app.js successfully!');
    """
    subprocess.run(["node", "-e", node_script], check=True)

    flag_str = "17291383220683.pages.dev"
    flag_hash = hashlib.sha256(flag_str.encode('utf-8')).hexdigest()
    print("\n" + "="*50)
    print(f"Round 4 Expected Flag: {flag_str}")
    print(f"Round 4 Flag Hash (SHA-256): {flag_hash}")
    print("="*50)

if __name__ == '__main__':
    main()
