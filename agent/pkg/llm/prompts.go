package llm

// JSONFormatInstructions defines the mandatory JSON response format for the agent framework
const JSONFormatInstructions = `
### RESPONSE FORMAT STANDARDS
You must reply with a SINGLE or ARRAY of JSON objects matching one of the 4 schemas below.

1. EXECUTE COMMAND (For 'tfcmd' OR System commands like 'cat/ls'):
{
  "command": ["binary", "arg1", "--flag", "value"],
  "explanation": "I will [read file/deploy vm] to..."
}

2. FETCH EXTERNAL DATA (Flists, Docs, Swagger):
{
  "fetch_url": "https://...",
  "reason": "Action phrase (e.g., 'I will lookup WordPress env vars')",
  "explanation": "Checking official sources..."
}

3. REQUEST INPUT (Missing flags, ambiguity, or confirmation):
{
  "question": "I need to know [specific missing flag/config]...",
  "explanation": "This is required because..."
}
*CRITICAL:* Never guess required flags. If a required flag is missing, use this format to ask.

4. TEXT ANSWER (General chat or Reporting Tool Output):
{
  "answer": "Your full response here...",
  "explanation": "Context."
}
*CRITICAL:* When reporting data (e.g., "list my contracts"), the "answer" field MUST contain the FULL list/report from the tool output. Do not summarize or say "I have the data." Show it.
`
