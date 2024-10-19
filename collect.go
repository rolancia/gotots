package gotots

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
)

// Collect initiates the type collection process starting from a given type 't'.
func Collect(ctx *Context, roots ...any) error {
	for _, root := range roots {
		tyof := reflect.TypeOf(root)
		if reflect.ValueOf(root).Kind() == reflect.Pointer {
			tyof = tyof.Elem()
		}
		if tyof.PkgPath() == "" {
			return errors.New("anonymous types are not supported for root type")
		}
		namePath := []string{ctx.config.FieldPackageNameToPrefix(tyof.PkgPath()) + tyof.Name()}
		if err := CollectType(ctx, tyof, namePath); err != nil {
			return err
		}
	}
	return nil
}

// WriteFromContext writes the collected TypeScript interfaces to the provided writer.
func WriteFromContext(ctx *Context, w io.Writer) error {
	for _, header := range ctx.customHeaders {
		if _, err := fmt.Fprintln(w, header); err != nil {
			return err
		}
	}
	if len(ctx.customHeaders) > 0 {
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	for _, sinfo := range ctx.Structs {
		if err := DescribeStruct(ctx, w, sinfo); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	return nil
}

// CollectType recursively collects type information for the given type 't'.
func CollectType(ctx *Context, t reflect.Type, namePath []string) error {
	if ctx.Cache[t] {
		return nil
	}
	ctx.Cache[t] = true

	switch t.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Array:
		elemType := t.Elem()
		newNamePath := namePath
		if elemType.Kind() == reflect.Struct && elemType.Name() == "" {
			newNamePath = append(newNamePath, "Elem")
		}
		return CollectType(ctx, elemType, newNamePath)
	case reflect.Map:
		keyType := t.Key()
		elemType := t.Elem()
		if err := CollectType(ctx, keyType, namePath); err != nil {
			return err
		}
		newElemNamePath := namePath
		if elemType.Kind() == reflect.Struct && elemType.Name() == "" {
			newElemNamePath = append(newElemNamePath, "Elem")
		}
		return CollectType(ctx, elemType, newElemNamePath)
	case reflect.Struct:
		// Skip anonymous structs
		if t.Name() == "" {
			return nil
		}

		var name string
		pkg := t.PkgPath()
		if pkg == "" {
			pkg = ctx.config.PackageNameForAnonymous
		}
		pkgPrefix := ctx.config.FieldPackageNameToPrefix(pkg)
		name = pkgPrefix + t.Name()
		ctx.TypeNames[t] = name

		sinfo := StructInfo{Type: t, GeneratedName: name}
		if err := CollectFields(ctx, &sinfo, t, namePath); err != nil {
			return err
		}
		ctx.Structs = append(ctx.Structs, sinfo)
	default:
		// Do nothing for other types
	}
	return nil
}

// StructInfo holds information about a collected struct.
type StructInfo struct {
	Type          reflect.Type
	Fields        []StructFieldInfo
	GeneratedName string
}

// StructFieldInfo holds information about a struct field.
type StructFieldInfo struct {
	Name       string
	Type       reflect.Type
	CustomType string
	JsonTag    string
}

// GoTypeToTsTypeMapKey converts a Go type to a TypeScript map key type.
func GoTypeToTsTypeMapKey(ctx *Context, t reflect.Type) (string, error) {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
		reflect.Float32, reflect.Float64:
		return "number", nil
	case reflect.String, reflect.Int64, reflect.Uint64:
		return "string", nil
	default:
		return "", errors.New("unsupported map key type " + t.String())
	}
}

// GoTypeToTSType converts a Go type to a TypeScript type.
func GoTypeToTSType(ctx *Context, t reflect.Type, fromPtr bool, depth int) (ty string, isOptional bool, err error) {
	indent := "    "
	if ctx.config.IndentWithTabs {
		indent = "\t"
	}

	switch t.Kind() {
	case reflect.Struct:
		if t.Name() == "" {
			var structDef strings.Builder
			structDef.WriteString("{\n")

			sinfo := StructInfo{Type: t}
			if err := CollectFields(ctx, &sinfo, t, []string{}); err != nil {
				return "", false, err
			}

			for _, field := range sinfo.Fields {
				fieldType, isOptional, _ := GoTypeToTSType(ctx, field.Type, false, depth+1)
				optionalMark := ""
				if isOptional {
					optionalMark = "?"
				}
				if field.JsonTag != "" {
					field.Name = field.JsonTag
				}
				// Adjust indentation based on depth
				structDef.WriteString(fmt.Sprintf("%s%s%s: %s;\n", strings.Repeat(indent, depth+1), field.Name, optionalMark, fieldType))
			}

			structDef.WriteString(strings.Repeat(indent, depth) + "}")
			return structDef.String(), fromPtr, nil
		}

		if name, ok := ctx.TypeNames[t]; ok {
			return name, fromPtr, nil
		}

		return "any", fromPtr, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
		reflect.Float32, reflect.Float64:
		return "number", fromPtr, nil
	case reflect.Int64, reflect.Uint64:
		return "bigint", fromPtr, nil
	case reflect.String:
		return "string", fromPtr, nil
	case reflect.Bool:
		return "boolean", fromPtr, nil
	case reflect.Ptr:
		return GoTypeToTSType(ctx, t.Elem(), true, depth)
	case reflect.Array, reflect.Slice:
		elemType, _, err := GoTypeToTSType(ctx, t.Elem(), false, depth)
		if err != nil {
			return "", true, err
		}
		// Set isOptional to true for arrays and slices
		return elemType + "[]", true, nil
	case reflect.Map:
		keyType, err := GoTypeToTsTypeMapKey(ctx, t.Key())
		if err != nil {
			return "", false, err
		}
		elemType, isOptional, err := GoTypeToTSType(ctx, t.Elem(), false, depth)
		if err != nil {
			return "", false, err
		}
		if isOptional {
			return fmt.Sprintf("{ [key: %s]: (%s | undefined) }", keyType, elemType), fromPtr, nil
		} else {
			return fmt.Sprintf("{ [key: %s]: %s }", keyType, elemType), fromPtr, nil
		}
	default:
		return "any", fromPtr, nil
	}
}

// CollectFields collects field information from a struct type.
func CollectFields(ctx *Context, sinfo *StructInfo, t reflect.Type, namePath []string) error {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")

		if field.Anonymous && (field.Type.Kind() == reflect.Struct || field.Type.Kind() == reflect.Ptr) {
			if jsonTag != "" && jsonTag != "-" {
				// Treat embedded field with json tag as a normal field
			} else {
				// Embedded field without json tag, flatten fields
				if err := CollectFields(ctx, sinfo, field.Type, namePath); err != nil {
					return err
				}
				continue
			}
		}

		sfield := StructFieldInfo{
			Name:       field.Name,
			Type:       field.Type,
			CustomType: field.Tag.Get("tstype"),
		}
		if jsonTag != "" {
			sfield.JsonTag = strings.Split(jsonTag, ",")[0]
		}
		if sfield.JsonTag == "-" {
			continue
		}
		if sfield.CustomType == "" {
			newNamePath := namePath
			switch field.Type.Kind() {
			case reflect.Struct:
				if field.Type.Name() == "" {
					newNamePath = append(newNamePath, field.Name)
				}
			case reflect.Slice, reflect.Array, reflect.Ptr:
				elemType := field.Type.Elem()
				if elemType.Kind() == reflect.Struct && elemType.Name() == "" {
					newNamePath = append(newNamePath, field.Name)
				}
			case reflect.Map:
				elemType := field.Type.Elem()
				if elemType.Kind() == reflect.Struct && elemType.Name() == "" {
					newNamePath = append(newNamePath, field.Name)
				}
			default:
			}
			if err := CollectType(ctx, field.Type, newNamePath); err != nil {
				return err
			}
		}
		sinfo.Fields = append(sinfo.Fields, sfield)
	}
	return nil
}

// DescribeStruct writes the TypeScript interface definition for a struct.
func DescribeStruct(ctx *Context, w io.Writer, s StructInfo) error {
	name := s.GeneratedName
	if _, err := fmt.Fprintf(w, "export interface %s {\n", name); err != nil {
		return err
	}
	for _, field := range s.Fields {
		tstype := field.CustomType
		isOptional := false
		if tstype == "" {
			var err error
			tstype, isOptional, err = GoTypeToTSType(ctx, field.Type, false, 1)
			if err != nil {
				return err
			}
		}
		fieldName := field.Name
		if field.JsonTag != "" {
			fieldName = field.JsonTag
		}
		optionalMark := ""
		if isOptional {
			optionalMark = "?"
		}
		indent := "    "
		if ctx.config.IndentWithTabs {
			indent = "\t"
		}
		if _, err := fmt.Fprintf(w, "%s%s%s: %s;\n", indent, fieldName, optionalMark, tstype); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, "}"); err != nil {
		return err
	}
	return nil
}
