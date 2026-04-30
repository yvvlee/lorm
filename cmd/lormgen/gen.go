package main

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"text/template"

	"github.com/samber/lo"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/imports"

	"github.com/yvvlee/lorm"
	"github.com/yvvlee/lorm/names"
)

const (
	lormPackage = "github.com/yvvlee/lorm"
)

var (
	//go:embed templates/model.tmpl
	modelTplStr string
	modelTpl    = template.Must(template.New("main").Parse(modelTplStr))
)

// Generator loads Go packages, extracts Lorm model metadata, and writes companion files.
type Generator struct {
	tableMapper names.Mapper
	fieldMapper names.Mapper
	tagKey      string
	fileSuffix  string

	fileSet *token.FileSet
}

func NewGenerator(
	tableMapper names.Mapper,
	fieldMapper names.Mapper,
	tagKey string,
	fileSuffix string,
) *Generator {
	return &Generator{
		tableMapper: tableMapper,
		fieldMapper: fieldMapper,
		tagKey:      tagKey,
		fileSuffix:  fileSuffix,
		fileSet:     token.NewFileSet(),
	}
}

// Generate emits one generated file for each source file that declares Lorm models.
func (g *Generator) Generate(files []string) error {
	for _, group := range groupFilesByDir(files) {
		pkgs, err := g.load(group)
		if err != nil {
			return err
		}
		if err := g.generatePackages(pkgs); err != nil {
			return err
		}
	}
	return nil
}

func groupFilesByDir(files []string) [][]string {
	if len(files) == 0 {
		return nil
	}
	byDir := make(map[string][]string)
	for _, file := range files {
		dir := filepath.Dir(file)
		byDir[dir] = append(byDir[dir], file)
	}
	dirs := lo.Keys(byDir)
	sort.Strings(dirs)
	groups := make([][]string, 0, len(dirs))
	for _, dir := range dirs {
		group := byDir[dir]
		sort.Strings(group)
		groups = append(groups, group)
	}
	return groups
}

func (g *Generator) generatePackages(pkgs []*packages.Package) error {
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			fileInfo, err := g.extractFile(pkg, file)
			if err != nil {
				return err
			}
			if fileInfo == nil {
				continue
			}
			generatedFilePath, err := g.generateFile(fileInfo)
			if err != nil {
				return err
			}
			fmt.Printf("Generated file: %s\n", generatedFilePath)
		}
	}
	return nil
}

func (g *Generator) generateFile(file *lorm.FileDescriptor) (string, error) {
	content, err := generateCode(file)
	if err != nil {
		return "", err
	}
	generatedFilePath := g.generatedFilePath(file.Path)
	formatted, err := imports.Process(generatedFilePath, content, nil)
	if err != nil {
		return "", fmt.Errorf("failed to format generated code: %w", err)
	}
	err = os.WriteFile(generatedFilePath, formatted, 0644)
	if err != nil {
		return "", err
	}
	return generatedFilePath, nil
}

func (g *Generator) generatedFilePath(originFile string) string {
	return originFile[:len(originFile)-3] + g.fileSuffix + ".go"
}

func (g *Generator) load(files []string) ([]*packages.Package, error) {
	// Request syntax plus type information so embedded structs and imported marker types can be resolved together.
	// Configure loading options
	cfg := &packages.Config{
		Mode: packages.NeedName | // Package name required
			packages.NeedFiles | // Need Go source file names that make up the package
			packages.NeedCompiledGoFiles | // Need Go source file names that participate in final compilation
			packages.NeedImports | // Need package dependencies
			packages.NeedDeps | // @Required Ensure transitive dependencies are resolved
			packages.NeedTypes | // Need package type information (*types.Package)
			packages.NeedTypesSizes | // Need size and alignment information for types
			packages.NeedSyntax | // Need package AST (*ast.File)
			packages.NeedTypesInfo, // Need detailed information after type checking (*types.Info)
		Fset: g.fileSet,
	}

	pkgs, err := packages.Load(cfg, files...)
	if err != nil {
		return nil, fmt.Errorf("failed to load packages: %v", err)
	}

	// Check for errors, such as syntax errors
	if packages.PrintErrors(pkgs) > 0 {
		return nil, errors.New("packages contain errors")
	}
	return pkgs, nil
}

// extractFile turns one parsed Go file into a descriptor when it imports lorm.
func (g *Generator) extractFile(pkg *packages.Package, file *ast.File) (*lorm.FileDescriptor, error) {
	lormImportSpec, ok := lo.Find(file.Imports, func(item *ast.ImportSpec) bool {
		return strings.Trim(item.Path.Value, "\"") == lormPackage
	})
	if !ok {
		// If lorm package is not imported, skip processing
		return nil, nil
	}
	tokenFile := g.fileSet.File(file.Pos())
	filePath := tokenFile.Name()
	// packages.Load records absolute paths; keep descriptors stable relative to where the CLI was invoked.
	fileRefPath := relativeToWorkingDir(filePath)

	// Respect import aliases both when detecting embedded markers and when generating method bodies.
	lormName := "lorm"
	if lormImportSpec.Name != nil {
		lormName = lormImportSpec.Name.Name
	}
	unimplementedTable := lormName + ".UnimplementedTable"
	unimplementedModel := lormName + ".UnimplementedModel"

	fileInfo := lorm.FileDescriptor{
		Path:            fileRefPath,
		LormImportAlias: lormName,
		Package:         file.Name.Name,
		Imports: lo.Map(file.Imports, func(item *ast.ImportSpec, _ int) *lorm.Import {
			var alias string
			if item.Name != nil {
				alias = item.Name.Name
			}
			return &lorm.Import{
				Path:  item.Path.Value,
				Alias: alias,
			}
		}),
		Structs: nil,
	}

	var extractErr error
	ast.Inspect(file, func(n ast.Node) bool {
		if extractErr != nil {
			return false
		}
		switch x := n.(type) {
		case *ast.GenDecl:
			if x.Tok == token.TYPE {
				for _, spec := range x.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					structType, ok := typeSpec.Type.(*ast.StructType)
					if !ok {
						continue
					}
					structInfo := &lorm.ModelDescriptor{
						Name: typeSpec.Name.Name,
					}
					var hasModel bool
					var fields []*ast.Field
					for _, field := range structType.Fields.List {
						// Embedded marker fields opt the struct into generation and may carry table metadata.
						if len(field.Names) > 0 {
							fields = append(fields, field)
							continue
						}
						fieldType, err := exprToString(field.Type)
						if err == nil && fieldType == unimplementedTable {
							hasModel = true
							structInfo.TableName, _ = parseTag(field, g.tagKey)
							if structInfo.TableName == "" {
								structInfo.TableName = g.tableMapper.ConvertName(structInfo.Name)
							}
							continue
						}
						if err == nil && fieldType == unimplementedModel {
							hasModel = true
							continue
						}
						fields = append(fields, field)
					}
					if !hasModel {
						continue
					}

					// Iterate through struct fields
					for _, field := range fields {
						if len(field.Names) == 0 {
							// Flatten named embedded structs so generated field accessors can treat them like direct members.
							embedFieldPrefix, _ := parseTag(field, g.tagKey)
							embedName, ensureType, embeddedPointer, structType := resolveEmbeddedStruct(pkg, field.Type)
							if structType == nil {
								extractErr = fmt.Errorf("unsupported embedded field %q in %s", exprSource(field.Type), structInfo.Name)
								return false
							}
							for _, embedField := range structType.Fields.List {
								fieldList, err := g.parseField(structInfo.Name+"."+embedName, embedField)
								if err != nil {
									extractErr = err
									return false
								}
								if len(fieldList) > 0 {
									for _, f := range fieldList {
										f.FullName = embedName + "." + f.Name
										f.DBField = embedFieldPrefix + f.DBField
										if embeddedPointer {
											f.EnsureFullName = embedName
											f.EnsureType = ensureType
										}
									}
									structInfo.Fields = append(structInfo.Fields, fieldList...)
								}
							}
						} else {
							// Regular field
							fieldList, err := g.parseField(structInfo.Name, field)
							if err != nil {
								extractErr = err
								return false
							}
							if len(fieldList) > 0 {
								structInfo.Fields = append(structInfo.Fields, fieldList...)
							}
						}
					}
					fileInfo.Structs = append(fileInfo.Structs, structInfo)
				}
				return false
			}
		}
		return true
	})
	if extractErr != nil {
		return nil, extractErr
	}
	return &fileInfo, nil
}

func relativeToWorkingDir(filePath string) string {
	fileRefPath, err := filepath.Rel(wd, filePath)
	if err == nil && !strings.HasPrefix(fileRefPath, ".."+string(filepath.Separator)) && fileRefPath != ".." {
		return fileRefPath
	}
	realWd, wdErr := filepath.EvalSymlinks(wd)
	realFilePath, fileErr := filepath.EvalSymlinks(filePath)
	if wdErr == nil && fileErr == nil {
		if rel, relErr := filepath.Rel(realWd, realFilePath); relErr == nil {
			return rel
		}
	}
	return fileRefPath
}

// parseField expands grouped declarations like "A, B string" into one descriptor per field name.
func (g *Generator) parseField(structName string, field *ast.Field) ([]*lorm.FieldDescriptor, error) {
	fieldType, err := exprToString(field.Type)
	if err != nil {
		return nil, fmt.Errorf("unsupported field type %q for %s.%s", exprSource(field.Type), structName, fieldNames(field))
	}

	dbField, flag := parseTag(field, g.tagKey)
	var fields []*lorm.FieldDescriptor
	for i, name := range field.Names {
		fieldInfo := &lorm.FieldDescriptor{
			Name:     name.Name,
			FullName: name.Name,
			DBField:  g.fieldMapper.ConvertName(name.Name),
		}
		if i == len(field.Names)-1 {
			fieldInfo.Flag = flag
			// When fields are declared in aggregation, the tag only takes effect for the last field, eg: fieldA, fieldB string `lorm:"field_b"`
			if dbField != "" {
				fieldInfo.DBField = dbField
			}
		}

		fieldInfo.Type = fieldType
		fields = append(fields, fieldInfo)
	}
	return fields, nil
}

// parseTag splits the lorm tag into an optional database field name and any flag tokens.
func parseTag(field *ast.Field, tagKey string) (dbField string, flag lorm.FieldFlag) {
	if field == nil || field.Tag == nil {
		return
	}
	tagString := reflect.StructTag(strings.Trim(field.Tag.Value, "`")).Get(tagKey)
	flags := lo.Uniq(strings.Split(tagString, ","))
	for fieldFlag, key := range lorm.FlagTagMap {
		if parseFlag(&flags, key) {
			flag |= fieldFlag
		}
	}
	if len(flags) > 0 {
		// After flag tokens are removed, the first remaining value is the explicit database field name.
		dbField = flags[0]
	}
	return
}

func parseFlag(flags *[]string, key string) bool {
	length := len(*flags)
	*flags = lo.Without(*flags, key)
	return length != len(*flags)
}

// exprToString normalizes the subset of Go field types that descriptors need to serialize.
func exprToString(expr ast.Expr) (string, error) {
	switch x := expr.(type) {
	case *ast.Ident:
		return x.Name, nil
	case *ast.SelectorExpr:
		selector, err := exprToString(x.X)
		if err != nil {
			return "", err
		}
		return selector + "." + x.Sel.Name, nil
	case *ast.StarExpr:
		elem, err := exprToString(x.X)
		if err != nil {
			return "", err
		}
		return "*" + elem, nil
	case *ast.ArrayType:
		elem, err := exprToString(x.Elt)
		if err != nil {
			return "", err
		}
		if x.Len == nil {
			return "[]" + elem, nil
		}
		return "[" + exprSource(x.Len) + "]" + elem, nil
	case *ast.MapType:
		key, err := exprToString(x.Key)
		if err != nil {
			return "", err
		}
		value, err := exprToString(x.Value)
		if err != nil {
			return "", err
		}
		return "map[" + key + "]" + value, nil
	default:
		return "", fmt.Errorf("unsupported field type %q", exprSource(expr))
	}
}

func fieldNames(field *ast.Field) string {
	names := lo.Map(field.Names, func(name *ast.Ident, _ int) string {
		return name.Name
	})
	return strings.Join(names, ", ")
}

func exprSource(expr ast.Expr) string {
	var buf bytes.Buffer
	if err := format.Node(&buf, token.NewFileSet(), expr); err != nil {
		return fmt.Sprintf("%T", expr)
	}
	return buf.String()
}

func resolveEmbeddedStruct(pkg *packages.Package, expr ast.Expr) (string, string, bool, *ast.StructType) {
	expr, pointer := unwrapPointerExpr(expr)
	embedName := embeddedFieldName(expr)
	if embedName == "" {
		return "", "", false, nil
	}
	ensureType, err := exprToString(expr)
	if err != nil {
		return "", "", false, nil
	}

	var obj types.Object
	switch x := expr.(type) {
	case *ast.Ident:
		if pkg != nil && pkg.TypesInfo != nil {
			obj = pkg.TypesInfo.Uses[x]
			if obj == nil {
				obj = pkg.TypesInfo.Defs[x]
			}
		}
		if obj == nil && x.Obj != nil {
			if ts, ok := x.Obj.Decl.(*ast.TypeSpec); ok {
				if structType, ok := ts.Type.(*ast.StructType); ok {
					return embedName, ensureType, pointer, structType
				}
			}
		}
	case *ast.SelectorExpr:
		if pkg != nil && pkg.TypesInfo != nil {
			obj = pkg.TypesInfo.Uses[x.Sel]
		}
	}
	if obj == nil {
		return "", "", false, nil
	}
	ts := findTypeSpecByObject(pkg, obj)
	if ts == nil {
		return "", "", false, nil
	}
	structType, _ := ts.Type.(*ast.StructType)
	return embedName, ensureType, pointer, structType
}

func unwrapPointerExpr(expr ast.Expr) (ast.Expr, bool) {
	pointer := false
	for {
		star, ok := expr.(*ast.StarExpr)
		if !ok {
			return expr, pointer
		}
		pointer = true
		expr = star.X
	}
}

func embeddedFieldName(expr ast.Expr) string {
	switch x := expr.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return x.Sel.Name
	default:
		return ""
	}
}

func findTypeSpecByObject(pkg *packages.Package, obj types.Object) *ast.TypeSpec {
	return findTypeSpecByObjectSeen(pkg, obj, make(map[*packages.Package]bool))
}

func findTypeSpecByObjectSeen(pkg *packages.Package, obj types.Object, seen map[*packages.Package]bool) *ast.TypeSpec {
	if pkg == nil || obj == nil || seen[pkg] {
		return nil
	}
	seen[pkg] = true
	if ts := findTypeSpecByPos(pkg, obj.Pos()); ts != nil {
		return ts
	}
	importPaths := lo.Keys(pkg.Imports)
	sort.Strings(importPaths)
	for _, importPath := range importPaths {
		if ts := findTypeSpecByObjectSeen(pkg.Imports[importPath], obj, seen); ts != nil {
			return ts
		}
	}
	return nil
}

func findTypeSpecByPos(pkg *packages.Package, pos token.Pos) *ast.TypeSpec {
	for _, f := range pkg.Syntax {
		if f.Pos() <= pos && pos <= f.End() {
			for _, decl := range f.Decls {
				if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
					for _, spec := range genDecl.Specs {
						if typeSpec, ok := spec.(*ast.TypeSpec); ok {
							if typeSpec.Name.Pos() == pos {
								return typeSpec
							}
						}
					}
				}
			}
		}
	}
	return nil
}

// generateCode renders the template and applies gofmt before the file hits disk.
func generateCode(fileInfo *lorm.FileDescriptor) ([]byte, error) {
	var buf bytes.Buffer
	err := modelTpl.Execute(&buf, fileInfo)
	if err != nil {
		return nil, fmt.Errorf("template execution failed: %v\n", err)
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to format generated code: %w", err)
	}
	return formatted, nil
}
