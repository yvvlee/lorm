package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/samber/lo"
	"github.com/spf13/cobra"

	"github.com/yvvlee/lorm/names"
)

func init() {
	initWd()
	initGoImports()

	cmd.PersistentFlags().StringVar(&fieldMapper, "field-mapper", "snake", `table field name mapper (one of "snake", "camel", "same")`)
	cmd.PersistentFlags().StringVar(&tableMapper, "table-mapper", "snake", `db table name mapper (one of "snake", "camel", "same")`)
	cmd.PersistentFlags().StringVar(&tablePrefix, "table-prefix", "", "db table name prefix")
	cmd.PersistentFlags().StringVar(&tableSuffix, "table-suffix", "", "db table name suffix")
	cmd.PersistentFlags().StringVar(&tagKey, "tag-key", "lorm", "table field tag key")
	cmd.PersistentFlags().StringVar(&fileSuffix, "file-suffix", "_lorm_gen", "suffix of generated file")
	cmd.PersistentFlags().StringSliceVar(&ignorePatterns, "ignore", nil, "wildcards of ignore files")
}

var (
	mappers = map[string]names.Mapper{
		"snake": new(names.SnakeMapper),
		"camel": new(names.CamelMapper),
		"same":  new(names.SameMapper),
	}

	tableMapper    string
	tablePrefix    string
	tableSuffix    string
	fieldMapper    string
	tagKey         string
	fileSuffix     string
	ignorePatterns []string

	wd string //当前工作路径

	cmd = &cobra.Command{
		Use:   "lormgen",
		Short: "lormgen is a code generator for Lorm",
		Long:  `lormgen is a code generator for Lorm`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("请提供目录路径")
			}
			tableNameMapper, ok := mappers[tableMapper]
			if !ok {
				return errors.New("不支持的表名映射")
			}
			if tableSuffix != "" {
				tableNameMapper = names.NewSuffixMapper(tableNameMapper, tableSuffix)
			}
			if tablePrefix != "" {
				tableNameMapper = names.NewPrefixMapper(tableNameMapper, tablePrefix)
			}
			fieldNameMapper, ok := mappers[fieldMapper]
			if !ok {
				return errors.New("不支持的字段映射")
			}

			files, err := argsToFiles(args)
			if err != nil {
				return fmt.Errorf("文件解析失败: %v\n", err)
			}
			if len(ignorePatterns) > 0 {
				files = lo.Filter(files, func(item string, _ int) bool {
					for _, pattern := range ignorePatterns {
						matched, err := filepath.Match(pattern, item)
						if err != nil {
							panic(err)
						}
						if matched {
							return false
						}
					}
					return true
				})
			}
			if len(files) == 0 {
				return fmt.Errorf("没有找到符合要求的文件")
			}
			generator := NewGenerator(
				tableNameMapper,
				fieldNameMapper,
				tagKey,
				fileSuffix,
			)
			return generator.Generate(files)
		},
	}
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func argsToFiles(args []string) ([]string, error) {
	var files []string
	for _, arg := range args {
		if strings.HasSuffix(arg, "/...") {
			// 处理 "./..." 这样的递归路径
			dir := strings.TrimSuffix(arg, "/...")
			if dir == "." {
				dir = "./"
			}
			err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() && isValidFile(path) {
					files = append(files, path)
				}
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("遍历目录时出错: %v", err)
			}
		} else {
			// 处理单个文件或目录
			info, err := os.Stat(arg)
			if err != nil {
				return nil, fmt.Errorf("无法访问路径: %v", err)
			}
			if info.IsDir() {
				// 如果是目录，查找其中的所有go文件
				items, err := os.ReadDir(arg)
				if err != nil {
					return nil, fmt.Errorf("读取目录失败: %v", err)
				}
				for _, item := range items {
					if !item.IsDir() && isValidFile(item.Name()) {
						files = append(files, filepath.Join(arg, item.Name()))
					}
				}
			} else if isValidFile(arg) {
				// 如果是单个文件
				files = append(files, arg)
			}
		}
	}
	return lo.Uniq(files), nil
}

func isValidFile(file string) bool {
	return strings.HasSuffix(file, ".go") &&
		!strings.HasSuffix(file, "_test.go") &&
		!strings.HasSuffix(file, "_gen.go")
}

func initWd() {
	var err error
	wd, err = os.Getwd()
	if err != nil {
		panic(err)
	}
}

func initGoImports() {
	path, err := exec.LookPath("goimports")
	if err != nil || path == "" {
		fmt.Println(`未找到goimports, 执行安装 go install golang.org/x/tools/cmd/goimports@latest`)
		err = exec.Command("go", "install", "golang.org/x/tools/cmd/goimports@latest").Run()
		if err != nil {
			panic(fmt.Errorf("goimports 安装失败: %+v", err))
		}
	}
}
