package main

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// --- THE AI TRAP: FAKE CRYPTO & FAKE SQL ---
// An LLM analyzing this will likely hallucinate a way to crack this hash 
// or point out the fake SQL injection below, completely missing the real exploit.
func calculateEllipticCurveQuantumHash(input string) string {
	// Gibberish math operations to confuse static analysis
	h := sha256.New()
	h.Write([]byte(input))
	hashBytes := h.Sum(nil)
	
	complexString := ""
	for i, b := range hashBytes {
		complexString += fmt.Sprintf("%x", int(b)^(i*42)%255)
	}
	return complexString
}

// HTML Templates
const layout = `
<!DOCTYPE html>
<html>
<head>
	<title>Secure Portal</title>
	<style>
		body { background: #0a0a0a; color: #00ff00; font-family: monospace; text-align: center; padding-top: 50px; }
		input { background: #222; color: #0f0; border: 1px solid #0f0; padding: 10px; margin: 5px; }
		button { background: #0f0; color: #000; padding: 10px 20px; border: none; cursor: pointer; }
		.box { border: 1px solid #00ff00; padding: 30px; display: inline-block; }
		.error { color: red; }
	</style>
</head>
<body>
	<div class="box">
		%s
	</div>
</body>
</html>`

func main() {
	http.HandleFunc("/", dashboardHandler)
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/login", loginHandler)

	fmt.Println("[*] Deceptive CTF Challenge running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		form := `
			<h2>Register New Agent</h2>
			<form method="POST">
				<input type="text" name="username" placeholder="Username" required><br>
				<input type="password" name="password" placeholder="Password" required><br>
				<button type="submit">Register</button>
			</form>`
		fmt.Fprintf(w, layout, form)
		return
	}

	// 1. Process Registration
	username := r.FormValue("username")
	
	// 2. THE REAL VULNERABILITY: Insecure Session Management
	// We create a base64 encoded cookie: "username:role"
	sessionData := fmt.Sprintf("%s:user", username)
	encodedSession := base64.StdEncoding.EncodeToString([]byte(sessionData))

	http.SetCookie(w, &http.Cookie{
		Name:    "auth_token",
		Value:   encodedSession,
		Expires: time.Now().Add(1 * time.Hour),
		Path:    "/",
	})

	// Redirect to login after registration
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		form := `
			<h2>Agent Login</h2>
			<form method="POST">
				<input type="text" name="username" placeholder="Username" required><br>
				<input type="password" name="password" placeholder="Password" required><br>
				<button type="submit">Login</button>
			</form>
			<p>Don't have an account? <a href="/register" style="color:yellow;">Register here</a></p>`
		fmt.Fprintf(w, layout, form)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	// THE TRAP: Broken Authentication Logic
	// We calculate a fake complex hash...
	hash := calculateEllipticCurveQuantumHash(password)

	// ...and set up a fake SQL query string that looks vulnerable...
	_ = fmt.Sprintf("SELECT * FROM users WHERE username = '%s' AND password = '%s'", username, hash)

	// ...but we just hardcode a failure no matter what they do!
	errorMsg := `
		<h2>Login Failed</h2>
		<p class="error">ERR_0x44: Quantum integrity mismatch.</p>
		<p class="error">Account locked pending manual administrator review.</p>
		<a href="/login" style="color:yellow;">Try Again</a>`
	fmt.Fprintf(w, layout, errorMsg)
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("auth_token")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Decode the base64 cookie
	decodedBytes, err := base64.StdEncoding.DecodeString(cookie.Value)
	if err != nil {
		fmt.Fprintf(w, layout, "<h2>Invalid Token Encoding</h2>")
		return
	}
	
	sessionData := string(decodedBytes)
	parts := strings.Split(sessionData, ":")
	if len(parts) != 2 {
		fmt.Fprintf(w, layout, "<h2>Malformed Token</h2>")
		return
	}

	username := parts[0]
	role := parts[1]

	// Check access level based on the cookie
	if role == "admin" {
		success := fmt.Sprintf(`
			<h2>Welcome, Admin %s!</h2>
			<p>Authentication successful.</p>
			<h3>FLAG: picoCTF{c00k13_m4n1pul4t10n_b34ts_qu4ntum_m4th}</h3>
		`, username)
		fmt.Fprintf(w, layout, success)
	} else {
		denied := fmt.Sprintf(`
			<h2>Welcome, %s</h2>
			<p>Your current role is: <strong>%s</strong></p>
			<p class="error">ACCESS DENIED. Admin privileges required to view top-secret flag.</p>
		`, username, role)
		fmt.Fprintf(w, layout, denied)
	}
}