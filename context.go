package gotots

import (
	"reflect"
)

func NewContext(cfg Config) *Context {
	return &Context{
		Cache:     make(map[reflect.Type]bool),
		TypeNames: make(map[reflect.Type]string),
		config:    cfg,
	}
}

type Context struct {
	Structs       []StructInfo
	Cache         map[reflect.Type]bool
	TypeNames     map[reflect.Type]string
	config        Config
	customHeaders []string
}

func (ctx *Context) AddCustomHeader(header string) {
	ctx.customHeaders = append(ctx.customHeaders, header)
}
