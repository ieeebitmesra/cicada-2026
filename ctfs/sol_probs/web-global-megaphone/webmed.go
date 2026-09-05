package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
)

// SecretData holds the flag that the template engine has access to.
type SecretData struct {
	Flag string
}

// PageData holds the data injected into our main HTML UI.
type PageData struct {
	Announcement template.HTML
}

// The main UI for the website (HTML + CSS)
const htmlTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Announcement Board</title>
    <style>
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background-color: #121212;
            color: #e0e0e0;
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
            margin: 0;
        }
        .container {
            background-color: #1e1e1e;
            padding: 40px;
            border-radius: 12px;
            box-shadow: 0 8px 16px rgba(0,0,0,0.5);
            max-width: 600px;
            width: 100%;
        }
        h1 {
            color: #ff9800;
            margin-top: 0;
        }
        p.subtitle {
            color: #aaaaaa;
            font-size: 0.95em;
            line-height: 1.5;
            margin-bottom: 25px;
        }
        textarea {
            width: 100%;
            padding: 15px;
            border-radius: 8px;
            border: 1px solid #333;
            background-color: #2c2c2c;
            color: #ffffff;
            box-sizing: border-box;
            font-family: monospace;
            resize: vertical;
        }
        textarea:focus {
            outline: none;
            border-color: #ff9800;
        }
        button {
            background-color: #ff9800;
            color: #121212;
            border: none;
            padding: 12px 24px;
            margin-top: 15px;
            border-radius: 6px;
            cursor: pointer;
            font-weight: bold;
            font-size: 1em;
            transition: background-color 0.2s;
        }
        button:hover {
            background-color: #e68a00;
        }
        .result-box {
            margin-top: 30px;
            padding: 20px;
            background-color: #252525;
            border-left: 5px solid #4caf50;
            border-radius: 6px;
        }
        .result-box h3 {
            margin-top: 0;
            color: #4caf50;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Global Megaphone 📣</h1>
        <p class="subtitle">
            I made a cool website where you can announce whatever you want! <br><br>
            I read about input sanitization, so now I remove any kind of characters that could be a problem :) I heard templating is a cool and modular way to build web apps!
        </p>
        
        <form method="POST" action="/">
            <textarea name="announcement" rows="5" placeholder="What's on your mind?"></textarea>
            <br>
            <button type="submit">Make Announcement</button>
        </form>

        {{if .Announcement}}
        <div class="result-box">
            <h3>Latest Announcement:</h3>
            <div>{{.Announcement}}</div>
        </div>
        {{end}}
    </div>
</body>
</html>
`

func handler(w http.ResponseWriter, r *http.Request) {
	// Parse the main UI template
	uiTmpl, err := template.New("index").Parse(htmlTemplate)
	if err != nil {
		http.Error(w, "Internal Server Error", 500)
		return
	}

	// Handle GET requests (just show the form)
	if r.Method == http.MethodGet {
		uiTmpl.Execute(w, PageData{})
		return
	}

	// Handle POST requests
	r.ParseForm()
	userInput := r.FormValue("announcement")

	// 1. The Flawed Sanitization
	// The developer removes the word "Flag" to prevent {{ .Flag }}
	sanitizedInput := strings.ReplaceAll(userInput, "Flag", "")

	// 2. The Vulnerability (SSTI)
	// The sanitized user input is compiled directly as a template
	userTmpl, err := template.New("user_input").Parse(sanitizedInput)
	
	var renderedAnnouncement string
	if err != nil {
		// If the user's template syntax is broken, show the error
		renderedAnnouncement = fmt.Sprintf("<em>Template Syntax Error: %s</em>", err.Error())
	} else {
		// Execute the user's template against our secret data
		var buf bytes.Buffer
		secret := SecretData{Flag: "PANTHEON{ssti_g0_t3mpl4t3_https://pwn-stonks-fmt.onrender.com}"}
		
		err = userTmpl.Execute(&buf, secret)
		if err != nil {
			renderedAnnouncement = "<em>Execution Error</em>"
		} else {
			renderedAnnouncement = buf.String()
		}
	}

	// 3. Render the main UI, passing in the processed user announcement
	// We use template.HTML to tell Go to render the HTML tags literally
	uiTmpl.Execute(w, PageData{
		Announcement: template.HTML(renderedAnnouncement),
	})
}

func main() {
	http.HandleFunc("/", handler)
	fmt.Println("[*] CTF Challenge listening on http://0.0.0.0:8080")
	fmt.Println("[*] Press Ctrl+C to stop.")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
