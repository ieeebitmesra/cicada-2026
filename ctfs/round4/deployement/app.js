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
    const _0xCIPHER = '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';

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
