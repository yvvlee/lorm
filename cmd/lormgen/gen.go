package main

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"

	"github.com/samber/lo"
	"github.com/yvvlee/lorm"
	"golang.org/x/tools/go/packages"

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

func (g *Generator) Generate(files []string) error {
	pkgs, err := g.load(files)
	if err != nil {
		return err
	}
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			fileInfo := g.extractFile(file)
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
	err = os.WriteFile(generatedFilePath, content, 0644)
	if err != nil {
		return "", err
	}
	err = exec.Command("goimports", "-w", generatedFilePath).Run()
	if err != nil {
		return "", err
	}
	return generatedFilePath, nil
}

func (g *Generator) generatedFilePath(originFile string) string {
	return originFile[:len(originFile)-3] + g.fileSuffix + ".go"
}

func (g *Generator) load(files []string) ([]*packages.Package, error) {
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

// extractFile extracts struct information from AST file
func (g *Generator) extractFile(file *ast.File) *lorm.FileDescriptor {
	lormImportSpec, ok := lo.Find(file.Imports, func(item *ast.ImportSpec) bool {
		return strings.Trim(item.Path.Value, "\"") == lormPackage
	})
	if !ok {
		//如果没有导入lorm包，则不处理
		return nil
	}
	tokenFile := g.fileSet.File(file.Pos())
	filePath := tokenFile.Name()
	fileRefPath, _ := filepath.Rel(wd, filePath)

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

	ast.Inspect(file, func(n ast.Node) bool {
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
					fields := lo.Filter(structType.Fields.List, func(field *ast.Field, _ int) bool {
						// Check if lorm.UnimplementedTable or lorm.UnimplementedModel is embedded
						if len(field.Names) > 0 {
							return true
						}
						fieldType := exprToString(field.Type)
						if fieldType == unimplementedTable {
							hasModel = true
							structInfo.TableName, _ = parseTag(field, g.tagKey)
							if structInfo.TableName == "" {
								structInfo.TableName = g.tableMapper.ConvertName(structInfo.Name)
							}
							return false
						}
						if fieldType == unimplementedModel {
							hasModel = true
							return false
						}
						return true
					})
					if !hasModel {
						continue
					}

					// 遍历结构体字段
					for _, field := range fields {
						if len(field.Names) == 0 {
							// Embedded field
							embedFieldPrefix, _ := parseTag(field, g.tagKey)
							if ident, ok := field.Type.(*ast.Ident); ok {
								if ts, ok := ident.Obj.Decl.(*ast.TypeSpec); ok {
									if st, ok := ts.Type.(*ast.StructType); ok {
										for _, embedField := range st.Fields.List {
											fieldList := g.parseField(embedField)
											if len(fieldList) > 0 {
												for _, f := range fieldList {
													f.FullName = ident.Name + "." + f.Name
													f.DBField = embedFieldPrefix + f.DBField
												}
												structInfo.Fields = append(structInfo.Fields, fieldList...)
											}
										}
									}
								}
							}
						} else {
							// Regular field
							fieldList := g.parseField(field)
							if len(fields) > 0 {
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
	return &fileInfo
}

func (g *Generator) parseField(field *ast.Field) []*lorm.FieldDescriptor {
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

		fieldInfo.Type = exprToString(field.Type)
		fields = append(fields, fieldInfo)
	}
	return fields
}

func parseTag(field *ast.Field, tagKey string) (filed string, flag lorm.FieldFlag) {
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
		filed = flags[0]
	}
	return
}

func parseFlag(flags *[]string, key string) bool {
	length := len(*flags)
	*flags = lo.Without(*flags, key)
	return length != len(*flags)
}

// exprToString converts an expression to a string
func exprToString(expr ast.Expr) string {
	switch x := expr.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return exprToString(x.X) + "." + x.Sel.Name
	case *ast.StarExpr:
		return "*" + exprToString(x.X)
	case *ast.ArrayType:
		return "[]" + exprToString(x.Elt)
	case *ast.MapType:
		return "map[" + exprToString(x.Key) + "]" + exprToString(x.Value)
	default:
		return ""
	}
}

// generateCode generates code for structs
func generateCode(fileInfo *lorm.FileDescriptor) ([]byte, error) {
	var buf bytes.Buffer
	err := modelTpl.Execute(&buf, fileInfo)
	if err != nil {
		return nil, fmt.Errorf("template execution failed: %v\n", err)
	}
	return format.Source(buf.Bytes())
}
