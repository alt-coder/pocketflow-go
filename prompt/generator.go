package prompt

import (
	"fmt"
	"reflect"
	"strings"
)

// GenerateStructuredPrompt creates an instruction prompt for parsing data into type T
// It analyzes the struct fields, yaml tags, and description tags to build comprehensive instructions
func GenerateStructuredPrompt[T any]() string {
	var zero T
	t := reflect.TypeOf(zero)

	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return fmt.Sprintf("Please format the output as a valid %s value.", t.Name())
	}

	var builder strings.Builder
	builder.WriteString("Please analyze the provided data and extract information in the following structured format:\n\n")

	// Add YAML format instruction if yaml tags are present
	if hasYamlTags(t) {
		builder.WriteString("Output the result in YAML format with the following structure:\n\n")
		builder.WriteString("```yaml\n")
		writeYamlStructure(t, &builder, 0)
		builder.WriteString("```\n\n")
	} else {
		builder.WriteString("Output the result in JSON format with the following structure:\n\n")
		builder.WriteString("```json\n")
		writeJsonStructure(t, &builder, 0)
		builder.WriteString("```\n\n")
	}

	// Add field descriptions
	builder.WriteString("Field descriptions:\n")
	writeFieldDescriptions(t, &builder, "")

	builder.WriteString("\nEnsure all fields are properly filled based on the available data. If a field cannot be determined from the data, use appropriate default values or leave empty as applicable.")

	return builder.String()
}

// hasYamlTags checks if the struct has yaml tags on any field
func hasYamlTags(t reflect.Type) bool {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if _, ok := field.Tag.Lookup("yaml"); ok {
			return true
		}
		// Check nested structs
		if field.Type.Kind() == reflect.Struct {
			if hasYamlTags(field.Type) {
				return true
			}
		}
		if field.Type.Kind() == reflect.Ptr && field.Type.Elem().Kind() == reflect.Struct {
			if hasYamlTags(field.Type.Elem()) {
				return true
			}
		}
	}
	return false
}

// writeYamlStructure writes the YAML structure representation
func writeYamlStructure(t reflect.Type, builder *strings.Builder, indent int) {
	indentStr := strings.Repeat("  ", indent)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		yamlTag := getYamlFieldName(field)
		if yamlTag == "-" {
			continue
		}

		builder.WriteString(fmt.Sprintf("%s%s: ", indentStr, yamlTag))

		fieldType := field.Type
		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}

		switch fieldType.Kind() {
		case reflect.Struct:
			builder.WriteString("\n")
			writeYamlStructure(fieldType, builder, indent+1)
		case reflect.Slice:
			elemType := fieldType.Elem()
			if elemType.Kind() == reflect.Ptr {
				elemType = elemType.Elem()
			}
			if elemType.Kind() == reflect.Struct {
				builder.WriteString("\n")
				builder.WriteString(fmt.Sprintf("%s  - ", indentStr))
				builder.WriteString("\n")
				writeYamlStructure(elemType, builder, indent+2)
			} else {
				builder.WriteString(fmt.Sprintf("[] # array of %s\n", elemType.Kind()))
			}
		default:
			builder.WriteString(fmt.Sprintf("\"\" # %s\n", fieldType.Kind()))
		}
	}
}

// writeJsonStructure writes the JSON structure representation
func writeJsonStructure(t reflect.Type, builder *strings.Builder, indent int) {
	indentStr := strings.Repeat("  ", indent)
	builder.WriteString("{\n")

	fieldCount := 0
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		jsonTag := getJsonFieldName(field)
		if jsonTag == "-" {
			continue
		}

		if fieldCount > 0 {
			builder.WriteString(",\n")
		}

		builder.WriteString(fmt.Sprintf("%s  \"%s\": ", indentStr, jsonTag))

		fieldType := field.Type
		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}

		switch fieldType.Kind() {
		case reflect.Struct:
			writeJsonStructure(fieldType, builder, indent+1)
		case reflect.Slice:
			elemType := fieldType.Elem()
			if elemType.Kind() == reflect.Ptr {
				elemType = elemType.Elem()
			}
			if elemType.Kind() == reflect.Struct {
				builder.WriteString("[\n")
				builder.WriteString(fmt.Sprintf("%s    ", indentStr))
				writeJsonStructure(elemType, builder, indent+2)
				builder.WriteString(fmt.Sprintf("\n%s  ]", indentStr))
			} else {
				builder.WriteString("[]")
			}
		case reflect.String:
			builder.WriteString("\"\"")
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			builder.WriteString("0")
		case reflect.Float32, reflect.Float64:
			builder.WriteString("0.0")
		case reflect.Bool:
			builder.WriteString("false")
		default:
			builder.WriteString("null")
		}

		fieldCount++
	}

	builder.WriteString(fmt.Sprintf("\n%s}", indentStr))
}

// writeFieldDescriptions writes detailed field descriptions
func writeFieldDescriptions(t reflect.Type, builder *strings.Builder, prefix string) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		fieldName := getFieldDisplayName(field)
		if fieldName == "-" {
			continue
		}

		fullFieldName := fieldName
		if prefix != "" {
			fullFieldName = prefix + "." + fieldName
		}

		// Get description from tag
		description := field.Tag.Get("description")
		if description == "" {
			description = fmt.Sprintf("Field of type %s", field.Type.String())
		}

		builder.WriteString(fmt.Sprintf("- %s: %s\n", fullFieldName, description))

		// Handle nested structs
		fieldType := field.Type
		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}

		if fieldType.Kind() == reflect.Struct {
			writeFieldDescriptions(fieldType, builder, fullFieldName)
		} else if fieldType.Kind() == reflect.Slice {
			elemType := fieldType.Elem()
			if elemType.Kind() == reflect.Ptr {
				elemType = elemType.Elem()
			}
			if elemType.Kind() == reflect.Struct {
				writeFieldDescriptions(elemType, builder, fullFieldName+"[]")
			}
		}
	}
}

// getYamlFieldName extracts the yaml field name from struct tag
func getYamlFieldName(field reflect.StructField) string {
	yamlTag := field.Tag.Get("yaml")
	if yamlTag == "" {
		return strings.ToLower(field.Name)
	}

	// Handle yaml tag options like "name,omitempty"
	parts := strings.Split(yamlTag, ",")
	if parts[0] == "" {
		return strings.ToLower(field.Name)
	}

	return parts[0]
}

// getJsonFieldName extracts the json field name from struct tag
func getJsonFieldName(field reflect.StructField) string {
	jsonTag := field.Tag.Get("json")
	if jsonTag == "" {
		return field.Name
	}

	// Handle json tag options like "name,omitempty"
	parts := strings.Split(jsonTag, ",")
	if parts[0] == "" {
		return field.Name
	}

	return parts[0]
}

// getFieldDisplayName gets the appropriate field name for display
func getFieldDisplayName(field reflect.StructField) string {
	// Prefer yaml tag, then json tag, then field name
	if yamlTag := field.Tag.Get("yaml"); yamlTag != "" {
		parts := strings.Split(yamlTag, ",")
		if parts[0] != "" {
			return parts[0]
		}
	}

	if jsonTag := field.Tag.Get("json"); jsonTag != "" {
		parts := strings.Split(jsonTag, ",")
		if parts[0] != "" {
			return parts[0]
		}
	}

	return field.Name
}

// ValidateStructForPrompt validates that type T is suitable for prompt generation
func ValidateStructForPrompt[T any]() error {
	var zero T
	t := reflect.TypeOf(zero)

	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return fmt.Errorf("type %s is not a struct", t.String())
	}

	return validateStructFields(t, "")
}

// validateStructFields recursively validates struct fields
func validateStructFields(t reflect.Type, prefix string) error {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if !field.IsExported() {
			continue
		}

		fieldPath := field.Name
		if prefix != "" {
			fieldPath = prefix + "." + field.Name
		}

		// Validate yaml tag if present
		if yamlTag := field.Tag.Get("yaml"); yamlTag != "" {
			if err := validateYamlTag(yamlTag, fieldPath); err != nil {
				return err
			}
		}

		// Recursively validate nested structs
		fieldType := field.Type
		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}

		if fieldType.Kind() == reflect.Struct {
			if err := validateStructFields(fieldType, fieldPath); err != nil {
				return err
			}
		} else if fieldType.Kind() == reflect.Slice {
			elemType := fieldType.Elem()
			if elemType.Kind() == reflect.Ptr {
				elemType = elemType.Elem()
			}
			if elemType.Kind() == reflect.Struct {
				if err := validateStructFields(elemType, fieldPath+"[]"); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// validateYamlTag validates yaml tag format
func validateYamlTag(tag, fieldPath string) error {
	if tag == "-" {
		return nil // Skip field
	}

	parts := strings.Split(tag, ",")
	fieldName := parts[0]

	if fieldName == "" {
		return nil // Use default field name
	}

	// Basic validation - yaml field names should be valid
	if strings.Contains(fieldName, " ") {
		return fmt.Errorf("invalid yaml tag for field %s: field name cannot contain spaces", fieldPath)
	}

	return nil
}
