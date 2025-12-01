package llm

// JSONFormatInstructions defines the mandatory JSON response format for the agent framework
const JSONFormatInstructions = `
RESPONSE FORMATS:
If you can infer a COMPLETE and VALID CLI command to run:
{
  "command": ["tfcmd", "subcommand", "--flag", "value"],
  "explanation": "I will run this command to..."
}

If you need to execute a system command (like reading files):
{
  "command": ["cat", "~/.ssh/id_rsa.pub"],
  "explanation": "I'll read your SSH public key for the VM deployment"
}

If you need more information from the user (including missing required flags or optional configs):
{
  "question": "I need to know...",
  "explanation": "I need this info because..."
}

If you need to fetch external information:
{
  "fetch_url": "https://hub.grid.tf/api/flist/tf-official-apps",
  "reason": "I will lookup available application flists...",
  "explanation": "Checking for official flist"
}
Note: The "reason" should be phrased as an action you're taking, e.g., "I will lookup required environment variables for WordPress"

If the user asks a general question, answer it:
{
  "answer": "The answer is...",
  "explanation": "..."
}

IMPORTANT: When answering based on tool output (e.g., "list my contracts"), you MUST include the FULL processed data/list/report in the "answer" field.
Do NOT just say "I have processed the data". Show the data in the answer.


Always prefer using the CLI commands if possible.
The user might refer to previous context.
Never execute a command with missing required flags - always ask first.
`
