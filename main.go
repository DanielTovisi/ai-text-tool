package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

const openAIURL = "https://api.openai.com/v1/chat/completions"
const model = "gpt-4o-mini" // change to a model you have access to

// --- OpenAI request/response types ---

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
}

type ChatChoice struct {
	Message ChatMessage `json:"message"`
}

type ChatResponse struct {
	Choices []ChatChoice `json:"choices"`
}

// --- API request/response types ---

type TextRequest struct {
	Text string `json:"text"`
}

type RewriteRequest struct {
	Text string `json:"text"`
	Tone string `json:"tone"`
}

type SummarizeResponse struct {
	Summary string `json:"summary"`
}

type KeywordsResponse struct {
	Keywords []string `json:"keywords"`
}

type RewriteResponse struct {
	Text string `json:"text"`
}

type QuestionsResponse struct {
	Questions []string `json:"questions"`
}

type TitlesResponse struct {
	Titles []string `json:"titles"`
}

type ExpandResponse struct {
	Text string `json:"text"`
}

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY env var is required")
	}

	mux := http.NewServeMux()

	// Web UI
	mux.HandleFunc("/", uiHandler)

	// API endpoints
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/summarize", withMethod("POST", summarizeHandler(apiKey)))
	mux.HandleFunc("/keywords", withMethod("POST", keywordsHandler(apiKey)))
	mux.HandleFunc("/rewrite", withMethod("POST", rewriteHandler(apiKey)))
	mux.HandleFunc("/questions", withMethod("POST", questionsHandler(apiKey)))
	mux.HandleFunc("/titles", withMethod("POST", titlesHandler(apiKey)))
	mux.HandleFunc("/expand", withMethod("POST", expandHandler(apiKey)))

	addr := ":8080"
	log.Printf("Server listening on %s", addr)
	if err := http.ListenAndServe(addr, logRequest(mux)); err != nil {
		log.Fatal(err)
	}
}

// --- UI handler (simple HTML + JS, no framework) ---

func uiHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(indexHTML))
}

// --- API Handlers ---

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func summarizeHandler(apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req TextRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		if req.Text == "" {
			http.Error(w, "`text` is required", http.StatusBadRequest)
			return
		}

		prompt := "Summarize the following text in 3–5 bullet points. Be concise and clear.\n\n" + req.Text
		out, err := callLLM(apiKey, prompt)
		if err != nil {
			log.Println("summarize error:", err)
			http.Error(w, "LLM error", http.StatusInternalServerError)
			return
		}

		resp := SummarizeResponse{Summary: out}
		writeJSON(w, http.StatusOK, resp)
	}
}

func keywordsHandler(apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req TextRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		if req.Text == "" {
			http.Error(w, "`text` is required", http.StatusBadRequest)
			return
		}

		prompt := `Extract 5–10 key keywords from the text below.
Return ONLY a JSON array of strings. Example: ["keyword1","keyword2"].

Text:
` + req.Text

		out, err := callLLM(apiKey, prompt)
		if err != nil {
			log.Println("keywords error:", err)
			http.Error(w, "LLM error", http.StatusInternalServerError)
			return
		}

		var kws []string
		if err := json.Unmarshal([]byte(out), &kws); err != nil {
			// fallback – try to be robust
			kws = []string{out}
		}

		resp := KeywordsResponse{Keywords: kws}
		writeJSON(w, http.StatusOK, resp)
	}
}

func rewriteHandler(apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RewriteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		if req.Text == "" {
			http.Error(w, "`text` is required", http.StatusBadRequest)
			return
		}
		tone := req.Tone
		if tone == "" {
			tone = "neutral"
		}

		prompt := fmt.Sprintf(
			"Rewrite the following text in a %s tone. Preserve the original meaning. Respond with ONLY the rewritten text.\n\n%s",
			tone, req.Text,
		)

		out, err := callLLM(apiKey, prompt)
		if err != nil {
			log.Println("rewrite error:", err)
			http.Error(w, "LLM error", http.StatusInternalServerError)
			return
		}

		resp := RewriteResponse{Text: out}
		writeJSON(w, http.StatusOK, resp)
	}
}

func questionsHandler(apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req TextRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		if req.Text == "" {
			http.Error(w, "`text` is required", http.StatusBadRequest)
			return
		}

		prompt := `From the text below, generate 5–10 clear, helpful questions.
Return ONLY a JSON array of strings. Example: ["Question 1?", "Question 2?"].

Text:
` + req.Text

		out, err := callLLM(apiKey, prompt)
		if err != nil {
			log.Println("questions error:", err)
			http.Error(w, "LLM error", http.StatusInternalServerError)
			return
		}

		var qs []string
		if err := json.Unmarshal([]byte(out), &qs); err != nil {
			// fallback – just put the raw output
			qs = []string{out}
		}

		resp := QuestionsResponse{Questions: qs}
		writeJSON(w, http.StatusOK, resp)
	}
}

func titlesHandler(apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req TextRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		if req.Text == "" {
			http.Error(w, "`text` is required", http.StatusBadRequest)
			return
		}

		prompt := `Generate 5 concise, engaging title ideas for the text below.
Return ONLY a JSON array of strings. Example: ["Title 1", "Title 2"].

Text:
` + req.Text

		out, err := callLLM(apiKey, prompt)
		if err != nil {
			log.Println("titles error:", err)
			http.Error(w, "LLM error", http.StatusInternalServerError)
			return
		}

		var titles []string
		if err := json.Unmarshal([]byte(out), &titles); err != nil {
			titles = []string{out}
		}

		resp := TitlesResponse{Titles: titles}
		writeJSON(w, http.StatusOK, resp)
	}
}

func expandHandler(apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req TextRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		if req.Text == "" {
			http.Error(w, "`text` is required", http.StatusBadRequest)
			return
		}

		prompt := `Expand and elaborate on the following text.
Add helpful explanations and details but keep it clear and readable.
Respond with ONLY the expanded text.

Text:
` + req.Text

		out, err := callLLM(apiKey, prompt)
		if err != nil {
			log.Println("expand error:", err)
			http.Error(w, "LLM error", http.StatusInternalServerError)
			return
		}

		resp := ExpandResponse{Text: out}
		writeJSON(w, http.StatusOK, resp)
	}
}

// --- LLM call helper ---

func callLLM(apiKey, prompt string) (string, error) {
	body := ChatRequest{
		Model: model,
		Messages: []ChatMessage{
			{Role: "system", Content: "You are a helpful text-processing assistant."},
			{Role: "user", Content: prompt},
		},
	}

	data, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", openAIURL, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenAI error: status=%d body=%s", resp.StatusCode, string(b))
	}

	var cr ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return "", err
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("no choices from LLM")
	}

	return cr.Choices[0].Message.Content, nil
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Println("writeJSON error:", err)
	}
}

func withMethod(method string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h(w, r)
	}
}

func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

// --- HTML UI (vanilla, no frameworks) ---

const indexHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <title>AI Text Tools</title>
  <style>
    body {
      font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      max-width: 1000px;
      margin: 40px auto;
      padding: 0 16px;
      background: #f5f5f7;
      color: #222;
    }
    h1 {
      text-align: center;
      margin-bottom: 8px;
    }
    p.subtitle {
      text-align: center;
      color: #6b7280;
      margin-bottom: 24px;
    }
    .card {
      background: white;
      padding: 16px 20px;
      border-radius: 12px;
      box-shadow: 0 4px 12px rgba(0,0,0,0.06);
      margin-bottom: 20px;
    }
    textarea {
      width: 100%;
      min-height: 150px;
      resize: vertical;
      padding: 10px;
      font-size: 14px;
      border-radius: 8px;
      border: 1px solid #ccc;
      box-sizing: border-box;
      font-family: inherit;
    }
    button {
      border: none;
      padding: 8px 14px;
      border-radius: 8px;
      font-size: 13px;
      cursor: pointer;
      margin-right: 8px;
      margin-bottom: 8px;
    }
    button.primary {
      background: #2563eb;
      color: white;
    }
    button.secondary {
      background: #e5e7eb;
      color: #111827;
    }
    button:disabled {
      opacity: 0.6;
      cursor: wait;
    }
    .label {
      font-weight: 600;
      margin-bottom: 4px;
      display: block;
    }
    pre {
      background: #111827;
      color: #e5e7eb;
      padding: 12px;
      border-radius: 8px;
      white-space: pre-wrap;
      word-wrap: break-word;
      font-size: 13px;
      max-height: 260px;
      overflow-y: auto;
    }
    .grid {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 16px;
    }
    @media (max-width: 800px) {
      .grid {
        grid-template-columns: 1fr;
      }
    }
    .status {
      font-size: 12px;
      color: #6b7280;
      margin-top: 6px;
      min-height: 18px;
    }
    select {
      padding: 6px 10px;
      border-radius: 8px;
      border: 1px solid #ccc;
      font-size: 13px;
      margin-left: 8px;
    }
  </style>
</head>
<body>
  <h1>AI Text Tools</h1>
  <p class="subtitle">Summarize, extract keywords, rewrite with tone, generate questions, titles, and expansions.</p>

  <div class="card">
    <label class="label" for="input">Input text</label>
    <textarea id="input" placeholder="Paste or type some text here..."></textarea>

    <div style="margin-top: 10px; margin-bottom: 8px;">
      <span class="label" style="display:inline; font-size:13px;">Rewrite tone:</span>
      <select id="tone">
        <option value="neutral">Neutral</option>
        <option value="formal">Formal</option>
        <option value="informal">Informal</option>
        <option value="friendly">Friendly</option>
        <option value="professional">Professional</option>
        <option value="persuasive">Persuasive</option>
      </select>
    </div>

    <div class="buttons">
      <button id="btnSummarize" class="primary">Summarize</button>
      <button id="btnKeywords" class="secondary">Keywords</button>
      <button id="btnRewrite" class="secondary">Rewrite</button>
      <button id="btnQuestions" class="secondary">Questions</button>
      <button id="btnTitles" class="secondary">Titles</button>
      <button id="btnExpand" class="secondary">Expand</button>
    </div>

    <div id="status" class="status"></div>
  </div>

  <div class="grid">
    <div class="card">
      <div class="label">Summary</div>
      <pre id="summaryOutput">–</pre>
    </div>

    <div class="card">
      <div class="label">Keywords</div>
      <pre id="keywordsOutput">–</pre>
    </div>

    <div class="card">
      <div class="label">Rewrite</div>
      <pre id="rewriteOutput">–</pre>
    </div>

    <div class="card">
      <div class="label">Questions</div>
      <pre id="questionsOutput">–</pre>
    </div>

    <div class="card">
      <div class="label">Titles</div>
      <pre id="titlesOutput">–</pre>
    </div>

    <div class="card">
      <div class="label">Expand</div>
      <pre id="expandOutput">–</pre>
    </div>
  </div>

  <script>
    const inputEl        = document.getElementById('input');
    const toneEl         = document.getElementById('tone');
    const btnSummarize   = document.getElementById('btnSummarize');
    const btnKeywords    = document.getElementById('btnKeywords');
    const btnRewrite     = document.getElementById('btnRewrite');
    const btnQuestions   = document.getElementById('btnQuestions');
    const btnTitles      = document.getElementById('btnTitles');
    const btnExpand      = document.getElementById('btnExpand');
    const summaryOutput  = document.getElementById('summaryOutput');
    const keywordsOutput = document.getElementById('keywordsOutput');
    const rewriteOutput  = document.getElementById('rewriteOutput');
    const questionsOutput= document.getElementById('questionsOutput');
    const titlesOutput   = document.getElementById('titlesOutput');
    const expandOutput   = document.getElementById('expandOutput');
    const statusEl       = document.getElementById('status');

    const allButtons = [
      btnSummarize,
      btnKeywords,
      btnRewrite,
      btnQuestions,
      btnTitles,
      btnExpand,
    ];

    function setLoading(isLoading, msg) {
      allButtons.forEach(b => b.disabled = isLoading);
      statusEl.textContent = isLoading ? (msg || 'Working...') : '';
    }

    async function callAPI(path, body) {
      const text = (body && body.text) || inputEl.value.trim();
      if (!text) {
        alert('Please enter some text first.');
        return null;
      }

      setLoading(true, 'Calling ' + path + ' ...');

      try {
        const res = await fetch(path, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(body || { text }),
        });
        if (!res.ok) {
          const errText = await res.text();
          throw new Error('HTTP ' + res.status + ': ' + errText);
        }
        const data = await res.json();
        setLoading(false);
        return data;
      } catch (err) {
        console.error(err);
        alert('Error: ' + err.message);
        setLoading(false, 'Error – see console.');
        return null;
      }
    }

    btnSummarize.addEventListener('click', async () => {
      const data = await callAPI('/summarize', { text: inputEl.value.trim() });
      if (!data) return;
      summaryOutput.textContent = data.summary || '(no summary)';
    });

    btnKeywords.addEventListener('click', async () => {
      const data = await callAPI('/keywords', { text: inputEl.value.trim() });
      if (!data) return;
      if (Array.isArray(data.keywords)) {
        keywordsOutput.textContent = data.keywords.join(', ');
      } else {
        keywordsOutput.textContent = JSON.stringify(data, null, 2);
      }
    });

    btnRewrite.addEventListener('click', async () => {
      const data = await callAPI('/rewrite', {
        text: inputEl.value.trim(),
        tone: toneEl.value,
      });
      if (!data) return;
      rewriteOutput.textContent = data.text || '(no rewrite)';
    });

    btnQuestions.addEventListener('click', async () => {
      const data = await callAPI('/questions', { text: inputEl.value.trim() });
      if (!data) return;
      if (Array.isArray(data.questions)) {
        questionsOutput.textContent = data.questions.map(q => '- ' + q).join('\n');
      } else {
        questionsOutput.textContent = JSON.stringify(data, null, 2);
      }
    });

    btnTitles.addEventListener('click', async () => {
      const data = await callAPI('/titles', { text: inputEl.value.trim() });
      if (!data) return;
      if (Array.isArray(data.titles)) {
        titlesOutput.textContent = data.titles.map(t => '- ' + t).join('\n');
      } else {
        titlesOutput.textContent = JSON.stringify(data, null, 2);
      }
    });

    btnExpand.addEventListener('click', async () => {
      const data = await callAPI('/expand', { text: inputEl.value.trim() });
      if (!data) return;
      expandOutput.textContent = data.text || '(no expansion)';
    });
  </script>
</body>
</html>
`
