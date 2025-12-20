// Package llm - extractor.go provides generic LLM-based structured extraction.
package llm

import (
	"fmt"
	"strings"
)

// ExtractionSchema defines the structure for LLM-based content extraction.
// It provides a reusable way to define what information to extract from text.
type ExtractionSchema struct {
	Name        string        // Schema name (e.g., "JobRequirements", "BrandVoice")
	Description string        // System prompt preamble describing the extraction task
	Fields      []SchemaField // Expected output fields
}

// SchemaField defines a single field in the extraction output.
type SchemaField struct {
	Name        string // JSON field name
	Type        string // Type hint: "string", "[]string", "map[string]string"
	Description string // Description for the LLM
	Required    bool   // Whether this field is required
}

// BuildExtractionPrompt constructs the LLM prompt from schema and input text.
func BuildExtractionPrompt(schema ExtractionSchema, inputText string) string {
	var sb strings.Builder

	// System description
	sb.WriteString(schema.Description)
	sb.WriteString("\n\n")

	// Output schema
	sb.WriteString("Return ONLY valid JSON matching this exact structure:\n{\n")
	for i, field := range schema.Fields {
		typeHint := field.Type
		if typeHint == "" {
			typeHint = "string"
		}
		requiredHint := ""
		if field.Required {
			requiredHint = " (required)"
		}
		sb.WriteString(fmt.Sprintf("  \"%s\": %s%s", field.Name, typeHint, requiredHint))
		if field.Description != "" {
			sb.WriteString(fmt.Sprintf(" // %s", field.Description))
		}
		if i < len(schema.Fields)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}
	sb.WriteString("}\n\n")

	// Instructions
	sb.WriteString("IMPORTANT:\n")
	sb.WriteString("- Extract information directly from the text, do not invent or summarize.\n")
	sb.WriteString("- Return ONLY the JSON object, no markdown, no explanation, no code blocks.\n\n")

	// Input text
	sb.WriteString("Input text:\n\"\"\"\n")
	sb.WriteString(inputText)
	sb.WriteString("\n\"\"\"\n")

	return sb.String()
}

// --- Predefined Schemas ---

// JobRequirementsSchema returns the extraction schema for job postings.
// Extracts team context, requirements, responsibilities, and administrative metadata.
func JobRequirementsSchema() ExtractionSchema {
	return ExtractionSchema{
		Name: "JobRequirements",
		Description: `You are an expert job posting parser. COPY TEXT VERBATIM - do not paraphrase, summarize, or reword.
Your task is to extract and categorize information from a raw job posting.
IMPORTANT: Preserve the exact wording from the original text.
Goal: Extract team context, requirements, responsibilities, and administrative metadata.
EXCLUDE: Application form fields, EEO statements, legal disclaimers, generic "About Company" boilerplate.`,
		Fields: []SchemaField{
			{
				Name:        "team_context",
				Type:        "\"string\"",
				Description: "Team name, organization, team description - include ALL context about the team/org verbatim",
				Required:    false,
			},
			{
				Name:        "requirements",
				Type:        "[\"string\"]",
				Description: "Technical requirements, qualifications, skills needed - copy each requirement verbatim",
				Required:    true,
			},
			{
				Name:        "responsibilities",
				Type:        "[\"string\"]",
				Description: "Job duties, day-to-day work - copy each responsibility verbatim",
				Required:    true,
			},
			{
				Name:        "nice_to_have",
				Type:        "[\"string\"]",
				Description: "Preferred skills, nice-to-have qualifications - copy verbatim",
				Required:    false,
			},
			{
				Name:        "admin_info",
				Type:        "{\"key\": \"value\"}",
				Description: "Salary, clearance, citizenship, location, job ID - extract key-value pairs",
				Required:    false,
			},
		},
	}
}

// BrandVoiceSchema returns the extraction schema for company brand voice analysis.
// Extracts tone, style rules, taboo phrases, values, and domain context.
func BrandVoiceSchema() ExtractionSchema {
	return ExtractionSchema{
		Name: "BrandVoice",
		Description: `You are an expert brand analyst. Your task is to extract brand voice characteristics from company text.
Analyze the tone, communication style, and values expressed in the content.`,
		Fields: []SchemaField{
			{
				Name:        "company",
				Type:        "\"string\"",
				Description: "Company name",
				Required:    true,
			},
			{
				Name:        "tone",
				Type:        "\"string\"",
				Description: "Brand tone (e.g., 'direct, metric-driven', 'collaborative, values-driven')",
				Required:    true,
			},
			{
				Name:        "style_rules",
				Type:        "[\"string\"]",
				Description: "Actionable style guidelines (e.g., 'lead with metrics', 'avoid hype')",
				Required:    true,
			},
			{
				Name:        "taboo_phrases",
				Type:        "[\"string\"]",
				Description: "Words/phrases the company avoids or criticizes",
				Required:    true,
			},
			{
				Name:        "domain_context",
				Type:        "\"string\"",
				Description: "Industry/domain context (e.g., 'B2B SaaS, infrastructure')",
				Required:    true,
			},
			{
				Name:        "values",
				Type:        "[\"string\"]",
				Description: "Core company values extracted from the text",
				Required:    true,
			},
		},
	}
}
