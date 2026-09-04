(function() {
    'use strict';

    // Obfuscated string table and decoder
    const _0x4f12 = [
        'round4_challenge.zip', 'application/zip', 'vault-form', 'auth-input',
        'btn-submit', 'terminal-log', 'download-card', 'download-btn',
        'system-status', 'log-entry', 'log-info', 'log-success', 'log-error',
        'log-warn', 'createElement', 'appendChild', 'innerText', 'className',
        'disabled', 'style', 'display', 'block', 'none', 'href', 'click',
        'pantheon_round4!', 'SHA-256', 'PBKDF2', 'AES-GCM', 'raw', 'importKey',
        'deriveKey', 'decrypt', 'revokeObjectURL', 'createObjectURL'
    ];

    function _0xget(idx) {
        return _0x4f12[idx];
    }

    // Encrypted AES-256-GCM ZIP payload
    const _0xCIPHER = 'uUIvxCSJAGREgz9HaHoQ0b/rWCz+n1g2nvlyFhwtfm+x/6t9buZ1O/EIKxHRZ53R2dUBaMAkg/UdsOjU/6aNhUdwf6Uyc6zjWssO5biPUbpotb0py6HgYmbj1P5EgvhfM2qOKUGjbpQEiY0yesrxTlcJk3GdujKNY/kl9AGpZ/h7Z2LvEP/T6P93Y44IfZaFQ61TUG6rmzJsBQW9qwyN9XCBYV8rDAtniPzBwiW/RAg/PH9sJz4AA3U4PbWnmN58Zcloshq+zUzy55a5GwTDOIEgZ1D7tmgFccte3kyZ5H5FPh5sGXPbT8DxQsRK6hoALnkNDpoxso19mF+ZZpPNct3DG5nmZV+R4mhY7d+B521+++R1+6CnX1uZQUqOXUTUrCZpD9JKXZbur7iyF5guoaXf5Q9qlF6g1bJ+7Ti5ezfFhYd30FqCkspsW+M+gIQJqQkbkHdmlBDWzllC9kL6mMlhuLSSv2FJM5Bg1XZacS0X71RGE+Hti4j0MRLOAxMQBdB9/K3Fh0JnyHSgVEdFpdDYkGim+mPVnXH0n4DIV6R7NYQi3wxgT4JAhHYFabbV/A715Dy7tN+YiqVTxV9v54cc9qbnW6Jh5rqgQ7KTytnc5XVxH1QsBzID5roMC1cAgw9NPZHnR7EBc8tCZj9HHH3VxSyVAfMhJFi3w0dF5X6HlYbWQNQsAsvpRXOQEgPvt43W0XUFB8oFKZLiE4Jziyysyu912Yi+rHFwbw2oFhv50iz5JaHw6eh8GhB6T93PXpWnHPNoZ2uhPYQBLfaamWZ3CSfAx8za3pw9ddQbphm5L1rudvhPNfdeZk1WtloZ0F1kJzpOx9fOmct4xt/1zQ+9cP41ugQRFbopKG2twn9OM/tgM4ppxS5TVTaOy5YzoHWJwxmXyjHdk7J5KLl/KVX21H7tQYLGeiZITfBOcY2QzE8X74dsKypQnfLIllgZbk2/UrRs3Fr4XWGnVh0r21SDQ8ZfxC9hK2eeC0zsjg2iSv6hdxBwLCHyf3nsPosNlvu4mAjRFLjcpbAC02sH+AI+g2nm/g==';

    // Cryptographic parameters
    const _0xSALT = new Uint8Array([112, 97, 110, 116, 104, 101, 111, 110, 95, 114, 111, 117, 110, 100, 52, 33]);
    const _0xIV = new Uint8Array([26, 63, 123, 156, 78, 130, 208, 101, 17, 148, 239, 40]);

    function _0xlog(msg, type) {
        const term = document.getElementById(_0xget(5));
        if (!term) return;
        const entry = document.createElement(_0xget(14));
        entry.className = _0xget(9) + ' ' + (type === 'err' ? _0xget(12) : type === 'succ' ? _0xget(11) : type === 'warn' ? _0xget(13) : _0xget(10));
        const now = new Date();
        const timeStr = '[' + String(now.getHours()).padStart(2, '0') + ':' + String(now.getMinutes()).padStart(2, '0') + ':' + String(now.getSeconds()).padStart(2, '0') + '] ';
        entry.innerText = timeStr + msg;
        term.appendChild(entry);
        term.scrollTop = term.scrollHeight;
    }

    async function _0x_validate_and_decrypt(inputKey) {
        const btn = document.getElementById(_0xget(4));
        btn.disabled = true;

        _0xlog('Receiving authorization sequence (' + inputKey.length + ' symbols)...', 'info');

        await new Promise(r => setTimeout(r, 250));
        _0xlog('Executing multi-stage S-box permutation & parity check...', 'info');

        await new Promise(r => setTimeout(r, 350));
        _0xlog('Deriving 256-bit AES master key via PBKDF2-HMAC-SHA256 (100,000 rounds)...', 'warn');

        try {
            // Convert base64 payload to bytes
            const binStr = atob(_0xCIPHER);
            const cipherBytes = new Uint8Array(binStr.length);
            for (let i = 0; i < binStr.length; i++) {
                cipherBytes[i] = binStr.charCodeAt(i);
            }

            const enc = new TextEncoder();
            const keyMaterial = await crypto.subtle.importKey(
                _0xget(29),
                enc.encode(inputKey),
                { name: _0xget(27) },
                false,
                [_0xget(31)]
            );

            const derivedKey = await crypto.subtle.deriveKey(
                {
                    name: _0xget(27),
                    salt: _0xSALT,
                    iterations: 100000,
                    hash: _0xget(26)
                },
                keyMaterial,
                { name: _0xget(28), length: 256 },
                false,
                [_0xget(32)]
            );

            const decryptedBuf = await crypto.subtle.decrypt(
                { name: _0xget(28), iv: _0xIV },
                derivedKey,
                cipherBytes
            );

            _0xlog('Cryptographic MAC tag verified! Integrity: 100%', 'succ');
            _0xlog('Unlocking and preparing ZIP payload stream...', 'succ');

            const blob = new Blob([decryptedBuf], { type: _0xget(1) });
            const url = URL.createObjectURL(blob);

            const dlCard = document.getElementById(_0xget(6));
            const dlBtn = document.getElementById(_0xget(7));
            const sysStatus = document.getElementById(_0xget(8));

            dlBtn.href = url;
            dlBtn.download = _0xget(0);
            dlCard.style.display = _0xget(21);
            sysStatus.innerText = '● VAULT UNLOCKED';
            sysStatus.style.borderColor = '#10b981';
            sysStatus.style.color = '#10b981';

            // Automatically trigger download
            dlBtn.click();
            _0xlog('Archive download initiated: ' + _0xget(0), 'succ');

        } catch (err) {
            _0xlog('CRITICAL ERROR: Authorization sequence rejected / MAC mismatch.', 'err');
            _0xlog('ACCESS DENIED. Vault Node 04 remains locked.', 'err');

            const sysStatus = document.getElementById(_0xget(8));
            sysStatus.innerText = '● ACCESS DENIED';
            sysStatus.style.borderColor = '#ef4444';
            sysStatus.style.color = '#ef4444';
        } finally {
            btn.disabled = false;
        }
    }

    window._0x_auth = function() {
        const input = document.getElementById(_0xget(3));
        if (!input) return;
        const val = input.value.trim().replace(/\s+/g, '');
        if (!val) {
            _0xlog('Input buffer empty. Please enter authorization sequence.', 'warn');
            return;
        }
        _0x_validate_and_decrypt(val);
    };

    // Auto-focus input on load
    window.addEventListener('DOMContentLoaded', () => {
        const inp = document.getElementById(_0xget(3));
        if (inp) inp.focus();
    });
})();
