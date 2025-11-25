📘 AI Text Tools (Go + OpenAI)

A lightweight, fast, and self-contained Go web application that provides multiple AI-powered text transformations using OpenAI’s Chat Completions API.
Includes a simple browser UI (no frameworks) and a REST API.

✨ Features
🔹 Text Processing Tools

Summarize — condense text into 3–5 bullet points

Keywords — extract 5–10 key terms

Rewrite — rewrite text in a chosen tone (formal, friendly, persuasive, etc.)

Questions — generate comprehension questions

Titles — produce 5 title ideas

Expand — expand and elaborate text

🔹 UI

Clean, simple HTML + vanilla JS

No build tools, no frameworks, no dependencies

Everything runs inside 1 Go server

🔹 Backend

Pure Go

Minimal dependencies (only stdlib)

7 REST endpoints

🚀 Demo (local)

Start the server:

export OPENAI_API_KEY="your-key-here"
go run main.go


Then open:

http://localhost:8080

🛠 API Endpoints
POST /summarize
{
  "text": "Your text here..."
}

POST /keywords
{
  "text": "Your text here..."
}

POST /rewrite
{
  "text": "Your text",
  "tone": "friendly"
}

POST /questions
{
  "text": "Your text"
}

POST /titles
{
  "text": "Your text"
}

POST /expand
{
  "text": "Your text"
}


All endpoints return JSON.

🧩 Project Structure
ai-text-tools/
├── main.go      # full backend + frontend UI
└── README.md    # this file


The entire app (backend + UI) is contained in main.go.
🧪 Example curl Commands
Summarize:
curl -X POST http://localhost:8080/summarize \
  -H "Content-Type: application/json" \
  -d '{"text":"Your text here"}'

Rewrite:
curl -X POST http://localhost:8080/rewrite \
  -H "Content-Type: application/json" \
  -d '{"text":"Hello world","tone":"friendly"}'

📜 License

MIT License

👤 Author

Daniel Tovisi
Built with ❤️ using Go + OpenAI
