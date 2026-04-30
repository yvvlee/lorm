package testdata

import "github.com/yvvlee/lorm"

type UnsupportedType struct {
	lorm.UnimplementedModel
	Callback func()
}
