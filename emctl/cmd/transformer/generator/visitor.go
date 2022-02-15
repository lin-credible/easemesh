/*
 * Copyright (c) 2021, MegaEase
 * All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package generator

import (
	"fmt"
	"go/ast"
	"strings"

	"github.com/dave/jennifer/jen"
	"github.com/pkg/errors"
)

type (
	meshClientVisitor struct {
		err          error
		getterID     string
		resourceType ResourceType
		builder      interfaceMethodBuilder
	}

	converter struct {
		err      error
		funcType *ast.FuncType
		funcName string
		imports  []*ast.ImportSpec
	}
)

// NewVisitor create an InterfaceVisitor object to generate code while traveling the interface.
func NewVisitor(resourceType ResourceType) InterfaceVisitor {
	return &meshClientVisitor{resourceType: resourceType, builder: &interfaceBuilder{}}
}

func (m *meshClientVisitor) visitorBegin(imports []*ast.ImportSpec, spec *InterfaceFileSpec) error {
	spec.Buf.PackageComment(`Copyright (c) 2021, MegaEase
All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.`)
	spec.Buf.Line()
	spec.Buf.PackageComment("code generated by github.com/megaease/easemeshctl/cmd/generator, DO NOT EDIT.")
	spec.Buf.Line()
	spec.Buf.Line()
	return nil
}

func (m *meshClientVisitor) visitorResourceGetterConcreatStruct(name string, spec *InterfaceFileSpec) error {
	spec.Buf.Type().Id(name).Struct(jen.Id("client").Qual("", "*meshClient"))
	m.getterID = name
	return nil
}

func (m *meshClientVisitor) visitorInterfaceConcreatStruct(name string, spec *InterfaceFileSpec) error {
	spec.Buf.Type().Id(name).Struct(jen.Id("client").Qual("", "*meshClient"))
	return nil
}

func (m *meshClientVisitor) visitorResourceGetterMethod(name string, method *ast.Field, imports []*ast.ImportSpec, spec *InterfaceFileSpec) error {
	var arguments, results []jen.Code
	err := covertFuncType(method, imports).extractArguments(&arguments).extractResults(&results).error()
	if err != nil {
		return errors.Wrapf(err, "extract arguments and result from method error")
	}

	spec.Buf.Func().Params(
		jen.Id(string(m.getterID[0])).Op("*").Id(m.getterID),
	).Id(name).Params(arguments...).Params(results...).BlockFunc(func(grp *jen.Group) {
		structName := strings.ToLower(name[0:1]) + name[1:] + "Interface"
		grp.Return(jen.Op("&").Id(structName).Values(jen.Dict{
			jen.Id("client"): jen.Id(strings.ToLower(name[0:1])).Dot("client"),
		}))
	})
	return nil
}

func (m *meshClientVisitor) visitorIntrefaceMethod(concreateStruct string, verb Verb, method *ast.Field, imports []*ast.ImportSpec, spec *InterfaceFileSpec) error {
	var err error

	info := &buildInfo{
		interfaceStructName: concreateStruct,
		method:              method,
		imports:             imports,
		buf:                 spec.Buf,
		resourceType:        m.resourceType,
		subResource:         spec.SubResource,
		resource2UrlMapping: spec.ResourceMapping,
	}

	switch verb {
	case Get:
		err = m.builder.buildGetMethod(info)
	case Patch:
		err = m.builder.buildPatchMethod(info)
	case Delete:
		err = m.builder.buildDeleteMethod(info)
	case List:
		err = m.builder.buildListMethod(info)
	case Create:
		err = m.builder.buildCreateMethod(info)
	}
	if err != nil {
		return errors.Wrapf(err, "build %s interface method error", verb)
	}
	return nil
}

func (m *meshClientVisitor) visitorEnd(spec *InterfaceFileSpec) error {
	return nil
}
func (m *meshClientVisitor) onError(e error) { m.err = e }

func covertFuncType(method *ast.Field, imports []*ast.ImportSpec) *converter {
	var err error
	funcType, ok := method.Type.(*ast.FuncType)
	if !ok {
		err = errors.Errorf("method should contain a functype")
		return &converter{}
	}

	if len(method.Names) == 0 {
		err = errors.Errorf("func name is reqired")
	}

	return &converter{funcType: funcType, err: err, funcName: method.Names[0].Name, imports: imports}
}

func (c *converter) funcIdens(fields []*ast.Field, withID bool) (codes []jen.Code) {
	for i, r := range fields {
		var statement *jen.Statement
		if len(r.Names) != 0 {
			statement = jen.Id(r.Names[0].Name)
		} else if withID {
			statement = jen.Id(fmt.Sprintf("args%d", i))
		}
		parsed := false
		for !parsed {
			switch r.Type.(type) {
			case *ast.Ident:
				iden := r.Type.(*ast.Ident)
				if statement == nil {
					statement = jen.Id(iden.Name)
				} else {
					statement.Id(iden.Name)
				}
				parsed = true
			case *ast.SelectorExpr:
				selectExpr := r.Type.(*ast.SelectorExpr)
				x := selectExpr.X.(*ast.Ident)
				path := c.fromImports(x.Name)
				if statement == nil {
					statement = jen.Qual(path, selectExpr.Sel.Name)
				} else {
					statement.Qual(path, selectExpr.Sel.Name)
				}
				parsed = true
			case *ast.StarExpr:
				starExpr := r.Type.(*ast.StarExpr)
				r.Type = starExpr.X
				if statement == nil {
					statement = jen.Op("*")
				} else {
					statement = statement.Op("*")
				}
			case *ast.ArrayType:
				arryExpr := r.Type.(*ast.ArrayType)
				r.Type = arryExpr.Elt
				if statement == nil {
					statement = jen.Op("[]")
				} else {
					statement = statement.Op("[]")
				}
			}
		}
		codes = append(codes, statement)
	}
	return
}

func (c *converter) fromImports(name string) (path string) {
	for _, im := range c.imports {
		if im.Path.Value == "" {
			continue
		}
		path = im.Path.Value[1 : len(im.Path.Value)-1]
		if im.Name != nil && im.Name.Name == name {
			return path
		}
		pathes := strings.Split(path, "/")
		if pathes[len(pathes)-1] == name {
			return
		}
	}

	return ""
}

func (c *converter) extractArguments(codes *[]jen.Code) *converter {
	if c.err == nil {
		*codes = append(*codes, c.funcIdens(c.funcType.Params.List, true)...)
	}
	return c
}

func (c *converter) extractResults(codes *[]jen.Code) *converter {
	if c.err == nil {
		*codes = append(*codes, c.funcIdens(c.funcType.Results.List, false)...)
	}
	return c
}

func (c *converter) extractFuncName(name *string) *converter {
	if c.err == nil {
		*name = c.funcName
	}
	return c
}

func (c *converter) error() error { return c.err }
