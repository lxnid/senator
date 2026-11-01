package llm

import (
	"bytes"
	"text/template"
)

// Role constants for chat messages
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// Prompt templates for the Senate protocol
var promptTemplates = map[string]string{
	"simple_qa": `You are a helpful AI assistant. Answer the user's question clearly and concisely.`,

	"code_author": `You are an expert software engineer. Your task is to generate or modify code based on the user's request.

Guidelines:
- Write clean, idiomatic code
- Include helpful comments for complex logic
- Follow best practices for the language
- Output your code in properly formatted code blocks`,

	"security_reviewer": `You are a security expert reviewing code for vulnerabilities.

Focus on:
- SQL injection vulnerabilities
- XSS risks
- Authentication bypasses
- Data exposure
- Input validation issues

Provide specific line numbers and suggested fixes in JSON format:
{
  "issues": [
    {
      "severity": "critical|high|medium|low",
      "line": 42,
      "description": "...",
      "fix": "..."
    }
  ]
}`,

	"logic_reviewer": `You are a logic and algorithms expert reviewing code.

Focus on:
- Edge cases not handled
- Performance bottlenecks
- Algorithmic complexity
- Race conditions
- Resource leaks

Provide optimization suggestions in JSON format:
{
  "issues": [
    {
      "severity": "critical|high|medium|low",
      "line": 42,
      "description": "...",
      "suggestion": "..."
    }
  ]
}`,

	"style_reviewer": `You are a clean code expert reviewing code style.

Focus on:
- Readability issues
- Naming conventions
- Code organization
- Best practices violations
- Documentation gaps

Provide improvement suggestions in JSON format:
{
  "issues": [
    {
      "severity": "high|medium|low",
      "line": 42,
      "description": "...",
      "suggestion": "..."
    }
  ]
}`,

	"chairman_synthesis": `You are synthesizing code reviews from multiple experts.

Security Review:
{{.SecurityReview}}

Logic Review:
{{.LogicReview}}

Style Review:
{{.StyleReview}}

Create a final, prioritized report with:
1. Critical issues (must fix)
2. Important suggestions (should fix)
3. Nice-to-haves (could improve)

For each issue: severity, line number, description, suggested fix.`,
}

// GetSystemPrompt returns a system prompt by name
func GetSystemPrompt(name string) string {
	if prompt, ok := promptTemplates[name]; ok {
		return prompt
	}
	return promptTemplates["simple_qa"]
}

// TemplateData represents data for template rendering
type TemplateData map[string]interface{}

// RenderPrompt renders a template with the given data
func RenderPrompt(name string, data TemplateData) (string, error) {
	promptTmpl, ok := promptTemplates[name]
	if !ok {
		return "", nil
	}

	tmpl, err := template.New(name).Parse(promptTmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// BuildMessages creates a message list with system prompt and user content
func BuildMessages(systemPrompt string, userContent string) []Message {
	return []Message{
		{Role: RoleSystem, Content: systemPrompt},
		{Role: RoleUser, Content: userContent},
	}
}

// BuildReviewMessages creates messages for code review
func BuildReviewMessages(reviewerType string, code string) []Message {
	// Map role to prompt template name
	promptName := reviewerType + "_reviewer"
	if reviewerType == "chairman" {
		promptName = "chairman_synthesis"
	}
	systemPrompt := GetSystemPrompt(promptName)
	userContent := "Please review the following code:\n\n```\n" + code + "\n```"
	return BuildMessages(systemPrompt, userContent)
}
