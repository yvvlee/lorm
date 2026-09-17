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
	lormPackage       = "github.com/yvvlee/lorm"
	defaultFileSuffix = "_lorm_gen"
)

func effectiveFileSuffix(suffix string) string {
	if suffix == "" {
		return defaultFileSuffix
	}
	return suffix
}

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
		fileSuffix:  effectiveFileSuffix(fileSuffix),
		fileSet:     token.NewFileSet(),
	}
}

// Generate emits one generated file for each source file that declares Lorm models.
func (g *Generator) Generate(files []string) error {
	var outputs []generatedFile
	for _, group := range groupFilesByDir(files) {
		pkgs, err := g.load(group)
		if err != nil {
			return err
		}
		pending, err := g.renderPackages(pkgs, group)
		if err != nil {
			return err
		}
		outputs = append(outputs, pending...)
	}
	// Do not touch output files until every package has loaded, validated and
	// rendered successfully. This includes errors in later directory groups.
	for _, output := range outputs {
		if err := output.write(); err != nil {
			return err
		}
		fmt.Printf("Generated file: %s\n", output.path)
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

type generatedFile struct {
	path    string
	content []byte
}

func (file generatedFile) write() error {
	if err := os.WriteFile(file.path, file.content, 0644); err != nil {
		return fmt.Errorf("write generated file %q: %w", file.path, err)
	}
	return nil
}

func (g *Generator) renderPackages(pkgs []*packages.Package, files []string) ([]generatedFile, error) {
	selected := make(map[string]bool, len(files))
	for _, file := range files {
		path, err := filepath.Abs(file)
		if err != nil {
			return nil, fmt.Errorf("resolve source file %q: %w", file, err)
		}
		selected[path] = true
	}
	var outputs []generatedFile
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			if !selected[g.fileSet.File(file.Pos()).Name()] {
				continue
			}
			fileInfo, err := g.extractFile(pkg, file)
			if err != nil {
				return nil, err
			}
			if fileInfo == nil {
				continue
			}
			output, err := g.renderFile(fileInfo)
			if err != nil {
				return nil, err
			}
			outputs = append(outputs, output)
		}
	}
	return outputs, nil
}

func (g *Generator) generateFile(file *lorm.FileDescriptor) (string, error) {
	output, err := g.renderFile(file)
	if err != nil {
		return "", err
	}
	if err := output.write(); err != nil {
		return "", err
	}
	return output.path, nil
}

func (g *Generator) renderFile(file *lorm.FileDescriptor) (generatedFile, error) {
	content, err := generateCode(file)
	if err != nil {
		return generatedFile{}, err
	}
	generatedFilePath := g.generatedFilePath(file.Path)
	if filepath.Clean(generatedFilePath) == filepath.Clean(file.Path) {
		return generatedFile{}, fmt.Errorf("generated file would overwrite source file %q", file.Path)
	}
	formatted, err := imports.Process(generatedFilePath, content, nil)
	if err != nil {
		return generatedFile{}, fmt.Errorf("failed to format generated code: %w", err)
	}
	return generatedFile{path: generatedFilePath, content: formatted}, nil
}

func (g *Generator) generatedFilePath(originFile string) string {
	return strings.TrimSuffix(originFile, ".go") + effectiveFileSuffix(g.fileSuffix) + ".go"
}

func (g *Generator) load(files []string) ([]*packages.Package, error) {
	// Match directory generation's source selection for type checking, even
	// when only one file was requested. Rendering still uses the original list.
	var err error
	files, err = g.packageSourceFiles(files)
	if err != nil {
		return nil, err
	}
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

	// A first generation cannot type-check code that already calls generated
	// methods. Keep the partial type information, but still fail on loading and
	// parsing errors.
	var loadErrors []error
	for _, pkg := range pkgs {
		for _, pkgErr := range pkg.Errors {
			if pkgErr.Kind != packages.TypeError {
				loadErrors = append(loadErrors, errors.New(pkgErr.Error()))
			}
		}
	}
	if len(loadErrors) > 0 {
		return nil, errors.Join(loadErrors...)
	}
	return pkgs, nil
}

func (g *Generator) packageSourceFiles(files []string) ([]string, error) {
	sources := make(map[string]bool, len(files))
	dirs := make(map[string]bool)
	for _, file := range files {
		path, err := filepath.Abs(file)
		if err != nil {
			return nil, fmt.Errorf("resolve source file %q: %w", file, err)
		}
		sources[path] = true
		dirs[filepath.Dir(path)] = true
	}
	for dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, fmt.Errorf("read source directory %q: %w", dir, err)
		}
		for _, entry := range entries {
			if !entry.IsDir() && isSourceFile(entry.Name(), g.fileSuffix) {
				sources[filepath.Join(dir, entry.Name())] = true
			}
		}
	}
	result := lo.Keys(sources)
	sort.Strings(result)
	return result, nil
}

// extractFile turns one parsed Go file into a descriptor when it declares a
// model that embeds an lorm marker type.
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
	buildConstraint, err := sourceBuildConstraint(fileRefPath, file)
	if err != nil {
		return nil, err
	}
	fileInfo.BuildConstraint = buildConstraint

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
							structInfo.TableName, err = parseNameTag(field, g.tagKey)
							if err != nil {
								extractErr = fmt.Errorf("invalid table tag for %s: %w", structInfo.Name, err)
								return false
							}
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
					if typeSpec.TypeParams != nil && typeSpec.TypeParams.NumFields() > 0 {
						extractErr = fmt.Errorf("model %s: generic models are not supported", structInfo.Name)
						return false
					}

					structInfo.Fields, extractErr = g.flattenFields(pkg, &fileInfo, structInfo.Name, fields, "", "", nil, make(map[*ast.StructType]bool))
					if extractErr != nil {
						return false
					}
					if err := validateModelDescriptor(structInfo); err != nil {
						extractErr = err
						return false
					}
					// Embedded fields are also direct members of the model, even
					// when none of their children become database columns.
					for _, field := range fields {
						if len(field.Names) == 0 {
							expr, _ := unwrapPointerExpr(field.Type)
							if err := validateGeneratedModelMethod(structInfo, embeddedFieldName(expr)); err != nil {
								extractErr = err
								return false
							}
						}
					}
					populateModelMetadata(structInfo)
					if structInfo.TableName != "" {
						for _, field := range structInfo.Fields {
							if field.ManagedTimeKind == "int" {
								fileInfo.Requires64BitInt = true
								break
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
	if len(fileInfo.Structs) == 0 {
		return nil, nil
	}
	return &fileInfo, nil
}

// flattenFields keeps the Go access path, database prefix, and pointer ancestors
// together while traversing embedded structs in declaration order.
func (g *Generator) flattenFields(pkg *packages.Package, file *lorm.FileDescriptor, modelName string, fields []*ast.Field, path, prefix string, parents []*lorm.FieldDescriptor, visiting map[*ast.StructType]bool) ([]*lorm.FieldDescriptor, error) {
	var result []*lorm.FieldDescriptor
	for _, field := range fields {
		if len(field.Names) > 0 {
			parsed, err := g.parseField(pkg, modelName+strings.TrimSuffix("."+path, "."), field)
			if err != nil {
				return nil, err
			}
			for _, f := range parsed {
				f.FullName = path + f.Name
				f.DBField = prefix + f.DBField
				if len(parents) > 0 {
					last := parents[len(parents)-1]
					f.EnsureFullName, f.EnsureType = last.EnsureFullName, last.EnsureType
					f.EnsureParents = append([]*lorm.FieldDescriptor(nil), parents[:len(parents)-1]...)
				}
			}
			result = append(result, parsed...)
			continue
		}

		expr, _ := unwrapPointerExpr(field.Type)
		if name := embeddedFieldName(expr); name != "" && !ast.IsExported(name) {
			continue
		}
		typ := typeOfExpr(pkg, expr)
		if typ != nil {
			if named, ok := types.Unalias(typ).(*types.Named); ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == lormPackage {
				if name := named.Obj().Name(); name == "UnimplementedTable" || name == "UnimplementedModel" {
					continue
				}
			}
		}
		embedName, _, pointer, embedded := resolveEmbeddedStruct(pkg, field.Type)
		if embedded == nil {
			return nil, fmt.Errorf("unsupported embedded field %q in %s.%s", exprSource(field.Type), modelName, strings.TrimSuffix(path, "."))
		}
		if visiting[embedded] {
			return nil, fmt.Errorf("recursive embedded field %s.%s%s", modelName, path, embedName)
		}
		embedPrefix, err := parseNameTag(field, g.tagKey)
		if err != nil {
			return nil, fmt.Errorf("invalid embedded field tag for %s.%s%s: %w", modelName, path, embedName, err)
		}
		embedParents := parents
		if pointer {
			ensureType := types.TypeString(typ, func(p *types.Package) string { return embeddedTypeQualifier(pkg, file, p) })
			embedParents = append(append([]*lorm.FieldDescriptor(nil), parents...), &lorm.FieldDescriptor{
				EnsureFullName: path + embedName,
				EnsureType:     ensureType,
			})
		}
		visiting[embedded] = true
		nested, err := g.flattenFields(pkg, file, modelName, embedded.Fields.List, path+embedName+".", prefix+embedPrefix, embedParents, visiting)
		delete(visiting, embedded)
		if err != nil {
			return nil, err
		}
		result = append(result, nested...)
	}
	return result, nil
}

// Nested embeddings can require a type from a package the model did not import.
func embeddedTypeQualifier(pkg *packages.Package, file *lorm.FileDescriptor, target *types.Package) string {
	if target == pkg.Types {
		return ""
	}
	used := make(map[string]bool)
	for _, imp := range file.Imports {
		alias := imp.Alias
		if alias == "" {
			if imported := pkg.Imports[strings.Trim(imp.Path, "\"")]; imported != nil {
				alias = imported.Name
			}
		}
		used[alias] = true
		if strings.Trim(imp.Path, "\"") == target.Path() && alias != "_" {
			if alias == "." {
				return ""
			}
			return alias
		}
	}
	base := "_lorm_embed_" + target.Name()
	alias := base
	for i := 2; used[alias] || pkg.Types.Scope().Lookup(alias) != nil; i++ {
		alias = fmt.Sprintf("%s%d", base, i)
	}
	file.Imports = append(file.Imports, &lorm.Import{Path: fmt.Sprintf("%q", target.Path()), Alias: alias})
	return alias
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
func (g *Generator) parseField(pkg *packages.Package, structName string, field *ast.Field) ([]*lorm.FieldDescriptor, error) {
	if !hasExportedName(field.Names) {
		return nil, nil
	}
	if len(field.Names) > 1 && field.Tag != nil {
		if _, exists := reflect.StructTag(strings.Trim(field.Tag.Value, "`")).Lookup(g.tagKey); exists {
			return nil, fmt.Errorf("invalid lorm tag for %s.%s: grouped fields with a lorm tag must be declared separately", structName, fieldNames(field))
		}
	}
	fieldType, err := exprToString(field.Type)
	if err != nil {
		return nil, fmt.Errorf("unsupported field type %q for %s.%s", exprSource(field.Type), structName, fieldNames(field))
	}

	dbField, flag, err := parseFieldTag(field, g.tagKey)
	if err != nil {
		return nil, fmt.Errorf("invalid lorm tag for %s.%s: %w", structName, fieldNames(field), err)
	}
	if flag.HasFlag(lorm.FlagAutoIncrement) && !flag.HasFlag(lorm.FlagPrimaryKey) {
		return nil, fmt.Errorf("invalid lorm tag for %s.%s: auto_increment requires primary_key", structName, fieldNames(field))
	}
	typ := typeOfExpr(pkg, field.Type)
	if isRawBytesType(typ) {
		return nil, fmt.Errorf("unsupported field type for %s.%s: sql.RawBytes borrows driver memory; use []byte instead", structName, fieldNames(field))
	}
	integerKind, integerBits, integerType := runtimeIntegerInfo(typ)
	managedFlags := 0
	for _, managedFlag := range []lorm.FieldFlag{lorm.FlagCreated, lorm.FlagUpdated, lorm.FlagVersion} {
		if flag.HasFlag(managedFlag) {
			managedFlags++
		}
	}
	if managedFlags > 1 {
		return nil, fmt.Errorf("invalid lorm tag for %s.%s: created, updated, and version are mutually exclusive", structName, fieldNames(field))
	}
	if flag.HasFlag(lorm.FlagPrimaryKey) && flag&(lorm.FlagUpdated|lorm.FlagVersion) != 0 {
		return nil, fmt.Errorf("invalid lorm tag for %s.%s: primary_key cannot be combined with updated or version", structName, fieldNames(field))
	}
	if flag.HasFlag(lorm.FlagAutoIncrement) && managedFlags > 0 {
		return nil, fmt.Errorf("invalid lorm tag for %s.%s: auto_increment cannot be combined with created, updated, or version", structName, fieldNames(field))
	}
	if flag.HasFlag(lorm.FlagJson) && (managedFlags > 0 || flag.HasFlag(lorm.FlagAutoIncrement)) {
		return nil, fmt.Errorf("invalid lorm tag for %s.%s: json cannot be combined with auto_increment, created, updated, or version", structName, fieldNames(field))
	}
	if flag.HasFlag(lorm.FlagAutoIncrement) && !integerType {
		return nil, fmt.Errorf("invalid lorm tag for %s.%s: auto_increment requires a built-in integer type or a pointer to one", structName, fieldNames(field))
	}
	if flag.HasFlag(lorm.FlagVersion) && !integerType {
		return nil, fmt.Errorf("invalid lorm tag for %s.%s: version requires a built-in integer type or a pointer to one", structName, fieldNames(field))
	}
	if flag.HasFlag(lorm.FlagVersion) && integerBits != 0 && integerBits < 32 {
		return nil, fmt.Errorf("invalid lorm tag for %s.%s: version requires an integer type of at least 32 bits or a pointer to one", structName, fieldNames(field))
	}
	managedTimeKind := ""
	if flag.HasFlag(lorm.FlagCreated) || flag.HasFlag(lorm.FlagUpdated) {
		managedTimeKind = managedTimeTypeKind(typ)
		if managedTimeKind == "" {
			typeName := "<unknown>"
			if typ != nil {
				typeName = types.TypeString(typ, func(p *types.Package) string { return p.Name() })
			}
			return nil, fmt.Errorf(
				"invalid lorm tag for %s.%s: created and updated require time.Time, sql.NullTime, int64, uint64, uint32, uint, int, string, or a pointer to one of these types; got %s",
				structName,
				fieldNames(field),
				typeName,
			)
		}
		if managedTimeKind == "int" {
			if pkg == nil || pkg.TypesSizes == nil {
				return nil, fmt.Errorf("cannot determine target int width for %s.%s", structName, fieldNames(field))
			}
			if !supportsManagedIntTime(pkg.TypesSizes) {
				return nil, fmt.Errorf("invalid lorm tag for %s.%s: int managed time fields require a 64-bit target", structName, fieldNames(field))
			}
		}
	}
	pointer := false
	if _, ok := field.Type.(*ast.StarExpr); ok {
		pointer = true
	}
	if typ != nil {
		_, pointer = types.Unalias(typ).Underlying().(*types.Pointer)
	}
	var fields []*lorm.FieldDescriptor
	for _, name := range field.Names {
		if !name.IsExported() {
			continue
		}
		fieldInfo := &lorm.FieldDescriptor{
			Name:     name.Name,
			FullName: name.Name,
			DBField:  g.fieldMapper.ConvertName(name.Name),
		}
		fieldInfo.Flag = flag
		if dbField != "" {
			fieldInfo.DBField = dbField
		}

		fieldInfo.Type = fieldType
		fieldInfo.Pointer = pointer
		fieldInfo.ManagedTimeKind = managedTimeKind
		fieldInfo.IntegerKind = integerKind
		fieldInfo.IntegerBits = integerBits
		fields = append(fields, fieldInfo)
	}
	return fields, nil
}

func hasExportedName(names []*ast.Ident) bool {
	for _, name := range names {
		if name.IsExported() {
			return true
		}
	}
	return false
}

func isRawBytesType(typ types.Type) bool {
	seen := make(map[types.Type]bool)
	for typ != nil {
		typ = types.Unalias(typ)
		if seen[typ] {
			return false
		}
		seen[typ] = true
		if pointer, ok := typ.Underlying().(*types.Pointer); ok {
			typ = pointer.Elem()
			continue
		}
		named, ok := typ.(*types.Named)
		return ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == "database/sql" && named.Obj().Name() == "RawBytes"
	}
	return false
}

func supportsManagedIntTime(sizes types.Sizes) bool {
	return sizes != nil && sizes.Sizeof(types.Typ[types.Int]) >= 8
}

func validateModelDescriptor(model *lorm.ModelDescriptor) error {
	columns := make(map[string]string, len(model.Fields))
	accessors := make(map[string]string, len(model.Fields))
	var versionField, autoIncrementField string
	for _, field := range model.Fields {
		if previous, exists := columns[field.DBField]; exists {
			return fmt.Errorf(
				"model %s has duplicate database column %q for fields %s and %s",
				model.Name,
				field.DBField,
				previous,
				field.FullName,
			)
		}
		columns[field.DBField] = field.FullName
		if field.Name == "All" || field.Name == "WithAlias" {
			return fmt.Errorf("model %s field %s conflicts with generated method %s_Fields.%s", model.Name, field.FullName, model.Name, field.Name)
		}
		if previous, exists := accessors[field.Name]; exists {
			return fmt.Errorf("model %s fields %s and %s generate the same column accessor %s_Fields.%s", model.Name, previous, field.FullName, model.Name, field.Name)
		}
		accessors[field.Name] = field.FullName
		root, _, _ := strings.Cut(field.FullName, ".")
		if err := validateGeneratedModelMethod(model, root); err != nil {
			return err
		}

		if field.Flag.HasFlag(lorm.FlagVersion) {
			if versionField != "" {
				return fmt.Errorf(
					"model %s has multiple version fields: %s and %s",
					model.Name,
					versionField,
					field.FullName,
				)
			}
			versionField = field.FullName
		}
		if field.Flag.HasFlag(lorm.FlagAutoIncrement) {
			if autoIncrementField != "" {
				return fmt.Errorf(
					"model %s has multiple auto-increment fields: %s and %s",
					model.Name,
					autoIncrementField,
					field.FullName,
				)
			}
			autoIncrementField = field.FullName
		}
	}
	if model.TableName != "" && len(model.Fields) == 1 && autoIncrementField != "" {
		return fmt.Errorf("model %s contains only auto-increment primary key %s; at least one other database column is required", model.Name, autoIncrementField)
	}
	return nil
}

// Only reserve methods emitted for this model. Promoted fields on embedded
// structs use explicit access paths and may share a model method's name.
func validateGeneratedModelMethod(model *lorm.ModelDescriptor, name string) error {
	var generated bool
	switch name {
	case "New", "LormFieldPtr", "LormFieldValue", "LormScan", "LormModelDescriptor", "LormCols":
		generated = true
	case "TableName":
		generated = model.TableName != ""
	case "LormBeforeInsert":
		generated = model.TableName != "" && model.NeedsBeforeInsertHook()
	case "LormAfterInsert":
		generated = model.TableName != "" && model.NeedsAfterInsertHook()
	case "LormBeforeUpdate":
		generated = model.TableName != "" && model.NeedsBeforeUpdateHook()
	case "LormAfterUpdate":
		generated = model.TableName != "" && model.NeedsAfterUpdateHook()
	}
	if generated {
		return fmt.Errorf("model %s field %s conflicts with generated method %s.%s", model.Name, name, model.Name, name)
	}
	return nil
}

func populateModelMetadata(model *lorm.ModelDescriptor) {
	model.PrimaryKeys = model.FlagFields(lorm.FlagPrimaryKey)
}

func managedTimeTypeKind(typ types.Type) string {
	if typ == nil {
		return ""
	}
	typ = types.Unalias(typ)
	if pointer, ok := typ.(*types.Pointer); ok {
		typ = types.Unalias(pointer.Elem())
	}

	switch value := typ.(type) {
	case *types.Basic:
		switch value.Kind() {
		case types.Int64:
			return "int64"
		case types.Uint64:
			return "uint64"
		case types.Uint32:
			return "uint32"
		case types.Uint:
			return "uint"
		case types.Int:
			return "int"
		case types.String:
			return "string"
		}
	case *types.Named:
		object := value.Obj()
		if object == nil || object.Pkg() == nil {
			return ""
		}
		path := object.Pkg().Path()
		if path == "time" && object.Name() == "Time" {
			return "time"
		}
		if path == "database/sql" && object.Name() == "NullTime" {
			return "null_time"
		}
	}
	return ""
}

func runtimeIntegerInfo(typ types.Type) (kind string, bits int, ok bool) {
	if typ == nil {
		return "", 0, false
	}
	typ = types.Unalias(typ)
	if pointer, ok := typ.(*types.Pointer); ok {
		typ = types.Unalias(pointer.Elem())
	}
	basic, ok := typ.(*types.Basic)
	if !ok {
		return "", 0, false
	}
	switch basic.Kind() {
	case types.Int:
		return "int", 0, true
	case types.Int8:
		return "int8", 8, true
	case types.Int16:
		return "int16", 16, true
	case types.Int32:
		return "int32", 32, true
	case types.Int64:
		return "int64", 64, true
	case types.Uint:
		return "uint", 0, true
	case types.Uint8:
		return "uint8", 8, true
	case types.Uint16:
		return "uint16", 16, true
	case types.Uint32:
		return "uint32", 32, true
	case types.Uint64:
		return "uint64", 64, true
	default:
		return "", 0, false
	}
}

func typeOfExpr(pkg *packages.Package, expr ast.Expr) types.Type {
	return typeOfExprSeen(pkg, expr, make(map[*packages.Package]bool))
}

func typeOfExprSeen(pkg *packages.Package, expr ast.Expr, seen map[*packages.Package]bool) types.Type {
	if pkg == nil || expr == nil || seen[pkg] {
		return nil
	}
	seen[pkg] = true
	if pkg.TypesInfo != nil {
		if typ := pkg.TypesInfo.TypeOf(expr); typ != nil {
			return typ
		}
	}

	importPaths := lo.Keys(pkg.Imports)
	sort.Strings(importPaths)
	for _, importPath := range importPaths {
		if typ := typeOfExprSeen(pkg.Imports[importPath], expr, seen); typ != nil {
			return typ
		}
	}
	return nil
}

// parseFieldTag splits a field tag into known flags and at most one database column.
func parseFieldTag(field *ast.Field, tagKey string) (dbField string, flag lorm.FieldFlag, err error) {
	if field == nil || field.Tag == nil {
		return
	}
	tagString, exists := reflect.StructTag(strings.Trim(field.Tag.Value, "`")).Lookup(tagKey)
	if !exists || tagString == "" {
		return
	}

	items := strings.Split(tagString, ",")
	for _, item := range items {
		if fieldFlag, ok := fieldFlagByTag(item); ok {
			flag |= fieldFlag
			continue
		}
		if item != "" && dbField == "" {
			dbField = item
			continue
		}
		return "", 0, fmt.Errorf("unrecognized tag item %q", item)
	}
	return
}

// parseNameTag reads table names and embedded-field prefixes, which accept one value and no flags.
func parseNameTag(field *ast.Field, tagKey string) (string, error) {
	if field == nil || field.Tag == nil {
		return "", nil
	}
	tagString, exists := reflect.StructTag(strings.Trim(field.Tag.Value, "`")).Lookup(tagKey)
	if !exists || tagString == "" {
		return "", nil
	}
	if strings.Contains(tagString, ",") {
		return "", fmt.Errorf("unrecognized tag item in %q; expected a single name", tagString)
	}
	return tagString, nil
}

func fieldFlagByTag(tag string) (lorm.FieldFlag, bool) {
	for flag, name := range lorm.FlagTagMap {
		if tag == name {
			return flag, true
		}
	}
	return 0, false
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
	case *ast.IndexExpr:
		base, err := exprToString(x.X)
		if err != nil {
			return "", err
		}
		index, err := exprToString(x.Index)
		if err != nil {
			return "", err
		}
		return base + "[" + index + "]", nil
	case *ast.IndexListExpr:
		base, err := exprToString(x.X)
		if err != nil {
			return "", err
		}
		indices := make([]string, 0, len(x.Indices))
		for _, indexExpr := range x.Indices {
			index, err := exprToString(indexExpr)
			if err != nil {
				return "", err
			}
			indices = append(indices, index)
		}
		return base + "[" + strings.Join(indices, ", ") + "]", nil
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

	typ := typeOfExpr(pkg, expr)
	if typ == nil {
		return "", "", false, nil
	}
	named, ok := types.Unalias(typ).(*types.Named)
	if !ok || named.TypeArgs().Len() != 0 {
		return "", "", false, nil
	}
	ts := findTypeSpecByObject(pkg, named.Obj())
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
	case *ast.IndexExpr:
		return embeddedFieldName(x.X)
	case *ast.IndexListExpr:
		return embeddedFieldName(x.X)
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
	for _, model := range fileInfo.Structs {
		if err := validateModelDescriptor(model); err != nil {
			return nil, err
		}
		populateModelMetadata(model)
	}
	var buf bytes.Buffer
	err := modelTpl.Execute(&buf, fileInfo)
	if err != nil {
		return nil, fmt.Errorf("template execution failed: %v", err)
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to format generated code: %w", err)
	}
	return formatted, nil
}
